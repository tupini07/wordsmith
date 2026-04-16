package editor

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings for the editor.
type KeyMap struct {
	// Cursor movement
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
	Home  key.Binding
	End   key.Binding
	PgUp  key.Binding
	PgDn  key.Binding

	// Jump to top/bottom
	GoTop    key.Binding
	GoBottom key.Binding

	// Cursor movement with selection
	ShiftUp    key.Binding
	ShiftDown  key.Binding
	ShiftLeft  key.Binding
	ShiftRight key.Binding
	ShiftHome  key.Binding
	ShiftEnd   key.Binding

	// Word movement
	CtrlLeft  key.Binding
	CtrlRight key.Binding

	// Word movement with selection
	CtrlShiftLeft  key.Binding
	CtrlShiftRight key.Binding

	// Select all
	SelectAll key.Binding

	// Editing
	Backspace      key.Binding
	Delete         key.Binding
	DeleteWordBack key.Binding
	Enter          key.Binding
	Tab            key.Binding
	ShiftTab       key.Binding

	// Clipboard
	Copy  key.Binding
	Cut   key.Binding
	Paste key.Binding

	// Markdown formatting
	Bold     key.Binding
	Italic   key.Binding
	Link     key.Binding
	Footnote key.Binding

	// File operations
	Save   key.Binding
	Reload key.Binding
	Quit   key.Binding

	// Undo/Redo
	Undo key.Binding
	Redo key.Binding

	// Navigation
	FileTree    key.Binding
	FuzzyFinder key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:    key.NewBinding(key.WithKeys("up")),
		Down:  key.NewBinding(key.WithKeys("down")),
		Left:  key.NewBinding(key.WithKeys("left")),
		Right: key.NewBinding(key.WithKeys("right")),
		Home:  key.NewBinding(key.WithKeys("home")),
		End:   key.NewBinding(key.WithKeys("end")),
		PgUp:  key.NewBinding(key.WithKeys("pgup")),
		PgDn:  key.NewBinding(key.WithKeys("pgdown")),

		GoBottom: key.NewBinding(key.WithKeys("ctrl+end")),
		GoTop:    key.NewBinding(key.WithKeys("ctrl+home")),

		ShiftUp:    key.NewBinding(key.WithKeys("shift+up")),
		ShiftDown:  key.NewBinding(key.WithKeys("shift+down")),
		ShiftLeft:  key.NewBinding(key.WithKeys("shift+left")),
		ShiftRight: key.NewBinding(key.WithKeys("shift+right")),
		ShiftHome:  key.NewBinding(key.WithKeys("shift+home")),
		ShiftEnd:   key.NewBinding(key.WithKeys("shift+end")),

		CtrlLeft:  key.NewBinding(key.WithKeys("ctrl+left")),
		CtrlRight: key.NewBinding(key.WithKeys("ctrl+right")),

		CtrlShiftLeft:  key.NewBinding(key.WithKeys("ctrl+shift+left")),
		CtrlShiftRight: key.NewBinding(key.WithKeys("ctrl+shift+right")),

		SelectAll: key.NewBinding(key.WithKeys("ctrl+a")),

		Backspace:      key.NewBinding(key.WithKeys("backspace")),
		Delete:         key.NewBinding(key.WithKeys("delete")),
		DeleteWordBack: key.NewBinding(key.WithKeys("ctrl+h", "ctrl+w")),
		Enter:     key.NewBinding(key.WithKeys("enter")),
		Tab:       key.NewBinding(key.WithKeys("tab")),
		ShiftTab:  key.NewBinding(key.WithKeys("shift+tab")),

		Copy:  key.NewBinding(key.WithKeys("ctrl+c")),
		Cut:   key.NewBinding(key.WithKeys("ctrl+x")),
		Paste: key.NewBinding(key.WithKeys("ctrl+v")),

		Bold:   key.NewBinding(key.WithKeys("alt+b")),
		Italic: key.NewBinding(key.WithKeys("alt+i")),
		Link:   key.NewBinding(key.WithKeys("ctrl+k")),
		Footnote: key.NewBinding(key.WithKeys("ctrl+d")),

		Save:   key.NewBinding(key.WithKeys("ctrl+s")),
		Reload: key.NewBinding(key.WithKeys("ctrl+r")),
		Quit:   key.NewBinding(key.WithKeys("ctrl+q")),

		Undo: key.NewBinding(key.WithKeys("ctrl+z")),
		Redo: key.NewBinding(key.WithKeys("ctrl+y")),

		FileTree:    key.NewBinding(key.WithKeys("ctrl+e")),
		FuzzyFinder: key.NewBinding(key.WithKeys("ctrl+p")),
	}
}
