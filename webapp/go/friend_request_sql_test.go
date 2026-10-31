package isureal

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Runs createFriendRequestQuery and transitionFriendRequestQuery through the cases the handlers used to check in
// separate statements (empty schema, ISUREAL_TEST_DSN). Rolled back at the end.
func TestFriendRequestQueries(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	tx := beginTestTx(t, db)
	defer tx.Rollback()

	u := map[string]string{}
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		u[name] = "00000000-0000-0000-0000-00000000000" + string(rune('1'+len(u)))
		if _, err := tx.ExecContext(ctx, `INSERT INTO users (id, username, secret_key_hash, display_name, region) VALUES ($1, $1, '', '', 'asia')`, u[name]); err != nil {
			t.Fatal(err)
		}
	}
	block := func(x, y string) {
		if _, err := tx.ExecContext(ctx, `INSERT INTO block_relations (id, blocker_id, blocked_id) VALUES ($1, $2, $3)`, uuid.NewString(), u[x], u[y]); err != nil {
			t.Fatal(err)
		}
	}
	create := func(actor, recipient string) (FriendRequest, error) {
		var r FriendRequest
		err := tx.GetContext(ctx, &r, createFriendRequestQuery, u[actor], u[recipient],
			min(u[actor], u[recipient]), max(u[actor], u[recipient]), uuid.NewString(), uuid.NewString())
		return r, err
	}
	friends := func(x, y string) bool {
		var ok bool
		if err := tx.GetContext(ctx, &ok, `SELECT EXISTS(SELECT 1 FROM friendships WHERE user_low_id=$1 AND user_high_id=$2)`, min(u[x], u[y]), max(u[x], u[y])); err != nil {
			t.Fatal(err)
		}
		return ok
	}
	type transition struct {
		Current string     `db:"current_status"`
		Blocked bool       `db:"blocked"`
		Status  *string    `db:"status"`
		Updated *time.Time `db:"updated_at"`
	}
	transit := func(r FriendRequest, status string) transition {
		var res transition
		if err := tx.GetContext(ctx, &res, transitionFriendRequestQuery,
			r.ID, r.RequesterID, r.RecipientID, status, uuid.NewString(), min(r.RequesterID, r.RecipientID), max(r.RequesterID, r.RecipientID)); err != nil {
			t.Fatal(err)
		}
		return res
	}
	conflict := func(name string, err error) {
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("%s: want no row (409), got %v", name, err)
		}
	}

	ab, err := create("a", "b")
	if err != nil || ab.Status != "pending" || ab.RequesterID != u["a"] {
		t.Fatalf("a->b: %+v %v", ab, err)
	}
	_, err = create("a", "b")
	conflict("a->b again", err)
	ba, err := create("b", "a")
	if err != nil || ba.ID != ab.ID || ba.Status != "accepted" || !friends("a", "b") {
		t.Fatalf("b->a accepts a's request: %+v %v", ba, err)
	}
	_, err = create("a", "b")
	conflict("a->b friends", err)
	block("c", "a")
	_, err = create("a", "c")
	conflict("a->c blocked", err)
	_, err = create("c", "a")
	conflict("c->a blocked", err)

	ad, _ := create("a", "d")
	if res := transit(ad, "accepted"); res.Status == nil || *res.Status != "accepted" || !friends("a", "d") {
		t.Fatalf("d accepts: %+v", res)
	}
	if res := transit(ad, "accepted"); res.Current != "accepted" || res.Status != nil {
		t.Fatalf("accept twice must be 403 without update: %+v", res)
	}

	ae, _ := create("a", "e")
	block("e", "a")
	if res := transit(ae, "accepted"); res.Current != "pending" || !res.Blocked || res.Status != nil || friends("a", "e") {
		t.Fatalf("blocked accept must be 409 without update: %+v", res)
	}
	if res := transit(ae, "rejected"); res.Status == nil || *res.Status != "rejected" || friends("a", "e") {
		t.Fatalf("blocked reject: %+v", res)
	}

	cd, _ := create("c", "d")
	if res := transit(cd, "cancelled"); res.Status == nil || *res.Status != "cancelled" || friends("c", "d") {
		t.Fatalf("cancel: %+v", res)
	}
}
