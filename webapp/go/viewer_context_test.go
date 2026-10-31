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

// A light viewer plus the edges addFriendsOfFriendsOf selects (author on one side, a friend on the other) must
// decide canView and canComment exactly as the full viewer does.
func TestLightViewerWithAuthorEdgesMatchesFullViewer(t *testing.T) {
	friendships := [][2]string{{"f1", "v"}, {"f2", "v"}, {"f1", "ff1"}, {"ff2", "f2"}, {"ff1", "x"}, {"b", "f1"}, {"f1", "f2"}}
	friends := map[string]bool{"f1": true, "f2": true}
	newViewer := func() viewerContext {
		return viewerContext{User: User{ID: "v"}, HasPostedToday: true, Friends: friends, FriendsOfFriends: map[string]bool{}, MutualFriends: map[string]int{}, Blocked: map[string]bool{"b": true}}
	}
	full := newViewer()
	full.addFriendFriendships(friendships)
	for _, author := range []string{"v", "f1", "f2", "ff1", "ff2", "x", "b"} {
		for _, visibility := range []string{"public", "friends", "friends_of_friends", "other"} {
			post := Post{AuthorID: author, Visibility: visibility}
			light := newViewer()
			if light.needsFriendsOfFriends(post) {
				var edges [][2]string
				for _, f := range friendships {
					if (f[0] == author && friends[f[1]]) || (f[1] == author && friends[f[0]]) {
						edges = append(edges, f)
					}
				}
				light.addFriendFriendships(edges)
			}
			if light.canView(post) != full.canView(post) || light.canComment(post) != full.canComment(post) {
				t.Errorf("post by %s (%s): light %v/%v, full %v/%v", author, visibility, light.canView(post), light.canComment(post), full.canView(post), full.canComment(post))
			}
		}
	}
}
