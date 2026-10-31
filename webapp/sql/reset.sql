-- 検索用に正規化した表示名。値は initializeHandler が Go で埋める（Go の case fold を SQL で再現できないため）
ALTER TABLE users ADD COLUMN IF NOT EXISTS display_name_search text;

-- 画像は圧縮済み（JPEG/PNG）で pglz では縮まず、試すだけで INSERT が 11ms → 5ms になる（#35）。全パーティションに効く
ALTER TABLE image_objects ALTER COLUMN data SET STORAGE EXTERNAL;

-- 追加インデックス（init.sh は reset.sql しか流さないのでここに書く）
CREATE INDEX IF NOT EXISTS realmojis_image_id ON realmojis(image_id);
CREATE INDEX IF NOT EXISTS posts_image_id ON posts(image_id);
CREATE INDEX IF NOT EXISTS login_sessions_token_hash ON login_sessions(token_hash);
CREATE INDEX IF NOT EXISTS login_sessions_user_id_active ON login_sessions(user_id) WHERE active;
CREATE INDEX IF NOT EXISTS users_username ON users(username);
-- User search matches substrings anywhere; without these it walks every user in username order (~4900 buffer hits per query)
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS users_username_trgm ON users USING gin (username gin_trgm_ops);
CREATE INDEX IF NOT EXISTS users_display_name_search_trgm ON users USING gin (display_name_search gin_trgm_ops);
CREATE INDEX IF NOT EXISTS friendships_user_high_id ON friendships(user_high_id);
CREATE INDEX IF NOT EXISTS block_relations_blocked_id ON block_relations(blocked_id);
CREATE INDEX IF NOT EXISTS follows_followee_id ON follows(followee_id);
CREATE INDEX IF NOT EXISTS friend_requests_recipient_id_status ON friend_requests(recipient_id, status);
CREATE INDEX IF NOT EXISTS friend_requests_requester_id_status ON friend_requests(requester_id, status);
CREATE INDEX IF NOT EXISTS posts_posting_window_id_author_id ON posts(posting_window_id, author_id);
-- BRIN, not btree: created_at follows the physical order (correlation ~1), and BRIN builds with one scan and adds no work to inserts
CREATE INDEX IF NOT EXISTS notifications_created_at_brin ON notifications USING brin(created_at);
CREATE INDEX IF NOT EXISTS comments_created_at_brin ON comments USING brin(created_at);
CREATE INDEX IF NOT EXISTS realmojis_created_at_brin ON realmojis USING brin(created_at);
CREATE INDEX IF NOT EXISTS posts_created_at_brin ON posts USING brin(created_at);
-- Not led by user_id: opening a window would touch one leaf per region user, each a full-page image after a checkpoint (#39).
-- No foreign keys: the window-open INSERT cannot violate them, and each check locks and dirties a users row.
CREATE UNIQUE INDEX IF NOT EXISTS notifications_posting_window_id_user_id ON notifications(posting_window_id, user_id);
DROP INDEX IF EXISTS notifications_user_id_created_at;
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_pkey, DROP CONSTRAINT IF EXISTS notifications_user_id_posting_window_id_key,
  DROP CONSTRAINT IF EXISTS notifications_user_id_fkey, DROP CONSTRAINT IF EXISTS notifications_posting_window_id_fkey;

SET session_replication_role = replica;

SELECT coalesce(created_at::text, 'infinity') AS boundary FROM initial_data_boundary \gset

BEGIN;

TRUNCATE ip_bans, login_sessions, administrator_sessions, friendships, friend_requests, follows, block_relations;

DELETE FROM notifications WHERE created_at > :'boundary';
DELETE FROM comments WHERE created_at > :'boundary';
DELETE FROM realmojis WHERE created_at > :'boundary';
DELETE FROM posts WHERE created_at > :'boundary';
DELETE FROM posting_windows WHERE created_at > :'boundary';
DO $$
DECLARE partition_name text;
BEGIN
  FOR partition_name IN
    SELECT c.relname FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
    WHERE i.inhparent = 'image_objects'::regclass
      AND c.relname > 'image_objects_' || to_char((SELECT created_at FROM initial_data_boundary) AT TIME ZONE 'UTC', 'YYYYMM')
  LOOP
    EXECUTE format('TRUNCATE %I', partition_name);
  END LOOP;
END $$;
-- ベンチで書く画像の TOAST は1つなので、投稿の INSERT が TOAST の拡張ロック (Lock:extend) を取り合う (#50)。
-- 上で空にした範囲パーティションを、id のハッシュで 8 つに分けて作り直す (TOAST と拡張ロックも 8 つになる)。
-- 分けた後は relkind が 'p' になるので、次の initialize からは何もしない。初期データのパーティションには触れない。
DO $$
DECLARE part record; k int;
BEGIN
  FOR part IN
    SELECT c.relname, pg_get_expr(c.relpartbound, c.oid) AS bound FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
    WHERE i.inhparent = 'image_objects'::regclass AND c.relkind = 'r' AND c.relname <> 'image_objects_default'
      AND c.relname > 'image_objects_' || to_char((SELECT created_at FROM initial_data_boundary) AT TIME ZONE 'UTC', 'YYYYMM')
  LOOP
    EXECUTE format('ALTER TABLE image_objects DETACH PARTITION %I', part.relname);
    EXECUTE format('DROP TABLE %I', part.relname);
    EXECUTE format('CREATE TABLE %I PARTITION OF image_objects %s PARTITION BY HASH (id)', part.relname, part.bound);
    FOR k IN 0..7 LOOP
      EXECUTE format('CREATE TABLE %I PARTITION OF %I FOR VALUES WITH (MODULUS 8, REMAINDER %s)', part.relname || '_h' || k, part.relname, k);
    END LOOP;
    EXECUTE format('ALTER TABLE %I ALTER COLUMN data SET STORAGE EXTERNAL', part.relname);
  END LOOP;
END $$;
DELETE FROM image_objects WHERE created_at > :'boundary';
DELETE FROM users WHERE created_at > :'boundary';

UPDATE users u
SET display_name = i.display_name, bio = i.bio, profile_image_id = i.profile_image_id, region = i.region,
    account_type = i.account_type, status = i.status, deleted_at = i.deleted_at, updated_at = i.updated_at
FROM initial_users i
WHERE u.id = i.id
  AND (u.display_name, u.bio, u.profile_image_id, u.region, u.account_type, u.status, u.deleted_at, u.updated_at)
      IS DISTINCT FROM (i.display_name, i.bio, i.profile_image_id, i.region, i.account_type, i.status, i.deleted_at, i.updated_at);

INSERT INTO friend_requests (id, requester_id, recipient_id, status, created_at, updated_at)
SELECT id, requester_id, recipient_id, status, created_at, updated_at FROM initial_friend_requests;
INSERT INTO friendships (id, user_low_id, user_high_id, source_request_id, created_at, updated_at)
SELECT id, user_low_id, user_high_id, source_request_id, created_at, updated_at FROM initial_friendships;
INSERT INTO follows (id, follower_id, followee_id, created_at, updated_at)
SELECT id, follower_id, followee_id, created_at, updated_at FROM initial_follows;
INSERT INTO block_relations (id, blocker_id, blocked_id, created_at)
SELECT id, blocker_id, blocked_id, created_at FROM initial_block_relations;

COMMIT;
