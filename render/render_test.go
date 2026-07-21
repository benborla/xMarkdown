package render

import (
	"strings"
	"testing"

	"github.com/benborla/xMarkdown/doc"
	"github.com/benborla/xMarkdown/theme"
)

func TestRenderSmoke(t *testing.T) {
	lines, err := Render([]byte("# Hello\n\nSome *styled* text here.\n"), 80, theme.BuiltinDark().Style)
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
	lines, err := Render([]byte(""), 80, theme.BuiltinDark().Style)
	if err != nil {
		t.Fatal(err)
	}
	if lines == nil {
		t.Fatal("expected non-nil lines for empty input")
	}
}

func TestRenderUsesThemeColors(t *testing.T) {
	dark, _ := Render([]byte("*emph*\n"), 80, theme.BuiltinDark().Style)
	light, _ := Render([]byte("*emph*\n"), 80, theme.BuiltinLight().Style)
	if strings.Join(dark, "\n") == strings.Join(light, "\n") {
		t.Fatal("dark and light themes should style output differently")
	}
}
