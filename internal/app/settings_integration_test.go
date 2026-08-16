package app

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tupini07/wordsmith/internal/config"
	"github.com/tupini07/wordsmith/internal/state"
)

func TestF2OpensAndSavesStructuredSettings(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg := config.Default()
	cfg.VaultPath = t.TempDir()
	m := New(cfg, state.State{}, "")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyF2})
	got := updated.(Model)
	if got.mode != ModeSettings || !got.settings.IsVisible() {
		t.Fatal("F2 did not open the settings modal")
	}
	if got.editor.FilePath() != "" {
		t.Fatalf("F2 opened an editor file: %q", got.editor.FilePath())
	}

	got.settings.draft.TabWidth = 2
	got.settings.draft.ContentWidth = 0
	got.settings.draft.AutosaveDelay = 750 * time.Millisecond
	got.settings.draft.Theme = "nord"
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	got = updated.(Model)

	if got.mode != ModeEditor || got.settings.IsVisible() {
		t.Fatal("saving settings did not return to the editor")
	}
	if got.cfg.TabWidth != 2 || got.cfg.ContentWidth != 0 ||
		got.cfg.AutosaveDelay != 750*time.Millisecond || got.activeTheme != "nord" {
		t.Fatalf("settings were not applied: %#v", got.cfg)
	}

	saved, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.TabWidth != 2 || saved.ContentWidth != 0 || saved.Theme != "nord" {
		t.Fatalf("settings were not persisted: %#v", saved)
	}
}

func TestVaultPathChangeIsPersistedButRequiresRestart(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	oldVault := t.TempDir()
	newVault := t.TempDir()
	cfg := config.Default()
	cfg.VaultPath = oldVault
	cfg.JournalFolder = "OldJournal"
	m := New(cfg, state.State{}, "")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyF2})
	got := updated.(Model)
	got.settings.draft.VaultPath = newVault
	got.settings.draft.JournalFolder = "NewJournal"
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	got = updated.(Model)

	if got.cfg.VaultPath != oldVault || got.cfg.JournalFolder != "OldJournal" {
		t.Fatalf("active path-sensitive config changed before restart: %#v", got.cfg)
	}
	if status := got.editor.SaveStatus(); !strings.Contains(status, "restart") {
		t.Fatalf("status = %q, want restart notice", status)
	}

	saved, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.VaultPath != newVault || saved.JournalFolder != "NewJournal" {
		t.Fatalf("persisted config = %#v", saved)
	}
}

func TestSettingsCancelLeavesConfigUntouched(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	cfg := config.Default()
	cfg.VaultPath = t.TempDir()
	m := New(cfg, state.State{}, "")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyF2})
	got := updated.(Model)
	got.settings.draft.TabWidth = 9
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyEscape})
	got = updated.(Model)

	if got.cfg.TabWidth != cfg.TabWidth {
		t.Fatalf("tab width changed to %d after cancel", got.cfg.TabWidth)
	}
	path, err := config.Path()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("config file was created on cancel: %v", err)
	}
}
