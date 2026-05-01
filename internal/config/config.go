package config

import (
	_ "embed"
	"os"
	"path/filepath"
	"time"
	"unicode/utf16"

	"gopkg.in/yaml.v3"
)

//go:embed config.example.yaml
var exampleConfig []byte

type Config struct {
	VaultPath       string        `yaml:"vault_path"`
	AutosaveDelay   time.Duration `yaml:"autosave_delay"`
	TabWidth        int           `yaml:"tab_width"`
	ContentWidth    int           `yaml:"content_width"`
	ShowLineNumbers bool          `yaml:"show_line_numbers"`
	Theme           string        `yaml:"theme"`
}

func Default() Config {
	return Config{
		VaultPath:      "",
		AutosaveDelay:  2 * time.Second,
		TabWidth:       4,
		ContentWidth:   80,
		ShowLineNumbers: false,
		Theme:          "gruvbox",
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

// EnsureExists creates the config file from the embedded example template
// if it doesn't already exist. Returns the config file path.
func EnsureExists() (string, error) {
	path, err := configPath()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(path); err == nil {
		return path, nil // already exists
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}

	// Write embedded example config
	if err := os.WriteFile(path, exampleConfig, 0644); err != nil {
		return "", err
	}

	return path, nil
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

	// yaml.v3 doesn't handle time.Duration directly, so we use a helper struct
	var raw struct {
		VaultPath       string `yaml:"vault_path"`
		AutosaveDelay   string `yaml:"autosave_delay"`
		TabWidth        int    `yaml:"tab_width"`
		ContentWidth    int    `yaml:"content_width"`
		ShowLineNumbers bool   `yaml:"show_line_numbers"`
		Theme           string `yaml:"theme"`
	}

	if err := yaml.Unmarshal(data, &raw); err != nil {
		return cfg, err
	}

	if raw.VaultPath != "" {
		cfg.VaultPath = raw.VaultPath
	}
	if raw.AutosaveDelay != "" {
		d, err := time.ParseDuration(raw.AutosaveDelay)
		if err == nil {
			cfg.AutosaveDelay = d
		}
	}
	if raw.TabWidth > 0 {
		cfg.TabWidth = raw.TabWidth
	}
	if raw.ContentWidth > 0 {
		cfg.ContentWidth = raw.ContentWidth
	}
	cfg.ShowLineNumbers = raw.ShowLineNumbers
	if raw.Theme != "" {
		cfg.Theme = raw.Theme
	}

	return cfg, nil
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
