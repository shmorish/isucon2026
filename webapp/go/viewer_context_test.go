package isureal

import (
	"reflect"
	"slices"
	"testing"
)

func TestJSONColumnsScanBytesAndString(t *testing.T) {
	var ids jsonStrings
	if err := ids.Scan([]byte(`["a","b"]`)); err != nil || !reflect.DeepEqual([]string(ids), []string{"a", "b"}) {
		t.Fatalf("bytes: %v %v", ids, err)
	}
	var empty jsonStrings
	if err := empty.Scan(`[]`); err != nil || len(empty) != 0 {
		t.Fatalf("string: %v %v", empty, err)
	}
	var pairs jsonStringPairs
	if err := pairs.Scan([]byte(`[["l","h"]]`)); err != nil || !reflect.DeepEqual([][2]string(pairs), [][2]string{{"l", "h"}}) {
		t.Fatalf("pairs: %v %v", pairs, err)
	}
	if err := ids.Scan(1); err == nil {
		t.Fatal("non-json source must fail")
	}
}

func TestAddFriendFriendshipsCountsMutualFriendsExceptViewer(t *testing.T) {
	viewer := viewerContext{User: User{ID: "v"}, Friends: map[string]bool{"a": true, "b": true}, FriendsOfFriends: map[string]bool{}, MutualFriends: map[string]int{}}
	viewer.addFriendFriendships([][2]string{{"a", "v"}, {"b", "v"}, {"a", "b"}, {"a", "x"}, {"b", "x"}, {"c", "d"}})
	wantFoF := map[string]bool{"v": true, "a": true, "b": true, "x": true}
	wantMutual := map[string]int{"a": 1, "b": 1, "x": 2}
	if !reflect.DeepEqual(viewer.FriendsOfFriends, wantFoF) || !reflect.DeepEqual(viewer.MutualFriends, wantMutual) {
		t.Fatalf("fof=%v mutual=%v", viewer.FriendsOfFriends, viewer.MutualFriends)
	}
}

func TestFeedCandidateFilterKeepsEveryVisibleFeedPost(t *testing.T) {
	viewer := viewerContext{
		User:             User{ID: "v"},
		HasPostedToday:   true,
		Friends:          map[string]bool{"f1": true, "f2": true},
		FriendsOfFriends: map[string]bool{"v": true, "f1": true, "ff1": true, "ff2": true},
		MutualFriends:    map[string]int{},
		Blocked:          map[string]bool{"ff2": true},
	}
	following := map[string]bool{"c1": true, "f2": true}
	authors := []string{"f1", "f2", "ff1", "ff2", "c1", "x"}
	visibilities := []string{"public", "friends", "friends_of_friends", "other"}
	for _, feed := range []string{"friends", "friends-of-friends", "public", "following"} {
		filter, args := feedCandidateFilter(viewer, following, feed)
		for _, author := range authors {
			for _, visibility := range visibilities {
				post := Post{AuthorID: author, Visibility: visibility}
				if !viewer.canView(post) || !matchesFeed(viewer, following, post, feed) {
					continue
				}
				var kept bool
				switch filter {
				case `AND p.visibility = 'public'`:
					kept = visibility == "public"
				case `AND p.author_id = ANY($4)`:
					kept = slices.Contains(args[0].([]string), author)
				default:
					t.Fatalf("unexpected filter %q", filter)
				}
				if !kept {
					t.Errorf("feed %s drops visible post by %s (%s)", feed, author, visibility)
				}
			}
		}
	}
}
