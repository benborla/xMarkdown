package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileGivesDefaults(t *testing.T) {
	cfg, err := loadFrom(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatalf("missing file should not error, got %v", err)
	}
	if cfg.Theme != "auto" || cfg.Numbers != "off" {
		t.Fatalf("cfg = %+v, want defaults auto/off", cfg)
	}
}

func TestLoadReadsValues(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(p, []byte("theme: gruvbox-light\nnumbers: relative\n"), 0o644)
	cfg, err := loadFrom(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "gruvbox-light" || cfg.Numbers != "relative" {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoadMalformedFallsBackWithError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(p, []byte(":\n\t: bad yaml ["), 0o644)
	cfg, err := loadFrom(p)
	if err == nil {
		t.Fatal("malformed yaml should return error for status warning")
	}
	if cfg.Theme != "auto" || cfg.Numbers != "off" {
		t.Fatalf("cfg = %+v, want defaults on malformed input", cfg)
	}
}

func TestDirRespectsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdgtest")
	if got := Dir(); got != "/tmp/xdgtest/xmd" {
		t.Fatalf("Dir = %q", got)
	}
}
