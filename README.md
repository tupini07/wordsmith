# Wordsmith ✍️

A cozy, minimal terminal markdown editor — inspired by [WordGrinder](https://cowlark.com/wordgrinder/) and built for distraction-free prose writing.

Wordsmith is designed for writers who keep notes and blog posts in markdown (e.g., in an Obsidian vault) and want a keyboard-driven, zen terminal experience for focused writing.

![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)

## Features

- **Markdown syntax highlighting** — bold, italic, headers, links, code, blockquotes, lists, and frontmatter are color-coded in a warm, cozy palette
- **Zen writing mode** — centered content column with configurable width, minimal chrome
- **Autosave** — debounced auto-saving with atomic writes (no data loss on crash)
- **Session persistence** — remembers your last opened file for seamless resume
- **Fuzzy file finder** — `Ctrl+P` to quickly search and open any markdown file in your vault
- **File tree sidebar** — `Ctrl+E` to browse your vault directory structure
- **Markdown hotkeys** — `Ctrl+B` bold, `Ctrl+I` italic, `Ctrl+K` link insertion
- **Word counter** — live word count in the status bar (excludes frontmatter)
- **Undo/Redo** — `Ctrl+Z` / `Ctrl+Y` with coalesced character grouping
- **Soft word wrapping** — prose wraps naturally at word boundaries
- **External change detection** — warns if a file was modified outside the editor

## Install

```bash
go install github.com/andreatupini/wordsmith@latest
```

Or build from source:

```bash
git clone https://github.com/andreatupini/wordsmith
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
```

See [config.example.yaml](config.example.yaml) for all options.

## Key Bindings

### Editing

| Key | Action |
|-----|--------|
| `Ctrl+B` | Bold — wraps word/selection in `**…**` |
| `Ctrl+I` | Italic — wraps word/selection in `*…*` |
| `Ctrl+K` | Link — inserts `[text](url)` |
| `Tab` | Indent (inserts spaces) |
| `Shift+Tab` | Outdent |
| `Ctrl+Z` | Undo |
| `Ctrl+Y` | Redo |

### Navigation

| Key | Action |
|-----|--------|
| `Ctrl+P` | Open fuzzy file finder |
| `Ctrl+E` | Toggle file tree sidebar |
| `Ctrl+Left/Right` | Move by word |
| `Home` / `End` | Start / end of line |
| `PgUp` / `PgDn` | Page up / down |
| `Shift+Arrow` | Select text |

### File Operations

| Key | Action |
|-----|--------|
| `Ctrl+S` | Force save |
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
