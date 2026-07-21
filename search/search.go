// Package search matches queries against ANSI-stripped rendered lines and
// highlights matches inside styled lines.
package search

import (
	"strings"

	"xmd/doc"
)

// Find returns line numbers whose ANSI-stripped text contains query
// (case-insensitive).
func Find(lines []string, query string) []int {
	q := strings.ToLower(query)
	var out []int
	for i, l := range lines {
		if strings.Contains(strings.ToLower(doc.StripANSI(l)), q) {
			out = append(out, i)
		}
	}
	return out
}
