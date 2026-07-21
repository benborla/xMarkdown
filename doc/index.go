// Package doc extracts headings and links from markdown source and locates
// them in glamour's rendered output by scanning ANSI-stripped lines.
package doc

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type Heading struct {
	Level int
	Text  string
	Line  int // rendered line number, -1 until matched
}

type Link struct {
	Text string
	URL  string
	Line int
}

type Index struct {
	Headings []Heading
	Links    []Link
}

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

// StripANSI removes SGR escape sequences from s.
func StripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// extract walks the markdown AST collecting headings and links in document order.
func extract(source []byte) (headings []Heading, links []Link) {
	md := goldmark.New()
	root := md.Parser().Parse(text.NewReader(source))
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Heading:
			headings = append(headings, Heading{Level: v.Level, Text: nodeText(v, source), Line: -1})
		case *ast.Link:
			links = append(links, Link{Text: nodeText(v, source), URL: string(v.Destination), Line: -1})
		case *ast.AutoLink:
			u := string(v.URL(source))
			links = append(links, Link{Text: u, URL: u, Line: -1})
		}
		return ast.WalkContinue, nil
	})
	return headings, links
}

// nodeText concatenates all text content beneath n.
func nodeText(n ast.Node, source []byte) string {
	var sb strings.Builder
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if t, ok := c.(*ast.Text); ok {
				sb.Write(t.Segment.Value(source))
			}
		}
		return ast.WalkContinue, nil
	})
	return sb.String()
}

// pos is a cursor into the stripped rendered lines: line number + byte column.
type pos struct{ line, col int }

// Build extracts anchors from source and locates each in rendered (ANSI-styled
// lines). Anchors whose text cannot be found are dropped silently.
func Build(source []byte, rendered []string) Index {
	headings, links := extract(source)
	stripped := make([]string, len(rendered))
	for i, l := range rendered {
		stripped[i] = StripANSI(l)
	}

	var ix Index
	cur := pos{0, 0}
	for _, h := range headings {
		p, ok := findFrom(stripped, cur, h.Text)
		if !ok {
			continue
		}
		h.Line = p.line
		ix.Headings = append(ix.Headings, h)
		cur = pos{p.line, p.col + len(h.Text)}
	}
	cur = pos{0, 0} // links use an independent cursor
	for _, l := range links {
		p, ok := findFrom(stripped, cur, l.Text)
		if !ok {
			continue
		}
		l.Line = p.line
		ix.Links = append(ix.Links, l)
		cur = pos{p.line, p.col + len(l.Text)}
	}
	return ix
}

// findFrom locates text at or after cur, searching the cursor line from its
// column first, then subsequent lines from column 0.
func findFrom(stripped []string, cur pos, text string) (pos, bool) {
	if text == "" || cur.line >= len(stripped) {
		return pos{}, false
	}
	if cur.col <= len(stripped[cur.line]) {
		if idx := strings.Index(stripped[cur.line][cur.col:], text); idx >= 0 {
			return pos{cur.line, cur.col + idx}, true
		}
	}
	for i := cur.line + 1; i < len(stripped); i++ {
		if idx := strings.Index(stripped[i], text); idx >= 0 {
			return pos{i, idx}, true
		}
	}
	// wrapped-text fallback — retry with a short prefix; anchor
	// granularity is the line, so a prefix hit is good enough.
	if r := []rune(text); len(r) > 16 {
		return findFrom(stripped, cur, string(r[:16]))
	}
	return pos{}, false
}

// NextHeading returns the line of the first heading after line, or -1.
func (ix Index) NextHeading(line int) int {
	for _, h := range ix.Headings {
		if h.Line > line {
			return h.Line
		}
	}
	return -1
}

// PrevHeading returns the line of the last heading before line, or -1.
func (ix Index) PrevHeading(line int) int {
	res := -1
	for _, h := range ix.Headings {
		if h.Line >= line {
			break
		}
		res = h.Line
	}
	return res
}
