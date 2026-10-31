package isureal

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The light viewer context plus the per-author friends-of-friends checks (postAndViewerQuery,
// loadFriendsOfFriends) must answer canView and canComment exactly as the context loaded with every friend
// friendship, for friends, friends of friends (edges stored in both orders), strangers, blocked users, deleted
// authors, ended windows and the viewer's own posts (empty schema, ISUREAL_TEST_DSN). Rolled back at the end.
func TestLightViewerContextMatchesFriendFriendships(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	tx := beginTestTx(t, db)
	defer tx.Rollback()

	key := strings.Repeat("ab", 32)
	u := map[string]string{}
	for _, name := range []string{"v", "f1", "f2", "ff1", "ff2", "ffblocked", "ffblocker", "stranger", "fdeleted", "ffdeleted"} {
		u[name] = uuid.NewString()
		status := "active"
		if strings.HasSuffix(name, "deleted") {
			status = "deleted"
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO users (id, username, secret_key_hash, display_name, region, status) VALUES ($1, $1, $2, '', 'asia', $3)`, u[name], key, status); err != nil {
			t.Fatal(err)
		}
	}
	// Edges as stored: user_low_id/user_high_id in both orders, so neither side may be assumed.
	friendship := func(low, high string) {
		if _, err := tx.ExecContext(ctx, `INSERT INTO friendships (id, user_low_id, user_high_id) VALUES ($1, $2, $3)`, uuid.NewString(), u[low], u[high]); err != nil {
			t.Fatal(err)
		}
	}
	friendship("v", "f1")
	friendship("f2", "v")
	friendship("f1", "ff1")
	friendship("ff2", "f2")
	friendship("ff2", "f1")
	friendship("f1", "ffblocked")
	friendship("ffblocker", "f2")
	friendship("v", "fdeleted")
	friendship("fdeleted", "ffdeleted")
	friendship("stranger", "ffdeleted")
	for _, pair := range [][2]string{{"v", "ffblocked"}, {"ffblocker", "v"}} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO block_relations (id, blocker_id, blocked_id) VALUES ($1, $2, $3)`, uuid.NewString(), u[pair[0]], u[pair[1]]); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	openWindow, endedWindow := uuid.NewString(), uuid.NewString()
	if _, err := tx.ExecContext(ctx, `INSERT INTO posting_windows (id, region, local_date, starts_at, on_time_deadline, ended_at) VALUES
		($1, 'asia', $3::date, $4, $4, NULL), ($2, 'asia', $3::date - 1, $4, $4, $4)`, openWindow, endedWindow, now, now); err != nil {
		t.Fatal(err)
	}
	type fixture struct {
		author, visibility, window string
		deleted                    bool
	}
	posts := map[string]fixture{}
	addPost := func(f fixture) {
		id := uuid.NewString()
		var deletedAt *time.Time
		if f.deleted {
			deletedAt = &now
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO posts (id, author_id, posting_window_id, image_id, visibility, location_kind, posted_at, deleted_at) VALUES ($1, $2, $3, $4, $5, 'none', $6, $7)`,
			id, u[f.author], f.window, uuid.NewString(), f.visibility, now, deletedAt); err != nil {
			t.Fatal(err)
		}
		posts[id] = f
	}
	addPost(fixture{author: "v", visibility: "friends", window: openWindow})
	for name := range u {
		for _, visibility := range []string{"public", "friends", "friends_of_friends", "other"} {
			addPost(fixture{author: name, visibility: visibility, window: openWindow})
		}
		addPost(fixture{author: name, visibility: "public", window: endedWindow})
		addPost(fixture{author: name, visibility: "public", window: openWindow, deleted: true})
	}
	tokenHash := "token-" + uuid.NewString()
	if _, err := tx.ExecContext(ctx, `INSERT INTO login_sessions (id, user_id, token_hash) VALUES ($1, $2, $3)`, uuid.NewString(), u["v"], tokenHash); err != nil {
		t.Fatal(err)
	}

	// Reference: the context with every friend friendship, through the session query the ranked feed uses.
	heavyRow, err := selectViewerContextBySession(ctx, tx, tokenHash, true)
	if err != nil {
		t.Fatal(err)
	}
	heavy, err := newViewerContext(heavyRow.viewerContextRow)
	if err != nil {
		t.Fatal(err)
	}
	// f1, f2 and fdeleted are friends but share no edge with another friend, so they are not friends of friends.
	wantFoF := map[string]bool{"v": true, "ff1": true, "ff2": true, "ffblocked": true, "ffblocker": true, "ffdeleted": true}
	for name, id := range u {
		if heavy.FriendsOfFriends[id] != wantFoF[name] {
			t.Errorf("heavy FriendsOfFriends[%s] = %v", name, heavy.FriendsOfFriends[id])
		}
	}
	if !heavy.HasPostedToday || !heavy.Friends[u["f1"]] || !heavy.Friends[u["f2"]] || !heavy.Blocked[u["ffblocked"]] || !heavy.Blocked[u["ffblocker"]] {
		t.Fatalf("heavy context: posted=%v friends=%v blocked=%v", heavy.HasPostedToday, heavy.Friends, heavy.Blocked)
	}

	// Light session context plus loadFriendsOfFriends, as the following feed does.
	lightRow, err := selectViewerContextBySession(ctx, tx, tokenHash, false)
	if err != nil {
		t.Fatal(err)
	}
	if lightRow.sessionColumns != heavyRow.sessionColumns || lightRow.HasPostedToday != heavyRow.HasPostedToday ||
		strings.Join(lightRow.FriendIDs, ",") != strings.Join(heavyRow.FriendIDs, ",") || strings.Join(lightRow.BlockedIDs, ",") != strings.Join(heavyRow.BlockedIDs, ",") {
		t.Fatalf("light row differs: %+v vs %+v", lightRow, heavyRow)
	}
	light, err := newViewerContext(lightRow.viewerContextRow)
	if err != nil {
		t.Fatal(err)
	}
	if len(light.FriendsOfFriends) != 0 || len(light.MutualFriends) != 0 {
		t.Fatalf("light context must start without friends of friends: %v", light.FriendsOfFriends)
	}
	var all []PostWithAuthor
	for id, f := range posts {
		all = append(all, PostWithAuthor{Post: Post{ID: id, AuthorID: u[f.author], Visibility: f.visibility}})
	}
	toCheck := friendsOfFriendsToCheck(light, all)
	for _, id := range toCheck {
		if id == u["v"] || id == u["f1"] || id == u["f2"] {
			t.Errorf("friendsOfFriendsToCheck includes a friend or the viewer: %s", id)
		}
	}
	if err := light.loadFriendsOfFriends(ctx, tx, toCheck); err != nil {
		t.Fatal(err)
	}
	for _, id := range toCheck {
		if light.FriendsOfFriends[id] != heavy.FriendsOfFriends[id] {
			t.Errorf("loadFriendsOfFriends[%s] = %v, heavy %v", id, light.FriendsOfFriends[id], heavy.FriendsOfFriends[id])
		}
	}
	if err := light.loadFriendsOfFriends(ctx, tx, nil); err != nil {
		t.Fatal(err)
	}

	// Single-statement post + viewer, as post detail, comments and RealMoji do.
	var visible, checked int
	for id, f := range posts {
		post, viewer, found, err := loadPostAndViewer(ctx, tx, u["v"], id)
		if err != nil {
			t.Fatal(err)
		}
		authorActive := !strings.HasSuffix(f.author, "deleted")
		if found != (authorActive && !f.deleted) {
			t.Errorf("post by %s (%s, deleted=%v): found=%v", f.author, f.visibility, f.deleted, found)
			continue
		}
		if !found {
			continue
		}
		checked++
		if post.ID != id || post.AuthorID != u[f.author] || post.Author.ID != u[f.author] || viewer.ID != u["v"] || len(post.Author.ImageKey) == 0 || len(viewer.ImageKey) == 0 {
			t.Errorf("post %s: wrong post or viewer row: %+v", id, post)
		}
		if viewer.FriendsOfFriends[post.AuthorID] != heavy.FriendsOfFriends[post.AuthorID] {
			t.Errorf("post by %s: author_friend_of_friend=%v, heavy %v", f.author, viewer.FriendsOfFriends[post.AuthorID], heavy.FriendsOfFriends[post.AuthorID])
		}
		if viewer.HasPostedToday != heavy.HasPostedToday || len(viewer.Friends) != len(heavy.Friends) || len(viewer.Blocked) != len(heavy.Blocked) {
			t.Errorf("post by %s: light viewer differs from heavy", f.author)
		}
		if got, want := viewer.canView(post.Post), heavy.canView(post.Post); got != want {
			t.Errorf("post by %s (%s, window ended=%v): canView=%v, heavy %v", f.author, f.visibility, post.WindowEndedAt != nil, got, want)
		}
		if got, want := viewer.canComment(post.Post), heavy.canComment(post.Post); got != want {
			t.Errorf("post by %s (%s): canComment=%v, heavy %v", f.author, f.visibility, got, want)
		}
		if got, want := light.canView(post.Post), heavy.canView(post.Post); got != want {
			t.Errorf("post by %s (%s): light+lazy canView=%v, heavy %v", f.author, f.visibility, got, want)
		}
		if viewer.canView(post.Post) {
			visible++
		}
		if allowed, err := canViewPost(ctx, tx, u["v"], id); err != nil || allowed != viewer.canView(post.Post) {
			t.Errorf("canViewPost(%s) = %v, %v", id, allowed, err)
		}
	}
	// v's own 6 posts (even "other" and the ended window), f1/f2 (public, friends, fof), ff1/ff2 (public, fof),
	// stranger (public); the blocked authors never, and the deleted authors' posts are not found.
	if checked != 1+8*5 || visible != 6+3+3+2+2+1 {
		t.Errorf("checked %d posts, %d visible", checked, visible)
	}
	if _, _, found, err := loadPostAndViewer(ctx, tx, u["fdeleted"], all[0].ID); err != nil || found {
		t.Errorf("deleted viewer: found=%v err=%v", found, err)
	}
	if _, _, found, err := loadPostAndViewer(ctx, tx, u["v"], uuid.NewString()); err != nil || found {
		t.Errorf("missing post: found=%v err=%v", found, err)
	}
}
