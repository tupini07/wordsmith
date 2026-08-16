package app

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tupini07/wordsmith/internal/config"
	"github.com/tupini07/wordsmith/internal/editor"
)

func TestSettingsModalEditsTypedValues(t *testing.T) {
	cfg := config.Default()
	cfg.VaultPath = t.TempDir()
	sm := newSettingsModal()
	sm.Show(cfg)

	sm.cursor = 2
	sm.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !sm.editing {
		t.Fatal("text field did not enter edit mode")
	}
	sm.input.SetValue("")
	sm.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if sm.err == "" || !sm.editing {
		t.Fatal("invalid value should keep editing and show an error")
	}
	sm.input.SetValue("DD-MM-YYYY")
	sm.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if sm.editing || sm.draft.JournalDateFormat != "DD-MM-YYYY" {
		t.Fatalf("date format was not applied: %#v", sm.draft)
	}

	sm.cursor = 3
	sm.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	sm.input.SetValue("750ms")
	sm.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if sm.draft.AutosaveDelay != 750*time.Millisecond {
		t.Fatalf("autosave delay = %v", sm.draft.AutosaveDelay)
	}

	sm.cursor = 5
	sm.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	sm.input.SetValue("0")
	sm.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if sm.draft.ContentWidth != 0 {
		t.Fatalf("content width = %d", sm.draft.ContentWidth)
	}
}

func TestSettingsModalTogglesAndCyclesChoices(t *testing.T) {
	sm := newSettingsModal()
	sm.Show(config.Default())

	sm.cursor = 6
	sm.HandleKey(tea.KeyMsg{Type: tea.KeySpace})
	if !sm.draft.ShowLineNumbers {
		t.Fatal("boolean field did not toggle")
	}

	sm.cursor = 7
	sm.HandleKey(tea.KeyMsg{Type: tea.KeyRight})
	if sm.draft.Theme != "nord" {
		t.Fatalf("theme = %q, want nord", sm.draft.Theme)
	}
	sm.HandleKey(tea.KeyMsg{Type: tea.KeyLeft})
	if sm.draft.Theme != "gruvbox" {
		t.Fatalf("theme = %q, want gruvbox", sm.draft.Theme)
	}
}

func TestSettingsModalSaveAndCancel(t *testing.T) {
	cfg := config.Default()
	sm := newSettingsModal()
	sm.Show(cfg)
	sm.draft.TabWidth = 2

	result, _ := sm.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	if result == nil || result.canceled || result.cfg.TabWidth != 2 {
		t.Fatalf("save result = %#v", result)
	}

	sm.Show(cfg)
	result, _ = sm.HandleKey(tea.KeyMsg{Type: tea.KeyEscape})
	if result == nil || !result.canceled || sm.IsVisible() {
		t.Fatalf("cancel result = %#v, visible = %v", result, sm.IsVisible())
	}
}

func TestSettingsModalViewListsCompleteSchema(t *testing.T) {
	sm := newSettingsModal()
	sm.Show(config.Default())
	view := sm.View(editor.GruvboxTheme(), 100, 30)

	for _, label := range []string{
		"Vault path",
		"Journal folder",
		"Journal date format",
		"Autosave delay",
		"Tab width",
		"Content width",
		"Show line numbers",
		"Theme",
	} {
		if !strings.Contains(view, label) {
			t.Errorf("view does not contain %q", label)
		}
	}
}
