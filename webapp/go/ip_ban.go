package isureal

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
)

const ipBanReloadInterval = 200 * time.Millisecond

type IPBanList struct {
	bans atomic.Pointer[map[string]time.Time]
}

type ipBan struct {
	IP          string    `db:"ip"`
	BannedUntil time.Time `db:"banned_until"`
}

// 200ms ごとに裏で ip_bans を読み直す。読めていない間・失敗した間は直前の一覧（初回は BAN なし）を使う
func (l *IPBanList) run(db *sqlx.DB, logger *slog.Logger) {
	ticker := time.NewTicker(ipBanReloadInterval)
	defer ticker.Stop()
	for ; ; <-ticker.C {
		bans, err := loadIPBans(context.Background(), db)
		if err != nil {
			logger.Error("failed to load ip bans", "error", err)
			continue
		}
		l.bans.Store(&bans)
	}
}

func (l *IPBanList) Banned(ip string) bool {
	bans := l.bans.Load()
	if bans == nil {
		return false
	}
	until, ok := (*bans)[ip]
	return ok && time.Now().Before(until)
}

func loadIPBans(ctx context.Context, db *sqlx.DB) (map[string]time.Time, error) {
	var rows []ipBan
	if err := db.SelectContext(ctx, &rows, `SELECT ip, banned_until FROM ip_bans WHERE banned_until > clock_timestamp()`); err != nil {
		return nil, err
	}
	bans := make(map[string]time.Time, len(rows))
	for _, row := range rows {
		bans[row.IP] = row.BannedUntil
	}
	return bans, nil
}

func extractClientIP(req *http.Request) string {
	if ip := req.Header.Get("X-Client-IP"); ip != "" {
		return ip
	}
	return echo.ExtractIPDirect()(req)
}

func IPBanMiddleware(h *HandlerContext) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if h.IPBans.Banned(c.RealIP()) {
				return NewAPIError(http.StatusForbidden, "ip_banned")
			}
			return next(c)
		}
	}
}
