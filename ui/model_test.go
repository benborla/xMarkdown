package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func newTestModel(t *testing.T, md string) Model {
	t.Helper()
	m := New("test.md", []byte(md))
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	res := nm.(Model)
	if len(res.lines) == 0 {
		t.Fatal("reflow produced no lines")
	}
	return res
}

func press(m Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		nm, _ := m.Update(msg)
		m = nm.(Model)
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
}

func TestScrollKeys(t *testing.T) {
	m := newTestModel(t, longDoc)
	vh := m.viewHeight()

	m = press(m, key("j"))
	if m.offset != 1 {
		t.Fatalf("after j: offset = %d, want 1", m.offset)
	}
	m = press(m, key("k"), key("k")) // clamps at 0
	if m.offset != 0 {
		t.Fatalf("after k k: offset = %d, want 0", m.offset)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.offset != vh/2 {
		t.Fatalf("after ctrl+d: offset = %d, want %d", m.offset, vh/2)
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyCtrlU})
	if m.offset != 0 {
		t.Fatalf("after ctrl+u: offset = %d, want 0", m.offset)
	}
	maxOffset := len(m.lines) - vh
	m = press(m, key("G"))
	if m.offset != maxOffset {
		t.Fatalf("after G: offset = %d, want %d", m.offset, maxOffset)
	}
	m = press(m, key("g"), key("g"))
	if m.offset != 0 {
		t.Fatalf("after gg: offset = %d, want 0", m.offset)
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
	if m.offset != first {
		t.Fatalf("after ]]: offset = %d, want %d (headings: %+v)", m.offset, first, m.index.Headings)
	}
	m = press(m, key("]"), key("]")) // → second heading
	if m.offset != second {
		t.Fatalf("after ]] ]]: offset = %d, want %d", m.offset, second)
	}
	m = press(m, key("["), key("[")) // back → first heading
	if m.offset != first {
		t.Fatalf("after [[: offset = %d, want %d", m.offset, first)
	}
}

func TestHeadingJumpAtBottomIsNoop(t *testing.T) {
	m := newTestModel(t, longDoc)
	m = press(m, key("G"))
	before := m.offset
	m = press(m, key("]"), key("]"))
	if m.offset != before {
		t.Fatalf("]] past last heading moved offset %d -> %d", before, m.offset)
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
	if m.offset != m.index.Headings[2].Line {
		t.Fatalf("offset = %d, want heading 2 line %d", m.offset, m.index.Headings[2].Line)
	}
}

func TestTOCEscCloses(t *testing.T) {
	m := newTestModel(t, longDoc)
	before := m.offset
	m = press(m, key("t"), tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeReading || m.offset != before {
		t.Fatal("esc should close TOC without jumping")
	}
}
