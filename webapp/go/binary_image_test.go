package isureal

import (
	"bytes"
	"context"
	"encoding/json"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func openExecModeTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("ISUREAL_TEST_DSN")
	if dsn == "" {
		t.Skip("ISUREAL_TEST_DSN is not set")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeExec
	db := sqlx.NewDb(stdlib.OpenDB(*cfg), "pgx")
	t.Cleanup(func() { db.Close() })
	return db
}

func testImage(n int) []byte {
	data := make([]byte, n)
	for i := range data {
		data[i] = byte(i)
	}
	rand.Shuffle(len(data), func(i, j int) { data[i], data[j] = data[j], data[i] })
	return data
}

func requireSameJSON(t *testing.T, name string, text, binary any) {
	t.Helper()
	a, err := json.Marshal(text)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(binary)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("%s differs:\n text:   %s\n binary: %s", name, a, b)
	}
}

type testImageMeta struct {
	SecretKeyHash string    `db:"secret_key_hash"`
	MediaType     string    `db:"media_type"`
	CreatedAt     time.Time `db:"created_at"`
	Visible       bool      `db:"visible"`
	Data          []byte    `db:"data"`
}

// Every statement passed binaryImage writes the same bytes and reads back the same values as the default text mode.
func TestBinaryImageMatchesTextMode(t *testing.T) {
	db := openExecModeTestDB(t)
	ctx := context.Background()
	tx := beginTestTx(t, db)
	region := "test-binary-" + uuid.NewString()[:8]

	userID := uuid.NewString()
	if _, err := tx.ExecContext(ctx, `INSERT INTO users (id, username, secret_key_hash, display_name, region) VALUES ($1, $1, 'k', 'd', $2)`, userID, region); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO posting_windows (id, region, local_date, starts_at, on_time_deadline) VALUES ($1, $2, '2099-01-01', $3, $4)`,
		uuid.NewString(), region, now.Add(-time.Minute), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	postImage := testImage(1_500_000)
	lat, lon := 35.681236, 139.767125
	postedAt := time.Now().UTC()
	imageID := uuid.NewString()
	var post Post
	if err := tx.GetContext(ctx, &post, createPostQuery, binaryImage, userID, region, imageID, postImage, postedAt, uuid.NewString(),
		"caption", "public", "exact", &lat, &lon, (*string)(nil)); err != nil {
		t.Fatal(err)
	}
	if d := post.PostedAt.Sub(postedAt); d < -time.Microsecond || d > time.Microsecond {
		t.Fatalf("posted_at %v, sent %v", post.PostedAt, postedAt)
	}
	textPost, err := getPost(ctx, tx, post.ID)
	if err != nil {
		t.Fatal(err)
	}
	requireSameJSON(t, "created post", textPost, post)

	var stored []byte
	if err := tx.GetContext(ctx, &stored, `SELECT data FROM image_objects WHERE id = $1`, imageID); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, postImage) {
		t.Fatal("post image bytes differ")
	}

	for name, q := range map[string]imageMetaQuery{"with data": imageMetaQueryWithData, "meta only": imageMetaQueryMetaOnly} {
		var text, binary testImageMeta
		if err := tx.GetContext(ctx, &text, q.byID, imageID); err != nil {
			t.Fatal(err)
		}
		if err := tx.GetContext(ctx, &binary, q.byID, binaryImage, imageID); err != nil {
			t.Fatal(err)
		}
		requireSameJSON(t, "image meta by id, "+name, text, binary)
		if err := tx.GetContext(ctx, &binary, q.byCreatedAt, binaryImage, imageID, binary.CreatedAt); err != nil {
			t.Fatal(err)
		}
		requireSameJSON(t, "image meta by created_at, "+name, text, binary)
		if name == "with data" && !bytes.Equal(binary.Data, postImage) {
			t.Fatal("served image bytes differ")
		}
	}

	type exportRow struct {
		ID   string `db:"id"`
		Data []byte `db:"data"`
	}
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`SELECT i.id, i.data FROM unnest($1::varchar[], $2::timestamptz[]) AS k(id, created_at) JOIN image_objects i ON i.id = k.id AND i.created_at = k.created_at`,
			[]any{[]string{imageID}, []time.Time{post.PostedAt}}},
		{`SELECT id, data FROM image_objects WHERE id = ANY($1)`, []any{[]string{imageID}}},
	} {
		var text, binary []exportRow
		if err := tx.SelectContext(ctx, &text, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
		if err := tx.SelectContext(ctx, &binary, q.sql, append([]any{binaryImage}, q.args...)...); err != nil {
			t.Fatal(err)
		}
		if len(binary) != 1 || !bytes.Equal(binary[0].Data, postImage) {
			t.Fatalf("export rows: %d", len(binary))
		}
		requireSameJSON(t, "export rows", text, binary)
	}

	profileImage, profileImageID := testImage(300_000), uuid.NewString()
	if _, err := tx.ExecContext(ctx, `INSERT INTO image_objects(id,owner_user_id,data,media_type) VALUES ($1,$2,$3,'image/jpeg')`, binaryImage, profileImageID, userID, profileImage); err != nil {
		t.Fatal(err)
	}
	realMojiImage, realMojiImageID := testImage(200_000), uuid.NewString()
	if _, err := tx.ExecContext(ctx, `
   WITH image AS (INSERT INTO image_objects(id,owner_user_id,data,media_type) VALUES($3,$2,$4,'image/jpeg'))
   UPDATE realmojis SET replaced_at=clock_timestamp() WHERE post_id=$1 AND reactor_id=$2 AND replaced_at IS NULL AND deleted_at IS NULL
  `, binaryImage, post.ID, userID, realMojiImageID, realMojiImage); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string][]byte{profileImageID: profileImage, realMojiImageID: realMojiImage} {
		if err := tx.GetContext(ctx, &stored, `SELECT data FROM image_objects WHERE id = $1`, id); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(stored, want) {
			t.Fatal("inserted image bytes differ")
		}
	}
}

// A connection that cached a statement's description keeps working after another session runs DDL like reset.sql's.
func TestBinaryImageSurvivesSchemaChange(t *testing.T) {
	db := openExecModeTestDB(t)
	ctx := context.Background()
	conn, err := db.Connx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	query := func() {
		t.Helper()
		var rows []struct {
			ID   string `db:"id"`
			Data []byte `db:"data"`
		}
		if err := conn.SelectContext(ctx, &rows, `SELECT id, data FROM image_objects WHERE id = ANY($1)`, binaryImage, []string{uuid.NewString()}); err != nil {
			t.Fatal(err)
		}
		var post Post
		err := conn.GetContext(ctx, &post, `SELECT `+postColumns+` FROM posts p JOIN posting_windows w ON w.id = p.posting_window_id WHERE p.id = $1`, binaryImage, uuid.NewString())
		if err != nil && err.Error() != "sql: no rows in result set" {
			t.Fatal(err)
		}
	}
	query()
	for _, ddl := range []string{
		`ALTER TABLE posts ADD COLUMN binary_image_test int`,
		`ALTER TABLE image_objects ALTER COLUMN data SET STORAGE EXTERNAL`,
		`TRUNCATE image_objects_202605`,
		`ALTER TABLE image_objects DETACH PARTITION image_objects_202605`,
		`DROP TABLE image_objects_202605`,
		`CREATE TABLE image_objects_202605 PARTITION OF image_objects FOR VALUES FROM ('2026-05-01 00:00:00+00') TO ('2026-06-01 00:00:00+00') PARTITION BY HASH (id)`,
		`CREATE TABLE image_objects_202605_h0 PARTITION OF image_objects_202605 FOR VALUES WITH (MODULUS 2, REMAINDER 0)`,
		`CREATE TABLE image_objects_202605_h1 PARTITION OF image_objects_202605 FOR VALUES WITH (MODULUS 2, REMAINDER 1)`,
		`ALTER TABLE posts DROP COLUMN binary_image_test`,
	} {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			t.Fatal(err)
		}
		query()
	}
}
