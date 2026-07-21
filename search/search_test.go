package search

import (
	"reflect"
	"testing"
)

func TestFind(t *testing.T) {
	lines := []string{
		"Alpha team",
		"\x1b[1mBravo\x1b[0m alpha",
		"charlie",
	}
	got := Find(lines, "alpha")
	want := []int{0, 1} // case-insensitive, matches styled line via stripping
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Find = %v, want %v", got, want)
	}
}

func TestFindNoMatch(t *testing.T) {
	if got := Find([]string{"nothing here"}, "zebra"); len(got) != 0 {
		t.Fatalf("Find = %v, want empty", got)
	}
}
