package isureal

import (
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestNotificationHubDispatch(t *testing.T) {
	hub := newNotificationHub(nil, nil)
	connectedAt := time.Date(2026, 10, 31, 1, 0, 0, 0, time.UTC)
	notified := hub.subscribe("user-a", "asia", connectedAt)
	moved := hub.subscribe("user-b", "asia", connectedAt)
	waiting := hub.subscribe("user-c", "asia", connectedAt)
	subs := []*notificationSubscriber{notified, moved, waiting}
	createdAt := connectedAt.Add(time.Second)

	hub.dispatch(subs, []notificationPollRow{
		{
			Index: 1, Valid: true,
			ID:              sql.NullString{String: "n-1", Valid: true},
			PostingWindowID: sql.NullString{String: "w-1", Valid: true},
			Region:          sql.NullString{String: "asia", Valid: true},
			CreatedAt:       sql.NullTime{Time: createdAt, Valid: true},
		},
		{Index: 2, Valid: false},
	})

	got := <-notified.result
	want := getNotificationsResponse{ID: "n-1", PostingWindowID: "w-1", Region: "asia", CreatedAt: createdAt}
	if got == nil || *got != want {
		t.Fatalf("notified subscriber got %+v, want %+v", got, want)
	}
	if got := <-moved.result; got != nil {
		t.Fatalf("invalid subscriber got %+v, want close", got)
	}
	select {
	case got := <-waiting.result:
		t.Fatalf("waiting subscriber got %+v, want nothing", got)
	default:
	}
	remaining := hub.snapshot()
	if len(remaining) != 1 || remaining[0] != waiting {
		t.Fatalf("remaining subscribers = %v, want only the waiting one", remaining)
	}
}

func TestNotificationPollArgsEncodeAsArrays(t *testing.T) {
	m := pgtype.NewMap()
	if _, err := m.Encode(pgtype.TextArrayOID, pgtype.BinaryFormatCode, []string{"a", "b"}, nil); err != nil {
		t.Fatalf("encode text[]: %v", err)
	}
	if _, err := m.Encode(pgtype.TimestamptzArrayOID, pgtype.BinaryFormatCode, []time.Time{time.Now()}, nil); err != nil {
		t.Fatalf("encode timestamptz[]: %v", err)
	}
}
