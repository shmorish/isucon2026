package isureal

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

const notificationPollInterval = 200 * time.Millisecond

type notificationSubscriber struct {
	userID      string
	region      string
	connectedAt time.Time
	result      chan *getNotificationsResponse
}

type notificationHub struct {
	db          *sqlx.DB
	logger      *slog.Logger
	mu          sync.Mutex
	subscribers map[*notificationSubscriber]struct{}
	// kick asks run to poll now instead of waiting for the next tick (window open). Buffered 1: kicks during a poll
	// collapse into one more poll right after it
	kick chan struct{}
}

type notificationPollRow struct {
	Index           int64          `db:"idx"`
	Valid           bool           `db:"valid"`
	ID              sql.NullString `db:"id"`
	PostingWindowID sql.NullString `db:"posting_window_id"`
	Region          sql.NullString `db:"region"`
	CreatedAt       sql.NullTime   `db:"created_at"`
}

// A notification is the region's active window that started after the subscriber connected, for a user who is still
// active in that region (#82). The window opener used to insert one row per active user with created_at = starts_at;
// only those rows of the active window, created after connecting, were ever delivered. The id is fixed per window and
// user so a reconnect sees the same one.
// ponytail: a user who leaves the region and comes back between connecting and the window opening gets a notification
// the row INSERT would not have made; the poll runs every 200ms, so that gap is tiny
const notificationPollQuery = `
SELECT s.idx, u.id IS NOT NULL AS valid, x.id, x.posting_window_id, x.region, x.created_at
FROM unnest($1::text[], $2::text[], $3::timestamptz[]) WITH ORDINALITY AS s(user_id, region, connected_at, idx)
LEFT JOIN users u ON u.id = s.user_id AND u.status = 'active' AND u.region = s.region
LEFT JOIN LATERAL (
  SELECT md5(w.id || ':' || s.user_id)::uuid::text AS id, w.id AS posting_window_id, w.region, w.starts_at AS created_at
  FROM posting_windows w
  WHERE w.region = s.region AND w.ended_at IS NULL AND w.starts_at > s.connected_at
  LIMIT 1
) x ON true
WHERE u.id IS NULL OR x.id IS NOT NULL
`

func newNotificationHub(db *sqlx.DB, logger *slog.Logger) *notificationHub {
	return &notificationHub{db: db, logger: logger, subscribers: map[*notificationSubscriber]struct{}{}, kick: make(chan struct{}, 1)}
}

func (hub *notificationHub) subscribe(userID, region string, connectedAt time.Time) *notificationSubscriber {
	sub := &notificationSubscriber{
		userID: userID, region: region, connectedAt: connectedAt,
		result: make(chan *getNotificationsResponse, 1),
	}
	hub.mu.Lock()
	hub.subscribers[sub] = struct{}{}
	hub.mu.Unlock()
	return sub
}

func (hub *notificationHub) unsubscribe(sub *notificationSubscriber) {
	hub.mu.Lock()
	delete(hub.subscribers, sub)
	hub.mu.Unlock()
}

func (hub *notificationHub) snapshot() []*notificationSubscriber {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	subs := make([]*notificationSubscriber, 0, len(hub.subscribers))
	for sub := range hub.subscribers {
		subs = append(subs, sub)
	}
	return subs
}

func (hub *notificationHub) run() {
	ticker := time.NewTicker(notificationPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-hub.kick:
		}
		if err := hub.poll(); err != nil {
			hub.logger.Error("failed to poll notifications", "error", err)
		}
	}
}

// pollSoon makes run poll right away. The poll is the same query as on a tick, only earlier, so what is delivered
// does not change: a just-committed window reaches its subscribers without waiting up to notificationPollInterval
func (hub *notificationHub) pollSoon() {
	select {
	case hub.kick <- struct{}{}:
	default:
	}
}

func (hub *notificationHub) poll() error {
	subs := hub.snapshot()
	if len(subs) == 0 {
		return nil
	}
	userIDs := make([]string, len(subs))
	regions := make([]string, len(subs))
	connectedAts := make([]time.Time, len(subs))
	for i, sub := range subs {
		userIDs[i] = sub.userID
		regions[i] = sub.region
		connectedAts[i] = sub.connectedAt
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var rows []notificationPollRow
	if err := hub.db.SelectContext(ctx, &rows, notificationPollQuery, userIDs, regions, connectedAts); err != nil {
		return err
	}
	hub.dispatch(subs, rows)
	return nil
}

func (hub *notificationHub) dispatch(subs []*notificationSubscriber, rows []notificationPollRow) {
	for _, row := range rows {
		if row.Index < 1 || row.Index > int64(len(subs)) {
			continue
		}
		sub := subs[row.Index-1]
		var result *getNotificationsResponse
		if row.Valid {
			result = &getNotificationsResponse{
				ID: row.ID.String, PostingWindowID: row.PostingWindowID.String,
				Region: row.Region.String, CreatedAt: row.CreatedAt.Time,
			}
		}
		hub.unsubscribe(sub)
		select {
		case sub.result <- result:
		default:
		}
	}
}
