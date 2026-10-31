package isureal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

type PostLocation struct {
	Kind      string   `json:"kind" db:"location_kind"`
	Latitude  *float64 `json:"latitude,omitempty" db:"latitude"`
	Longitude *float64 `json:"longitude,omitempty" db:"longitude"`
	Locality  *string  `json:"locality,omitempty" db:"locality"`
}

func (location PostLocation) valid() bool {
	switch location.Kind {
	case "none":
		return location.Latitude == nil && location.Longitude == nil && location.Locality == nil
	case "exact":
		return location.Latitude != nil && location.Longitude != nil && location.Locality == nil &&
			*location.Latitude >= -90 && *location.Latitude <= 90 && *location.Longitude >= -180 && *location.Longitude <= 180
	case "approximate":
		return location.Latitude == nil && location.Longitude == nil && location.Locality != nil && validUserText(*location.Locality, 1, 100)
	default:
		return false
	}
}

type Post struct {
	ID              string `db:"id"`
	AuthorID        string `db:"author_id"`
	PostingWindowID string `db:"posting_window_id"`
	ImageID         string `db:"image_id"`
	Caption         string `db:"caption"`
	Visibility      string `db:"visibility"`
	PostLocation
	PostedAt       time.Time  `db:"posted_at"`
	OnTimeDeadline time.Time  `db:"on_time_deadline"`
	WindowEndedAt  *time.Time `db:"window_ended_at"`
}

const postColumns = `p.id, p.author_id, p.posting_window_id, p.image_id, p.caption, p.visibility,
  p.location_kind, p.latitude, p.longitude, p.locality, p.posted_at,
  w.on_time_deadline, w.ended_at AS window_ended_at`

func getPost(ctx context.Context, db sqlx.QueryerContext, postID string) (Post, error) {
	var post Post
	err := sqlx.GetContext(ctx, db, &post, `
  SELECT `+postColumns+`
  FROM posts p JOIN posting_windows w ON w.id = p.posting_window_id
  WHERE p.id = $1 AND p.deleted_at IS NULL
 `, postID)
	return post, err
}

type PostWithAuthor struct {
	Post
	Author User `db:"author"`
}

func selectPostsWithAuthor(ctx context.Context, db sqlx.QueryerContext, conditions string, args ...any) ([]PostWithAuthor, error) {
	var posts []PostWithAuthor
	if err := sqlx.SelectContext(ctx, db, &posts, `
  SELECT `+postColumns+`, `+userColumnsAs("u", "author")+`
  FROM posts p
  JOIN posting_windows w ON w.id = p.posting_window_id
  JOIN users u ON u.id = p.author_id AND u.status = 'active'
  WHERE p.deleted_at IS NULL `+conditions, args...); err != nil {
		return nil, err
	}
	for i := range posts {
		if err := posts[i].Author.setImageKey(); err != nil {
			return nil, err
		}
	}
	return posts, nil
}

func getPostWithAuthor(ctx context.Context, db sqlx.QueryerContext, postID string) (PostWithAuthor, error) {
	posts, err := selectPostsWithAuthor(ctx, db, `AND p.id = $1`, postID)
	if err != nil {
		return PostWithAuthor{}, err
	}
	if len(posts) == 0 {
		return PostWithAuthor{}, sql.ErrNoRows
	}
	return posts[0], nil
}

type viewerContext struct {
	User
	HasPostedToday   bool
	Friends          map[string]bool
	FriendsOfFriends map[string]bool
	MutualFriends    map[string]int
	Blocked          map[string]bool
}

type jsonStrings []string

func (values *jsonStrings) Scan(src any) error {
	return scanJSON(src, (*[]string)(values))
}

type jsonStringPairs [][2]string

func (values *jsonStringPairs) Scan(src any) error {
	return scanJSON(src, (*[][2]string)(values))
}

func scanJSON(src any, dest any) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, dest)
	case string:
		return json.Unmarshal([]byte(v), dest)
	default:
		return fmt.Errorf("unsupported json column type %T", src)
	}
}

var viewerContextQuery = viewerContextSQL("", "$1", "", "")

// viewerContextBySessionQuery is viewerContextQuery for the viewer of the session whose token hash is $1.
// sess and me use the same active-user conditions, so a valid session always has its me row.
var viewerContextBySessionQuery = viewerContextSQL(sessionCTE+", ", "(SELECT user_id FROM sess)", sessionSelect+", ", "sess, ")

// "Posted today" means a post in any window sharing the local_date, regardless of region. The same rule is spelled out separately (not shared, to keep each query plan) in viewerContextSQL, activeWindowWithPostCountsQuery, getMyCurrentPostHandler, getCurrentPostingWindowHandler and deletePostHandler; change them together.
func viewerContextSQL(with, viewerID, extraColumns, extraFrom string) string {
	return `
  WITH ` + with + `me AS (
    SELECT ` + strings.Join(userColumns, ", ") + ` FROM users WHERE id = ` + viewerID + ` AND status = 'active'
  ), friend_ids AS (
    SELECT CASE WHEN user_low_id = ` + viewerID + ` THEN user_high_id ELSE user_low_id END AS id
    FROM friendships WHERE user_low_id = ` + viewerID + ` OR user_high_id = ` + viewerID + `
  )
  SELECT ` + extraColumns + `me.*,
    EXISTS(
      SELECT 1 FROM posts p
      JOIN posting_windows posted ON posted.id = p.posting_window_id
      JOIN posting_windows current ON current.region = me.region AND current.ended_at IS NULL
                                  AND current.local_date = posted.local_date
      WHERE p.author_id = me.id AND p.deleted_at IS NULL
    ) AS has_posted_today,
    COALESCE((
      SELECT json_agg(CASE WHEN blocker_id = ` + viewerID + ` THEN blocked_id ELSE blocker_id END)
      FROM block_relations WHERE blocker_id = ` + viewerID + ` OR blocked_id = ` + viewerID + `
    ), '[]'::json) AS blocked_ids,
    COALESCE((SELECT json_agg(id) FROM friend_ids), '[]'::json) AS friend_ids,
    COALESCE((
      SELECT json_agg(json_build_array(user_low_id, user_high_id)) FROM friendships
      WHERE user_low_id = ANY(ARRAY(SELECT id FROM friend_ids)) OR user_high_id = ANY(ARRAY(SELECT id FROM friend_ids))
    ), '[]'::json) AS friend_friendships
  FROM ` + extraFrom + `me
`
}

type viewerContextRow struct {
	User
	HasPostedToday    bool            `db:"has_posted_today"`
	BlockedIDs        jsonStrings     `db:"blocked_ids"`
	FriendIDs         jsonStrings     `db:"friend_ids"`
	FriendFriendships jsonStringPairs `db:"friend_friendships"`
}

func loadViewerContext(ctx context.Context, db sqlx.QueryerContext, viewerID string) (viewerContext, error) {
	var row viewerContextRow
	if err := sqlx.GetContext(ctx, db, &row, viewerContextQuery, viewerID); err != nil {
		return viewerContext{}, err
	}
	return newViewerContext(row)
}

type viewerContextBySessionRow struct {
	sessionColumns
	viewerContextRow
}

// selectViewerContextBySession returns sql.ErrNoRows when the session is not valid. The caller checks the session
// (acceptFoldedSession) before newViewerContext, whose errors must not win over 403 region_mismatch.
func selectViewerContextBySession(ctx context.Context, db sqlx.QueryerContext, tokenHash string) (viewerContextBySessionRow, error) {
	var row viewerContextBySessionRow
	err := sqlx.GetContext(ctx, db, &row, viewerContextBySessionQuery, tokenHash)
	return row, err
}

func newViewerContext(row viewerContextRow) (viewerContext, error) {
	viewer := viewerContext{Friends: map[string]bool{}, FriendsOfFriends: map[string]bool{}, MutualFriends: map[string]int{}}
	viewer.User = row.User
	if err := viewer.User.setImageKey(); err != nil {
		return viewer, err
	}
	viewer.HasPostedToday = row.HasPostedToday
	viewer.Blocked = make(map[string]bool, len(row.BlockedIDs))
	for _, id := range row.BlockedIDs {
		viewer.Blocked[id] = true
	}
	for _, id := range row.FriendIDs {
		viewer.Friends[id] = true
	}
	viewer.addFriendFriendships(row.FriendFriendships)
	return viewer, nil
}

func (viewer viewerContext) addFriendFriendships(friendships [][2]string) {
	for _, friendship := range friendships {
		userLowID, userHighID := friendship[0], friendship[1]
		if viewer.Friends[userLowID] {
			viewer.FriendsOfFriends[userHighID] = true
			if userHighID != viewer.ID {
				viewer.MutualFriends[userHighID]++
			}
		}
		if viewer.Friends[userHighID] {
			viewer.FriendsOfFriends[userLowID] = true
			if userLowID != viewer.ID {
				viewer.MutualFriends[userLowID]++
			}
		}
	}
}

func (viewer viewerContext) canView(post Post) bool {
	if post.AuthorID == viewer.ID {
		return true
	}
	if post.WindowEndedAt != nil || !viewer.HasPostedToday || viewer.Blocked[post.AuthorID] {
		return false
	}
	switch post.Visibility {
	case "public":
		return true
	case "friends":
		return viewer.Friends[post.AuthorID]
	case "friends_of_friends":
		return viewer.Friends[post.AuthorID] || viewer.FriendsOfFriends[post.AuthorID]
	default:
		return false
	}
}

// loadPostAndViewer reads the post and the viewer with two concurrent
// statements. found is false when either the post or the viewer is missing.
// A missing post wins over any viewer error.
func loadPostAndViewer(ctx context.Context, db *sqlx.DB, viewerID, postID string) (post PostWithAuthor, viewer viewerContext, found bool, err error) {
	var viewerErr error
	var wg sync.WaitGroup
	wg.Go(func() { viewer, viewerErr = loadViewerContext(ctx, db, viewerID) })
	post, err = getPostWithAuthor(ctx, db, postID)
	wg.Wait()
	if errors.Is(err, sql.ErrNoRows) {
		return post, viewer, false, nil
	}
	if err != nil {
		return post, viewer, false, err
	}
	if errors.Is(viewerErr, sql.ErrNoRows) {
		return post, viewer, false, nil
	}
	if viewerErr != nil {
		return post, viewer, false, viewerErr
	}
	return post, viewer, true, nil
}

func canViewPost(ctx context.Context, db *sqlx.DB, viewerID, postID string) (bool, error) {
	post, viewer, found, err := loadPostAndViewer(ctx, db, viewerID, postID)
	if err != nil {
		return false, err
	}
	return found && viewer.canView(post.Post), nil
}

func getFriendIDs(ctx context.Context, db sqlx.QueryerContext, userID string) ([]string, error) {
	var friendships []struct {
		UserLowID  string `db:"user_low_id"`
		UserHighID string `db:"user_high_id"`
	}
	if err := sqlx.SelectContext(ctx, db, &friendships, `SELECT user_low_id, user_high_id FROM friendships WHERE user_low_id = $1 OR user_high_id = $1`, userID); err != nil {
		return nil, err
	}
	var ids []string
	for _, friendship := range friendships {
		if friendship.UserLowID == userID {
			ids = append(ids, friendship.UserHighID)
		} else {
			ids = append(ids, friendship.UserLowID)
		}
	}
	return ids, nil
}

// Every place that row-locks users. Changing a mode or the order here changes
// what these exclude each other from; check all of them together.
//   - lockUser, lockUserRow: own row, FOR NO KEY UPDATE
//   - post create (createPostLockSQL in post_api.go): own row, FOR NO KEY UPDATE, joined to the session lookup
//   - account deletion (account_deletion_api.go): own row, FOR NO KEY UPDATE, status='active'
//   - profile image set/delete (profile_image_api.go): own row, FOR NO KEY UPDATE, reads profile_image_id
//   - login (user_api.go): row by username, FOR NO KEY UPDATE
//   - logout (session.go): own row, FOR NO KEY UPDATE inside a CTE
//   - lockUserPair (relationship.go): actor FOR NO KEY UPDATE, target FOR SHARE, status='active', smaller id first
//   - comment create (comment_api.go): both rows FOR SHARE, smaller id first, no status condition
//
// They differ in columns read, conditions and errors, so they are not merged.
func lockUser(ctx context.Context, tx *sqlx.Tx, userID string) (User, error) {
	var user User
	err := tx.GetContext(ctx, &user, `SELECT `+strings.Join(userColumns, ", ")+` FROM users WHERE id = $1 FOR NO KEY UPDATE`, userID)
	return user, err
}

const lockUserRowSQL = `SELECT id FROM users WHERE id = $1 FOR NO KEY UPDATE`

// lockUserRow is lockUser without reading the row; a missing row is sql.ErrNoRows.
func lockUserRow(ctx context.Context, tx *sqlx.Tx, userID string) error {
	var lockedID string
	return tx.GetContext(ctx, &lockedID, lockUserRowSQL, userID)
}

// requireActive is not folded into lockUser's WHERE so handlers that read the user late keep their error precedence.
func (user *User) requireActive() error {
	if user.Status != "active" {
		return sql.ErrNoRows
	}
	return user.setImageKey()
}

// Not merged into the users lock statement: the counts need a snapshot taken after that lock is granted.
// "Posted today" means a post in any window sharing the local_date, regardless of region. The same rule is spelled out separately (not shared, to keep each query plan) in viewerContextSQL, activeWindowWithPostCountsQuery, getMyCurrentPostHandler, getCurrentPostingWindowHandler and deletePostHandler; change them together.
const activeWindowWithPostCountsQuery = `
  WITH w AS (
    SELECT id, local_date, starts_at FROM posting_windows WHERE region = $2 AND ended_at IS NULL FOR SHARE
  )
  SELECT w.id, w.local_date, w.starts_at, c.active, c.deleted
  FROM w CROSS JOIN LATERAL (
    SELECT count(*) FILTER (WHERE p.deleted_at IS NULL) AS active,
           count(*) FILTER (WHERE p.deleted_at IS NOT NULL) AS deleted
    FROM posts p JOIN posting_windows pw ON pw.id = p.posting_window_id
    WHERE p.author_id = $1 AND pw.local_date = w.local_date
  ) c
`
