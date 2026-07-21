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
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"

	"github.com/benborla/xMarkdown/doc"
	"github.com/benborla/xMarkdown/render"
	"github.com/benborla/xMarkdown/search"
	"github.com/benborla/xMarkdown/theme"
)

type mode int

const (
	modeReading mode = iota
	modeSearchInput
	modeTOC
)

// scrolloff is the vim-style margin: the viewport scrolls once the cursor
// gets within this many lines of an edge.
const scrolloff = 3

// NumberMode selects the line-number gutter style.
type NumberMode int

const (
	NumbersOff NumberMode = iota
	NumbersAbsolute
	NumbersRelative
)

// gutterReserve is the content-width reserve when numbers are on.
// ponytail: fixed 5 cols fits 4-digit line counts; docs rendering to >9999
// lines overflow the row by the extra digits — widen if that ever matters.
const gutterReserve = 5

type Model struct {
	path   string
	source []byte
	theme  theme.Theme

	width, height int
	lines         []string
	index         doc.Index

	mode    mode
	offset  int
	cursor  int
	pending string // first key of two-key sequences: g, ], [

	searchInput string
	query       string
	matches     []int
	matchIdx    int

	tocIdx  int
	linkIdx int

	loading   bool
	spin      int
	renderSeq int // tags async renders so stale results are dropped

	status  string
	numbers NumberMode
	dark    bool
}

// renderDoneMsg carries the result of an async render.
type renderDoneMsg struct {
	seq   int
	lines []string
	index doc.Index
	err   error
}

type spinTickMsg struct{}

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func spinTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return spinTickMsg{} })
}

// Options configures the Model at construction time.
type Options struct {
	Theme   theme.Theme
	Numbers NumberMode
	Dark    bool   // active mode, used by :theme resolution
	Warning string // initial status message (config/theme load problems)
}

func New(path string, source []byte, opts Options) Model {
	return Model{
		path: path, source: source,
		theme: opts.Theme, numbers: opts.Numbers, dark: opts.Dark,
		status:  opts.Warning,
		linkIdx: -1, matchIdx: -1,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, m.startRender()
	case renderDoneMsg:
		if msg.seq != m.renderSeq {
			return m, nil // stale render from before a resize
		}
		m.loading = false
		if msg.err != nil {
			m.status = "render error: " + msg.err.Error()
			return m, nil
		}
		m.applyRender(msg.lines, msg.index)
		return m, nil
	case spinTickMsg:
		if !m.loading {
			return m, nil
		}
		m.spin++
		return m, spinTick()
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

// startRender kicks off an async render at the current width and starts the
// spinner. The seq tag lets Update drop results that a newer render obsoleted.
func (m *Model) startRender() tea.Cmd {
	if m.width <= 0 {
		return nil
	}
	m.loading = true
	m.renderSeq++
	seq, src, w, style := m.renderSeq, m.source, m.width-m.gutterWidth(), m.theme.Style
	return tea.Batch(
		func() tea.Msg {
			lines, err := render.Render(src, w, style)
			if err != nil {
				return renderDoneMsg{seq: seq, err: err}
			}
			return renderDoneMsg{seq: seq, lines: lines, index: doc.Build(src, lines)}
		},
		spinTick(),
	)
}

// applyRender installs freshly rendered lines, preserving state where possible
// (rerunning any active search — line numbers shift on resize).
func (m *Model) applyRender(lines []string, ix doc.Index) {
	m.lines = lines
	m.index = ix
	m.linkIdx = -1
	if m.query != "" {
		m.matches = search.Find(m.lines, m.query)
		m.matchIdx = -1
		if len(m.matches) > 0 {
			m.selectMatchNear(m.cursor) // keep the reader near their match
		}
	}
	m.ensureVisible()
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

func (m *Model) clampCursor() {
	if m.cursor > len(m.lines)-1 {
		m.cursor = len(m.lines) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// ensureVisible scrolls the minimum needed to keep the cursor within
// scrolloff lines of the viewport edges.
func (m *Model) ensureVisible() {
	m.clampCursor()
	vh := m.viewHeight()
	if vh <= 0 || len(m.lines) == 0 {
		return
	}
	off := scrolloff
	if vh <= 2*off {
		off = (vh - 1) / 2
	}
	if m.cursor < m.offset+off {
		m.offset = m.cursor - off
	}
	if m.cursor > m.offset+vh-1-off {
		m.offset = m.cursor - vh + 1 + off
	}
	m.clamp()
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
		m.cursor++
	case "k", "up":
		m.cursor--
	case "ctrl+d":
		m.cursor += m.viewHeight() / 2
	case "ctrl+u":
		m.cursor -= m.viewHeight() / 2
	case "ctrl+f", " ":
		m.cursor += m.viewHeight()
	case "ctrl+b":
		m.cursor -= m.viewHeight()
	case "g":
		if pending == "g" {
			m.cursor = 0
		} else {
			m.pending = "g"
		}
	case "G":
		m.cursor = len(m.lines) - 1
	case "]":
		if pending == "]" {
			if ln := m.index.NextHeading(m.cursor); ln >= 0 {
				m.cursor = ln
			}
		} else {
			m.pending = "]"
		}
	case "[":
		if pending == "[" {
			if ln := m.index.PrevHeading(m.cursor); ln >= 0 {
				m.cursor = ln
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
	m.ensureVisible()
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
	m.cursor = m.matches[m.matchIdx]
	m.status = fmt.Sprintf("match %d/%d", m.matchIdx+1, n)
	m.ensureVisible()
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
	m.cursor = link.Line
	m.status = link.URL
	m.ensureVisible()
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
	m.cursor = 0
	m.query = ""
	m.matches = nil
	m.matchIdx = -1
	m.linkIdx = -1
	return m, m.startRender()
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
		switch msg.Type {
		case tea.KeyRunes:
			m.searchInput += string(msg.Runes)
		case tea.KeySpace:
			m.searchInput += " "
		}
	}
	return m, nil
}

func (m *Model) commitSearch() {
	if m.searchInput == "" {
		return // empty query would match every line
	}
	m.query = m.searchInput
	m.matches = search.Find(m.lines, m.query)
	m.matchIdx = -1
	if len(m.matches) == 0 {
		m.status = "no matches: " + m.query
		return
	}
	m.selectMatchNear(m.cursor)
}

// selectMatchNear implements the vim behavior of jumping to the first match at
// or after line, wrapping to the top match, and moves the cursor there.
// Requires len(m.matches) > 0.
func (m *Model) selectMatchNear(line int) {
	m.matchIdx = -1
	for i, ln := range m.matches {
		if ln >= line {
			m.matchIdx = i
			break
		}
	}
	if m.matchIdx < 0 {
		m.matchIdx = 0 // wrap to top
	}
	m.cursor = m.matches[m.matchIdx]
	m.status = fmt.Sprintf("match %d/%d", m.matchIdx+1, len(m.matches))
	m.ensureVisible()
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
			m.cursor = m.index.Headings[m.tocIdx].Line
			m.ensureVisible()
		}
		m.mode = modeReading
	}
	return m, nil
}

// hexSeq converts "#rrggbb" to an SGR sequence; bg selects background.
// ponytail: truecolor assumed — 256/16-color terminals get approximations
// from the terminal itself or raw truecolor codes; degrade via termenv if
// anyone complains.
func hexSeq(hex string, bg bool) string {
	c := termenv.RGBColor(hex)
	return "\x1b[" + c.Sequence(bg) + "m"
}

func (m Model) gutterWidth() int {
	if m.numbers == NumbersOff {
		return 0
	}
	w := len(fmt.Sprintf("%d", len(m.lines))) + 1
	if w < gutterReserve {
		w = gutterReserve
	}
	return w
}

func (m Model) gutter(i int) string {
	if m.numbers == NumbersOff {
		return ""
	}
	n := i + 1
	fg := m.theme.UI.LinenrFG
	if i == m.cursor {
		fg = m.theme.UI.LinenrCursorFG // cursor row: absolute number, accent color
	} else if m.numbers == NumbersRelative {
		n = i - m.cursor
		if n < 0 {
			n = -n
		}
	}
	return fmt.Sprintf("%s%*d\x1b[0m ", hexSeq(fg, false), m.gutterWidth()-1, n)
}

// setNumbers switches the gutter mode, re-rendering only when the gutter
// appears or disappears (content width change).
func (m *Model) setNumbers(n NumberMode) tea.Cmd {
	wasOn := m.numbers != NumbersOff
	m.numbers = n
	if wasOn != (n != NumbersOff) {
		return m.startRender()
	}
	return nil
}

// cursorlineify tints a full row: sets the background up front, re-applies it
// after every SGR reset inside the line, and pads to the viewport width.
func (m Model) cursorlineify(line string) string {
	bg := hexSeq(m.theme.UI.CursorlineBG, true)
	s := bg + strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+bg)
	// Re-apply bg after \x1b[49m (default-background reset emitted by
	// search.HighlightStyled at the end of each match), so the cursorline
	// tint is not lost between match-end and row-end.
	// The injected bg sequences use \x1b[48;2;...m, not \x1b[49m, so there
	// is no double-processing hazard.
	s = strings.ReplaceAll(s, "\x1b[49m", "\x1b[49m"+bg)
	// ponytail: pad counts runes, not cells — CJK double-width rows over-pad;
	// switch to go-runewidth if that ever matters.
	pad := m.width - len([]rune(doc.StripANSI(line)))
	if pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s + "\x1b[0m"
}

func (m Model) View() string {
	if m.mode == modeTOC {
		return m.viewTOC()
	}
	if m.loading && len(m.lines) == 0 {
		// initial load: nothing to show behind the spinner yet
		vh := m.viewHeight()
		rows := make([]string, vh)
		if vh > 0 {
			rows[vh/2] = "  " + m.spinnerLabel()
		}
		return strings.Join(rows, "\n") + "\n" + m.statusLine()
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
			line = search.HighlightStyled(line, m.query, hexSeq(m.theme.UI.SearchBG, true))
		}
		if i == linkLine {
			line = search.Highlight(line, linkText)
		}
		line = m.gutter(i) + line
		if i == m.cursor {
			line = m.cursorlineify(line)
		}
		visible = append(visible, line)
	}
	for len(visible) < vh {
		visible = append(visible, "")
	}
	return strings.Join(visible, "\n") + "\n" + m.statusLine()
}

func (m Model) spinnerLabel() string {
	return spinFrames[m.spin%len(spinFrames)] + " rendering " + m.path + "…"
}

func (m Model) statusLine() string {
	var content string
	switch {
	case m.mode == modeSearchInput:
		content = "/" + m.searchInput
	case m.loading:
		content = m.spinnerLabel()
	default:
		pct := 100
		if len(m.lines) > 1 {
			pct = m.cursor * 100 / (len(m.lines) - 1)
		}
		left := m.status
		if left == "" {
			left = m.path
		}
		content = fmt.Sprintf("%s  %d%%", left, pct)
	}
	if pad := m.width - len([]rune(content)); pad > 0 {
		content += strings.Repeat(" ", pad)
	}
	return hexSeq(m.theme.UI.StatusFG, false) + hexSeq(m.theme.UI.StatusBG, true) + content + "\x1b[0m"
}

func (m Model) viewTOC() string {
	var b strings.Builder
	b.WriteString("Table of Contents\n\n")
	if len(m.index.Headings) == 0 {
		b.WriteString("  (no headings)\n")
	}
	for i, h := range m.index.Headings {
		row := strings.Repeat("  ", h.Level-1) + h.Text
		if i == m.tocIdx {
			b.WriteString(hexSeq(m.theme.UI.TOCSelectedFG, false) + "> " + row + "\x1b[0m\n")
		} else {
			b.WriteString("  " + row + "\n")
		}
	}
	b.WriteString("\n[enter] jump  [esc] close")
	return b.String()
}
