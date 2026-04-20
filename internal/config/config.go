package config

import (
	_ "embed"
	"os"
	"path/filepath"
	"time"

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
