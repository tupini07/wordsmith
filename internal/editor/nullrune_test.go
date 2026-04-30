package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNullRuneFilterPreservesSelection(t *testing.T) {
	// Create an editor with content
	m := New(4, 80, NordTheme())
	m.SetSize(80, 24)
	m.buffer.SetContent("Hello World")
	m.focused = true
	m.rewrap()

	// Select text with shift+right (5 chars)
	for i := 0; i < 5; i++ {
		msg := tea.KeyMsg{Type: tea.KeyShiftRight}
		m, _ = m.Update(msg)
	}

	if !m.hasSelection {
		t.Fatal("expected selection after shift+right")
	}

	// Simulate bare Ctrl press on Windows: KeyRunes with null char
	nullMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}}
	m, _ = m.Update(nullMsg)

	// Selection must survive
	if !m.hasSelection {
		t.Fatal("null-rune event destroyed selection (bare modifier key)")
	}

	// Content must be unchanged
	if got := m.buffer.Content(); got != "Hello World" {
		t.Fatalf("content corrupted by null rune: %q", got)
	}
}

func TestNullRuneFilterWithAlt(t *testing.T) {
	// Simulate bare Alt press on Windows: KeyRunes with null char + Alt flag
	m := New(4, 80, NordTheme())
	m.SetSize(80, 24)
	m.buffer.SetContent("Test content")
	m.focused = true
	m.rewrap()

	// Select some text
	for i := 0; i < 4; i++ {
		msg := tea.KeyMsg{Type: tea.KeyShiftRight}
		m, _ = m.Update(msg)
	}

	if !m.hasSelection {
		t.Fatal("expected selection")
	}

	// Bare Alt key on Windows
	altNull := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0}, Alt: true}
	m, _ = m.Update(altNull)

	if !m.hasSelection {
		t.Fatal("bare Alt key destroyed selection")
	}

	if got := m.buffer.Content(); got != "Test content" {
		t.Fatalf("content corrupted by bare Alt: %q", got)
	}
}
