package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/benborla/xMarkdown/render"
	"github.com/benborla/xMarkdown/theme"
	"github.com/benborla/xMarkdown/ui"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: xmd <file.md>")
		os.Exit(1)
	}
	path := os.Args[1]
	source, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "xmd:", err)
		os.Exit(1)
	}

	// Piped output: dump rendered markdown, no TUI.
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		lines, err := render.Render(source, 80, theme.BuiltinDark().Style)
		if err != nil {
			fmt.Fprintln(os.Stderr, "xmd:", err)
			os.Exit(1)
		}
		for _, l := range lines {
			fmt.Println(l)
		}
		return
	}

	p := tea.NewProgram(ui.New(path, source, theme.BuiltinDark()), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "xmd:", err)
		os.Exit(1)
	}
}
