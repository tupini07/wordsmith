package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const maxRecentFiles = 20
const maxCursorPositions = 100

// CursorPos stores a cursor position for a file.
type CursorPos struct {
	Line int `json:"line"`
	Col  int `json:"col"`
}

type State struct {
	LastFile        string               `json:"last_file"`
	RecentFiles     []string             `json:"recent_files"`
	CursorPositions map[string]CursorPos `json:"cursor_positions,omitempty"`
}

func statePath() (string, error) {
	// Use XDG state directory: ~/.local/state/wordsmith/state.json
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		stateDir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(stateDir, "wordsmith", "state.json"), nil
}

func Load() (State, error) {
	var st State

	path, err := statePath()
	if err != nil {
		return st, nil
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return st, nil
	}
	if err != nil {
		return st, err
	}

	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, nil // corrupt state file, start fresh
	}

	return st, nil
}

func (s *State) Save() error {
	path, err := statePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write: write to temp file, then rename
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// SetLastFile updates the last opened file and adds it to recent files.
func (s *State) SetLastFile(relPath string) {
	s.LastFile = relPath

	// Remove from recent files if already present
	filtered := make([]string, 0, len(s.RecentFiles))
	for _, f := range s.RecentFiles {
		if f != relPath {
			filtered = append(filtered, f)
		}
	}

	// Prepend to recent files
	s.RecentFiles = append([]string{relPath}, filtered...)

	// Trim to max
	if len(s.RecentFiles) > maxRecentFiles {
		s.RecentFiles = s.RecentFiles[:maxRecentFiles]
	}
}

// SetCursorPos saves the cursor position for a file.
func (s *State) SetCursorPos(relPath string, line, col int) {
	if s.CursorPositions == nil {
		s.CursorPositions = make(map[string]CursorPos)
	}
	s.CursorPositions[relPath] = CursorPos{Line: line, Col: col}

	// Evict oldest entries if over limit
	if len(s.CursorPositions) > maxCursorPositions {
		// Keep only entries that are in recent files
		kept := make(map[string]CursorPos)
		for _, f := range s.RecentFiles {
			if pos, ok := s.CursorPositions[f]; ok {
				kept[f] = pos
			}
		}
		s.CursorPositions = kept
	}
}

// GetCursorPos returns the saved cursor position for a file, or (0,0) if none.
func (s *State) GetCursorPos(relPath string) (line, col int) {
	if s.CursorPositions == nil {
		return 0, 0
	}
	pos, ok := s.CursorPositions[relPath]
	if !ok {
		return 0, 0
	}
	return pos.Line, pos.Col
}
