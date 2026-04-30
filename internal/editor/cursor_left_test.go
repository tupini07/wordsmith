package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCursorLeftFromLineStart(t *testing.T) {
	m := New(4, 80, NordTheme())
	m.SetSize(80, 24)
	m.buffer.SetContent("---\nFirst of, let me")
	m.focused = true
	m.rewrap()

	// Place cursor at start of line 1
	m.cursorLine = 1
	m.cursorCol = 0

	// Press left
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	m, _ = m.Update(msg)

	// Cursor should be at end of line 0 (col 3)
	if m.cursorLine != 0 || m.cursorCol != 3 {
		t.Fatalf("expected cursor at (0,3), got (%d,%d)", m.cursorLine, m.cursorCol)
	}

	// Verify cursor at end of visual line
	visRow, _ := m.wrap.LogicalToVisual(m.cursorLine, m.cursorCol)
	vl := m.wrap.VisualLines[visRow]
	localCol := m.cursorCol - vl.LogicalCol
	if localCol != len(vl.Runes) {
		t.Errorf("cursor not at end of visual line runes")
	}
}

func TestCursorLeftFromLineStartWithEmptyLine(t *testing.T) {
	m := New(4, 80, NordTheme())
	m.SetSize(80, 24)
	m.buffer.SetContent("---\n\nFirst of, let me")
	m.focused = true
	m.rewrap()

	// Place cursor at start of line 2
	m.cursorLine = 2
	m.cursorCol = 0

	// Press left
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	m, _ = m.Update(msg)

	if m.cursorLine != 1 || m.cursorCol != 0 {
		t.Fatalf("expected cursor at (1,0), got (%d,%d)", m.cursorLine, m.cursorCol)
	}

	visRow, visCol := m.wrap.LogicalToVisual(m.cursorLine, m.cursorCol)
	t.Logf("Cursor at empty line - visual: row=%d col=%d", visRow, visCol)
	vl := m.wrap.VisualLines[visRow]
	t.Logf("Visual line: LogicalCol=%d, Runes=%q (len=%d)", vl.LogicalCol, string(vl.Runes), len(vl.Runes))
}

func TestCursorLeftWrappedLine(t *testing.T) {
	m := New(4, 30, NordTheme())
	m.SetSize(30, 24)
	m.buffer.SetContent("---\n\nFirst of, let me give you some advice. Long term planning.")
	m.focused = true
	m.rewrap()

	// Cursor at start of wrapped paragraph, press left
	m.cursorLine = 2
	m.cursorCol = 0
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	m, _ = m.Update(msg)

	// Should move to end of empty line 1
	if m.cursorLine != 1 || m.cursorCol != 0 {
		t.Fatalf("expected (1,0), got (%d,%d)", m.cursorLine, m.cursorCol)
	}

	// Verify cursor would be rendered (cursorVisCol >= 0)
	visRow, _ := m.wrap.LogicalToVisual(m.cursorLine, m.cursorCol)
	vl := m.wrap.VisualLines[visRow]
	localCol := m.cursorCol - vl.LogicalCol
	if localCol != len(vl.Runes) {
		t.Errorf("cursor not at end-of-line position")
	}
}

func TestCursorLeftCRLFFile(t *testing.T) {
	m := New(4, 80, NordTheme())
	m.SetSize(80, 24)
	// Simulate a CRLF file
	m.buffer.SetContent("---\r\n\r\nFirst of, let me give you some advice.")
	m.focused = true
	m.rewrap()

	// Verify CRLF was normalized: line 0 should be "---" (3 chars, no \r)
	if m.buffer.LineLen(0) != 3 {
		t.Fatalf("expected line 0 len=3, got %d (CRLF not stripped)", m.buffer.LineLen(0))
	}
	line0 := m.buffer.Line(0)
	for _, r := range line0 {
		if r == '\r' {
			t.Fatal("line 0 still contains \\r after SetContent")
		}
	}

	// Line 1 should be empty
	if m.buffer.LineLen(1) != 0 {
		t.Fatalf("expected line 1 len=0, got %d", m.buffer.LineLen(1))
	}

	// Cursor at start of line 2, press left
	m.cursorLine = 2
	m.cursorCol = 0
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	m, _ = m.Update(msg)

	// Should be at end of empty line 1 (col 0)
	if m.cursorLine != 1 || m.cursorCol != 0 {
		t.Fatalf("expected (1,0), got (%d,%d)", m.cursorLine, m.cursorCol)
	}
}
