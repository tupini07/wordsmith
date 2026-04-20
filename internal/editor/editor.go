package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/tupini07/wordsmith/internal/clipboard"
)

// SaveMsg is sent when the file should be saved.
type SaveMsg struct{}

// SavedMsg is sent after a successful save.
type SavedMsg struct{}

// AutosaveTickMsg triggers an autosave check.
type AutosaveTickMsg struct{}

// FileChangedExternallyMsg is sent when external file change is detected.
type FileChangedExternallyMsg struct{}

// FileWatchTickMsg triggers a periodic check for external file changes.
type FileWatchTickMsg struct{}

// Model is the Bubble Tea model for the editor component.
type Model struct {
	buffer    *Buffer
	keymap    KeyMap
	theme     Theme

	// Cursor position (logical)
	cursorLine int
	cursorCol  int
	preferredCol int // for stable vertical movement

	// Selection: anchor is where selection started, cursor is where it ends.
	// If both are -1, no selection.
	selAnchorLine int
	selAnchorCol  int
	hasSelection  bool

	// Viewport
	scrollOffset int // visual line offset
	width        int
	fullWidth    int // full terminal width (for chrome: title, status, padding)
	height       int // editor area height (excluding title/status)

	// Mouse: editor area offset within terminal for coordinate translation
	editorOffsetX int
	editorOffsetY int
	mouseDown     bool // left button is held (for drag-select)
	clickCount    int  // 1=single, 2=double (word), 3=triple (line)
	lastClickTime time.Time
	lastClickX    int
	lastClickY    int
	// For word/line drag-select: remember the initially selected range anchor
	wordSelAnchorStart int // col start of initially selected word/line
	wordSelAnchorEnd   int // col end of initially selected word/line
	wordSelAnchorLine  int // logical line of initial click

	// Wrapping
	wrap WrapResult

	// File
	filePath     string
	vaultPath    string // vault root for relative path display
	fileMtime    time.Time
	saveStatus   string
	saveStatusAt time.Time

	// Autosave
	autosaveDelay time.Duration
	pendingSave   bool

	// Config
	tabWidth            int
	contentWidth        int
	typewriterHighlight bool

	// External change tracking
	externallyChanged bool // file was modified externally, pending user action

	// State
	focused bool
}

// New creates a new editor model.
func New(tabWidth, contentWidth int, theme Theme) Model {
	return Model{
		buffer:        NewBuffer(),
		keymap:        DefaultKeyMap(),
		theme:         theme,
		selAnchorLine: -1,
		selAnchorCol:  -1,
		autosaveDelay: 2 * time.Second,
		tabWidth:      tabWidth,
		contentWidth:  contentWidth,
		focused:       true,
	}
}

// SetSize sets the editor dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	if m.fullWidth == 0 {
		m.fullWidth = width
	}
	m.rewrap()
}

// SetFullWidth sets the full terminal width (used for chrome elements like title/status bars).
func (m *Model) SetFullWidth(w int) {
	m.fullWidth = w
}

// ThemeMarginStyle returns the editor's margin style (for background fill).
func (m Model) ThemeMarginStyle() lipgloss.Style {
	return m.theme.Margin
}

// SetTheme hot-swaps the color theme.
func (m *Model) SetTheme(t Theme) {
	m.theme = t
}

// SetTabWidth changes the tab/indent width.
func (m *Model) SetTabWidth(w int) {
	m.tabWidth = w
}

// SetContentWidth changes the zen-mode content column width and rewraps.
func (m *Model) SetContentWidth(w int) {
	m.contentWidth = w
	m.rewrap()
}

// SetFilePath sets the current file path.
func (m *Model) SetFilePath(path string) {
	m.filePath = path
}

// SetVaultPath sets the vault root for relative title display.
func (m *Model) SetVaultPath(path string) {
	m.vaultPath = path
}

// FilePath returns the current file path.
func (m Model) FilePath() string {
	return m.filePath
}

// CursorPos returns the current cursor line and column.
func (m Model) CursorPos() (line, col int) {
	return m.cursorLine, m.cursorCol
}

// SetCursorPos moves the cursor to the given line and column, clamping to valid bounds.
func (m *Model) SetCursorPos(line, col int) {
	m.cursorLine = line
	m.cursorCol = col
	m.clampCursor()
	m.ensureCursorVisible()
}

// IsDirty returns true if the buffer has unsaved changes.
func (m Model) IsDirty() bool {
	return m.buffer.IsDirty()
}

// WordCount returns the word count.
func (m Model) WordCount() int {
	return m.buffer.WordCount()
}

// SaveStatus returns the current save status string.
func (m Model) SaveStatus() string {
	if m.saveStatus != "" && time.Since(m.saveStatusAt) < 3*time.Second {
		return m.saveStatus
	}
	if m.buffer.IsDirty() {
		return "Modified"
	}
	return ""
}

// SetAutosaveDelay sets the autosave delay.
func (m *Model) SetAutosaveDelay(d time.Duration) {
	m.autosaveDelay = d
}

// SetEditorOffset tells the editor where its content area starts within
// the terminal, so mouse coordinates can be translated correctly.
func (m *Model) SetEditorOffset(x, y int) {
	m.editorOffsetX = x
	m.editorOffsetY = y
}

// SetTypewriterHighlight enables/disables highlighting of the active line.
func (m *Model) SetTypewriterHighlight(on bool) {
	m.typewriterHighlight = on
}

// SetStatus sets a temporary status message in the status bar.
func (m *Model) SetStatus(msg string) {
	m.saveStatus = msg
	m.saveStatusAt = time.Now()
}

// Theme returns the editor's theme.
func (m Model) Theme() Theme {
	return m.theme
}

// LoadFile loads a file into the buffer.
func (m *Model) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	m.buffer.SetContent(string(data))
	m.filePath = path
	m.cursorLine = 0
	m.cursorCol = 0
	m.hasSelection = false
	m.scrollOffset = 0
	m.rewrap()
	m.updatePreferredCol()

	// Record mtime
	info, err := os.Stat(path)
	if err == nil {
		m.fileMtime = info.ModTime()
	}

	return nil
}

// SaveFile saves the buffer to disk using atomic write.
func (m *Model) SaveFile() error {
	if m.filePath == "" {
		return fmt.Errorf("no file path set")
	}

	// Check for external changes
	info, err := os.Stat(m.filePath)
	if err == nil && !m.fileMtime.IsZero() && info.ModTime().After(m.fileMtime) {
		m.externallyChanged = true
		m.saveStatus = "⚠ File changed externally — Ctrl+S to overwrite, Ctrl+R to reload"
		m.saveStatusAt = time.Now()
		return fmt.Errorf("file changed externally")
	}

	content := m.buffer.Content()

	// Ensure directory exists
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Atomic write: temp file + rename
	tmp := m.filePath + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, m.filePath); err != nil {
		os.Remove(tmp)
		return err
	}

	m.buffer.ClearDirty()
	m.saveStatus = "Saved"
	m.saveStatusAt = time.Now()

	// Update mtime
	info, err = os.Stat(m.filePath)
	if err == nil {
		m.fileMtime = info.ModTime()
	}

	return nil
}

// ForceSaveFile saves the buffer ignoring external modification checks.
func (m *Model) ForceSaveFile() error {
	if m.filePath == "" {
		return fmt.Errorf("no file path set")
	}

	content := m.buffer.Content()

	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp := m.filePath + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, m.filePath); err != nil {
		os.Remove(tmp)
		return err
	}

	m.buffer.ClearDirty()
	m.externallyChanged = false
	m.saveStatus = "Saved (overwritten)"
	m.saveStatusAt = time.Now()

	info, err := os.Stat(m.filePath)
	if err == nil {
		m.fileMtime = info.ModTime()
	}
	return nil
}

// ReloadFile re-reads the current file from disk, replacing buffer contents.
func (m *Model) ReloadFile() error {
	if m.filePath == "" {
		return fmt.Errorf("no file path set")
	}
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	m.buffer.SetContent(string(data))
	m.externallyChanged = false
	m.rewrap()

	// Clamp cursor to new content bounds
	if m.cursorLine >= m.buffer.LineCount() {
		m.cursorLine = m.buffer.LineCount() - 1
	}
	if m.cursorLine < 0 {
		m.cursorLine = 0
	}
	if m.cursorCol > m.buffer.LineLen(m.cursorLine) {
		m.cursorCol = m.buffer.LineLen(m.cursorLine)
	}
	m.clearSelection()
	m.updatePreferredCol()
	m.ensureCursorVisible()

	info, err := os.Stat(m.filePath)
	if err == nil {
		m.fileMtime = info.ModTime()
	}

	m.saveStatus = "Reloaded"
	m.saveStatusAt = time.Now()
	return nil
}

// CheckExternalChange checks if the file was modified externally.
// Returns true if it was changed and the editor should respond.
func (m *Model) CheckExternalChange() bool {
	if m.filePath == "" || m.fileMtime.IsZero() {
		return false
	}
	info, err := os.Stat(m.filePath)
	if err != nil {
		return false
	}
	return info.ModTime().After(m.fileMtime)
}

// IsExternallyChanged returns whether an external change has been detected.
func (m Model) IsExternallyChanged() bool {
	return m.externallyChanged
}

// Content returns the buffer content as a string.
func (m Model) Content() string {
	return m.buffer.Content()
}

func (m *Model) rewrap() {
	editWidth := m.editWidth()
	m.wrap = WrapLines(m.bufferLines(), editWidth)
}

func (m Model) editWidth() int {
	w := m.width
	if m.contentWidth > 0 && m.contentWidth < w {
		w = m.contentWidth
	}
	if w <= 0 {
		w = 80
	}
	return w
}

func (m Model) bufferLines() [][]rune {
	lines := make([][]rune, m.buffer.LineCount())
	for i := range lines {
		lines[i] = m.buffer.Line(i)
	}
	return lines
}

// SetFocused sets whether the editor is focused.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.MouseMsg:
		return m.handleMouseMsg(msg)
	case AutosaveTickMsg:
		if m.pendingSave && m.buffer.IsDirty() {
			m.pendingSave = false
			if err := m.SaveFile(); err != nil {
				m.saveStatus = "Save failed"
				m.saveStatusAt = time.Now()
			}
		}
		return m, nil

	case FileWatchTickMsg:
		if m.CheckExternalChange() {
			if !m.buffer.IsDirty() {
				// No local edits — auto-reload silently
				m.ReloadFile()
			} else if !m.externallyChanged {
				// Local edits exist — flag for user action
				m.externallyChanged = true
				m.saveStatus = "⚠ File changed externally — Ctrl+S to overwrite, Ctrl+R to reload"
				m.saveStatusAt = time.Now()
			}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	km := m.keymap

	switch {
	// Quit
	case key.Matches(msg, km.Quit):
		return m, tea.Quit

	// Save (force-save if externally changed)
	case key.Matches(msg, km.Save):
		if m.externallyChanged {
			if err := m.ForceSaveFile(); err != nil {
				m.saveStatus = "Save failed"
				m.saveStatusAt = time.Now()
			}
		} else {
			if err := m.SaveFile(); err != nil && !m.externallyChanged {
				m.saveStatus = "Save failed"
				m.saveStatusAt = time.Now()
			}
		}
		return m, nil

	// Reload from disk
	case key.Matches(msg, km.Reload):
		if err := m.ReloadFile(); err != nil {
			m.SetStatus("Reload failed: " + err.Error())
		}
		return m, nil

	// Undo
	case key.Matches(msg, km.Undo):
		if line, col, ok := m.buffer.Undo(); ok {
			m.cursorLine = line
			m.cursorCol = col
			m.clearSelection()
			m.rewrap()
			m.updatePreferredCol()
			m.ensureCursorVisible()
		}
		return m, nil

	// Redo
	case key.Matches(msg, km.Redo):
		if line, col, ok := m.buffer.Redo(); ok {
			m.cursorLine = line
			m.cursorCol = col
			m.clearSelection()
			m.rewrap()
			m.updatePreferredCol()
			m.ensureCursorVisible()
		}
		return m, nil

	// Bold
	case key.Matches(msg, km.Bold):
		m.wrapWithMarkers("**")
		return m, m.scheduleAutosave()

	// Italic
	case key.Matches(msg, km.Italic):
		m.wrapWithMarkers("*")
		return m, m.scheduleAutosave()

	// Link
	case key.Matches(msg, km.Link):
		m.insertLink()
		return m, m.scheduleAutosave()

	// Footnote
	case key.Matches(msg, km.Footnote):
		m.handleFootnote()
		return m, m.scheduleAutosave()

	// Movement with selection
	case key.Matches(msg, km.ShiftUp):
		m.startOrExtendSelection()
		m.moveCursorUp()
		return m, nil
	case key.Matches(msg, km.ShiftDown):
		m.startOrExtendSelection()
		m.moveCursorDown()
		return m, nil
	case key.Matches(msg, km.ShiftLeft):
		m.startOrExtendSelection()
		m.moveCursorLeft()
		return m, nil
	case key.Matches(msg, km.ShiftRight):
		m.startOrExtendSelection()
		m.moveCursorRight()
		return m, nil
	case key.Matches(msg, km.ShiftHome):
		m.startOrExtendSelection()
		m.moveToVisualLineStart()
		return m, nil
	case key.Matches(msg, km.ShiftEnd):
		m.startOrExtendSelection()
		m.moveToVisualLineEnd()
		return m, nil

	// Movement (clears selection)
	case key.Matches(msg, km.Up):
		m.clearSelection()
		m.moveCursorUp()
		return m, nil
	case key.Matches(msg, km.Down):
		m.clearSelection()
		m.moveCursorDown()
		return m, nil
	case key.Matches(msg, km.Left):
		if m.hasSelection {
			sl, sc, _, _ := m.selectionRange()
			m.cursorLine, m.cursorCol = sl, sc
			m.clearSelection()
		} else {
			m.moveCursorLeft()
		}
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return m, nil
	case key.Matches(msg, km.Right):
		if m.hasSelection {
			_, _, el, ec := m.selectionRange()
			m.cursorLine, m.cursorCol = el, ec
			m.clearSelection()
		} else {
			m.moveCursorRight()
		}
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return m, nil
	case key.Matches(msg, km.Home):
		m.clearSelection()
		m.moveToVisualLineStart()
		return m, nil
	case key.Matches(msg, km.End):
		m.clearSelection()
		m.moveToVisualLineEnd()
		return m, nil
	case key.Matches(msg, km.PgUp):
		m.clearSelection()
		for i := 0; i < m.height; i++ {
			m.moveCursorUp()
		}
		return m, nil
	case key.Matches(msg, km.PgDn):
		m.clearSelection()
		for i := 0; i < m.height; i++ {
			m.moveCursorDown()
		}
		return m, nil
	case key.Matches(msg, km.CtrlLeft):
		m.clearSelection()
		m.moveCursorWordLeft()
		return m, nil
	case key.Matches(msg, km.CtrlRight):
		m.clearSelection()
		m.moveCursorWordRight()
		return m, nil
	case key.Matches(msg, km.CtrlShiftLeft):
		m.startOrExtendSelection()
		m.moveCursorWordLeft()
		return m, nil
	case key.Matches(msg, km.CtrlShiftRight):
		m.startOrExtendSelection()
		m.moveCursorWordRight()
		return m, nil
	case key.Matches(msg, km.GoBottom):
		m.clearSelection()
		m.cursorLine = m.buffer.LineCount() - 1
		m.cursorCol = m.buffer.LineLen(m.cursorLine)
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return m, nil
	case key.Matches(msg, km.GoTop):
		m.clearSelection()
		m.cursorLine = 0
		m.cursorCol = 0
		m.scrollOffset = 0
		m.updatePreferredCol()
		return m, nil

	case key.Matches(msg, km.SelectAll):
		lastLine := m.buffer.LineCount() - 1
		m.selAnchorLine = 0
		m.selAnchorCol = 0
		m.cursorLine = lastLine
		m.cursorCol = m.buffer.LineLen(lastLine)
		m.hasSelection = true
		m.ensureCursorVisible()
		m.updatePreferredCol()
		return m, nil

	// Editing
	case key.Matches(msg, km.Backspace):
		if m.hasSelection {
			m.deleteSelection()
		} else {
			newLine, newCol, ok := m.buffer.Backspace(m.cursorLine, m.cursorCol)
			if ok {
				m.cursorLine = newLine
				m.cursorCol = newCol
			}
		}
		m.clearSelection()
		m.rewrap()
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return m, m.scheduleAutosave()

	case key.Matches(msg, km.Delete):
		if m.hasSelection {
			m.deleteSelection()
		} else {
			m.buffer.DeleteChar(m.cursorLine, m.cursorCol)
		}
		m.clearSelection()
		m.rewrap()
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return m, m.scheduleAutosave()

	case key.Matches(msg, km.DeleteWordBack):
		if m.hasSelection {
			m.deleteSelection()
		} else {
			// Delete from cursor back to start of previous word
			startCol := m.cursorCol
			line := m.buffer.Line(m.cursorLine)
			col := m.cursorCol
			// Skip whitespace backwards
			for col > 0 && unicode.IsSpace(line[col-1]) {
				col--
			}
			// Skip word chars backwards
			for col > 0 && !unicode.IsSpace(line[col-1]) {
				col--
			}
			if col < startCol {
				m.buffer.DeleteRange(m.cursorLine, col, m.cursorLine, startCol)
				m.cursorCol = col
			} else if m.cursorLine > 0 {
				// At start of line — join with previous line (delete the newline)
				prevLine := m.cursorLine - 1
				prevLen := m.buffer.LineLen(prevLine)
				m.buffer.DeleteRange(prevLine, prevLen, m.cursorLine, 0)
				m.cursorLine = prevLine
				m.cursorCol = prevLen
			}
		}
		m.clearSelection()
		m.rewrap()
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return m, m.scheduleAutosave()

	// Delete word forward (alt+d)
	case msg.Alt && msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && (msg.Runes[0] == 'd' || msg.Runes[0] == 'D'):
		if m.hasSelection {
			m.deleteSelection()
		} else {
			line := m.buffer.Line(m.cursorLine)
			lineLen := len(line)
			col := m.cursorCol
			// Skip whitespace forward
			for col < lineLen && unicode.IsSpace(line[col]) {
				col++
			}
			// Skip word chars forward
			for col < lineLen && !unicode.IsSpace(line[col]) {
				col++
			}
			if col > m.cursorCol {
				m.buffer.DeleteRange(m.cursorLine, m.cursorCol, m.cursorLine, col)
			} else if m.cursorLine < m.buffer.LineCount()-1 {
				// At end of line — join with next line
				m.buffer.DeleteRange(m.cursorLine, m.cursorCol, m.cursorLine+1, 0)
			}
		}
		m.clearSelection()
		m.rewrap()
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return m, m.scheduleAutosave()

	case key.Matches(msg, km.Copy):
		if m.hasSelection {
			clipboard.Write(m.selectedText())
		} else {
			// No selection — copy the current line
			line := string(m.buffer.Line(m.cursorLine))
			clipboard.Write(line)
		}
		return m, nil

	case key.Matches(msg, km.Cut):
		if m.hasSelection {
			clipboard.Write(m.selectedText())
			m.deleteSelection()
		} else {
			// No selection — cut the current line
			line := string(m.buffer.Line(m.cursorLine))
			clipboard.Write(line)
			if m.buffer.LineCount() > 1 {
				if m.cursorLine < m.buffer.LineCount()-1 {
					m.buffer.DeleteRange(m.cursorLine, 0, m.cursorLine+1, 0)
				} else {
					prevLen := m.buffer.LineLen(m.cursorLine - 1)
					m.buffer.DeleteRange(m.cursorLine-1, prevLen, m.cursorLine, m.buffer.LineLen(m.cursorLine))
					m.cursorLine--
				}
				m.cursorCol = 0
			} else {
				m.buffer.DeleteRange(0, 0, 0, m.buffer.LineLen(0))
				m.cursorCol = 0
			}
		}
		m.clearSelection()
		m.rewrap()
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return m, m.scheduleAutosave()

	case key.Matches(msg, km.Paste):
		text, err := clipboard.Read()
		if err == nil && text != "" {
			if m.hasSelection {
				m.deleteSelection()
			}
			m.buffer.BeginUndoGroup()
			runes := normalizePastedText([]rune(text))
			for _, r := range runes {
				if r == '\n' {
					m.buffer.InsertNewline(m.cursorLine, m.cursorCol)
					m.cursorLine++
					m.cursorCol = 0
				} else if r == '\r' {
					// skip
				} else {
					m.buffer.InsertChar(m.cursorLine, m.cursorCol, r)
					m.cursorCol++
				}
			}
			m.buffer.EndUndoGroup()
		}
		m.clearSelection()
		m.rewrap()
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return m, m.scheduleAutosave()

	case key.Matches(msg, km.Enter):
		m.deleteSelection()
		m.buffer.InsertNewline(m.cursorLine, m.cursorCol)
		m.cursorLine++
		m.cursorCol = 0
		m.clearSelection()
		m.rewrap()
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return m, m.scheduleAutosave()

	case key.Matches(msg, km.Tab):
		m.indent()
		m.rewrap()
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return m, m.scheduleAutosave()

	case key.Matches(msg, km.ShiftTab):
		m.outdent()
		m.rewrap()
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return m, m.scheduleAutosave()

	default:
		// Regular character input
		if msg.Type == tea.KeyRunes {
			m.deleteSelection()

			runes := msg.Runes
			// Detect paste: either Bubble Tea's Paste flag is set, or
			// the runes contain newlines (Windows Terminal sends pasted
			// text as plain runes without the bracketed-paste flag).
			isPaste := msg.Paste || runesContainNewline(runes)
			if isPaste {
				runes = normalizePastedText(runes)
				m.buffer.BeginUndoGroup()
			}
			for _, r := range runes {
				if r == '\n' {
					m.buffer.InsertNewline(m.cursorLine, m.cursorCol)
					m.cursorLine++
					m.cursorCol = 0
				} else if r == '\r' {
					// skip carriage returns
				} else {
					m.buffer.InsertChar(m.cursorLine, m.cursorCol, r)
					m.cursorCol++
				}
			}
			if isPaste {
				m.buffer.EndUndoGroup()
			}
			m.clearSelection()
			m.rewrap()
			m.updatePreferredCol()
			m.ensureCursorVisible()
			return m, m.scheduleAutosave()
		}
		if msg.Type == tea.KeySpace {
			m.deleteSelection()
			m.buffer.InsertChar(m.cursorLine, m.cursorCol, ' ')
			m.cursorCol++
			m.clearSelection()
			m.rewrap()
			m.updatePreferredCol()
			m.ensureCursorVisible()
			return m, m.scheduleAutosave()
		}
	}

	return m, nil
}

// handleMouseMsg processes mouse events: click to place cursor, drag to
// select, and scroll wheel to scroll.
func (m Model) handleMouseMsg(msg tea.MouseMsg) (Model, tea.Cmd) {
	// Translate terminal coordinates to editor-local coordinates
	localX := msg.X - m.editorOffsetX
	localY := msg.Y - m.editorOffsetY

	// Account for zen-mode left margin
	leftMargin := 0
	if m.contentWidth > 0 && m.width > m.contentWidth {
		leftMargin = (m.width - m.contentWidth) / 2
	}
	col := localX - leftMargin

	// Visual row = local screen Y + scroll offset
	visRow := localY + m.scrollOffset

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.scrollOffset -= 3
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
		return m, nil

	case tea.MouseButtonWheelDown:
		maxScroll := m.wrap.VisualLineCount() - m.height
		if maxScroll < 0 {
			maxScroll = 0
		}
		m.scrollOffset += 3
		if m.scrollOffset > maxScroll {
			m.scrollOffset = maxScroll
		}
		return m, nil

	case tea.MouseButtonLeft:
		switch msg.Action {
		case tea.MouseActionPress:
			// Out-of-bounds check
			if localX < 0 || localY < 0 || localY >= m.height || col < 0 {
				return m, nil
			}
			if visRow < 0 || visRow >= m.wrap.VisualLineCount() {
				return m, nil
			}

			logLine, logCol := m.wrap.VisualToLogical(visRow, col)

			// Detect multi-click (double/triple)
			now := time.Now()
			const multiClickWindow = 400 * time.Millisecond
			sameSpot := msg.X == m.lastClickX && msg.Y == m.lastClickY
			if sameSpot && now.Sub(m.lastClickTime) < multiClickWindow {
				m.clickCount++
				if m.clickCount > 3 {
					m.clickCount = 3
				}
			} else {
				m.clickCount = 1
			}
			m.lastClickTime = now
			m.lastClickX = msg.X
			m.lastClickY = msg.Y
			m.mouseDown = true

			switch m.clickCount {
			case 2: // Double-click: select word
				ws, we := m.wordBoundsAt(logLine, logCol)
				m.selAnchorLine = logLine
				m.selAnchorCol = ws
				m.cursorLine = logLine
				m.cursorCol = we
				m.hasSelection = true
				m.wordSelAnchorStart = ws
				m.wordSelAnchorEnd = we
				m.wordSelAnchorLine = logLine
			case 3: // Triple-click: select whole logical line
				m.selAnchorLine = logLine
				m.selAnchorCol = 0
				m.cursorLine = logLine
				m.cursorCol = m.buffer.LineLen(logLine)
				m.hasSelection = true
				m.wordSelAnchorStart = 0
				m.wordSelAnchorEnd = m.buffer.LineLen(logLine)
				m.wordSelAnchorLine = logLine
			default: // Single click: place cursor
				m.cursorLine = logLine
				m.cursorCol = logCol
				m.clampCursor()
				m.clearSelection()
			}
			m.updatePreferredCol()
			return m, nil

		case tea.MouseActionMotion:
			if !m.mouseDown {
				return m, nil
			}
			// Clamp Y to editor bounds for drag
			if localY < 0 {
				localY = 0
			}
			if localY >= m.height {
				localY = m.height - 1
			}
			visRow = localY + m.scrollOffset
			if visRow < 0 {
				visRow = 0
			}
			if visRow >= m.wrap.VisualLineCount() {
				visRow = m.wrap.VisualLineCount() - 1
			}
			if col < 0 {
				col = 0
			}

			logLine, logCol := m.wrap.VisualToLogical(visRow, col)

			switch m.clickCount {
			case 2: // Word-granularity drag
				ws, we := m.wordBoundsAt(logLine, logCol)
				anchorLine := m.wordSelAnchorLine
				anchorStart := m.wordSelAnchorStart
				anchorEnd := m.wordSelAnchorEnd
				// Determine direction: is drag target before or after anchor?
				before := logLine < anchorLine || (logLine == anchorLine && ws < anchorStart)
				if before {
					m.selAnchorLine = anchorLine
					m.selAnchorCol = anchorEnd
					m.cursorLine = logLine
					m.cursorCol = ws
				} else {
					m.selAnchorLine = anchorLine
					m.selAnchorCol = anchorStart
					m.cursorLine = logLine
					m.cursorCol = we
				}
				m.hasSelection = true
			case 3: // Line-granularity drag
				anchorLine := m.wordSelAnchorLine
				if logLine < anchorLine {
					m.selAnchorLine = anchorLine
					m.selAnchorCol = m.buffer.LineLen(anchorLine)
					m.cursorLine = logLine
					m.cursorCol = 0
				} else {
					m.selAnchorLine = anchorLine
					m.selAnchorCol = 0
					m.cursorLine = logLine
					m.cursorCol = m.buffer.LineLen(logLine)
				}
				m.hasSelection = true
			default: // Character-granularity drag
				if !m.hasSelection {
					m.selAnchorLine = m.cursorLine
					m.selAnchorCol = m.cursorCol
					m.hasSelection = true
				}
				m.cursorLine = logLine
				m.cursorCol = logCol
				m.clampCursor()
			}
			m.updatePreferredCol()
			m.ensureCursorVisible()
			return m, nil

		case tea.MouseActionRelease:
			m.mouseDown = false
			return m, nil
		}

	case tea.MouseButtonNone:
		if msg.Action == tea.MouseActionRelease {
			m.mouseDown = false
		}
		return m, nil
	}

	return m, nil
}

// Cursor movement

// updatePreferredCol computes and stores the visual column for stable
// vertical cursor movement across soft-wrapped lines.
func (m *Model) updatePreferredCol() {
	_, visCol := m.wrap.LogicalToVisual(m.cursorLine, m.cursorCol)
	m.preferredCol = visCol
}

// snapToVisualRow ensures the cursor actually lands on the given target visual
// row. When VisualToLogical returns a boundary position that LogicalToVisual
// maps to the next visual row, this steps the cursor back so it stays on the
// intended row.
func (m *Model) snapToVisualRow(targetRow int) {
	actual, _ := m.wrap.LogicalToVisual(m.cursorLine, m.cursorCol)
	if actual == targetRow {
		return
	}
	// Place cursor at the last rune position of the target visual line
	vl := m.wrap.VisualLines[targetRow]
	end := len(vl.Runes)
	lr := m.wrap.LineMap[vl.LogicalLine]
	if targetRow < lr.End-1 && end > 0 {
		end-- // stay on this visual row, not the next one's start
	}
	m.cursorLine = vl.LogicalLine
	m.cursorCol = vl.LogicalCol + end
	m.clampCursor()
}

func (m *Model) moveCursorUp() {
	visRow, _ := m.wrap.LogicalToVisual(m.cursorLine, m.cursorCol)
	if visRow > 0 {
		targetRow := visRow - 1
		logLine, logCol := m.wrap.VisualToLogical(targetRow, m.preferredCol)
		m.cursorLine = logLine
		m.cursorCol = logCol
		m.clampCursor()
		m.snapToVisualRow(targetRow)
		m.ensureCursorVisible()
	}
}

func (m *Model) moveCursorDown() {
	visRow, _ := m.wrap.LogicalToVisual(m.cursorLine, m.cursorCol)
	if visRow < m.wrap.VisualLineCount()-1 {
		targetRow := visRow + 1
		logLine, logCol := m.wrap.VisualToLogical(targetRow, m.preferredCol)
		m.cursorLine = logLine
		m.cursorCol = logCol
		m.clampCursor()
		m.snapToVisualRow(targetRow)
		m.ensureCursorVisible()
	}
}

func (m *Model) moveCursorLeft() {
	if m.cursorCol > 0 {
		m.cursorCol--
	} else if m.cursorLine > 0 {
		m.cursorLine--
		m.cursorCol = m.buffer.LineLen(m.cursorLine)
	}
	m.updatePreferredCol()
	m.ensureCursorVisible()
}

func (m *Model) moveCursorRight() {
	if m.cursorCol < m.buffer.LineLen(m.cursorLine) {
		m.cursorCol++
	} else if m.cursorLine < m.buffer.LineCount()-1 {
		m.cursorLine++
		m.cursorCol = 0
	}
	m.updatePreferredCol()
	m.ensureCursorVisible()
}

// moveToVisualLineStart moves cursor to the start of the current visual
// (soft-wrapped) line.
func (m *Model) moveToVisualLineStart() {
	visRow, _ := m.wrap.LogicalToVisual(m.cursorLine, m.cursorCol)
	if visRow >= 0 && visRow < m.wrap.VisualLineCount() {
		vl := m.wrap.VisualLines[visRow]
		m.cursorCol = vl.LogicalCol
	} else {
		m.cursorCol = 0
	}
	m.updatePreferredCol()
}

// moveToVisualLineEnd moves cursor to the end of the current visual
// (soft-wrapped) line.
func (m *Model) moveToVisualLineEnd() {
	visRow, _ := m.wrap.LogicalToVisual(m.cursorLine, m.cursorCol)
	if visRow >= 0 && visRow < m.wrap.VisualLineCount() {
		vl := m.wrap.VisualLines[visRow]
		endCol := vl.LogicalCol + len(vl.Runes)

		// If this is not the last visual sub-line for this logical line,
		// trim trailing whitespace so the cursor lands on the last visible
		// character rather than wrapping to the next sub-line.
		isLastSubLine := visRow+1 >= m.wrap.VisualLineCount() ||
			m.wrap.VisualLines[visRow+1].LogicalLine != vl.LogicalLine
		if !isLastSubLine {
			runes := vl.Runes
			trimmed := len(runes)
			for trimmed > 0 && unicode.IsSpace(runes[trimmed-1]) {
				trimmed--
			}
			if trimmed > 0 {
				endCol = vl.LogicalCol + trimmed
			}
		}

		m.cursorCol = endCol
	} else {
		m.cursorCol = m.buffer.LineLen(m.cursorLine)
	}
	m.updatePreferredCol()
}

func (m *Model) moveCursorWordLeft() {
	line := m.buffer.Line(m.cursorLine)
	col := m.cursorCol

	// If at start of line, jump to end of previous line
	if col == 0 && m.cursorLine > 0 {
		m.cursorLine--
		m.cursorCol = m.buffer.LineLen(m.cursorLine)
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return
	}

	// Skip whitespace backwards
	for col > 0 && unicode.IsSpace(line[col-1]) {
		col--
	}
	// Skip word chars backwards → land at start of word
	if col > 0 && isWordChar(line[col-1]) {
		for col > 0 && isWordChar(line[col-1]) {
			col--
		}
	} else if col > 0 {
		// Skip a single punctuation char
		col--
	}
	m.cursorCol = col
	m.updatePreferredCol()
	m.ensureCursorVisible()
}

// wordBoundsAt returns the start (inclusive) and end (exclusive) column
// of the word at the given column on the given line.
func (m *Model) wordBoundsAt(line, col int) (int, int) {
	runes := m.buffer.Line(line)
	n := len(runes)
	if n == 0 {
		return 0, 0
	}
	if col >= n {
		col = n - 1
	}
	if col < 0 {
		col = 0
	}
	charAtCol := runes[col]
	start, end := col, col
	if isWordChar(charAtCol) {
		for start > 0 && isWordChar(runes[start-1]) {
			start--
		}
		for end < n && isWordChar(runes[end]) {
			end++
		}
	} else if unicode.IsSpace(charAtCol) {
		// On whitespace — select the whitespace run
		for start > 0 && unicode.IsSpace(runes[start-1]) {
			start--
		}
		for end < n && unicode.IsSpace(runes[end]) {
			end++
		}
	} else {
		// On punctuation — select the single character
		end = col + 1
	}
	return start, end
}

func (m *Model) moveCursorWordRight() {
	line := m.buffer.Line(m.cursorLine)
	col := m.cursorCol
	n := len(line)

	// If at end of line, jump to start of next line
	if col >= n && m.cursorLine < m.buffer.LineCount()-1 {
		m.cursorLine++
		m.cursorCol = 0
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return
	}

	// Skip whitespace forward
	for col < n && unicode.IsSpace(line[col]) {
		col++
	}
	// Skip word chars forward → land at end of word
	if col < n && isWordChar(line[col]) {
		for col < n && isWordChar(line[col]) {
			col++
		}
	} else if col < n {
		// Skip a single punctuation char
		col++
	}
	m.cursorCol = col
	m.updatePreferredCol()
	m.ensureCursorVisible()
}

func (m *Model) clampCursor() {
	if m.cursorLine < 0 {
		m.cursorLine = 0
	}
	if m.cursorLine >= m.buffer.LineCount() {
		m.cursorLine = m.buffer.LineCount() - 1
	}
	if m.cursorCol < 0 {
		m.cursorCol = 0
	}
	lineLen := m.buffer.LineLen(m.cursorLine)
	if m.cursorCol > lineLen {
		m.cursorCol = lineLen
	}
}

func (m *Model) ensureCursorVisible() {
	if m.height <= 0 {
		return
	}
	visRow, _ := m.wrap.LogicalToVisual(m.cursorLine, m.cursorCol)

	// Typewriter scrolling: keep cursor at ~75% of viewport height.
	// Only activates once there's enough content above to fill the space.
	anchor := m.height * 3 / 4
	if anchor < 1 {
		anchor = 1
	}
	ideal := visRow - anchor
	if ideal < 0 {
		ideal = 0
	}
	maxScroll := m.wrap.VisualLineCount() - m.height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if ideal > maxScroll {
		ideal = maxScroll
	}
	m.scrollOffset = ideal
}

// Selection

func (m *Model) startOrExtendSelection() {
	if !m.hasSelection {
		m.selAnchorLine = m.cursorLine
		m.selAnchorCol = m.cursorCol
		m.hasSelection = true
	}
}

func (m *Model) clearSelection() {
	m.hasSelection = false
	m.selAnchorLine = -1
	m.selAnchorCol = -1
}

func (m *Model) selectionRange() (startLine, startCol, endLine, endCol int) {
	if !m.hasSelection {
		return -1, -1, -1, -1
	}
	startLine, startCol = m.selAnchorLine, m.selAnchorCol
	endLine, endCol = m.cursorLine, m.cursorCol
	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, startCol, endLine, endCol = endLine, endCol, startLine, startCol
	}
	return
}

func (m *Model) selectedText() string {
	if !m.hasSelection {
		return ""
	}
	sl, sc, el, ec := m.selectionRange()
	// Read from buffer without deleting
	var sb strings.Builder
	if sl == el {
		line := m.buffer.Line(sl)
		if sc < len(line) && ec <= len(line) {
			sb.WriteString(string(line[sc:ec]))
		}
	} else {
		first := m.buffer.Line(sl)
		if sc < len(first) {
			sb.WriteString(string(first[sc:]))
		}
		sb.WriteRune('\n')
		for i := sl + 1; i < el; i++ {
			sb.WriteString(string(m.buffer.Line(i)))
			sb.WriteRune('\n')
		}
		last := m.buffer.Line(el)
		if ec <= len(last) {
			sb.WriteString(string(last[:ec]))
		}
	}
	return sb.String()
}

func (m *Model) deleteSelection() {
	if !m.hasSelection {
		return
	}
	sl, sc, el, ec := m.selectionRange()
	m.buffer.DeleteRange(sl, sc, el, ec)
	m.cursorLine = sl
	m.cursorCol = sc
	m.clearSelection()
	m.rewrap()
	m.updatePreferredCol()
}

// Markdown formatting

func (m *Model) wrapWithMarkers(marker string) {
	markerLen := len([]rune(marker))

	if m.hasSelection {
		text := m.selectedText()
		sl, sc, _, _ := m.selectionRange()

		m.buffer.BeginUndoGroup()
		// Toggle: if selection is already wrapped, unwrap it
		if len(text) >= 2*len(marker) &&
			text[:len(marker)] == marker && text[len(text)-len(marker):] == marker {
			inner := text[len(marker) : len(text)-len(marker)]
			m.deleteSelection()
			endLine, endCol := m.buffer.InsertText(sl, sc, inner)
			m.cursorLine = endLine
			m.cursorCol = endCol
		} else {
			m.deleteSelection()
			wrapped := marker + text + marker
			endLine, endCol := m.buffer.InsertText(sl, sc, wrapped)
			m.cursorLine = endLine
			m.cursorCol = endCol
		}
		m.buffer.EndUndoGroup()
		m.clearSelection()
		m.rewrap()
		m.updatePreferredCol()
		m.ensureCursorVisible()
		return
	}

	// No selection — behavior depends on cursor context
	line := m.buffer.Line(m.cursorLine)
	lineLen := len(line)

	// Check if cursor is sitting right between empty markers (e.g. **|** or *|*)
	// In this case, remove the empty markers entirely.
	if m.cursorCol >= markerLen && m.cursorCol+markerLen <= lineLen {
		before := string(line[m.cursorCol-markerLen : m.cursorCol])
		after := string(line[m.cursorCol : m.cursorCol+markerLen])
		if before == marker && after == marker {
			m.buffer.DeleteRange(m.cursorLine, m.cursorCol-markerLen, m.cursorLine, m.cursorCol+markerLen)
			m.cursorCol -= markerLen
			m.rewrap()
			m.updatePreferredCol()
			return
		}
	}

	// Check if cursor is right before closing markers with a word char to the left.
	// This handles the case where the user is at the end of formatted text
	// (e.g., "**some text here|**") and wants to exit the markers.
	if m.cursorCol > 0 && m.cursorCol+markerLen <= lineLen && isWordChar(line[m.cursorCol-1]) {
		after := string(line[m.cursorCol : m.cursorCol+markerLen])
		if after == marker {
			m.cursorCol += markerLen
			m.rewrap()
			m.updatePreferredCol()
			return
		}
	}

	// Try to find the word under/before cursor.
	// WordAt won't find it if cursor is at end (past last char), so check col-1
	// — but only if col-1 is actually a word character (avoid finding distant words).
	start, end := m.buffer.WordAt(m.cursorLine, m.cursorCol)
	if start == end && m.cursorCol > 0 && m.cursorCol-1 < lineLen && isWordChar(line[m.cursorCol-1]) {
		start, end = m.buffer.WordAt(m.cursorLine, m.cursorCol-1)
	}

	if start < end {
		// We have a word. Check if it's already wrapped with markers.
		mStart := start - markerLen
		mEnd := end + markerLen
		alreadyWrapped := mStart >= 0 && mEnd <= lineLen &&
			string(line[mStart:start]) == marker &&
			string(line[end:mEnd]) == marker

		// For bold (**), avoid stripping when part of bold-italic (***)
		if alreadyWrapped && marker == "**" {
			if (mStart > 0 && line[mStart-1] == '*') ||
				(mEnd < lineLen && line[mEnd] == '*') {
				alreadyWrapped = false
			}
		}

		if alreadyWrapped {
			// Cursor at end of word: just move past closing markers
			if m.cursorCol >= end {
				m.cursorCol = mEnd
			} else {
				// Cursor inside word: remove the markers (unwrap)
				m.buffer.BeginUndoGroup()
				m.buffer.DeleteRange(m.cursorLine, end, m.cursorLine, mEnd)
				m.buffer.DeleteRange(m.cursorLine, mStart, m.cursorLine, start)
				m.buffer.EndUndoGroup()
				// Adjust cursor for removed prefix markers
				m.cursorCol = m.cursorCol - markerLen
				if m.cursorCol < mStart {
					m.cursorCol = mStart
				}
			}
		} else {
			// Wrap the word and move cursor past closing markers
			m.buffer.BeginUndoGroup()
			word := string(line[start:end])
			m.buffer.DeleteRange(m.cursorLine, start, m.cursorLine, end)
			wrapped := marker + word + marker
			_, endCol := m.buffer.InsertText(m.cursorLine, start, wrapped)
			m.buffer.EndUndoGroup()
			m.cursorCol = endCol
		}
	} else {
		// No word — insert empty markers and place cursor between them
		m.buffer.BeginUndoGroup()
		markerRunes := []rune(marker)
		for _, r := range markerRunes {
			m.buffer.InsertChar(m.cursorLine, m.cursorCol, r)
			m.cursorCol++
		}
		for _, r := range markerRunes {
			m.buffer.InsertChar(m.cursorLine, m.cursorCol, r)
		}
		m.buffer.EndUndoGroup()
	}
	m.rewrap()
	m.updatePreferredCol()
	m.ensureCursorVisible()
}

func (m *Model) insertLink() {
	if m.hasSelection {
		text := m.selectedText()
		sl, sc, _, _ := m.selectionRange()
		m.deleteSelection()

		link := "[" + text + "]()"
		endLine, endCol := m.buffer.InsertText(sl, sc, link)
		// Place cursor inside the ()
		m.cursorLine = endLine
		m.cursorCol = endCol - 1
	} else {
		start, end := m.buffer.WordAt(m.cursorLine, m.cursorCol)
		if start < end {
			line := m.buffer.Line(m.cursorLine)
			word := string(line[start:end])
			m.buffer.DeleteRange(m.cursorLine, start, m.cursorLine, end)
			link := "[" + word + "]()"
			endLine, endCol := m.buffer.InsertText(m.cursorLine, start, link)
			m.cursorLine = endLine
			m.cursorCol = endCol - 1
		} else {
			link := "[]()"
			_, endCol := m.buffer.InsertText(m.cursorLine, m.cursorCol, link)
			// Place cursor inside []
			m.cursorCol = endCol - 3
		}
	}
	m.rewrap()
	m.updatePreferredCol()
	m.ensureCursorVisible()
}

func (m *Model) handleFootnote() {
	// Check if cursor is on a footnote reference [^id]
	if id, refStart, refEnd := m.footnoteRefAt(m.cursorLine, m.cursorCol); id != "" {
		// Jump to definition
		if defLine, defCol := m.findFootnoteDef(id); defLine >= 0 {
			m.cursorLine = defLine
			m.cursorCol = defCol
			m.clearSelection()
			m.rewrap()
			m.updatePreferredCol()
			m.ensureCursorVisible()
			return
		}
		// Definition doesn't exist — create it
		_ = refStart
		_ = refEnd
		m.appendFootnoteDef(id)
		return
	}

	// Check if cursor is on a footnote definition line [^id]:
	if id := m.footnoteDefLineID(m.cursorLine); id != "" {
		// Jump back to first reference
		if refLine, refCol := m.findFootnoteRef(id); refLine >= 0 {
			m.cursorLine = refLine
			m.cursorCol = refCol
			m.clearSelection()
			m.rewrap()
			m.updatePreferredCol()
			m.ensureCursorVisible()
			return
		}
	}

	// Neither — insert a new footnote
	id := m.nextFootnoteID()
	ref := fmt.Sprintf("[^%s]", id)
	refRunes := []rune(ref)
	for _, r := range refRunes {
		m.buffer.InsertChar(m.cursorLine, m.cursorCol, r)
		m.cursorCol++
	}
	m.appendFootnoteDef(id)
}

// footnoteRefAt checks if the cursor is inside a [^...] reference.
// Returns the footnote id and the start/end columns of the reference.
func (m *Model) footnoteRefAt(line, col int) (id string, start, end int) {
	row := m.buffer.Line(line)
	if len(row) == 0 {
		return "", 0, 0
	}

	// Clamp col to valid range for searching
	searchCol := col
	if searchCol >= len(row) {
		searchCol = len(row) - 1
	}
	if searchCol < 0 {
		return "", 0, 0
	}

	// Scan left to find `[^`
	left := searchCol
	for left > 0 && row[left] != '[' {
		if row[left] == ']' && left < searchCol {
			break // passed a closing bracket
		}
		left--
	}

	if left+1 >= len(row) || row[left] != '[' || row[left+1] != '^' {
		return "", 0, 0
	}

	// Scan right to find `]`
	right := left + 2
	for right < len(row) && row[right] != ']' {
		right++
	}
	if right >= len(row) {
		return "", 0, 0
	}

	// Verify cursor is within [^ ... ]
	if col < left || col > right {
		return "", 0, 0
	}

	id = string(row[left+2 : right])
	if id == "" {
		return "", 0, 0
	}
	return id, left, right + 1
}

// footnoteDefLineID checks if a line is a footnote definition (starts with [^id]:).
func (m *Model) footnoteDefLineID(line int) string {
	row := m.buffer.Line(line)
	s := string(row)
	s = strings.TrimLeft(s, " \t")
	if !strings.HasPrefix(s, "[^") {
		return ""
	}
	end := strings.Index(s, "]:")
	if end < 0 {
		return ""
	}
	return s[2:end]
}

// findFootnoteDef finds the line where [^id]: is defined.
// Returns (line, col) where col points after "[^id]: ".
func (m *Model) findFootnoteDef(id string) (int, int) {
	prefix := "[^" + id + "]:"
	for i := 0; i < m.buffer.LineCount(); i++ {
		s := strings.TrimLeft(string(m.buffer.Line(i)), " \t")
		if strings.HasPrefix(s, prefix) {
			// Position cursor at end of prefix (start of content)
			fullLine := string(m.buffer.Line(i))
			idx := strings.Index(fullLine, prefix)
			col := idx + len(prefix)
			// Skip a space after the colon if present
			if col < len([]rune(fullLine)) && fullLine[col] == ' ' {
				col++
			}
			return i, col
		}
	}
	return -1, 0
}

// findFootnoteRef finds the first occurrence of [^id] (not followed by :) in the document.
func (m *Model) findFootnoteRef(id string) (int, int) {
	ref := "[^" + id + "]"
	def := "[^" + id + "]:"
	for i := 0; i < m.buffer.LineCount(); i++ {
		s := string(m.buffer.Line(i))
		idx := strings.Index(s, ref)
		if idx >= 0 {
			// Make sure it's not the definition line
			if !strings.Contains(s, def) || strings.Index(s, def) != idx {
				return i, idx
			}
		}
	}
	return -1, 0
}

// nextFootnoteID scans the document for [^N] patterns and returns the next number.
func (m *Model) nextFootnoteID() string {
	max := 0
	for i := 0; i < m.buffer.LineCount(); i++ {
		s := string(m.buffer.Line(i))
		pos := 0
		for {
			idx := strings.Index(s[pos:], "[^")
			if idx < 0 {
				break
			}
			start := pos + idx + 2
			end := strings.Index(s[start:], "]")
			if end < 0 {
				break
			}
			numStr := s[start : start+end]
			n := 0
			for _, c := range numStr {
				if c >= '0' && c <= '9' {
					n = n*10 + int(c-'0')
				} else {
					n = -1
					break
				}
			}
			if n > max {
				max = n
			}
			pos = start + end + 1
		}
	}
	return fmt.Sprintf("%d", max+1)
}

// appendFootnoteDef adds a [^id]: definition at the end of the file and jumps to it.
func (m *Model) appendFootnoteDef(id string) {
	lastLine := m.buffer.LineCount() - 1
	lastLineLen := m.buffer.LineLen(lastLine)

	// Add blank line if the last line isn't empty
	if lastLineLen > 0 {
		m.buffer.InsertNewline(lastLine, lastLineLen)
		lastLine++
	}

	// Add the definition line
	def := fmt.Sprintf("[^%s]: ", id)
	m.buffer.InsertNewline(lastLine, m.buffer.LineLen(lastLine))
	lastLine++
	m.buffer.InsertText(lastLine, 0, def)

	m.cursorLine = lastLine
	m.cursorCol = len([]rune(def))
	m.clearSelection()
	m.rewrap()
	m.updatePreferredCol()
	m.ensureCursorVisible()
}

func (m *Model) indent() {
	spaces := strings.Repeat(" ", m.tabWidth)

	if m.hasSelection {
		sl, _, el, _ := m.selectionRange()
		for i := sl; i <= el; i++ {
			m.buffer.InsertText(i, 0, spaces)
		}
		m.cursorCol += m.tabWidth
		m.selAnchorCol += m.tabWidth
	} else {
		for _, r := range spaces {
			m.buffer.InsertChar(m.cursorLine, m.cursorCol, r)
			m.cursorCol++
		}
	}
}

func (m *Model) outdent() {
	if m.hasSelection {
		sl, _, el, _ := m.selectionRange()
		for i := sl; i <= el; i++ {
			m.removeLeadingSpaces(i)
		}
	} else {
		m.removeLeadingSpaces(m.cursorLine)
	}
	m.clampCursor()
}

func (m *Model) removeLeadingSpaces(line int) {
	row := m.buffer.Line(line)
	count := 0
	for _, r := range row {
		if r == ' ' && count < m.tabWidth {
			count++
		} else if r == '\t' && count < 1 {
			count = m.tabWidth
			break
		} else {
			break
		}
	}
	if count > 0 {
		m.buffer.DeleteRange(line, 0, line, count)
		if line == m.cursorLine && m.cursorCol >= count {
			m.cursorCol -= count
		} else if line == m.cursorLine {
			m.cursorCol = 0
		}
	}
}

func (m Model) scheduleAutosave() tea.Cmd {
	m.pendingSave = true
	return tea.Tick(m.autosaveDelay, func(t time.Time) tea.Msg {
		return AutosaveTickMsg{}
	})
}

// FileWatchCmd returns a command that ticks every 2 seconds to check for
// external file changes.
func FileWatchCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return FileWatchTickMsg{}
	})
}

func runesContainNewline(runes []rune) bool {
	for _, r := range runes {
		if r == '\n' || r == '\r' {
			return true
		}
	}
	return false
}

// normalizePastedText collapses hard-wrapped lines into paragraphs.
// Single newlines between non-blank lines become spaces (unwrapping
// hard-wrapped text), while blank lines (paragraph breaks) and lines
// starting with markdown block syntax are preserved.
func normalizePastedText(runes []rune) []rune {
	text := string(runes)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimRight(text, "\r")
	lines := strings.Split(text, "\n")

	if len(lines) <= 1 {
		return []rune(text)
	}

	var buf strings.Builder
	for i, line := range lines {
		buf.WriteString(line)
		if i == len(lines)-1 {
			break
		}

		next := ""
		if i+1 < len(lines) {
			next = lines[i+1]
		}

		// Keep the newline when:
		// - current line is blank (paragraph break)
		// - next line is blank (paragraph break)
		// - next line starts with markdown block-level syntax
		if line == "" || next == "" || isBlockStart(next) {
			buf.WriteByte('\n')
		} else {
			// Collapse the hard wrap: join with a space unless the line
			// already ends with one.
			if len(line) > 0 && line[len(line)-1] != ' ' {
				buf.WriteByte(' ')
			}
		}
	}
	return []rune(buf.String())
}

// isBlockStart returns true if the line starts a new markdown block element.
func isBlockStart(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" {
		return false
	}
	switch trimmed[0] {
	case '#': // heading
		return true
	case '-', '*', '+': // list item or hr
		return true
	case '>': // blockquote
		return true
	case '|': // table row
		return true
	case '`': // possible code fence
		return strings.HasPrefix(trimmed, "```")
	case '~': // possible code fence
		return strings.HasPrefix(trimmed, "~~~")
	}
	// Ordered list: digits followed by . or )
	if len(trimmed) >= 2 && trimmed[0] >= '0' && trimmed[0] <= '9' {
		for j := 1; j < len(trimmed); j++ {
			if trimmed[j] >= '0' && trimmed[j] <= '9' {
				continue
			}
			if trimmed[j] == '.' || trimmed[j] == ')' {
				return true
			}
			break
		}
	}
	return false
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	editWidth := m.editWidth()
	leftMargin := 0
	if m.contentWidth > 0 && m.width > m.contentWidth {
		leftMargin = (m.width - m.contentWidth) / 2
	}

	// Determine highlight state (frontmatter / code block) for visible lines.
	hlState := HighlightState{}
	if m.scrollOffset > 0 {
		for i := 0; i < len(m.wrap.VisualLines) && i < m.scrollOffset; i++ {
			vl := m.wrap.VisualLines[i]
			if vl.LogicalCol == 0 {
				line := m.buffer.Line(vl.LogicalLine)
				trimmed := strings.TrimSpace(string(line))

				// Code fences
				if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
					hlState.InCodeBlock = !hlState.InCodeBlock
					continue
				}

				if trimmed == "---" {
					if hlState.InFrontmatter {
						hlState.InFrontmatter = false
					} else if vl.LogicalLine == 0 {
						hlState.InFrontmatter = true
					}
				}
			}
		}

		// If scroll starts mid-way through a blockquote logical line, seed InBlockquote
		firstVl := m.wrap.VisualLines[m.scrollOffset]
		if firstVl.LogicalCol > 0 {
			line := m.buffer.Line(firstVl.LogicalLine)
			trimmed := strings.TrimSpace(string(line))
			if len(trimmed) > 0 && trimmed[0] == '>' {
				hlState.InBlockquote = true
			}
		}
	}

	// Get selection range for rendering
	selSL, selSC, selEL, selEC := m.selectionRange()

	var lines []string
	visEnd := m.scrollOffset + m.height
	if visEnd > len(m.wrap.VisualLines) {
		visEnd = len(m.wrap.VisualLines)
	}

	lastLogicalLine := -1
	for vi := m.scrollOffset; vi < visEnd; vi++ {
		vl := m.wrap.VisualLines[vi]

		// Only update frontmatter state at start of logical line
		if vl.LogicalLine != lastLogicalLine && vl.LogicalCol == 0 {
			lastLogicalLine = vl.LogicalLine
		}

		// Get the runes for this visual line
		lineRunes := vl.Runes

		// Clear per-logical-line state at the start of each new logical line
		lineHlState := hlState
		if vl.LogicalCol == 0 {
			lineHlState.InBlockquote = false
		}

		// Highlight
		tokens, newState := HighlightLine(lineRunes, m.theme, lineHlState, vl.LogicalLine)
		if vl.LogicalCol == 0 {
			hlState = newState
		}

		// Determine cursor position on this visual line
		cursorVisCol := -1
		if m.focused && vl.LogicalLine == m.cursorLine {
			localCol := m.cursorCol - vl.LogicalCol
			if localCol >= 0 && localCol < len(lineRunes) {
				cursorVisCol = localCol
			} else if localCol == len(lineRunes) {
				// Cursor at end of visual line — only show here if this is the
				// last visual sub-line for this logical line (otherwise it
				// belongs at col 0 of the next sub-line).
				isLastSubLine := vi+1 >= len(m.wrap.VisualLines) ||
					m.wrap.VisualLines[vi+1].LogicalLine != vl.LogicalLine
				if isLastSubLine {
					cursorVisCol = localCol
				}
			}
		}

		// Determine selection range on this visual line
		selStart, selEnd := -1, -1
		if m.hasSelection {
			lineStartL := vl.LogicalLine
			lineStartC := vl.LogicalCol
			lineEndC := vl.LogicalCol + len(vl.Runes)

			// Check if this visual line overlaps with the selection
			if lineStartL >= selSL && lineStartL <= selEL {
				var rangeStart, rangeEnd int
				if lineStartL == selSL {
					rangeStart = selSC
				} else {
					rangeStart = 0
				}
				if lineStartL == selEL {
					rangeEnd = selEC
				} else {
					rangeEnd = m.buffer.LineLen(lineStartL)
				}

				// Map to local visual line coordinates
				localStart := rangeStart - lineStartC
				localEnd := rangeEnd - lineStartC
				if localStart < 0 {
					localStart = 0
				}
				if localEnd > lineEndC-lineStartC {
					localEnd = lineEndC - lineStartC
				}
				if localStart < localEnd {
					selStart = localStart
					selEnd = localEnd
				}
			}
		}

		// Active line highlighting (only the visual sub-line containing the cursor)
		isActiveLine := m.typewriterHighlight && m.focused && cursorVisCol >= 0

		rendered := RenderTokens(tokens, cursorVisCol, selStart, selEnd, m.theme, isActiveLine)

		// Choose padding style: active line uses highlighted bg, others use normal margin
		padStyle := m.theme.Margin
		if isActiveLine {
			padStyle = m.theme.ActiveLine
		}

		// Pad to width and add margin
		renderedWidth := lipgloss.Width(rendered)
		if renderedWidth < editWidth {
			rendered += padStyle.Render(strings.Repeat(" ", editWidth-renderedWidth))
		}

		if leftMargin > 0 {
			rendered = padStyle.Render(strings.Repeat(" ", leftMargin)) + rendered
		}

		// Fill any remaining space to the right edge
		totalWidth := lipgloss.Width(rendered)
		if totalWidth < m.width {
			rendered += padStyle.Render(strings.Repeat(" ", m.width-totalWidth))
		}

		lines = append(lines, rendered)
	}

	// Fill remaining lines with background-colored empty lines
	for len(lines) < m.height {
		lines = append(lines, m.theme.Margin.Render(strings.Repeat(" ", m.width)))
	}

	return strings.Join(lines, "\n")
}

// chromeWidth returns the width for chrome elements (title, status, padding).
func (m Model) chromeWidth() int {
	if m.fullWidth > 0 {
		return m.fullWidth
	}
	return m.width
}

// PaddingLine returns an empty line styled with the editor background.
func (m Model) PaddingLine() string {
	return m.theme.Margin.Render(strings.Repeat(" ", m.chromeWidth()))
}

// TitleView renders the title bar.
func (m Model) TitleView() string {
	cw := m.chromeWidth()
	title := m.filePath
	if title == "" {
		title = "[No File]"
	} else {
		// Show vault-relative breadcrumb path (strip .md extension)
		display := title
		if m.vaultPath != "" {
			if rel, err := filepath.Rel(m.vaultPath, title); err == nil {
				display = rel
			}
		}
		display = strings.TrimSuffix(display, ".md")
		// Convert path separators to breadcrumb arrows
		parts := strings.Split(filepath.ToSlash(display), "/")
		title = strings.Join(parts, " / ")
	}

	if len(title) > cw-4 {
		title = "…" + title[len(title)-cw+5:]
	}

	return m.theme.TitleBar.
		Width(cw).
		Render(title)
}

// StatusView renders the status bar.
func (m Model) StatusView() string {
	cw := m.chromeWidth()
	wc := fmt.Sprintf("Words: %d", m.WordCount())
	status := m.SaveStatus()

	left := wc
	if status != "" {
		left += " │ " + status
	}

	// Right side: cursor position
	right := fmt.Sprintf("Ln %d, Col %d", m.cursorLine+1, m.cursorCol+1)

	gap := cw - runewidth.StringWidth(left) - runewidth.StringWidth(right) - 2
	if gap < 1 {
		gap = 1
	}

	bar := left + strings.Repeat(" ", gap) + right

	return m.theme.StatusBar.
		Width(cw).
		Render(bar)
}
