package isureal

import (
	"slices"
	"testing"
)

// 候補は消さずに足すだけなので、表示名を変えた後も古い名前と新しい名前の両方で候補に残る（最終の判定は SQL）
func TestUserSearchCandidatesKeepOldAndNewTexts(t *testing.T) {
	x := &userSearchIndex{texts: map[string][]string{}}
	if _, ok := x.candidates("a"); ok {
		t.Fatal("an index that was never loaded must not narrow the search")
	}
	x.loaded = true
	x.add("u1", "alice", normalizeSearchText("Alice"))
	x.add("u1", normalizeSearchText("Wonder"))
	x.add("u2", "bob", "bob")
	for q, want := range map[string][]string{"ali": {"u1"}, "wonder": {"u1"}, "bo": {"u2"}, "zzz": nil} {
		got, ok := x.candidates(q)
		slices.Sort(got)
		if !ok || !slices.Equal(got, want) {
			t.Fatalf("candidates(%q) = %v, %v; want %v", q, got, ok, want)
		}
	}
}
