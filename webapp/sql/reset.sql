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
-- 画像の created_at をプロフィール画像から逆引きする（imageCreatedAtLookupQuery）
CREATE INDEX IF NOT EXISTS users_profile_image_id ON users(profile_image_id);
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
-- Memories/export load the active RealMojis of up to ~50 posts at once; a post's rows are scattered over the heap, so each
-- row cost a random heap read (16 posts: 8.5 of 9.1 ms on disk, #92). Covering every selected column makes it an
-- index-only scan over the post's 1-2 leaf pages. Comments get no such index: body is up to 1000 runes (~3 KB),
-- over the 2704-byte btree tuple limit, so a long comment's INSERT could fail.
CREATE INDEX IF NOT EXISTS realmojis_post_active_cover ON realmojis (post_id)
  INCLUDE (id, reactor_id, image_id, reaction_type, created_at, replaced_at, deleted_at)
  WHERE replaced_at IS NULL AND deleted_at IS NULL;
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
-- 上で空にした範囲パーティションのうち、ベンチと追試で書く月 (境界の日〜翌月) だけを id のハッシュで 16 に分けて作り直す
-- (TOAST と拡張ロックも 16 になる)。8 では、全リージョンの投稿を s1 の 60 接続から入れるようになって、1つの TOAST に
-- 10 前後の INSERT が並ぶ瞬間が出た (#50)。それより後の月 (202612〜202706) は空のまま使わないので消し、default に任せる。
-- 子テーブルが増えると、created_at でも id でも絞れないクエリ (エクスポートの unnest JOIN など) が全部の子をロックして
-- fast-path (16個) を溢れ、LWLock:LockManager を取り合う (#61)。id で引くクエリ (RealMoji・プロフィール画像) はハッシュの子を
-- 1つに絞れるので 16 にしても増えず、消す7つの分だけ減る。子テーブルは 50 → 59。
-- 分けた月の子の数が 16 でないとき、または後の月が残っているときだけ作り直す・消すので、次の initialize からは何もしない。
-- 初期データのパーティションと default には触れない。
DO $$
DECLARE part record; k int; modulus constant int := 16;
BEGIN
  FOR part IN
    SELECT c.relname, pg_get_expr(c.relpartbound, c.oid) AS bound,
      (SELECT count(*) FROM pg_inherits h WHERE h.inhparent = c.oid) AS children,
      c.relname <= 'image_objects_' || to_char((SELECT created_at FROM initial_data_boundary) AT TIME ZONE 'UTC' + interval '1 month', 'YYYYMM') AS want_hash
    FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
    WHERE i.inhparent = 'image_objects'::regclass AND c.relname <> 'image_objects_default'
      AND c.relname > 'image_objects_' || to_char((SELECT created_at FROM initial_data_boundary) AT TIME ZONE 'UTC', 'YYYYMM')
  LOOP
    CONTINUE WHEN part.want_hash AND part.children = modulus;
    EXECUTE format('ALTER TABLE image_objects DETACH PARTITION %I', part.relname);
    EXECUTE format('DROP TABLE %I', part.relname);
    CONTINUE WHEN NOT part.want_hash;
    EXECUTE format('CREATE TABLE %I PARTITION OF image_objects %s PARTITION BY HASH (id)', part.relname, part.bound);
    FOR k IN 0..modulus - 1 LOOP
      EXECUTE format('CREATE TABLE %I PARTITION OF %I FOR VALUES WITH (MODULUS %s, REMAINDER %s)', part.relname || '_h' || k, part.relname, modulus, k);
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
