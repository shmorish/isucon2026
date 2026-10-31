package isureal

import (
	"reflect"
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
