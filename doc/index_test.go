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

func TestBuildPositions(t *testing.T) {
	src := []byte(`# Title

[docs](https://example.com)

## Setup

## Setup
`)
	// Hand-crafted "rendered" lines (unstyled — StripANSI is a no-op on them).
	rendered := []string{
		"",                             // 0
		"  Title",                      // 1
		"",                             // 2
		"  docs (https://example.com)", // 3
		"",                             // 4
		"  Setup",                      // 5
		"",                             // 6
		"  Setup",                      // 7
	}
	ix := Build(src, rendered)

	wantHeadingLines := []int{1, 5, 7} // duplicate "Setup" resolves in order
	if len(ix.Headings) != 3 {
		t.Fatalf("got %d headings, want 3: %+v", len(ix.Headings), ix.Headings)
	}
	for i, want := range wantHeadingLines {
		if ix.Headings[i].Line != want {
			t.Errorf("heading %d line = %d, want %d", i, ix.Headings[i].Line, want)
		}
	}

	if len(ix.Links) != 1 || ix.Links[0].Line != 3 {
		t.Fatalf("links = %+v, want one link at line 3", ix.Links)
	}
}

func TestBuildDropsUnmatched(t *testing.T) {
	src := []byte("# Ghost\n\n# Real\n")
	rendered := []string{"  Real"} // "Ghost" absent from rendered output
	ix := Build(src, rendered)
	if len(ix.Headings) != 1 || ix.Headings[0].Text != "Real" || ix.Headings[0].Line != 0 {
		t.Fatalf("headings = %+v, want only Real at line 0", ix.Headings)
	}
}

func TestBuildTwoLinksSameLine(t *testing.T) {
	src := []byte("[a](x.md) and [b](y.md)\n")
	rendered := []string{"  a (x.md) and b (y.md)"}
	ix := Build(src, rendered)
	if len(ix.Links) != 2 {
		t.Fatalf("links = %+v, want 2", ix.Links)
	}
	if ix.Links[0].Line != 0 || ix.Links[1].Line != 0 {
		t.Errorf("both links should anchor at line 0: %+v", ix.Links)
	}
}

func TestHeadingNavigation(t *testing.T) {
	ix := Index{Headings: []Heading{{Line: 1}, {Line: 5}, {Line: 9}}}
	if got := ix.NextHeading(1); got != 5 {
		t.Errorf("NextHeading(1) = %d, want 5", got)
	}
	if got := ix.NextHeading(9); got != -1 {
		t.Errorf("NextHeading(9) = %d, want -1", got)
	}
	if got := ix.PrevHeading(5); got != 1 {
		t.Errorf("PrevHeading(5) = %d, want 1", got)
	}
	if got := ix.PrevHeading(0); got != -1 {
		t.Errorf("PrevHeading(0) = %d, want -1", got)
	}
}
