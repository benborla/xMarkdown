package search

import (
	"reflect"
	"testing"
	"unicode/utf8"
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

// Ⱥ U+023A is 2 bytes but its lowercase ⱥ U+2C65 is 3 bytes, so indexes into
// the lowered line drift past the byte map built on the original. This used to
// panic with index out of range.
func TestHighlightExpandingCaseRuneNoPanic(t *testing.T) {
	got := Highlight("ȺȺȺ x", "x")
	want := "ȺȺȺ \x1b[7mx\x1b[27m"
	if got != want {
		t.Fatalf("Highlight = %q, want %q", got, want)
	}
}

// İ U+0130 is 2 bytes but lowercases to 1-byte i. This used to mis-highlight
// and split the İ rune mid-sequence, producing invalid UTF-8.
func TestHighlightShrinkingCaseRune(t *testing.T) {
	got := Highlight("aİbc", "ib")
	want := "a\x1b[7mİb\x1b[27mc"
	if got != want {
		t.Fatalf("Highlight = %q, want %q", got, want)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("Highlight produced invalid UTF-8: %q", got)
	}
}

func TestHighlightMultibyteQueryOnStyledLine(t *testing.T) {
	raw := "\x1b[1mȺ\x1b[0m and ⱥ tail"
	got := Highlight(raw, "ⱥ")
	want := "\x1b[1m\x1b[7mȺ\x1b[27m\x1b[0m and ⱥ tail"
	if got != want {
		t.Fatalf("Highlight = %q, want %q", got, want)
	}
}
