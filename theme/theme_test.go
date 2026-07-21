package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltinsParse(t *testing.T) {
	d := BuiltinDark()
	l := BuiltinLight()
	if d.UI.CursorlineBG == "" || l.UI.CursorlineBG == "" {
		t.Fatal("builtin themes must define all xmd UI colors")
	}
	if d.UI.CursorlineBG == l.UI.CursorlineBG {
		t.Fatal("dark and light should differ")
	}
	if len(d.Style) == 0 {
		t.Fatal("Style must carry raw JSON for glamour")
	}
}

func TestResolveAuto(t *testing.T) {
	th, err := Resolve("auto", true)
	if err != nil || th.Name != "gruvbox-dark" {
		t.Fatalf("Resolve(auto, dark) = %v, %v", th.Name, err)
	}
	th, err = Resolve("", false)
	if err != nil || th.Name != "gruvbox-light" {
		t.Fatalf("Resolve('', light) = %v, %v", th.Name, err)
	}
}

func TestResolveBuiltinByName(t *testing.T) {
	th, err := Resolve("gruvbox-light", true) // explicit name beats dark flag
	if err != nil || th.Name != "gruvbox-light" {
		t.Fatalf("got %v, %v", th.Name, err)
	}
}

func TestResolveCustomFileWithDefaultsMerge(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	themesDir := filepath.Join(dir, "xmd", "themes")
	os.MkdirAll(themesDir, 0o755)
	// custom theme sets only one xmd field; the rest must merge from defaults
	os.WriteFile(filepath.Join(themesDir, "mytheme.json"),
		[]byte(`{"document":{"color":"#ffffff"},"xmd":{"cursorline_bg":"#123456"}}`), 0o644)

	th, err := Resolve("mytheme", true)
	if err != nil {
		t.Fatal(err)
	}
	if th.UI.CursorlineBG != "#123456" {
		t.Fatalf("custom field not honored: %+v", th.UI)
	}
	if th.UI.LinenrFG != BuiltinDark().UI.LinenrFG {
		t.Fatalf("missing fields must merge from dark defaults: %+v", th.UI)
	}
}

func TestResolveDirectPath(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.json")
	os.WriteFile(p, []byte(`{"xmd":{}}`), 0o644)
	if _, err := Resolve(p, true); err != nil {
		t.Fatal(err)
	}
}

func TestResolveUnknownErrors(t *testing.T) {
	_, err := Resolve("no-such-theme", true)
	if err == nil || !strings.Contains(err.Error(), "no-such-theme") {
		t.Fatalf("want descriptive error, got %v", err)
	}
}

func TestResolveMalformedErrors(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(p, []byte("{not json"), 0o644)
	if _, err := Resolve(p, true); err == nil {
		t.Fatal("malformed theme JSON must error")
	}
}
