package isureal

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// The row that references an image (post, realmoji, user) carries the image's created_at, so GET /api/images finds the
// image with imageCreatedAtLookupQuery + byCreatedAt (one partition) and never needs the id-only query.
func TestImageCreatedAtMatchesReferrer(t *testing.T) {
	db := openExecModeTestDB(t)
	ctx := context.Background()
	tx := beginTestTx(t, db)
	region := "test-created-" + uuid.NewString()[:8]

	authorID, reactorID := uuid.NewString(), uuid.NewString()
	for _, id := range []string{authorID, reactorID} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO users (id, username, secret_key_hash, display_name, region) VALUES ($1, $1, 'k', 'd', $2)`, id, region); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO posting_windows (id, region, local_date, starts_at, on_time_deadline) VALUES ($1, $2, '2099-01-01', $3, $4)`,
		uuid.NewString(), region, now.Add(-time.Minute), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	lat, lon := 35.681236, 139.767125
	postImageID := uuid.NewString()
	var post Post
	if err := tx.GetContext(ctx, &post, createPostQuery, binaryImage, authorID, region, postImageID, testImage(1000), time.Now().UTC(), uuid.NewString(),
		"caption", "public", "exact", &lat, &lon, (*string)(nil)); err != nil {
		t.Fatal(err)
	}
	requireFoundByReferrer(t, tx, "post image", postImageID, post.PostedAt)

	realMojiImageID, sent := uuid.NewString(), time.Now().UTC()
	if _, err := tx.ExecContext(ctx, replaceRealMojiImageQuery, binaryImage, post.ID, reactorID, realMojiImageID, testImage(1000), sent); err != nil {
		t.Fatal(err)
	}
	var reaction RealMoji
	if err := tx.GetContext(ctx, &reaction, createRealMojiQuery, uuid.NewString(), post.ID, reactorID, realMojiImageID, "like", sent); err != nil {
		t.Fatal(err)
	}
	requireMicrosecondsOf(t, "realmoji created_at", reaction.CreatedAt, sent)
	requireFoundByReferrer(t, tx, "realmoji image", realMojiImageID, reaction.CreatedAt)

	// Replacing the realmoji hides the old image at once and the new one is found the same way.
	replacementImageID, sent := uuid.NewString(), time.Now().UTC()
	if _, err := tx.ExecContext(ctx, replaceRealMojiImageQuery, binaryImage, post.ID, reactorID, replacementImageID, testImage(1000), sent); err != nil {
		t.Fatal(err)
	}
	if err := tx.GetContext(ctx, &reaction, createRealMojiQuery, uuid.NewString(), post.ID, reactorID, replacementImageID, "smile", sent); err != nil {
		t.Fatal(err)
	}
	requireFoundByReferrer(t, tx, "replacement realmoji image", replacementImageID, reaction.CreatedAt)
	var replaced testImageMeta
	if err := tx.GetContext(ctx, &replaced, imageMetaQueryMetaOnly.byID, binaryImage, realMojiImageID); err != nil {
		t.Fatal(err)
	}
	if replaced.Visible {
		t.Fatal("replaced realmoji image is still visible")
	}

	for i, size := range []int{1000, 2000} {
		var old profileImageRef
		if err := tx.GetContext(ctx, &old, lockProfileImageQuery, authorID); err != nil {
			t.Fatal(err)
		}
		if (old.ImageID != nil) != (i > 0) {
			t.Fatalf("profile image before upload %d: %v", i, old.ImageID)
		}
		profileImageID, sent := uuid.NewString(), time.Now().UTC()
		var updatedAt time.Time
		if err := tx.GetContext(ctx, &updatedAt, putProfileImageQuery, binaryImage, authorID, profileImageID, testImage(size), sent); err != nil {
			t.Fatal(err)
		}
		requireMicrosecondsOf(t, "users.updated_at", updatedAt, sent)
		if err := deleteImageObject(ctx, tx, old); err != nil {
			t.Fatal(err)
		}
		requireFoundByReferrer(t, tx, "profile image", profileImageID, updatedAt)
		if old.ImageID != nil {
			var n int
			if err := tx.GetContext(ctx, &n, `SELECT count(*) FROM image_objects WHERE id = $1`, *old.ImageID); err != nil {
				t.Fatal(err)
			}
			if n != 0 {
				t.Fatal("replaced profile image was not deleted")
			}
		}
	}

	// A profile image whose user row was touched later (updated_at moved) is still deleted, by id.
	var current profileImageRef
	if err := tx.GetContext(ctx, &current, lockProfileImageQuery, authorID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET updated_at=clock_timestamp() WHERE id=$1`, authorID); err != nil {
		t.Fatal(err)
	}
	var touched profileImageRef
	if err := tx.GetContext(ctx, &touched, lockProfileImageQuery, authorID); err != nil {
		t.Fatal(err)
	}
	if touched.UpdatedAt.Equal(current.UpdatedAt) {
		t.Fatal("updated_at did not move")
	}
	if err := deleteImageObject(ctx, tx, touched); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := tx.GetContext(ctx, &n, `SELECT count(*) FROM image_objects WHERE id = $1`, *touched.ImageID); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("profile image with a moved updated_at was not deleted")
	}
}

// The DB stores microseconds; the value read back must be the sent time truncated to that, as the response shows it.
func requireMicrosecondsOf(t *testing.T, name string, stored, sent time.Time) {
	t.Helper()
	if stored.Nanosecond()%1000 != 0 {
		t.Fatalf("%s %v is not whole microseconds", name, stored)
	}
	if d := sent.Sub(stored); d < 0 || d >= time.Microsecond {
		t.Fatalf("%s %v, sent %v", name, stored, sent)
	}
}

func requireFoundByReferrer(t *testing.T, tx *sqlx.Tx, name, imageID string, want time.Time) {
	t.Helper()
	ctx := context.Background()
	var referenced sql.NullTime
	if err := tx.GetContext(ctx, &referenced, imageCreatedAtLookupQuery, imageID); err != nil {
		t.Fatal(err)
	}
	if !referenced.Valid || !referenced.Time.Equal(want) {
		t.Fatalf("%s: referrer time %v, want %v", name, referenced, want)
	}
	for q, meta := range map[string]imageMetaQuery{"meta only": imageMetaQueryMetaOnly, "with data": imageMetaQueryWithData} {
		var found testImageMeta
		err := tx.GetContext(ctx, &found, meta.byCreatedAt, binaryImage, imageID, referenced.Time)
		if errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("%s (%s): not found by created_at %v", name, q, referenced.Time)
		}
		if err != nil {
			t.Fatal(err)
		}
		if !found.Visible || !found.CreatedAt.Equal(want) {
			t.Fatalf("%s (%s): visible %v created_at %v", name, q, found.Visible, found.CreatedAt)
		}
	}
}
