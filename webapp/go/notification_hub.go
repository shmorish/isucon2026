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
}

type notificationPollRow struct {
	Index           int64          `db:"idx"`
	Valid           bool           `db:"valid"`
	ID              sql.NullString `db:"id"`
	PostingWindowID sql.NullString `db:"posting_window_id"`
	Region          sql.NullString `db:"region"`
	CreatedAt       sql.NullTime   `db:"created_at"`
}

const notificationPollQuery = `
SELECT s.idx, u.id IS NOT NULL AS valid, x.id, x.posting_window_id, x.region, x.created_at
FROM unnest($1::text[], $2::text[], $3::timestamptz[]) WITH ORDINALITY AS s(user_id, region, connected_at, idx)
LEFT JOIN users u ON u.id = s.user_id AND u.status = 'active' AND u.region = s.region
LEFT JOIN LATERAL (
  SELECT n.id, n.posting_window_id, w.region, n.created_at
  FROM notifications n JOIN posting_windows w ON w.id = n.posting_window_id
  WHERE n.user_id = s.user_id AND n.created_at > s.connected_at AND w.ended_at IS NULL
  ORDER BY n.created_at, n.id LIMIT 1
) x ON true
WHERE u.id IS NULL OR x.id IS NOT NULL
`

func newNotificationHub(db *sqlx.DB, logger *slog.Logger) *notificationHub {
	return &notificationHub{db: db, logger: logger, subscribers: map[*notificationSubscriber]struct{}{}}
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
	for range ticker.C {
		if err := hub.poll(); err != nil {
			hub.logger.Error("failed to poll notifications", "error", err)
		}
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
