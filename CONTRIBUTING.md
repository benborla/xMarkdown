# Contributing to xmd

Thanks for your interest in contributing. Issues and pull requests are welcome.

## Getting started

```sh
git clone <repo-url>
cd xmd
go build -o xmd .
go test ./...
```

Requires Go 1.21+. No other tooling needed.

## Project layout

| Path | Responsibility |
|------|----------------|
| `main.go` | CLI entry, TTY detection, pipe dump mode |
| `ui/` | Bubble Tea model — all key handling, modes, viewport |
| `render/` | Glamour wrapper: markdown → styled terminal lines |
| `doc/` | Goldmark AST extraction + matching anchors to rendered lines |
| `search/` | Query matching and ANSI-aware highlighting |

## Making changes

1. Fork and create a feature branch off `main`.
2. Write a failing test first, then the fix or feature (the codebase is
   test-driven; `ui` tests drive the model as a pure function — see
   `ui/model_test.go` for the `press`/`run` helpers).
3. Keep the change minimal. This project deliberately avoids features it
   doesn't need yet — if you're unsure whether something belongs, open an
   issue first.
4. Before pushing, make sure all of these are clean:

```sh
go test ./...
go vet ./...
gofmt -l .
```

5. Open a pull request with a short description of what and why.

## Commit messages

Conventional Commits style, subject ≤ 50 chars:

```
feat: add heading jump wrap-around
fix: accept space in search input
test: cover TOC empty-document case
docs: clarify install instructions
```

Body only when the "why" isn't obvious from the diff.

## Reporting bugs

Open an issue with:

- The markdown file (or minimal snippet) that triggers the problem
- Your terminal emulator and OS
- What you expected vs. what happened

## License

By contributing, you agree that your contributions will be licensed under the
[MIT License](LICENSE).
