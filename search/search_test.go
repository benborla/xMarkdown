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

func TestHighlightPlain(t *testing.T) {
	got := Highlight("hello world", "world")
	want := "hello \x1b[7mworld\x1b[27m"
	if got != want {
		t.Fatalf("Highlight = %q, want %q", got, want)
	}
}

func TestHighlightStyledLine(t *testing.T) {
	raw := "\x1b[1mHello\x1b[0m world"
	got := Highlight(raw, "world")
	want := "\x1b[1mHello\x1b[0m \x1b[7mworld\x1b[27m"
	if got != want {
		t.Fatalf("Highlight = %q, want %q", got, want)
	}
}

func TestHighlightCaseInsensitive(t *testing.T) {
	got := Highlight("Hello World", "world")
	want := "Hello \x1b[7mWorld\x1b[27m"
	if got != want {
		t.Fatalf("Highlight = %q, want %q", got, want)
	}
}

func TestHighlightNoMatch(t *testing.T) {
	raw := "\x1b[1mplain\x1b[0m"
	if got := Highlight(raw, "zebra"); got != raw {
		t.Fatalf("Highlight = %q, want unchanged %q", got, raw)
	}
}

func TestHighlightEmptyQuery(t *testing.T) {
	if got := Highlight("abc", ""); got != "abc" {
		t.Fatalf("Highlight = %q, want unchanged", got)
	}
}
