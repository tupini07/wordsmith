package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tupini07/wordsmith/internal/config"
	"github.com/tupini07/wordsmith/internal/editor"
	"github.com/tupini07/wordsmith/internal/filetree"
	"github.com/tupini07/wordsmith/internal/finder"
	"github.com/tupini07/wordsmith/internal/state"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppMode represents the current application mode.
type AppMode int

const (
	ModeEditor AppMode = iota
	ModeFileTree
	ModeFuzzyFinder
	ModeThemePicker
)

const fileTreeWidth = 30

// Model is the top-level application model.
type Model struct {
	mode        AppMode
	editor      editor.Model
	tree        filetree.Model
	finder      finder.Model
	picker      themePicker
	cfg         config.Config
	state       state.State
	width       int
	height      int
	initFile    string
	loaded      bool
	configPath  string // cached config file path for detecting config edits
	activeTheme string // runtime theme name (may differ from config)
}

// New creates a new app model.
func New(cfg config.Config, st state.State, filePath string) Model {
	theme := editor.ThemeByName(cfg.Theme)
	ed := editor.New(cfg.TabWidth, cfg.ContentWidth, theme)
	ed.SetAutosaveDelay(cfg.AutosaveDelay)
	ed.SetVaultPath(cfg.VaultPath)

	tree := filetree.New(cfg.VaultPath)
	tree.SetThemeColors(theme.Bg, theme.ChromeBg, theme.Fg, theme.AccentColor, theme.DirColor)

	fnd := finder.New(cfg.VaultPath)
	fnd.SetThemeColors(theme.Bg, theme.ChromeBg, theme.ChromeBg, theme.Fg, theme.AccentColor, theme.DimColor)

	return Model{
		mode:        ModeEditor,
		editor:      ed,
		tree:        tree,
		finder:      fnd,
		picker:      newThemePicker(),
		cfg:         cfg,
		state:       st,
		initFile:    filePath,
		activeTheme: cfg.Theme,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return editor.FileWatchCmd()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()

		// Load initial file on first resize (we need dimensions first)
		if !m.loaded {
			m.loaded = true
			if m.initFile != "" {
				if err := m.editor.LoadFile(m.initFile); err == nil {
					rel := m.cfg.RelFilePath(m.initFile)
					m.state.SetLastFile(rel)
					// Restore cursor position
					line, col := m.state.GetCursorPos(rel)
					if line > 0 || col > 0 {
						m.editor.SetCursorPos(line, col)
					}
				}
			} else if m.cfg.VaultPath != "" {
				// No file specified — show fuzzy finder
				m.mode = ModeFuzzyFinder
				m.finder.SetRecentFiles(m.state.RecentFiles)
				cmd := m.finder.Show()
				m.editor.SetFocused(false)
				return m, cmd
			}
		}
		return m, nil

	case finder.FileScanCompleteMsg:
		m.finder.HandleScanComplete(msg.Files)
		return m, nil

	case finder.FileCreateMsg:
		return m.createAndOpenFile(msg.Path)

	case finder.FileRenameMsg:
		return m.renameFile(msg.OldPath, msg.NewPath)

	case filetree.CreateInDirMsg:
		// Open finder pre-filled with the directory path
		m.mode = ModeFuzzyFinder
		m.finder.SetRecentFiles(m.state.RecentFiles)
		cmd := m.finder.ShowWithQuery(msg.Dir)
		m.tree.Hide()
		m.editor.SetFocused(false)
		m.updateSizes()
		return m, cmd

	case finder.FileSelectedMsg:
		return m.openFile(msg.Path)

	case editor.FileWatchTickMsg:
		m.editor, _ = m.editor.Update(msg)
		return m, editor.FileWatchCmd()

	case filetree.FileSelectedMsg:
		return m.openFile(msg.Path)

	case tea.KeyMsg:
		// Theme picker gets first-priority key handling
		if m.mode == ModeThemePicker {
			result, preview := m.picker.HandleKey(msg)
			if preview != "" {
				m.applyTheme(preview)
			}
			if result != nil {
				m.picker.Hide()
				m.mode = ModeEditor
				m.editor.SetFocused(true)
				m.applyTheme(result.theme)
				if result.confirmed {
					m.editor.SetStatus("Theme: " + result.theme)
				}
			}
			return m, nil
		}

		// Global keybindings that work in any mode
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+q"))):
			// Save before quitting
			if m.editor.IsDirty() {
				m.editor.SaveFile()
			}
			return m, tea.Quit

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+e"))):
			if m.mode == ModeFileTree {
				m.mode = ModeEditor
				m.tree.Hide()
				m.editor.SetFocused(true)
			} else {
				m.mode = ModeFileTree
				m.tree.Show()
				m.editor.SetFocused(false)
			}
			m.updateSizes()
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+p"))):
			if m.mode == ModeFuzzyFinder {
				m.mode = ModeEditor
				m.finder.Hide()
				m.editor.SetFocused(true)
				return m, nil
			} else {
				m.mode = ModeFuzzyFinder
				m.finder.SetRecentFiles(m.state.RecentFiles)
				cmd := m.finder.Show()
				m.editor.SetFocused(false)
				return m, cmd
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("f2"))):
			// Open config file for editing
			cfgPath, err := config.EnsureExists()
			if err != nil {
				m.editor.SetStatus("Config error: " + err.Error())
				return m, nil
			}
			// Save current file before switching
			if m.editor.IsDirty() {
				m.editor.SaveFile()
			}
			m.configPath = cfgPath
			return m.openFile(cfgPath)

		case key.Matches(msg, key.NewBinding(key.WithKeys("f3"))):
			// Rename current file
			if fp := m.editor.FilePath(); fp != "" {
				if m.editor.IsDirty() {
					m.editor.SaveFile()
				}
				m.mode = ModeFuzzyFinder
				m.finder.ShowRename(fp)
				m.editor.SetFocused(false)
				m.updateSizes()
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("f4"))):
			// Toggle theme picker
			if m.mode == ModeThemePicker {
				// Close and revert (same as Esc)
				m.applyTheme(m.picker.OriginalTheme())
				m.picker.Hide()
				m.mode = ModeEditor
				m.editor.SetFocused(true)
			} else {
				m.mode = ModeThemePicker
				m.picker.Show(m.activeTheme)
				m.editor.SetFocused(false)
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			if m.mode == ModeThemePicker {
				m.applyTheme(m.picker.OriginalTheme())
				m.picker.Hide()
				m.mode = ModeEditor
				m.editor.SetFocused(true)
				return m, nil
			}
			if m.mode != ModeEditor {
				m.mode = ModeEditor
				m.tree.Hide()
				m.finder.Hide()
				m.editor.SetFocused(true)
				m.updateSizes()
				return m, nil
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
			// Route to editor for copy (don't quit)
			if m.mode == ModeEditor {
				m.editor, _ = m.editor.Update(msg)
			}
			return m, nil
		}
	case tea.MouseMsg:
		// When in file tree mode, clicks in the editor area switch to editor
		if m.mode == ModeFileTree && msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			treeW := fileTreeWidth
			if msg.X >= treeW {
				m.mode = ModeEditor
				m.tree.Hide()
				m.editor.SetFocused(true)
				m.updateSizes()
				// Forward the click to the editor
				m.editor, _ = m.editor.Update(msg)
				return m, nil
			}
		}
		// When in file tree mode, scroll wheel in editor area scrolls the editor
		if m.mode == ModeFileTree &&
			(msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown) {
			treeW := fileTreeWidth
			if msg.X >= treeW {
				m.editor, _ = m.editor.Update(msg)
				return m, nil
			}
		}
	}

	// Route to active component
	var cmd tea.Cmd

	switch m.mode {
	case ModeEditor:
		m.editor, cmd = m.editor.Update(msg)
	case ModeFileTree:
		// Intercept 'l' to reveal current file in tree
		if kmsg, ok := msg.(tea.KeyMsg); ok && kmsg.String() == "l" {
			if fp := m.editor.FilePath(); fp != "" {
				m.tree.RevealFile(fp)
			}
			return m, nil
		}
		m.tree, cmd = m.tree.Update(msg)
	case ModeFuzzyFinder:
		m.finder, cmd = m.finder.Update(msg)
	}

	return m, cmd
}

func (m *Model) openFile(path string) (Model, tea.Cmd) {
	// If we're leaving the config file, save it and reload settings
	prevPath := m.editor.FilePath()
	wasConfig := m.configPath != "" && prevPath == m.configPath

	// Save cursor position for the file we're leaving
	if prevPath != "" && prevPath != m.configPath {
		line, col := m.editor.CursorPos()
		rel := m.cfg.RelFilePath(prevPath)
		m.state.SetCursorPos(rel, line, col)
	}

	if err := m.editor.LoadFile(path); err != nil {
		return *m, nil
	}

	// Don't track the config file as the "last opened file"
	if path != m.configPath {
		rel := m.cfg.RelFilePath(path)
		m.state.SetLastFile(rel)

		// Restore cursor position
		line, col := m.state.GetCursorPos(rel)
		if line > 0 || col > 0 {
			m.editor.SetCursorPos(line, col)
		}
	}

	// Switch back to editor mode
	m.mode = ModeEditor
	m.tree.Hide()
	m.finder.Hide()
	m.editor.SetFocused(true)
	m.updateSizes()

	if wasConfig {
		m.reloadConfig()
	}

	return *m, nil
}

func (m *Model) createAndOpenFile(absPath string) (tea.Model, tea.Cmd) {
	// Validate the path stays inside the vault
	absPath = filepath.Clean(absPath)
	vaultAbs := filepath.Clean(m.cfg.VaultPath)
	if !strings.HasPrefix(absPath, vaultAbs+string(filepath.Separator)) {
		m.editor.SetStatus("Cannot create file outside vault")
		return m, nil
	}

	// Create parent directories
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		m.editor.SetStatus("Error: " + err.Error())
		return m, nil
	}

	// Create file exclusively — never truncate an existing file
	f, err := os.OpenFile(absPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			// File was created between the finder check and now — just open it
			return m.openFile(absPath)
		}
		m.editor.SetStatus("Error: " + err.Error())
		return m, nil
	}
	f.Close()

	// Add to finder's cached file list
	rel := m.cfg.RelFilePath(absPath)
	m.finder.AddFile(rel)

	return m.openFile(absPath)
}

func (m *Model) renameFile(oldPath, newPath string) (tea.Model, tea.Cmd) {
	// Validate new path stays inside the vault
	newPath = filepath.Clean(newPath)
	vaultAbs := filepath.Clean(m.cfg.VaultPath)
	if !strings.HasPrefix(newPath, vaultAbs+string(filepath.Separator)) {
		m.editor.SetStatus("Cannot rename file outside vault")
		m.mode = ModeEditor
		m.editor.SetFocused(true)
		return m, nil
	}

	// Don't rename to same path
	if oldPath == newPath {
		m.mode = ModeEditor
		m.editor.SetFocused(true)
		return m, nil
	}

	// Create parent directories for the new path
	dir := filepath.Dir(newPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		m.editor.SetStatus("Error: " + err.Error())
		m.mode = ModeEditor
		m.editor.SetFocused(true)
		return m, nil
	}

	// Check the destination doesn't already exist
	if _, err := os.Stat(newPath); err == nil {
		m.editor.SetStatus("File already exists")
		m.mode = ModeEditor
		m.editor.SetFocused(true)
		return m, nil
	}

	// Perform the rename
	if err := os.Rename(oldPath, newPath); err != nil {
		m.editor.SetStatus("Rename error: " + err.Error())
		m.mode = ModeEditor
		m.editor.SetFocused(true)
		return m, nil
	}

	// Update the editor to point at the new path
	m.editor.SetFilePath(newPath)
	m.editor.SetStatus("Renamed to " + m.cfg.RelFilePath(newPath))

	// Update recent files state
	rel := m.cfg.RelFilePath(newPath)
	m.state.SetLastFile(rel)

	m.mode = ModeEditor
	m.editor.SetFocused(true)
	m.updateSizes()
	return m, nil
}

func (m *Model) updateSizes() {
	// Reserve 1 line for title, 1 for status, 2 for padding
	editorHeight := m.height - 4
	if editorHeight < 1 {
		editorHeight = 1
	}

	treeW := 0
	if m.tree.IsVisible() {
		treeW = fileTreeWidth
		m.tree.SetSize(treeW, editorHeight)
	}

	editorWidth := m.width - treeW
	if editorWidth < 10 {
		editorWidth = 10
	}

	m.editor.SetSize(editorWidth, editorHeight)
	m.editor.SetFullWidth(m.width)
	m.editor.SetEditorOffset(treeW, 2) // Y=2 for title bar + padding row
	m.finder.SetSize(m.width, m.height)
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Title bar
	title := m.editor.TitleView()
	pad := m.editor.PaddingLine()

	// Editor content
	editorView := m.editor.View()

	// File tree (if visible)
	if m.tree.IsVisible() {
		treeView := m.tree.View()
		// Merge tree and editor side by side (editor height excludes padding)
		editorView = mergeSideBySide(treeView, editorView, m.height-4, m.width, m.editor.ThemeMarginStyle())
	}

	// Status bar
	status := m.editor.StatusView()

	composed := title + "\n" + pad + "\n" + editorView + "\n" + pad + "\n" + status

	// Fuzzy finder overlay
	if m.finder.IsVisible() {
		finderView := m.finder.View()
		if finderView != "" {
			return overlayCenter(composed, finderView, m.width, m.height)
		}
	}

	// Theme picker overlay (renders its own full-screen view)
	if m.picker.IsVisible() {
		theme := editor.ThemeByName(m.activeTheme)
		return m.picker.View(theme, m.width, m.height)
	}

	return composed
}

// SaveState saves the current session state.
func (m Model) SaveState() error {
	// Save current cursor position
	if fp := m.editor.FilePath(); fp != "" && fp != m.configPath {
		line, col := m.editor.CursorPos()
		rel := m.cfg.RelFilePath(fp)
		m.state.SetCursorPos(rel, line, col)
	}
	return m.state.Save()
}

// applyTheme applies a theme by name to all components.
func (m *Model) applyTheme(name string) {
	theme := editor.ThemeByName(name)
	m.activeTheme = name
	m.editor.SetTheme(theme)
	m.tree.SetThemeColors(theme.Bg, theme.ChromeBg, theme.Fg, theme.AccentColor, theme.DirColor)
	m.finder.SetThemeColors(theme.Bg, theme.ChromeBg, theme.ChromeBg, theme.Fg, theme.AccentColor, theme.DimColor)
}

// reloadConfig re-reads the config file and hot-reloads settings that can
// change at runtime (theme, tab width, content width, typewriter highlight,
// autosave delay). vault_path changes require a restart.
func (m *Model) reloadConfig() {
	newCfg, err := config.Load()
	if err != nil {
		m.editor.SetStatus("Config reload failed: " + err.Error())
		return
	}

	// Theme from config always wins on reload (user explicitly saved it)
	if newCfg.Theme != m.activeTheme {
		m.applyTheme(newCfg.Theme)
	}
	if newCfg.TabWidth != m.cfg.TabWidth {
		m.editor.SetTabWidth(newCfg.TabWidth)
	}
	if newCfg.ContentWidth != m.cfg.ContentWidth {
		m.editor.SetContentWidth(newCfg.ContentWidth)
	}
	if newCfg.AutosaveDelay != m.cfg.AutosaveDelay {
		m.editor.SetAutosaveDelay(newCfg.AutosaveDelay)
	}

	m.cfg = newCfg
	m.updateSizes() // content width may have changed
	m.editor.SetStatus("Config reloaded")
}

// mergeSideBySide renders two views side by side, filling to fullWidth.
func mergeSideBySide(left, right string, height, fullWidth int, fillStyle lipgloss.Style) string {
	leftLines := splitLines(left, height)
	rightLines := splitLines(right, height)

	var result string
	for i := 0; i < height; i++ {
		if i > 0 {
			result += "\n"
		}
		l := ""
		r := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		line := l + r
		// Fill any remaining space to the right edge
		lineWidth := lipgloss.Width(line)
		if lineWidth < fullWidth {
			line += fillStyle.Render(strings.Repeat(" ", fullWidth-lineWidth))
		}
		result += line
	}
	return result
}

// overlayCenter places an overlay on top of a background.
func overlayCenter(bg, overlay string, width, height int) string {
	_ = width
	_ = height
	// Simple overlay: just return the background with the overlay rendered on top
	// The finder's View already handles positioning
	bgLines := splitLines(bg, height)
	overlayLines := splitLines(overlay, height)

	var result string
	for i := 0; i < height; i++ {
		if i > 0 {
			result += "\n"
		}
		if i < len(overlayLines) && overlayLines[i] != "" {
			result += overlayLines[i]
		} else if i < len(bgLines) {
			result += bgLines[i]
		}
	}
	return result
}

func splitLines(s string, minLen int) []string {
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	for len(lines) < minLen {
		lines = append(lines, "")
	}
	return lines
}
