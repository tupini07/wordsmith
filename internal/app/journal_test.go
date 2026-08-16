package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tupini07/wordsmith/internal/config"
	"github.com/tupini07/wordsmith/internal/state"
)

func TestAltJCreatesAndOpensTodayJournal(t *testing.T) {
	vault := t.TempDir()
	at := time.Date(2026, time.August, 16, 8, 44, 0, 0, time.Local)
	cfg := config.Default()
	cfg.VaultPath = vault
	cfg.JournalFolder = "Journal"

	m := New(cfg, state.State{}, "")
	m.now = func() time.Time { return at }
	m.mode = ModeFuzzyFinder
	m.finder.Show()

	updated, _ := m.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'j'},
		Alt:   true,
	})
	got := updated.(Model)

	wantPath := filepath.Join(vault, "Journal", "2026-08-16.md")
	if got.editor.FilePath() != wantPath {
		t.Fatalf("opened path = %q, want %q", got.editor.FilePath(), wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("journal file was not created: %v", err)
	}
	if got.mode != ModeEditor || got.finder.IsVisible() {
		t.Fatal("journal command did not return focus to the editor")
	}
	if got.state.LastFile != filepath.Join("Journal", "2026-08-16.md") {
		t.Fatalf("last file = %q", got.state.LastFile)
	}
	if len(got.state.RecentFiles) != 1 || got.state.RecentFiles[0] != got.state.LastFile {
		t.Fatalf("recent files = %#v", got.state.RecentFiles)
	}
}

func TestJournalOpensExistingEntryWithoutTruncating(t *testing.T) {
	vault := t.TempDir()
	cfg := config.Default()
	cfg.VaultPath = vault
	cfg.JournalFolder = "Journal"
	at := time.Date(2026, time.August, 16, 0, 0, 0, 0, time.Local)

	path, err := cfg.JournalFilePath(at)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const content = "# Existing entry\n\nDo not truncate me.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m := New(cfg, state.State{}, "")
	m.now = func() time.Time { return at }
	updated, _ := m.openTodayJournal()
	got := updated.(Model)

	if got.editor.FilePath() != path {
		t.Fatalf("opened path = %q, want %q", got.editor.FilePath(), path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Fatalf("existing content changed to %q", data)
	}
}

func TestJournalSavesDirtyFileBeforeSwitching(t *testing.T) {
	vault := t.TempDir()
	current := filepath.Join(vault, "current.md")
	if err := os.WriteFile(current, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.VaultPath = vault
	cfg.JournalFolder = "Journal"
	m := New(cfg, state.State{}, "")
	m.now = func() time.Time {
		return time.Date(2026, time.August, 16, 0, 0, 0, 0, time.Local)
	}
	if err := m.editor.LoadFile(current); err != nil {
		t.Fatal(err)
	}
	m.editor, _ = m.editor.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if !m.editor.IsDirty() {
		t.Fatal("editor should be dirty before opening journal")
	}

	m.openTodayJournal()

	data, err := os.ReadFile(current)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "xoriginal" {
		t.Fatalf("saved content = %q, want xoriginal", data)
	}
}

func TestJournalDisabledShowsStatusAndClosesOverlay(t *testing.T) {
	cfg := config.Default()
	cfg.VaultPath = t.TempDir()
	m := New(cfg, state.State{}, "")
	m.mode = ModeFuzzyFinder
	m.finder.Show()

	updated, _ := m.openTodayJournal()
	got := updated.(Model)

	if got.editor.FilePath() != "" {
		t.Fatalf("opened unexpected file %q", got.editor.FilePath())
	}
	if got.mode != ModeEditor || got.finder.IsVisible() {
		t.Fatal("disabled journal should return to the editor")
	}
	if status := got.editor.SaveStatus(); !strings.Contains(status, "journal folder is not configured") {
		t.Fatalf("status = %q", status)
	}
}

func TestJournalRejectsDirectoryAtEntryPath(t *testing.T) {
	vault := t.TempDir()
	cfg := config.Default()
	cfg.VaultPath = vault
	cfg.JournalFolder = "Journal"
	at := time.Date(2026, time.August, 16, 0, 0, 0, 0, time.Local)
	path, err := cfg.JournalFilePath(at)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}

	m := New(cfg, state.State{}, "")
	m.now = func() time.Time { return at }
	updated, _ := m.openTodayJournal()
	got := updated.(Model)

	if status := got.editor.SaveStatus(); !strings.Contains(status, "Journal path is a directory") {
		t.Fatalf("status = %q", status)
	}
	if got.editor.FilePath() != "" {
		t.Fatalf("opened unexpected file %q", got.editor.FilePath())
	}
}
