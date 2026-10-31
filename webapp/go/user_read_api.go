package isureal

import (
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/labstack/echo/v5"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

type searchUsersRequest struct {
	Q              string `query:"q"`
	BeforeUsername string `query:"beforeUsername"`
	BeforeID       string `query:"beforeId"`
}

type searchUsersResponse struct {
	Items   []relationshipUserResponse `json:"items"`
	HasMore bool                       `json:"hasMore"`
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func normalizeSearchText(value string) string {
	return norm.NFC.String(cases.Fold().String(norm.NFC.String(value)))
}

func searchUsersHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var input searchUsersRequest
		if err := c.Bind(&input); err != nil {
			return err
		}
		if !validUserText(input.Q, 1, 100) {
			return NewAPIError(http.StatusBadRequest, "invalid_relationship")
		}
		if (input.BeforeUsername == "") != (input.BeforeID == "") {
			return NewAPIError(http.StatusBadRequest, "invalid_pagination")
		}
		ctx := c.Request().Context()
		viewer := GetSession(c).UserID
		query := normalizeSearchText(input.Q)
		// username は [a-z0-9_.-] だけなので正規化しても変わらない。display_name は書き込み時に正規化した display_name_search と比べる
		// 候補を s1 のメモリで絞り（userSearchIndex）、最終の判定と並び順は今までどおり DB で行う
		var users []User
		args := []any{viewer, input.BeforeUsername, input.BeforeID, "%" + likeEscaper.Replace(query) + "%"}
		candidateFilter := ""
		if ids, ok := userSearch.candidates(query); ok {
			if len(ids) == 0 {
				return c.JSON(http.StatusOK, searchUsersResponse{Items: []relationshipUserResponse{}, HasMore: false})
			}
			candidateFilter = " AND id = ANY($5)"
			args = append(args, ids)
		}
		if err := h.DB.SelectContext(ctx, &users, `SELECT `+strings.Join(userColumns, ", ")+` FROM users u
			WHERE status='active' AND id<>$1 AND ($2='' OR (username,id)>($2,$3))
			  AND (username LIKE $4 OR display_name_search LIKE $4)`+candidateFilter+`
			  AND NOT EXISTS (SELECT 1 FROM block_relations b WHERE (b.blocker_id=$1 AND b.blocked_id=u.id) OR (b.blocker_id=u.id AND b.blocked_id=$1))
			ORDER BY username, id LIMIT 51`, args...); err != nil {
			return err
		}
		hasMore := len(users) > 50
		if hasMore {
			users = users[:50]
		}
		for i := range users {
			if err := users[i].setImageKey(); err != nil {
				return err
			}
			if users[i].AccountType != "certified_public" {
				users[i].Bio = ""
			}
		}
		prefetchImageSignatures(ctx, imagesToSignForUsers(users))
		items := make([]relationshipUserResponse, 0, len(users))
		for _, user := range users {
			items = append(items, newRelationshipUserResponse(user))
		}
		return c.JSON(http.StatusOK, searchUsersResponse{Items: items, HasMore: hasMore})
	}
}

type getUserRequest struct {
	UserID string `param:"userId"`
}

type getUserResponse relationshipUserResponse

func getUserHandler(h *HandlerContext) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var input getUserRequest
		if err := c.Bind(&input); err != nil {
			return err
		}
		ctx := c.Request().Context()
		viewer := GetSession(c).UserID
		user, err := getUser(ctx, h.DB, input.UserID)
		if errors.Is(err, sql.ErrNoRows) {
			return NewAPIError(http.StatusNotFound, "relationship_not_found")
		}
		if err != nil {
			return err
		}
		blocked, err := isBlocked(ctx, h.DB, viewer, user.ID)
		if err != nil {
			return err
		}
		if blocked {
			return NewAPIError(http.StatusNotFound, "relationship_not_found")
		}
		friends, err := getFriendIDs(ctx, h.DB, viewer)
		if err != nil {
			return err
		}
		if user.AccountType != "certified_public" && !slices.Contains(friends, user.ID) {
			user.Bio = ""
		}
		prefetchImageSignatures(ctx, appendImagesToSign(nil, user))
		return c.JSON(http.StatusOK, getUserResponse(newRelationshipUserResponse(user)))
	}
}

func newRelationshipUserResponse(user User) relationshipUserResponse {
	return relationshipUserResponse{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, ProfileImageURL: newGetMeResponse(user).ProfileImageURL, Bio: user.Bio, AccountType: user.AccountType}
}
