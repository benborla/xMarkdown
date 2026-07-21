// Package search matches queries against ANSI-stripped rendered lines and
// highlights matches inside styled lines.
package search

import (
	"strings"
	"unicode"
	"unicode/utf8"

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

// Highlight wraps the first case-insensitive occurrence of query in raw (an
// ANSI-styled line) with reverse video. Returns raw unchanged if no match.
// ponytail: a [0m reset inside the match region cancels the reverse video
// mid-match — acceptable; fix with full SGR state tracking if it ever matters.
func Highlight(raw, query string) string {
	if query == "" {
		return raw
	}
	stripped, byteMap := stripWithMap(raw)
	// Lowercasing can change a rune's byte length (e.g. Ⱥ U+023A is 2 bytes,
	// ⱥ U+2C65 is 3; İ U+0130 is 2, i is 1), so an index into the lowered
	// string cannot be used on stripped directly. Build the lowered string
	// rune-by-rune alongside a map from each lowered byte to the stripped
	// byte offset of the rune it came from.
	var low strings.Builder
	var lowMap []int
	for bi, r := range stripped {
		lr := unicode.ToLower(r)
		for j := 0; j < utf8.RuneLen(lr); j++ {
			lowMap = append(lowMap, bi)
		}
		low.WriteRune(lr)
	}
	lq := strings.ToLower(query)
	idx := strings.Index(low.String(), lq)
	if idx < 0 {
		return raw
	}
	startStripped := lowMap[idx]
	lastStripped := lowMap[idx+len(lq)-1]
	_, w := utf8.DecodeRuneInString(stripped[lastStripped:])
	start := byteMap[startStripped]
	end := byteMap[lastStripped+w-1] + 1
	return raw[:start] + "\x1b[7m" + raw[start:end] + "\x1b[27m" + raw[end:]
}

// stripWithMap strips ANSI codes and returns, for each byte of the stripped
// string, its index in the raw string.
func stripWithMap(raw string) (string, []int) {
	var b strings.Builder
	var m []int
	i := 0
	for i < len(raw) {
		if loc := ansiPrefixLen(raw[i:]); loc > 0 {
			i += loc
			continue
		}
		b.WriteByte(raw[i])
		m = append(m, i)
		i++
	}
	return b.String(), m
}

// ansiPrefixLen returns the length of an SGR escape sequence at the start of s,
// or 0 if s does not start with one.
func ansiPrefixLen(s string) int {
	if !strings.HasPrefix(s, "\x1b[") {
		return 0
	}
	for j := 2; j < len(s); j++ {
		c := s[j]
		if c == 'm' {
			return j + 1
		}
		if (c < '0' || c > '9') && c != ';' {
			return 0
		}
	}
	return 0
}
