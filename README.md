# Wordsmith

A minimal terminal markdown editor — inspired by [WordGrinder](https://cowlark.com/wordgrinder/) and built for distraction-free prose writing.

Wordsmith is designed for writers who keep notes and blog posts in markdown (e.g., in an Obsidian vault) and want a keyboard-driven, zen terminal experience for focused writing.


## Features

- **Markdown syntax highlighting** — special styling for bold, italic, headers, links, code, blockquotes, lists, and frontmatter
- **Zen writing mode** — centered content column with configurable width
- **Autosave** — auto-saving with atomic writes
- **Session persistence** — remembers your last opened file for seamless resume
- **Fuzzy file finder** — `Ctrl+P` to quickly search and open any markdown file in your vault, or create new files
- **File tree sidebar** — `Ctrl+E` to browse your vault directory structure, press `n` to create a new file
- **Markdown hotkeys** — `Ctrl+B` bold, `Ctrl+I` italic, `Ctrl+K` link insertion
- **Word counter** — live word count in the status bar
- **Undo/Redo** — `Ctrl+Z` / `Ctrl+Y` with coalesced character grouping


## Install

```bash
go install github.com/tupini07/wordsmith@latest
```

Or build from source:

```bash
git clone https://github.com/tupini07/wordsmith
cd wordsmith
go build -o wordsmith .
```

## Usage

```bash
# Open a specific file
wordsmith path/to/file.md

# Open last edited file (from session state)
wordsmith

# If no file and no session state, opens the fuzzy finder
```

## Configuration

Create a config file at `~/.config/wordsmith/config.yaml`:

```yaml
vault_path: "/path/to/folder/with/markdown/notes"
autosave_delay: "2s"
tab_width: 4
content_width: 80
theme: "default"
```

See [config.example.yaml](config.example.yaml) for all options.

## Themes

Wordsmith ships with 4 built-in themes. Set `theme` in your config file:

| Theme | Description |
|-------|-------------|
| `default` | Warm, cozy palette — cream text, coral headings, teal links |
| `nord` | Cool blue-gray palette inspired by [Nord](https://www.nordtheme.com/) |
| `dracula` | Dark purple-accented palette inspired by [Dracula](https://draculatheme.com/) |
| `gruvbox` | Warm retro palette inspired by [Gruvbox](https://github.com/morhetz/gruvbox) |

All themes set explicit backgrounds on every element to prevent your terminal's native background from bleeding through.

## Key Bindings

### Editing

| Key | Action |
|-----|--------|
| `Alt+B` | Bold — toggle `**…**` around word/selection |
| `Alt+I` | Italic — toggle `*…*` around word/selection |
| `Ctrl+K` | Link — inserts `[text](url)` |
| `Ctrl+D` | Footnote — insert, or jump between ref ↔ definition |
| `Ctrl+Backspace` / `Ctrl+W` | Delete previous word |
| `Alt+D` | Delete next word |
| `Ctrl+A` | Select all |
| `Ctrl+C` | Copy selection (or current line if no selection) |
| `Ctrl+X` | Cut selection (or current line if no selection) |
| `Ctrl+V` | Paste from system clipboard |
| `Tab` | Indent (inserts spaces) |
| `Shift+Tab` | Outdent |
| `Ctrl+Z` | Undo |
| `Ctrl+Y` | Redo |

### Navigation

| Key | Action |
|-----|--------|
| `Ctrl+P` | Open fuzzy file finder (type a new name to create) |
| `Ctrl+E` | Toggle file tree sidebar |
| `n` (in file tree) | Create new file in selected directory |
| `Ctrl+Left/Right` | Move by word |
| `Ctrl+Shift+Left/Right` | Select by word |
| `Home` / `End` | Start / end of line |
| `PgUp` / `PgDn` | Page up / down |
| `Ctrl+Home` | Go to top of file |
| `Ctrl+End` | Go to end of file |
| `Shift+Arrow` | Select text |

### Mouse

| Action | Effect |
|--------|--------|
| Click | Move cursor to position |
| Click + drag | Select text |
| Scroll wheel | Scroll viewport |

### File Operations

| Key | Action |
|-----|--------|
| `Ctrl+S` | Save (or overwrite if file changed externally) |
| `Ctrl+R` | Reload file from disk |
| `F2` | Open config file for editing (hot-reloads on close) |
| `F3` | Rename current file |
| `Ctrl+Q` | Quit |

## Session State

Wordsmith remembers your last opened file in `~/.local/state/wordsmith/state.json`. Launch `wordsmith` without arguments to resume where you left off.

## Architecture

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) (TUI framework) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) (styling).

```
internal/
├── app/         # Top-level model, mode switching
├── editor/      # Core editor: buffer, cursor, wrapping, highlighting
├── finder/      # Fuzzy file finder (Ctrl+P)
├── filetree/    # File tree sidebar (Ctrl+E)
├── config/      # YAML configuration
└── state/       # Session persistence
```

## License

MIT
