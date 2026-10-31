package isureal

import (
	"strings"
	"testing"
)

// The lock statements must stay byte-identical to the literals they replaced:
// changing a class number or key expression in one place breaks the exclusion
// between opening a posting window and registration/deletion/region changes.
func TestLockSQLUnchanged(t *testing.T) {
	cases := []struct{ name, got, want string }{
		{"lockRegionUsers", lockRegionUsersSQL, `SELECT pg_advisory_xact_lock(1, hashtext($1))`},
		{"lockRegionUsersShared", lockRegionUsersSharedSQL, `SELECT pg_advisory_xact_lock_shared(1, hashtext($1))`},
		{"lockAllRegionUsersShared", lockAllRegionUsersSharedSQL, `SELECT pg_advisory_xact_lock_shared(1, hashtext(r)) FROM unnest(ARRAY['america','asia','europe']) AS r`},
		{"registration", lockRegistrationSQL, `SELECT pg_advisory_xact_lock(2, hashtext($2)) FROM (SELECT pg_advisory_xact_lock_shared(1, hashtext($1)) OFFSET 0) AS region_lock`},
		{"lockUserRow", lockUserRowSQL, `SELECT id FROM users WHERE id = $1 FOR NO KEY UPDATE`},
		{"lockRegionPosts", lockRegionPostsSQL, `SELECT pg_advisory_xact_lock(3, hashtext($1))`},
		{"lockRegionPostsShared", lockRegionPostsSharedSQL, `SELECT pg_advisory_xact_lock_shared(3, hashtext($1))`},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s:\n got  %s\n want %s", c.name, c.got, c.want)
		}
	}
	if want := `pg_advisory_xact_lock_shared(3, hashtext(u.region))::text AS region_posts_lock`; !strings.Contains(createPostLockSQL, want) {
		t.Errorf("createPostLockSQL does not take %s", want)
	}
	for _, region := range []string{"america", "asia", "europe"} {
		if !validRegion(region) {
			t.Errorf("lockAllRegionUsersShared locks %q, which validRegion rejects", region)
		}
	}
}
