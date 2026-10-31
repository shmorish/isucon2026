package isureal

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
)

type Session struct {
	Region    string    `db:"region" json:"-"`
	Token     string    `db:"-" json:"-"`
	UserID    string    `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	CreatedAt time.Time `db:"created_at"`
}

func GetSession(c *echo.Context) *Session {
	sess, ok := c.Get("session").(*Session)
	if !ok {
		return nil
	}
	return sess
}

// sessionTokenHashFromHeader returns "" when there is no Authorization header.
func sessionTokenHashFromHeader(c *echo.Context) (string, error) {
	authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
	if authHeader == "" {
		return "", nil
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.Logger().Warn("Invalid authorization header format(not started with Bearer)")
		return "", NewAPIError(http.StatusUnauthorized, "invalid_session")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	tokenHashBytes := sha256.Sum256([]byte(token))
	return hex.EncodeToString(tokenHashBytes[:]), nil
}

func getSessionFromHeader(c *echo.Context, db *sqlx.DB) (*Session, error) {
	tokenHash, err := sessionTokenHashFromHeader(c)
	if tokenHash == "" {
		return nil, err
	}
	var session Session
	err = db.GetContext(c.Request().Context(), &session, `
		SELECT s.user_id, s.token_hash, s.created_at, u.region
		FROM login_sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.active = true AND u.status = 'active'
	`, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &session, nil
}

func SessionMiddleware(handlerContext *HandlerContext, authRequired bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			session, err := getSessionFromHeader(c, handlerContext.DB)
			if err != nil {
				return err
			}
			if session == nil && authRequired {
				return NewAPIError(http.StatusUnauthorized, "invalid_session")
			}
			c.Set("session", session)
			return next(c)
		}
	}
}

// FoldedSessionMiddleware is SessionMiddleware(h, true) + RegionMiddleware(h) for handlers that look the session up
// in their first query (sessionCTE + sessionSelect, token hash as $1) and then call acceptFoldedSession.
func FoldedSessionMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			tokenHash, err := sessionTokenHashFromHeader(c)
			if err != nil {
				return err
			}
			if tokenHash == "" {
				return NewAPIError(http.StatusUnauthorized, "invalid_session")
			}
			c.Set("session_token_hash", tokenHash)
			return next(c)
		}
	}
}

func foldedSessionTokenHash(c *echo.Context) string {
	tokenHash, _ := c.Get("session_token_hash").(string)
	return tokenHash
}

// LIMIT 1: the separate lookup took the first matching row the same way.
const sessionCTE = `sess AS (
		SELECT s.user_id, s.token_hash, s.created_at, u.region
		FROM login_sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.active = true AND u.status = 'active'
		LIMIT 1
	)`

const sessionSelect = `sess.user_id AS sess_user_id, sess.token_hash AS sess_token_hash, sess.created_at AS sess_created_at, sess.region AS sess_region`

type sessionColumns struct {
	SessUserID    string    `db:"sess_user_id"`
	SessTokenHash string    `db:"sess_token_hash"`
	SessCreatedAt time.Time `db:"sess_created_at"`
	SessRegion    string    `db:"sess_region"`
}

// acceptFoldedSession gives the responses SessionMiddleware and RegionMiddleware gave. found is false when the
// folded query returned no row (no valid session).
func acceptFoldedSession(c *echo.Context, h *HandlerContext, found bool, cols sessionColumns) error {
	if !found {
		return NewAPIError(http.StatusUnauthorized, "invalid_session")
	}
	c.Set("session", &Session{UserID: cols.SessUserID, TokenHash: cols.SessTokenHash, CreatedAt: cols.SessCreatedAt, Region: cols.SessRegion})
	if cols.SessRegion != h.Config.Region {
		return NewAPIError(http.StatusForbidden, "region_mismatch")
	}
	return nil
}

// requireFoldedSession runs the standalone session lookup, for handlers that return (e.g. a 400) before their
// folded query: 401/403 must still win over those errors.
func requireFoldedSession(c *echo.Context, h *HandlerContext) error {
	var cols sessionColumns
	err := h.DB.GetContext(c.Request().Context(), &cols, `WITH `+sessionCTE+` SELECT `+sessionSelect+` FROM sess`, foldedSessionTokenHash(c))
	if errors.Is(err, sql.ErrNoRows) {
		return acceptFoldedSession(c, h, false, cols)
	}
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	return acceptFoldedSession(c, h, true, cols)
}

func InvalidateLoginSession(db *sqlx.DB, userID string, tokenHash string) error {
	// One autocommit statement instead of BEGIN/lock/UPDATE/COMMIT. The main query reads locked, and the UPDATE CTE it does not
	// reference only runs after the main query has finished, so the user row is locked before any session row, as before.
	// The statement's snapshot predates the lock wait, which only matters for rows created meanwhile: the target session
	// already exists, and a concurrent change to it is rechecked against the latest row version.
	var lockedUserID string
	if err := db.Get(&lockedUserID, `
		WITH locked AS (SELECT id FROM users WHERE id = $1 FOR NO KEY UPDATE),
		logout AS (
			UPDATE login_sessions
			SET active = false, invalidated_at = clock_timestamp(), reason = 'logout'
			WHERE user_id = (SELECT id FROM locked) AND active = true AND token_hash = $2
		)
		SELECT id FROM locked
	`, userID, tokenHash); err != nil {
		return fmt.Errorf("failed to lock user: %w", err)
	}

	return nil
}

func createLoginSessionTx(ctx context.Context, tx *sqlx.Tx, userID string) (*Session, error) {
	randomToken := uuid.NewString()
	tokenHashBytes := sha256.Sum256([]byte(randomToken))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])
	// users の行は呼び出し側（loginHandler）がロック済み。古いセッションの無効化と新しいセッションの作成を1往復で行う
	var session Session
	if err := tx.GetContext(ctx, &session, `
  WITH invalidated AS (
   UPDATE login_sessions
   SET active = false, invalidated_at = clock_timestamp(), reason = 'replaced'
   WHERE user_id = $2 AND active = true
  )
  INSERT INTO login_sessions (id, user_id, token_hash)
  VALUES ($1, $2, $3)
  RETURNING user_id, token_hash, created_at
 `, uuid.NewString(), userID, tokenHash); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	session.Token = randomToken
	return &session, nil
}
