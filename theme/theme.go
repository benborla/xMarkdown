// Package theme resolves xmd themes: single JSON files carrying a glamour
// StyleConfig plus an "xmd" key with UI chrome colors.
package theme

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/muesli/termenv"

	"github.com/benborla/xMarkdown/config"
)

//go:embed gruvbox-dark.json
var gruvboxDark []byte

//go:embed gruvbox-light.json
var gruvboxLight []byte

// UI holds xmd chrome colors (the "xmd" key of a theme file).
type UI struct {
	CursorlineBG   string `json:"cursorline_bg"`
	LinenrFG       string `json:"linenr_fg"`
	LinenrCursorFG string `json:"linenr_cursor_fg"`
	StatusFG       string `json:"status_fg"`
	StatusBG       string `json:"status_bg"`
	TOCSelectedFG  string `json:"toc_selected_fg"`
	SearchBG       string `json:"search_bg"`
}

// Theme is a resolved theme: raw style JSON for glamour (which ignores the
// "xmd" key) plus the parsed UI colors.
type Theme struct {
	Name  string
	Style []byte
	UI    UI
}

// DetectDark reports whether the terminal background is dark. Query failures
// default to dark per spec.
func DetectDark() bool {
	return termenv.HasDarkBackground()
}

func BuiltinDark() Theme  { t, _ := fromBytes("gruvbox-dark", gruvboxDark, true); return t }
func BuiltinLight() Theme { t, _ := fromBytes("gruvbox-light", gruvboxLight, false); return t }

// Resolve maps a theme spec to a Theme. "" or "auto" picks gruvbox by
// terminal darkness; a builtin name, a name under <config>/themes/<name>.json,
// or a filesystem path are also accepted.
func Resolve(spec string, dark bool) (Theme, error) {
	switch spec {
	case "", "auto":
		if dark {
			return BuiltinDark(), nil
		}
		return BuiltinLight(), nil
	case "gruvbox-dark":
		return BuiltinDark(), nil
	case "gruvbox-light":
		return BuiltinLight(), nil
	}
	path := spec
	if !strings.ContainsAny(spec, "/.") {
		path = filepath.Join(config.Dir(), "themes", spec+".json")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, fmt.Errorf("theme %q: %w", spec, err)
	}
	return fromBytes(spec, data, dark)
}

func fromBytes(name string, data []byte, dark bool) (Theme, error) {
	var envelope struct {
		XMD UI `json:"xmd"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Theme{}, fmt.Errorf("theme %s: %w", name, err)
	}
	ui := envelope.XMD
	fillDefaults(&ui, dark)
	return Theme{Name: name, Style: data, UI: ui}, nil
}

// fillDefaults merges missing UI fields from the builtin matching the active
// mode, so custom themes only need to override what they care about.
func fillDefaults(ui *UI, dark bool) {
	src := gruvboxDark
	if !dark {
		src = gruvboxLight
	}
	var def struct {
		XMD UI `json:"xmd"`
	}
	json.Unmarshal(src, &def)
	d := def.XMD
	if ui.CursorlineBG == "" {
		ui.CursorlineBG = d.CursorlineBG
	}
	if ui.LinenrFG == "" {
		ui.LinenrFG = d.LinenrFG
	}
	if ui.LinenrCursorFG == "" {
		ui.LinenrCursorFG = d.LinenrCursorFG
	}
	if ui.StatusFG == "" {
		ui.StatusFG = d.StatusFG
	}
	if ui.StatusBG == "" {
		ui.StatusBG = d.StatusBG
	}
	if ui.TOCSelectedFG == "" {
		ui.TOCSelectedFG = d.TOCSelectedFG
	}
	if ui.SearchBG == "" {
		ui.SearchBG = d.SearchBG
	}
}
