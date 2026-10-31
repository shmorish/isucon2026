package isureal

import (
	"context"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
)

type getMemoriesRequest struct {
	Month          string     `query:"month"`
	BeforePostedAt *time.Time `query:"beforePostedAt"`
	BeforeID       string     `query:"beforeId"`
}

type memoryPostResponse struct {
	createPostResponse
	RealMojis []replaceRealMojiResponse `json:"realmojis"`
	Comments  []createCommentResponse   `json:"comments"`
}

type getMemoriesResponse struct {
	Items   []memoryPostResponse `json:"items"`
	HasMore bool                 `json:"hasMore"`
}

type getMemoriesByDateRequest struct {
	Date           string     `param:"date"`
	BeforePostedAt *time.Time `query:"beforePostedAt"`
	BeforeID       string     `query:"beforeId"`
}

type getMemoriesByDateResponse getMemoriesResponse

func getMemoriesHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var input getMemoriesRequest
		var start time.Time
		err := c.Bind(&input)
		if err == nil {
			start, err = time.Parse("2006-01", input.Month)
			if err != nil {
				err = NewAPIError(http.StatusBadRequest, "invalid_memory_period")
			}
		}
		if err == nil {
			err = validateMemoryPagination(input.BeforePostedAt, input.BeforeID)
		}
		if err := foldedSessionBefore(c, h, err); err != nil {
			return err
		}
		response, err := listMemories(c, h, start, start.AddDate(0, 1, 0), input.BeforePostedAt, input.BeforeID)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, response)
	}
}

func getMemoriesByDateHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var input getMemoriesByDateRequest
		var start time.Time
		err := c.Bind(&input)
		if err == nil {
			start, err = time.Parse(time.DateOnly, input.Date)
			if err != nil {
				err = NewAPIError(http.StatusBadRequest, "invalid_memory_period")
			}
		}
		if err == nil {
			err = validateMemoryPagination(input.BeforePostedAt, input.BeforeID)
		}
		if err := foldedSessionBefore(c, h, err); err != nil {
			return err
		}
		response, err := listMemories(c, h, start, start.AddDate(0, 0, 1), input.BeforePostedAt, input.BeforeID)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, getMemoriesByDateResponse(response))
	}
}

func validateMemoryPagination(beforeAt *time.Time, beforeID string) error {
	if (beforeAt == nil) != (beforeID == "") {
		return NewAPIError(http.StatusBadRequest, "invalid_pagination")
	}
	return nil
}

// listMemories looks the session up in the same statement as the posts (FoldedSessionMiddleware), so the posts cost no
// extra round trip. With no posts the statement returns no row either way, so only then the session is checked alone.
func listMemories(c *echo.Context, h *HandlerContext, start, end time.Time, beforeAt *time.Time, beforeID string) (getMemoriesResponse, error) {
	var response getMemoriesResponse
	ctx := c.Request().Context()
	var rows []struct {
		sessionColumns
		PostWithAuthor
	}
	if err := h.DB.SelectContext(ctx, &rows, `
  WITH `+sessionCTE+`
  SELECT `+sessionSelect+`, `+postColumns+`, `+userColumnsAs("u", "author")+`
  FROM sess
  JOIN posts p ON p.author_id = sess.user_id
  JOIN posting_windows w ON w.id = p.posting_window_id
  JOIN users u ON u.id = p.author_id AND u.status = 'active'
  WHERE p.deleted_at IS NULL AND w.local_date >= $2 AND w.local_date < $3
  AND ($4::timestamptz IS NULL OR (p.posted_at, p.id) < ($4, $5))
  ORDER BY p.posted_at DESC, p.id DESC LIMIT 51
 `, foldedSessionTokenHash(c), start, end, beforeAt, beforeID); err != nil {
		return response, err
	}
	if len(rows) == 0 {
		if err := requireFoldedSession(c, h); err != nil {
			return response, err
		}
	} else if err := acceptFoldedSession(c, h, true, rows[0].sessionColumns); err != nil {
		return response, err
	}
	response.HasMore = len(rows) > 50
	if response.HasMore {
		rows = rows[:50]
	}
	posts := make([]PostWithAuthor, 0, len(rows))
	for _, row := range rows {
		if err := row.Author.setImageKey(); err != nil {
			return response, err
		}
		posts = append(posts, row.PostWithAuthor)
	}
	items, err := loadMemoryItems(ctx, h.DB, posts)
	if err != nil {
		return response, err
	}
	response.Items = items
	return response, nil
}

func loadMemoryItems(ctx context.Context, db sqlx.QueryerContext, posts []PostWithAuthor) ([]memoryPostResponse, error) {
	items := make([]memoryPostResponse, 0, len(posts))
	if len(posts) == 0 {
		return items, nil
	}
	postIDs := make([]string, len(posts))
	for i, post := range posts {
		postIDs[i] = post.ID
	}

	query, args, err := sqlx.In(`AND r.post_id IN (?) ORDER BY r.created_at DESC, r.id DESC`, postIDs)
	if err != nil {
		return nil, err
	}
	reactions, err := selectCurrentRealMojis(ctx, db, sqlx.Rebind(sqlx.DOLLAR, query), args...)
	if err != nil {
		return nil, err
	}
	query, args, err = sqlx.In(`AND c.post_id IN (?) ORDER BY c.created_at, c.id`, postIDs)
	if err != nil {
		return nil, err
	}
	comments, err := selectCommentsWithAuthor(ctx, db, sqlx.Rebind(sqlx.DOLLAR, query), args...)
	if err != nil {
		return nil, err
	}

	// 署名は組み立てる前に1往復でまとめて取り寄せる（#48）
	toSign := make([]imageToSign, 0, 2*(len(posts)+len(reactions))+len(comments))
	for _, post := range posts {
		toSign = appendImagesToSign(toSign, post.Author, post.ImageID)
	}
	for _, reaction := range reactions {
		toSign = appendImagesToSign(toSign, reaction.Reactor, reaction.ImageID)
	}
	for _, comment := range comments {
		toSign = appendImagesToSign(toSign, comment.Author)
	}
	prefetchImageSignatures(ctx, toSign)

	itemByPostID := make(map[string]*memoryPostResponse, len(posts))
	for _, post := range posts {
		items = append(items, memoryPostResponse{
			createPostResponse: newPostResponse(post.Post, post.Author),
			RealMojis:          make([]replaceRealMojiResponse, 0),
			Comments:           make([]createCommentResponse, 0),
		})
		itemByPostID[post.ID] = &items[len(items)-1]
	}
	for _, reaction := range reactions {
		item := itemByPostID[reaction.PostID]
		if len(item.RealMojis) < 50 {
			item.RealMojis = append(item.RealMojis, newRealMojiResponse(reaction.RealMoji, reaction.Reactor))
		}
	}
	for _, comment := range comments {
		item := itemByPostID[comment.PostID]
		if len(item.Comments) < 50 {
			item.Comments = append(item.Comments, newCommentResponse(comment.Comment, comment.Author))
		}
	}
	return items, nil
}
