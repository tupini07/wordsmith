package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultJournalConfig(t *testing.T) {
	cfg := Default()

	if cfg.JournalFolder != "" {
		t.Fatalf("JournalFolder = %q, want disabled", cfg.JournalFolder)
	}
	if cfg.JournalDateFormat != "YYYY-MM-DD" {
		t.Fatalf("JournalDateFormat = %q, want YYYY-MM-DD", cfg.JournalDateFormat)
	}
}

func TestLoadJournalConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	path := filepath.Join(configHome, "wordsmith", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	data := []byte("vault_path: /notes\njournal_folder: Daily\njournal_date_format: DD-MM-YYYY\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.JournalFolder != "Daily" {
		t.Fatalf("JournalFolder = %q, want Daily", cfg.JournalFolder)
	}
	if cfg.JournalDateFormat != "DD-MM-YYYY" {
		t.Fatalf("JournalDateFormat = %q, want DD-MM-YYYY", cfg.JournalDateFormat)
	}
}

func TestSaveRoundTripIncludesCompleteConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	cfg := Config{
		VaultPath:         "/notes",
		JournalFolder:     "Daily",
		JournalDateFormat: "DD-MM-YYYY",
		AutosaveDelay:     750 * time.Millisecond,
		TabWidth:          2,
		ContentWidth:      0,
		ShowLineNumbers:   true,
		Theme:             "nord",
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != cfg {
		t.Fatalf("Load() = %#v, want %#v", got, cfg)
	}

	cfg.Theme = "dracula"
	if err := Save(cfg); err != nil {
		t.Fatalf("overwrite config: %v", err)
	}
	got, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != cfg {
		t.Fatalf("Load() after overwrite = %#v, want %#v", got, cfg)
	}

	data, err := os.ReadFile(filepath.Join(configHome, "wordsmith", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"vault_path:",
		"journal_folder:",
		"journal_date_format:",
		"autosave_delay:",
		"tab_width:",
		"content_width:",
		"show_line_numbers:",
		"theme:",
	} {
		if !strings.Contains(string(data), key) {
			t.Errorf("saved config does not contain %q", key)
		}
	}
}

func TestJournalFilePath(t *testing.T) {
	vault := t.TempDir()
	at := time.Date(2026, time.August, 6, 23, 30, 0, 0, time.Local)

	tests := []struct {
		name    string
		folder  string
		format  string
		wantRel string
	}{
		{
			name:    "default format",
			folder:  "Journal",
			wantRel: filepath.Join("Journal", "2026-08-06.md"),
		},
		{
			name:    "supported tokens",
			folder:  "Daily",
			format:  "DD-MM-YY",
			wantRel: filepath.Join("Daily", "06-08-26.md"),
		},
		{
			name:    "absolute folder inside vault",
			folder:  filepath.Join(vault, "Notes", "Journal"),
			format:  "YYYY_MM_DD",
			wantRel: filepath.Join("Notes", "Journal", "2026_08_06.md"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.VaultPath = vault
			cfg.JournalFolder = tt.folder
			cfg.JournalDateFormat = tt.format

			got, err := cfg.JournalFilePath(at)
			if err != nil {
				t.Fatal(err)
			}
			want := filepath.Join(vault, tt.wantRel)
			if got != want {
				t.Fatalf("JournalFilePath() = %q, want %q", got, want)
			}
		})
	}
}

func TestJournalFilePathRejectsInvalidConfig(t *testing.T) {
	vault := t.TempDir()
	outside := t.TempDir()
	at := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.Local)

	tests := []struct {
		name   string
		vault  string
		folder string
		format string
	}{
		{name: "missing journal folder", vault: vault},
		{name: "missing vault", folder: "Journal"},
		{name: "absolute folder outside vault", vault: vault, folder: outside},
		{name: "relative folder escapes vault", vault: vault, folder: filepath.Join("..", "Journal")},
		{name: "format creates directories", vault: vault, folder: "Journal", format: "YYYY/MM/DD"},
		{name: "format is current directory", vault: vault, folder: "Journal", format: "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.VaultPath = tt.vault
			cfg.JournalFolder = tt.folder
			if tt.format != "" {
				cfg.JournalDateFormat = tt.format
			}

			if _, err := cfg.JournalFilePath(at); err == nil {
				t.Fatal("JournalFilePath() returned nil error")
			}
		})
	}
}

func TestIsPathInVault(t *testing.T) {
	vault := t.TempDir()
	cfg := Default()
	cfg.VaultPath = vault

	if !cfg.IsPathInVault(vault) {
		t.Fatal("vault root should be in vault")
	}
	if !cfg.IsPathInVault(filepath.Join(vault, "Journal", "entry.md")) {
		t.Fatal("vault descendant should be in vault")
	}
	if cfg.IsPathInVault(filepath.Join(filepath.Dir(vault), "outside.md")) {
		t.Fatal("sibling path should not be in vault")
	}
}

func TestIsPathInVaultRejectsSymlinkEscape(t *testing.T) {
	vault := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(vault, "Journal")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	cfg := Default()
	cfg.VaultPath = vault
	if cfg.IsPathInVault(filepath.Join(link, "2026-08-16.md")) {
		t.Fatal("path through an outside-vault symlink should not be in vault")
	}
}
