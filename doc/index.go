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
