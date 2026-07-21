// Package ui implements the Bubble Tea model: a vim-navigable viewport over
// glamour-rendered markdown.
package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"xmd/doc"
	"xmd/render"
	"xmd/search"
)

type mode int

const (
	modeReading mode = iota
	modeSearchInput
	modeTOC
)

type Model struct {
	path   string
	source []byte

	width, height int
	lines         []string
	index         doc.Index

	mode    mode
	offset  int
	pending string // first key of two-key sequences: g, ], [

	searchInput string
	query       string
	matches     []int
	matchIdx    int

	tocIdx  int
	linkIdx int

	status string
}

func New(path string, source []byte) Model {
	return Model{path: path, source: source, linkIdx: -1, matchIdx: -1}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.reflow()
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeSearchInput:
			return m.updateSearchInput(msg)
		case modeTOC:
			return m.updateTOC(msg)
		default:
			return m.updateReading(msg)
		}
	}
	return m, nil
}

// reflow re-renders and re-indexes at the current width, preserving state
// where possible (rerunning any active search — line numbers shift on resize).
func (m *Model) reflow() {
	if m.width <= 0 {
		return
	}
	lines, err := render.Render(m.source, m.width)
	if err != nil {
		m.status = "render error: " + err.Error()
		return
	}
	m.lines = lines
	m.index = doc.Build(m.source, lines)
	m.linkIdx = -1
	if m.query != "" {
		m.matches = search.Find(m.lines, m.query)
		m.matchIdx = -1
	}
	m.clamp()
}

func (m Model) viewHeight() int {
	if m.height <= 1 {
		return m.height
	}
	return m.height - 1 // reserve status line
}

func (m *Model) clamp() {
	max := len(m.lines) - m.viewHeight()
	if max < 0 {
		max = 0
	}
	if m.offset > max {
		m.offset = max
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m Model) updateReading(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()
	pending := m.pending
	m.pending = ""
	m.status = ""
	switch keyStr {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		m.offset++
	case "k", "up":
		m.offset--
	case "ctrl+d":
		m.offset += m.viewHeight() / 2
	case "ctrl+u":
		m.offset -= m.viewHeight() / 2
	case "ctrl+f", " ":
		m.offset += m.viewHeight()
	case "ctrl+b":
		m.offset -= m.viewHeight()
	case "g":
		if pending == "g" {
			m.offset = 0
		} else {
			m.pending = "g"
		}
	case "G":
		m.offset = len(m.lines)
	case "]":
		if pending == "]" {
			if ln := m.index.NextHeading(m.offset); ln >= 0 {
				m.offset = ln
			}
		} else {
			m.pending = "]"
		}
	case "[":
		if pending == "[" {
			if ln := m.index.PrevHeading(m.offset); ln >= 0 {
				m.offset = ln
			}
		} else {
			m.pending = "["
		}
	case "/":
		m.mode = modeSearchInput
		m.searchInput = ""
	case "n":
		m.jumpMatch(1)
	case "N":
		m.jumpMatch(-1)
	case "t":
		m.mode = modeTOC
		m.tocIdx = 0
	case "tab":
		m.cycleLink(1)
	case "shift+tab":
		m.cycleLink(-1)
	case "enter":
		return m.followLink()
	case "esc":
		m.linkIdx = -1
		m.query = ""
		m.matches = nil
		m.matchIdx = -1
	}
	m.clamp()
	return m, nil
}

func (m *Model) jumpMatch(dir int) {
	n := len(m.matches)
	if n == 0 {
		m.status = "no matches"
		return
	}
	if m.matchIdx < 0 {
		if dir > 0 {
			m.matchIdx = 0
		} else {
			m.matchIdx = n - 1
		}
	} else {
		m.matchIdx = ((m.matchIdx+dir)%n + n) % n
	}
	m.offset = m.matches[m.matchIdx]
	m.status = fmt.Sprintf("match %d/%d", m.matchIdx+1, n)
	m.clamp()
}

func (m *Model) cycleLink(dir int) {
	n := len(m.index.Links)
	if n == 0 {
		m.status = "no links"
		return
	}
	if m.linkIdx < 0 {
		if dir > 0 {
			m.linkIdx = 0
		} else {
			m.linkIdx = n - 1
		}
	} else {
		m.linkIdx = ((m.linkIdx+dir)%n + n) % n
	}
	link := m.index.Links[m.linkIdx]
	if link.Line < m.offset || link.Line >= m.offset+m.viewHeight() {
		m.offset = link.Line
	}
	m.status = link.URL
	m.clamp()
}

// openBrowser is a package var so tests can stub it.
var openBrowser = func(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (m Model) followLink() (tea.Model, tea.Cmd) {
	if m.linkIdx < 0 || m.linkIdx >= len(m.index.Links) {
		return m, nil
	}
	url := m.index.Links[m.linkIdx].URL
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		if err := openBrowser(url); err != nil {
			m.status = "open failed: " + err.Error()
		} else {
			m.status = "opened " + url
		}
		return m, nil
	}
	target := url
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(m.path), target)
	}
	src, err := os.ReadFile(target)
	if err != nil {
		m.status = "cannot open: " + url
		return m, nil
	}
	m.path = target
	m.source = src
	m.offset = 0
	m.query = ""
	m.matches = nil
	m.matchIdx = -1
	m.reflow()
	return m, nil
}

func (m Model) updateSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeReading
	case "enter":
		m.mode = modeReading
		m.commitSearch()
	case "backspace":
		if len(m.searchInput) > 0 {
			m.searchInput = m.searchInput[:len(m.searchInput)-1]
		}
	case "ctrl+c":
		return m, tea.Quit
	default:
		if msg.Type == tea.KeyRunes {
			m.searchInput += string(msg.Runes)
		}
	}
	return m, nil
}

func (m *Model) commitSearch() {
	m.query = m.searchInput
	m.matches = search.Find(m.lines, m.query)
	m.matchIdx = -1
	if len(m.matches) == 0 {
		m.status = "no matches: " + m.query
		return
	}
	// vim behavior: jump to first match at or after the current position
	for i, ln := range m.matches {
		if ln >= m.offset {
			m.matchIdx = i
			break
		}
	}
	if m.matchIdx < 0 {
		m.matchIdx = 0 // wrap to top
	}
	m.offset = m.matches[m.matchIdx]
	m.status = fmt.Sprintf("match %d/%d", m.matchIdx+1, len(m.matches))
	m.clamp()
}

func (m Model) updateTOC(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "t", "q":
		m.mode = modeReading
	case "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.tocIdx < len(m.index.Headings)-1 {
			m.tocIdx++
		}
	case "k", "up":
		if m.tocIdx > 0 {
			m.tocIdx--
		}
	case "enter":
		if len(m.index.Headings) > 0 {
			m.offset = m.index.Headings[m.tocIdx].Line
			m.clamp()
		}
		m.mode = modeReading
	}
	return m, nil
}

func (m Model) View() string {
	if m.mode == modeTOC {
		return m.viewTOC()
	}
	vh := m.viewHeight()
	end := m.offset + vh
	if end > len(m.lines) {
		end = len(m.lines)
	}
	linkLine, linkText := -1, ""
	if m.linkIdx >= 0 && m.linkIdx < len(m.index.Links) {
		linkLine = m.index.Links[m.linkIdx].Line
		linkText = m.index.Links[m.linkIdx].Text
	}
	visible := make([]string, 0, vh)
	for i := m.offset; i < end; i++ {
		line := m.lines[i]
		if m.matchIdx >= 0 && m.matchIdx < len(m.matches) && i == m.matches[m.matchIdx] {
			line = search.Highlight(line, m.query)
		}
		if i == linkLine {
			line = search.Highlight(line, linkText)
		}
		visible = append(visible, line)
	}
	for len(visible) < vh {
		visible = append(visible, "")
	}
	return strings.Join(visible, "\n") + "\n" + m.statusLine()
}

func (m Model) statusLine() string {
	if m.mode == modeSearchInput {
		return "/" + m.searchInput
	}
	pct := 100
	if max := len(m.lines) - m.viewHeight(); max > 0 {
		pct = m.offset * 100 / max
	}
	left := m.status
	if left == "" {
		left = m.path
	}
	return fmt.Sprintf("%s  %d%%", left, pct)
}

func (m Model) viewTOC() string {
	var b strings.Builder
	b.WriteString("Table of Contents\n\n")
	if len(m.index.Headings) == 0 {
		b.WriteString("  (no headings)\n")
	}
	for i, h := range m.index.Headings {
		cursor := "  "
		if i == m.tocIdx {
			cursor = "> "
		}
		b.WriteString(cursor + strings.Repeat("  ", h.Level-1) + h.Text + "\n")
	}
	b.WriteString("\n[enter] jump  [esc] close")
	return b.String()
}
