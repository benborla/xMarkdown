# xmd

A vim-navigable markdown reader for the terminal.

Like [Glow](https://github.com/charmbracelet/glow), but instead of passively
scrolling rendered output, you navigate it: jump between headings, search with
`/`, pop open a table of contents, and follow links — all without leaving the
keyboard.

## Install

Grab a binary from [releases](https://github.com/benborla/xMarkdown/releases), or:

```sh
go install github.com/benborla/xMarkdown@latest   # installs as `xMarkdown`
```

Or build from source:

```sh
git clone https://github.com/benborla/xMarkdown.git
cd xMarkdown
go build -o xmd .
```

Requires Go 1.21+.

## Usage

```sh
xmd README.md
```

Piping works too — renders and exits, no TUI:

```sh
xmd README.md | less -R
```

## Keys

| Key | Action |
|-----|--------|
| `j` / `k`, arrows | line down / up |
| `ctrl-d` / `ctrl-u` | half page down / up |
| `ctrl-f` / `ctrl-b`, `space` | full page down / up |
| `gg` / `G` | top / bottom |
| `]]` / `[[` | next / previous heading |
| `/` | search (case-insensitive), `enter` to run |
| `n` / `N` | next / previous match |
| `t` | table of contents — `j`/`k` select, `enter` jump |
| `Tab` / `Shift-Tab` | cycle link highlight |
| `Enter` | follow highlighted link (`.md` opens in place, URLs open in browser) |
| `esc` | dismiss search / overlay / link highlight |
| `:` | command mode — `:set nu`, `:set rnu`, `:set nonu`, `:theme <name>`, `:q` |
| `q` / `ctrl-c` | quit |

## Themes & config

xmd ships gruvbox dark and light and picks one automatically from your
terminal background. Override with `--theme gruvbox-light`, `--theme
/path/to/theme.json`, or in `~/.config/xmd/config.yaml`:

```yaml
theme: auto      # auto | gruvbox-dark | gruvbox-light | <custom-name> | /path.json
numbers: off     # off | absolute | relative
```

A theme is a single JSON file: a [glamour style](https://github.com/charmbracelet/glamour/tree/master/styles)
plus an `"xmd"` key for UI chrome (cursorline, line numbers, status bar, TOC,
search highlight). Drop custom themes in `~/.config/xmd/themes/<name>.json`
and select them by name. Missing `"xmd"` fields inherit gruvbox defaults.

## How it works

[Glamour](https://github.com/charmbracelet/glamour) renders the whole document
into styled terminal lines; [goldmark](https://github.com/yuin/goldmark)
parses the same source into an AST to extract headings and links, which are
matched against the rendered output to build a jump index. A
[Bubble Tea](https://github.com/charmbracelet/bubbletea) loop drives the
viewport.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
