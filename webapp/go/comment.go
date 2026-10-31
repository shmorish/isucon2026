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
	post, viewer, found, err := loadPostAndViewer(ctx, db, viewerID, postID)
	if err != nil {
		return false, err
	}
	return found && viewer.canComment(post.Post), nil
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
