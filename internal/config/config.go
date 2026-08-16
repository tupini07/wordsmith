package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"

	"gopkg.in/yaml.v3"
)

type Config struct {
	VaultPath         string        `yaml:"vault_path"`
	JournalFolder     string        `yaml:"journal_folder"`
	JournalDateFormat string        `yaml:"journal_date_format"`
	AutosaveDelay     time.Duration `yaml:"autosave_delay"`
	TabWidth          int           `yaml:"tab_width"`
	ContentWidth      int           `yaml:"content_width"`
	ShowLineNumbers   bool          `yaml:"show_line_numbers"`
	Theme             string        `yaml:"theme"`
}

type fileConfig struct {
	VaultPath         string `yaml:"vault_path"`
	JournalFolder     string `yaml:"journal_folder"`
	JournalDateFormat string `yaml:"journal_date_format"`
	AutosaveDelay     string `yaml:"autosave_delay"`
	TabWidth          *int   `yaml:"tab_width"`
	ContentWidth      *int   `yaml:"content_width"`
	ShowLineNumbers   *bool  `yaml:"show_line_numbers"`
	Theme             string `yaml:"theme"`
}

func Default() Config {
	return Config{
		VaultPath:         "",
		JournalFolder:     "",
		JournalDateFormat: "YYYY-MM-DD",
		AutosaveDelay:     2 * time.Second,
		TabWidth:          4,
		ContentWidth:      80,
		ShowLineNumbers:   false,
		Theme:             "gruvbox",
	}
}

func configPath() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, "wordsmith", "config.yaml"), nil
}

// Path returns the config file path (may not exist yet).
func Path() (string, error) {
	return configPath()
}

func Load() (Config, error) {
	cfg := Default()

	path, err := configPath()
	if err != nil {
		return cfg, nil // use defaults if we can't find config dir
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil // no config file, use defaults
	}
	if err != nil {
		return cfg, err
	}

	// Handle UTF-16 and BOM encodings (common on Windows)
	data = normalizeEncoding(data)

	// yaml.v3 doesn't handle time.Duration directly. Pointer scalar fields also
	// distinguish omitted values from explicit zero/false values.
	var raw fileConfig

	if err := yaml.Unmarshal(data, &raw); err != nil {
		return cfg, err
	}

	if raw.VaultPath != "" {
		cfg.VaultPath = raw.VaultPath
	}
	if raw.JournalFolder != "" {
		cfg.JournalFolder = raw.JournalFolder
	}
	if raw.JournalDateFormat != "" {
		cfg.JournalDateFormat = raw.JournalDateFormat
	}
	if raw.AutosaveDelay != "" {
		d, err := time.ParseDuration(raw.AutosaveDelay)
		if err == nil {
			cfg.AutosaveDelay = d
		}
	}
	if raw.TabWidth != nil && *raw.TabWidth > 0 {
		cfg.TabWidth = *raw.TabWidth
	}
	if raw.ContentWidth != nil && *raw.ContentWidth >= 0 {
		cfg.ContentWidth = *raw.ContentWidth
	}
	if raw.ShowLineNumbers != nil {
		cfg.ShowLineNumbers = *raw.ShowLineNumbers
	}
	if raw.Theme != "" {
		cfg.Theme = raw.Theme
	}

	return cfg, nil
}

// Save writes every supported setting to the config file atomically.
func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	raw := fileConfig{
		VaultPath:         cfg.VaultPath,
		JournalFolder:     cfg.JournalFolder,
		JournalDateFormat: cfg.JournalDateFormat,
		AutosaveDelay:     cfg.AutosaveDelay.String(),
		TabWidth:          &cfg.TabWidth,
		ContentWidth:      &cfg.ContentWidth,
		ShowLineNumbers:   &cfg.ShowLineNumbers,
		Theme:             cfg.Theme,
	}
	data, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// normalizeEncoding detects UTF-16 and BOM-marked files and converts to plain UTF-8 bytes.
func normalizeEncoding(data []byte) []byte {
	if len(data) >= 2 {
		// UTF-16 LE BOM
		if data[0] == 0xFF && data[1] == 0xFE {
			return []byte(decodeUTF16LE(data[2:]))
		}
		// UTF-16 BE BOM
		if data[0] == 0xFE && data[1] == 0xFF {
			return []byte(decodeUTF16BE(data[2:]))
		}
	}
	// UTF-8 BOM — strip it
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}
	// Heuristic: detect BOM-less UTF-16 LE (every other byte null)
	if len(data) >= 4 && looksLikeUTF16LE(data) {
		return []byte(decodeUTF16LE(data))
	}
	return data
}

func looksLikeUTF16LE(data []byte) bool {
	if len(data) < 4 || len(data)%2 != 0 {
		return false
	}
	check := len(data)
	if check > 200 {
		check = 200
	}
	nulls := 0
	for i := 1; i < check; i += 2 {
		if data[i] == 0 {
			nulls++
		}
	}
	return nulls > (check/2)*8/10
}

func decodeUTF16LE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	u16s := make([]uint16, len(data)/2)
	for i := range u16s {
		u16s[i] = uint16(data[i*2]) | uint16(data[i*2+1])<<8
	}
	return string(utf16.Decode(u16s))
}

func decodeUTF16BE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	u16s := make([]uint16, len(data)/2)
	for i := range u16s {
		u16s[i] = uint16(data[i*2])<<8 | uint16(data[i*2+1])
	}
	return string(utf16.Decode(u16s))
}

// AbsFilePath converts a vault-relative path to an absolute path.
func (c Config) AbsFilePath(relPath string) string {
	if c.VaultPath == "" {
		return relPath
	}
	return filepath.Join(c.VaultPath, relPath)
}

// RelFilePath converts an absolute path to a vault-relative path.
func (c Config) RelFilePath(absPath string) string {
	if c.VaultPath == "" {
		return absPath
	}
	rel, err := filepath.Rel(c.VaultPath, absPath)
	if err != nil {
		return absPath
	}
	return rel
}

// IsPathInVault reports whether path is the vault root or one of its descendants.
func (c Config) IsPathInVault(path string) bool {
	if c.VaultPath == "" || path == "" {
		return false
	}

	vaultAbs, err := canonicalPath(c.VaultPath)
	if err != nil {
		return false
	}
	pathAbs, err := canonicalPath(path)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(vaultAbs, pathAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// canonicalPath resolves symlinks in the existing portion of a path while
// preserving any trailing components that have not been created yet.
func canonicalPath(path string) (string, error) {
	current, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	var missing []string
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

// JournalFilePath returns the absolute path for the journal entry at the given time.
func (c Config) JournalFilePath(at time.Time) (string, error) {
	if strings.TrimSpace(c.JournalFolder) == "" {
		return "", fmt.Errorf("journal folder is not configured")
	}
	if strings.TrimSpace(c.VaultPath) == "" {
		return "", fmt.Errorf("vault path is not configured")
	}

	vaultAbs, err := filepath.Abs(c.VaultPath)
	if err != nil {
		return "", fmt.Errorf("resolve vault path: %w", err)
	}

	folder := c.JournalFolder
	if !filepath.IsAbs(folder) {
		folder = filepath.Join(vaultAbs, folder)
	}
	folderAbs, err := filepath.Abs(folder)
	if err != nil {
		return "", fmt.Errorf("resolve journal folder: %w", err)
	}
	if !c.IsPathInVault(folderAbs) {
		return "", fmt.Errorf("journal folder must be inside the vault")
	}

	format := c.JournalDateFormat
	if format == "" {
		format = "YYYY-MM-DD"
	}
	name := strings.NewReplacer(
		"YYYY", at.Format("2006"),
		"YY", at.Format("06"),
		"MM", at.Format("01"),
		"DD", at.Format("02"),
	).Replace(format)

	if name == "" || name == "." || name == ".." ||
		strings.ContainsAny(name, `/\`) || strings.ContainsRune(name, '\x00') {
		return "", fmt.Errorf("journal date format produces an invalid filename")
	}

	return filepath.Join(folderAbs, name+".md"), nil
}
