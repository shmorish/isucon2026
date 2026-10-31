package isureal

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

type Comment struct {
	ID        string     `db:"id"`
	PostID    string     `db:"post_id"`
	AuthorID  string     `db:"author_id"`
	Body      string     `db:"body"`
	CreatedAt time.Time  `db:"created_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

func getComment(ctx context.Context, db sqlx.QueryerContext, id string) (Comment, error) {
	var comment Comment
	err := sqlx.GetContext(ctx, db, &comment, `SELECT id,post_id,author_id,body,created_at,deleted_at FROM comments WHERE id=$1`, id)
	return comment, err
}

type CommentWithAuthor struct {
	Comment
	Author User `db:"author"`
}

func selectCommentsWithAuthor(ctx context.Context, db sqlx.QueryerContext, conditions string, args ...any) ([]CommentWithAuthor, error) {
	var comments []CommentWithAuthor
	if err := sqlx.SelectContext(ctx, db, &comments, `
  SELECT c.id, c.post_id, c.author_id, c.body, c.created_at, c.deleted_at, `+userColumnsAs("u", "author")+`
  FROM comments c JOIN users u ON u.id = c.author_id AND u.status = 'active'
  WHERE c.deleted_at IS NULL `+conditions, args...); err != nil {
		return nil, err
	}
	for i := range comments {
		if err := comments[i].Author.setImageKey(); err != nil {
			return nil, err
		}
	}
	return comments, nil
}

func canComment(ctx context.Context, db sqlx.QueryerContext, viewerID, postID string) (bool, error) {
	post, err, viewer, viewerErr := loadPostAndViewer(ctx, db, viewerID, postID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	err = viewerErr
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return viewer.canComment(post.Post), nil
}

func (viewer viewerContext) canComment(post Post) bool {
	if !viewer.canView(post) || post.WindowEndedAt != nil {
		return false
	}
	if viewer.ID == post.AuthorID {
		return viewer.HasPostedToday
	}
	if post.Visibility == "public" {
		return true
	}
	return viewer.Friends[post.AuthorID]
}

// beginTxWithPost sends BEGIN while reading the post outside the transaction.
// BEGIN takes no snapshot or lock, so this is the same as reading the post
// first. A missing post is interaction_forbidden. The returned tx, when not
// nil, must be rolled back by the caller even if err is set.
func beginTxWithPost(ctx context.Context, db *sqlx.DB, postID string) (*sqlx.Tx, Post, error) {
	var tx *sqlx.Tx
	var txErr error
	var wg sync.WaitGroup
	wg.Go(func() { tx, txErr = db.BeginTxx(ctx, nil) })
	post, err := getPost(ctx, db, postID)
	wg.Wait()
	if errors.Is(err, sql.ErrNoRows) {
		return tx, post, NewAPIError(http.StatusForbidden, "interaction_forbidden")
	}
	if err != nil {
		return tx, post, err
	}
	return tx, post, txErr
}

// loadPostAndViewerTx returns what loadPostAndViewer returns inside a
// transaction, but in one statement instead of two. found is false when
// either the post or the viewer is missing, where canViewPost and canComment
// both answer false.
func loadPostAndViewerTx(ctx context.Context, tx *sqlx.Tx, viewerID, postID string) (post PostWithAuthor, viewer viewerContext, found bool, err error) {
	var row struct {
		PostWithAuthor
		Viewer            User            `db:"viewer"`
		HasPostedToday    bool            `db:"has_posted_today"`
		BlockedIDs        jsonStrings     `db:"blocked_ids"`
		FriendIDs         jsonStrings     `db:"friend_ids"`
		FriendFriendships jsonStringPairs `db:"friend_friendships"`
	}
	err = tx.GetContext(ctx, &row, `
  WITH v AS (`+viewerContextQuery+`)
  SELECT `+postColumns+`, `+userColumnsAs("u", "author")+`, `+userColumnsAs("v", "viewer")+`,
    v.has_posted_today, v.blocked_ids, v.friend_ids, v.friend_friendships
  FROM posts p
  JOIN posting_windows w ON w.id = p.posting_window_id
  JOIN users u ON u.id = p.author_id AND u.status = 'active'
  CROSS JOIN v
  WHERE p.deleted_at IS NULL AND p.id = $2`, viewerID, postID)
	if errors.Is(err, sql.ErrNoRows) {
		return post, viewer, false, nil
	}
	if err != nil {
		return post, viewer, false, err
	}
	post = row.PostWithAuthor
	if err := post.Author.setImageKey(); err != nil {
		return post, viewer, false, err
	}
	viewer = viewerContext{User: row.Viewer, HasPostedToday: row.HasPostedToday, Friends: map[string]bool{}, FriendsOfFriends: map[string]bool{}, MutualFriends: map[string]int{}}
	if err := viewer.User.setImageKey(); err != nil {
		return post, viewer, false, err
	}
	viewer.Blocked = make(map[string]bool, len(row.BlockedIDs))
	for _, id := range row.BlockedIDs {
		viewer.Blocked[id] = true
	}
	for _, id := range row.FriendIDs {
		viewer.Friends[id] = true
	}
	viewer.addFriendFriendships(row.FriendFriendships)
	return post, viewer, true, nil
}
