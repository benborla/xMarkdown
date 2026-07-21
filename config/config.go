// Package config loads xmd's user configuration file.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Theme   string `yaml:"theme"`   // auto | builtin name | custom name | path
	Numbers string `yaml:"numbers"` // off | absolute | relative
}

var defaults = Config{Theme: "auto", Numbers: "off"}

// Dir returns xmd's config directory, honoring XDG_CONFIG_HOME.
func Dir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "xmd")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "xmd")
}

// Load reads <Dir>/config.yaml. A missing file returns defaults and nil
// error; a malformed file returns defaults and the parse error so the caller
// can show a status-line warning.
func Load() (Config, error) {
	return loadFrom(filepath.Join(Dir(), "config.yaml"))
}

func loadFrom(path string) (Config, error) {
	cfg := defaults
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaults, err
	}
	if cfg.Theme == "" {
		cfg.Theme = defaults.Theme
	}
	if cfg.Numbers == "" {
		cfg.Numbers = defaults.Numbers
	}
	return cfg, nil
}
