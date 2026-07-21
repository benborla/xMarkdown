package render

import (
	"strings"
	"testing"

	"github.com/benborla/xMarkdown/doc"
)

func TestRenderSmoke(t *testing.T) {
	lines, err := Render([]byte("# Hello\n\nSome *styled* text here.\n"), 80)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 2 {
		t.Fatalf("expected multiple lines, got %d", len(lines))
	}
	joined := doc.StripANSI(strings.Join(lines, "\n"))
	if !strings.Contains(joined, "Hello") {
		t.Errorf("rendered output missing heading text:\n%s", joined)
	}
	if !strings.Contains(joined, "styled") {
		t.Errorf("rendered output missing body text:\n%s", joined)
	}
}

func TestRenderEmpty(t *testing.T) {
	lines, err := Render([]byte(""), 80)
	if err != nil {
		t.Fatal(err)
	}
	if lines == nil {
		t.Fatal("expected non-nil lines for empty input")
	}
}
