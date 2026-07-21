package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/benborla/xMarkdown/doc"
	"github.com/benborla/xMarkdown/theme"
)

// longDoc renders to well over 10 lines at width 40.
const longDoc = `# One

first section text

## Two

second section text

## Three

third section text with alpha keyword

## Four

fourth section text

## Five

fifth section text
`

func key(k string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

// run executes a command synchronously and feeds resulting messages back into
// the model until the command chain settles, mimicking the tea runtime.
func run(m Model, cmd tea.Cmd) Model {
	if cmd == nil {
		return m
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			m = run(m, c)
		}
		return m
	}
	nm, next := m.Update(msg)
	return run(nm.(Model), next)
}

func newTestModel(t *testing.T, md string) Model {
	t.Helper()
	m := New("test.md", []byte(md), Options{Theme: theme.BuiltinDark()})
	nm, cmd := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	res := run(nm.(Model), cmd)
	if len(res.lines) == 0 {
		t.Fatal("reflow produced no lines")
	}
	return res
}

func press(m Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		nm, cmd := m.Update(msg)
		m = run(nm.(Model), cmd)
	}
	return m
}

func TestWindowSizeRendersAndIndexes(t *testing.T) {
	m := newTestModel(t, longDoc)
	if len(m.index.Headings) != 5 {
		t.Fatalf("got %d headings, want 5: %+v", len(m.index.Headings), m.index.Headings)
	}
	if m.offset != 0 {
		t.Fatalf("initial offset = %d, want 0", m.offset)
	}
	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d", m.cursor)
	}
}

func TestCursorKeys(t *testing.T) {
	m := newTestModel(t, longDoc)
	vh := m.viewHeight()

	m = press(m, key("j"))
	if m.cursor != 1 || m.offset != 0 {
		t.Fatalf("after j: cursor=%d offset=%d, want cursor=1 offset=0", m.cursor, m.offset)
	}
	m = press(m, key("k"), key("k")) // clamps at 0
	if m.cursor != 0 {
		t.Fatalf("after k k: cursor = %d, want 0", m.cursor)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.cursor != vh/2 {
		t.Fatalf("after ctrl+d: cursor = %d, want %d", m.cursor, vh/2)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyCtrlU})
	if m.cursor != 0 {
		t.Fatalf("after ctrl+u: cursor = %d, want 0", m.cursor)
	}
	m = press(m, key("G"))
	if m.cursor != len(m.lines)-1 {
		t.Fatalf("after G: cursor = %d, want %d", m.cursor, len(m.lines)-1)
	}
	if m.offset != len(m.lines)-vh {
		t.Fatalf("after G: offset = %d, want %d", m.offset, len(m.lines)-vh)
	}
	m = press(m, key("g"), key("g"))
	if m.cursor != 0 || m.offset != 0 {
		t.Fatalf("after gg: cursor=%d offset=%d", m.cursor, m.offset)
	}
}

func TestScrollOnlyAtMargin(t *testing.T) {
	m := newTestModel(t, longDoc)
	vh := m.viewHeight()
	edge := vh - 1 - scrolloff // last cursor row before scrolling starts
	for i := 0; i < edge; i++ {
		m = press(m, key("j"))
	}
	if m.offset != 0 {
		t.Fatalf("offset moved too early: %d (cursor %d)", m.offset, m.cursor)
	}
	m = press(m, key("j"))
	if m.offset != 1 {
		t.Fatalf("offset should scroll by 1 at margin, got %d", m.offset)
	}
}

func TestJumpKeepsScrolloff(t *testing.T) {
	m := newTestModel(t, longDoc)
	last := m.index.Headings[len(m.index.Headings)-1].Line
	m = press(m, key("]"), key("]"), key("]"), key("]"), key("]"), key("]"),
		key("]"), key("]"), key("]"), key("]")) // jump through all headings
	if m.cursor < last {
		t.Fatalf("cursor = %d, want at last heading %d", m.cursor, last)
	}
	if m.cursor-m.offset > m.viewHeight()-1 {
		t.Fatal("cursor left the viewport")
	}
}

func TestViewSlicesVisibleLines(t *testing.T) {
	m := newTestModel(t, longDoc)
	view := m.View()
	rows := strings.Split(view, "\n")
	if len(rows) != 10 { // viewHeight lines + status line
		t.Fatalf("view has %d rows, want 10", len(rows))
	}
}

func TestQuitKeys(t *testing.T) {
	m := newTestModel(t, longDoc)
	_, cmd := m.Update(key("q"))
	if cmd == nil {
		t.Fatal("q should return tea.Quit")
	}
}

func TestHeadingJumps(t *testing.T) {
	m := newTestModel(t, longDoc)
	first := m.index.Headings[0].Line
	second := m.index.Headings[1].Line
	if first <= 0 {
		t.Fatalf("expected glamour top padding to place first heading below line 0, got %d", first)
	}

	m = press(m, key("]"), key("]")) // from line 0 → first heading
	if m.cursor != first {
		t.Fatalf("after ]]: cursor = %d, want %d (headings: %+v)", m.cursor, first, m.index.Headings)
	}
	m = press(m, key("]"), key("]")) // → second heading
	if m.cursor != second {
		t.Fatalf("after ]] ]]: cursor = %d, want %d", m.cursor, second)
	}
	m = press(m, key("["), key("[")) // back → first heading
	if m.cursor != first {
		t.Fatalf("after [[: cursor = %d, want %d", m.cursor, first)
	}
}

func TestHeadingJumpAtBottomIsNoop(t *testing.T) {
	m := newTestModel(t, longDoc)
	m = press(m, key("G"))
	before := m.cursor
	m = press(m, key("]"), key("]"))
	if m.cursor != before {
		t.Fatalf("]] past last heading moved cursor %d -> %d", before, m.cursor)
	}
}

func TestTOCJump(t *testing.T) {
	m := newTestModel(t, longDoc)
	m = press(m, key("t"))
	if m.mode != modeTOC {
		t.Fatal("t should enter TOC mode")
	}
	view := m.View()
	if !strings.Contains(view, "Table of Contents") {
		t.Fatalf("TOC view missing title:\n%s", view)
	}
	m = press(m, key("j"), key("j"), tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeReading {
		t.Fatal("enter should return to reading mode")
	}
	if m.cursor != m.index.Headings[2].Line {
		t.Fatalf("cursor = %d, want heading 2 line %d", m.cursor, m.index.Headings[2].Line)
	}
}

func TestTOCEscCloses(t *testing.T) {
	m := newTestModel(t, longDoc)
	before := m.cursor
	m = press(m, key("t"), tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeReading || m.cursor != before {
		t.Fatal("esc should close TOC without jumping")
	}
}

func typeString(m Model, s string) Model {
	for _, r := range s {
		m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func TestSearchJumpsToMatch(t *testing.T) {
	m := newTestModel(t, longDoc)
	m = press(m, key("/"))
	if m.mode != modeSearchInput {
		t.Fatal("/ should enter search input mode")
	}
	if !strings.Contains(m.statusLine(), "/") {
		t.Fatalf("status line should echo search input, got %q", m.statusLine())
	}
	m = typeString(m, "alpha")
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeReading {
		t.Fatal("enter should commit search")
	}
	if len(m.matches) != 1 {
		t.Fatalf("matches = %v, want exactly 1 (longDoc has one 'alpha')", m.matches)
	}
	if m.cursor != m.matches[0] {
		t.Fatalf("cursor = %d, want match line %d", m.cursor, m.matches[0])
	}
}

func TestSearchWrapsWithN(t *testing.T) {
	m := newTestModel(t, longDoc)
	m = press(m, key("/"))
	m = typeString(m, "section")
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if len(m.matches) < 2 {
		t.Fatalf("need multiple matches for wrap test, got %v", m.matches)
	}
	start := m.matchIdx
	for range m.matches {
		m = press(m, key("n"))
	}
	if m.matchIdx != start {
		t.Fatalf("n should wrap around: idx = %d, want %d", m.matchIdx, start)
	}
}

func TestSearchNoMatches(t *testing.T) {
	m := newTestModel(t, longDoc)
	before := m.cursor
	m = press(m, key("/"))
	m = typeString(m, "zebra")
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.cursor != before {
		t.Fatal("no-match search should not move cursor")
	}
	if !strings.Contains(m.statusLine(), "no matches") {
		t.Fatalf("status = %q, want no-matches message", m.statusLine())
	}
}

func TestSearchHighlightInView(t *testing.T) {
	m := newTestModel(t, longDoc)
	m = press(m, key("/"))
	m = typeString(m, "alpha")
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(m.View(), "\x1b[48;2;") {
		t.Fatal("view should contain themed background highlight for current match")
	}
}

// TestCursorlineTintAfterSearchMatchReset verifies that the cursorline
// background is re-applied after the \x1b[49m that search.HighlightStyled
// emits at the end of each match, so the tint does not drop between the
// match close and the row end.
func TestCursorlineTintAfterSearchMatchReset(t *testing.T) {
	m := newTestModel(t, longDoc)
	m = press(m, key("/"))
	m = typeString(m, "alpha")
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})

	// The cursor should be on the match row.
	if len(m.matches) != 1 {
		t.Fatalf("matches = %v, want exactly 1", m.matches)
	}
	if m.cursor != m.matches[0] {
		t.Fatalf("cursor = %d, want match line %d", m.cursor, m.matches[0])
	}

	rows := strings.Split(m.View(), "\n")
	cursorRow := rows[m.cursor-m.offset]

	// The search match ends with \x1b[27m\x1b[49m (reverse off + default bg).
	// cursorlineify must re-inject the cursorline bg after that \x1b[49m.
	const wantSeq = "\x1b[49m\x1b[48;2;60;56;54m" // gruvbox-dark cursorline_bg = #3c3836
	if !strings.Contains(cursorRow, wantSeq) {
		t.Fatalf("cursorline tint missing after search-match background reset;\n"+
			"want %q somewhere in row:\n%q", wantSeq, cursorRow)
	}
}

func TestSearchEscCancels(t *testing.T) {
	m := newTestModel(t, longDoc)
	m = press(m, key("/"))
	m = typeString(m, "alp")
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeReading || m.query != "" {
		t.Fatal("esc should cancel search input without committing")
	}
}

func TestSearchInputAcceptsSpace(t *testing.T) {
	m := newTestModel(t, longDoc)
	m = press(m, key("/"))
	m = typeString(m, "first")
	m = press(m, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	m = typeString(m, "section")
	if m.searchInput != "first section" {
		t.Fatalf("searchInput = %q, want %q", m.searchInput, "first section")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	if len(m.matches) != 1 {
		t.Fatalf("matches = %v, want 1 line containing %q", m.matches, "first section")
	}
}

func TestSearchEmptyCommitIsNoop(t *testing.T) {
	m := newTestModel(t, longDoc)
	before := m.offset
	m = press(m, key("/"), tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeReading {
		t.Fatal("enter should leave search input mode")
	}
	if len(m.matches) != 0 || m.query != "" || m.matchIdx != -1 {
		t.Fatalf("empty commit set search state: query=%q matches=%v matchIdx=%d",
			m.query, m.matches, m.matchIdx)
	}
	if m.offset != before {
		t.Fatalf("empty commit moved offset %d -> %d", before, m.offset)
	}
}

func TestResizeKeepsSearchMatch(t *testing.T) {
	m := newTestModel(t, longDoc)
	m = press(m, key("/"))
	m = typeString(m, "alpha")
	m = press(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = press(m, tea.WindowSizeMsg{Width: 60, Height: 12})
	if m.matchIdx < 0 {
		t.Fatalf("resize dropped current match: matchIdx = %d, matches = %v", m.matchIdx, m.matches)
	}
	// clamp may pull offset below the match line near the end of the doc;
	// the contract is that the selected match stays visible.
	if ml := m.matches[m.matchIdx]; ml < m.offset || ml >= m.offset+m.viewHeight() {
		t.Fatalf("match line %d not visible in window [%d, %d)", ml, m.offset, m.offset+m.viewHeight())
	}
	if !strings.Contains(m.View(), "\x1b[48;2;") {
		t.Fatal("match highlight should survive resize")
	}
}

const linkDoc = `# Links

Visit [example](https://example.com) or read [other](other.md).
`

func TestLinkCycle(t *testing.T) {
	m := newTestModel(t, linkDoc)
	if len(m.index.Links) != 2 {
		t.Fatalf("links = %+v, want 2", m.index.Links)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.linkIdx != 0 {
		t.Fatalf("tab: linkIdx = %d, want 0", m.linkIdx)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.linkIdx != 1 {
		t.Fatalf("tab tab: linkIdx = %d, want 1", m.linkIdx)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.linkIdx != 0 {
		t.Fatalf("tab wrap: linkIdx = %d, want 0", m.linkIdx)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.linkIdx != 1 {
		t.Fatalf("shift+tab: linkIdx = %d, want 1", m.linkIdx)
	}
	if !strings.Contains(m.View(), "\x1b[7m") {
		t.Fatal("highlighted link should appear reverse-video in view")
	}
}

func TestFollowURLOpensBrowser(t *testing.T) {
	opened := ""
	orig := openBrowser
	openBrowser = func(url string) error { opened = url; return nil }
	defer func() { openBrowser = orig }()

	m := newTestModel(t, linkDoc)
	m = press(m, tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyEnter})
	if opened != "https://example.com" {
		t.Fatalf("opened = %q, want https://example.com", opened)
	}
}

func TestFollowMarkdownLinkLoadsInPlace(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "other.md"), []byte("# Other Doc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.md")
	if err := os.WriteFile(mainPath, []byte(linkDoc), 0o644); err != nil {
		t.Fatal(err)
	}

	m := New(mainPath, []byte(linkDoc), Options{Theme: theme.BuiltinDark()})
	nm, cmd := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	m = run(nm.(Model), cmd)
	// second link is other.md
	m = press(m, tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyEnter})

	if m.path != filepath.Join(dir, "other.md") {
		t.Fatalf("path = %q, want other.md loaded", m.path)
	}
	if !strings.Contains(doc.StripANSI(m.View()), "Other Doc") {
		t.Fatal("view should show the followed document")
	}
	if m.cursor != 0 {
		t.Fatalf("cursor = %d, want 0 after following", m.cursor)
	}
}

func TestFollowMissingFileShowsError(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.md")
	if err := os.WriteFile(mainPath, []byte(linkDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	m := New(mainPath, []byte(linkDoc), Options{Theme: theme.BuiltinDark()})
	nm, cmd := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	m = run(nm.(Model), cmd)
	m = press(m, tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(m.statusLine(), "cannot open") {
		t.Fatalf("status = %q, want cannot-open error", m.statusLine())
	}
	if m.path != mainPath {
		t.Fatal("view should stay on original document")
	}
}

func TestLoadingSpinnerShownUntilRenderDone(t *testing.T) {
	m := New("test.md", []byte(longDoc), Options{Theme: theme.BuiltinDark()})
	nm, cmd := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	m = nm.(Model)
	if !m.loading {
		t.Fatal("should be loading after WindowSizeMsg")
	}
	if !strings.Contains(m.View(), "rendering") {
		t.Fatalf("loading view should show rendering message:\n%s", m.View())
	}
	m = run(m, cmd)
	if m.loading {
		t.Fatal("loading should clear after render completes")
	}
	if len(m.lines) == 0 {
		t.Fatal("lines should be populated after render")
	}
}

func TestStaleRenderDropped(t *testing.T) {
	m := New("test.md", []byte(longDoc), Options{Theme: theme.BuiltinDark()})
	nm, cmd1 := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	m = nm.(Model)
	nm, cmd2 := m.Update(tea.WindowSizeMsg{Width: 60, Height: 10})
	m = nm.(Model)
	m = run(m, cmd2) // newer render lands first
	wantLines := len(m.lines)
	m = run(m, cmd1) // stale width-40 result must be ignored
	if len(m.lines) != wantLines {
		t.Fatalf("stale render applied: %d lines, want %d", len(m.lines), wantLines)
	}
	if m.loading {
		t.Fatal("loading should stay cleared after stale result")
	}
}

func TestCursorlineTintOnCursorRow(t *testing.T) {
	m := newTestModel(t, longDoc)
	m = press(m, key("j"), key("j"))
	rows := strings.Split(m.View(), "\n")
	cursorRow := rows[m.cursor-m.offset]
	if !strings.Contains(cursorRow, "\x1b[48;2;") {
		t.Fatalf("cursor row should carry a background tint: %q", cursorRow)
	}
	otherRow := rows[m.cursor-m.offset+1]
	if strings.Contains(otherRow, "\x1b[48;2;60;56;54m") { // gruvbox-dark cursorline
		t.Fatalf("non-cursor row should not carry cursorline tint: %q", otherRow)
	}
}

func TestStatusLineThemed(t *testing.T) {
	m := newTestModel(t, longDoc)
	if !strings.Contains(m.statusLine(), "\x1b[") {
		t.Fatal("status line should be styled")
	}
}

func TestGutterAbsoluteAndRelative(t *testing.T) {
	m := newTestModel(t, longDoc)
	m.numbers = NumbersAbsolute
	rows := strings.Split(m.View(), "\n")
	first := doc.StripANSI(rows[0])
	if !strings.HasPrefix(strings.TrimLeft(first, " "), "1 ") {
		t.Fatalf("absolute gutter should start with 1: %q", first)
	}

	m.numbers = NumbersRelative
	m = press(m, key("j"), key("j"))
	rows = strings.Split(m.View(), "\n")
	cursorRow := doc.StripANSI(rows[m.cursor-m.offset])
	if !strings.HasPrefix(strings.TrimLeft(cursorRow, " "), "3 ") {
		t.Fatalf("relative mode shows absolute number on cursor row: %q", cursorRow)
	}
	above := doc.StripANSI(rows[m.cursor-m.offset-1])
	if !strings.HasPrefix(strings.TrimLeft(above, " "), "1 ") {
		t.Fatalf("row above cursor should show relative 1: %q", above)
	}
}

func TestGutterOffByDefault(t *testing.T) {
	m := newTestModel(t, longDoc)
	rows := strings.Split(m.View(), "\n")
	if strings.HasPrefix(strings.TrimLeft(doc.StripANSI(rows[0]), " "), "1 ") {
		t.Fatal("gutter must be off by default")
	}
}
