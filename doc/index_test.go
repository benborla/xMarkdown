package doc

import "testing"

func TestStripANSI(t *testing.T) {
	got := StripANSI("\x1b[1mbold\x1b[0m plain")
	want := "bold plain"
	if got != want {
		t.Fatalf("StripANSI = %q, want %q", got, want)
	}
}

func TestExtract(t *testing.T) {
	src := []byte(`# Title

see [docs](https://example.com) and [guide](guide.md)

## Setup

visit <https://auto.link>
`)
	headings, links := extract(src)

	wantHeadings := []struct {
		level int
		text  string
	}{{1, "Title"}, {2, "Setup"}}
	if len(headings) != len(wantHeadings) {
		t.Fatalf("got %d headings, want %d: %+v", len(headings), len(wantHeadings), headings)
	}
	for i, w := range wantHeadings {
		if headings[i].Level != w.level || headings[i].Text != w.text {
			t.Errorf("heading %d = %+v, want level=%d text=%q", i, headings[i], w.level, w.text)
		}
	}

	wantLinks := []struct{ text, url string }{
		{"docs", "https://example.com"},
		{"guide", "guide.md"},
		{"https://auto.link", "https://auto.link"},
	}
	if len(links) != len(wantLinks) {
		t.Fatalf("got %d links, want %d: %+v", len(links), len(wantLinks), links)
	}
	for i, w := range wantLinks {
		if links[i].Text != w.text || links[i].URL != w.url {
			t.Errorf("link %d = %+v, want text=%q url=%q", i, links[i], w.text, w.url)
		}
	}
}
