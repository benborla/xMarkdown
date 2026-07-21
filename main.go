package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/benborla/xMarkdown/config"
	"github.com/benborla/xMarkdown/render"
	"github.com/benborla/xMarkdown/theme"
	"github.com/benborla/xMarkdown/ui"
)

func main() {
	themeFlag := flag.String("theme", "", "theme: auto | gruvbox-dark | gruvbox-light | <name> | <path>")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: xmd [--theme <theme>] <file.md>")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	path := flag.Arg(0)
	source, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "xmd:", err)
		os.Exit(1)
	}

	var warning string
	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		warning = "config: " + cfgErr.Error()
	}
	spec := cfg.Theme
	if *themeFlag != "" {
		spec = *themeFlag
	}
	dark := theme.DetectDark()
	th, err := theme.Resolve(spec, dark)
	if err != nil {
		warning = err.Error() + " (using auto)"
		th, _ = theme.Resolve("auto", dark)
	}

	// Piped output: dump rendered markdown, no TUI.
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		if warning != "" {
			fmt.Fprintln(os.Stderr, "xmd: "+warning)
		}
		lines, err := render.Render(source, 80, th.Style)
		if err != nil {
			fmt.Fprintln(os.Stderr, "xmd:", err)
			os.Exit(1)
		}
		for _, l := range lines {
			fmt.Println(l)
		}
		return
	}

	numbers := ui.NumbersOff
	switch cfg.Numbers {
	case "absolute":
		numbers = ui.NumbersAbsolute
	case "relative":
		numbers = ui.NumbersRelative
	}

	p := tea.NewProgram(ui.New(path, source, ui.Options{
		Theme:   th,
		Numbers: numbers,
		Dark:    dark,
		Warning: warning,
	}), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "xmd:", err)
		os.Exit(1)
	}
}
