// Package render turns markdown source into styled terminal lines via glamour.
package render

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

// Render renders markdown to ANSI-styled lines wrapped at width, using the
// given glamour style JSON (a theme's Style bytes).
func Render(source []byte, width int, styleJSON []byte) ([]string, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(styleJSON),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}
	out, err := r.RenderBytes(source)
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimRight(string(out), "\n"), "\n"), nil
}
