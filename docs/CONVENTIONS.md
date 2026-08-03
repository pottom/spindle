# spindle — conventions

A Spotify Connect remote control TUI. Currently in **prototype phase** — the goal is
resolving the four risks listed in `DESIGN.md`, not production readiness.

## Commands

- `make run-mock` — mock backend, no auth and no network. Use this during development.
- `make run` — live Spotify API.
- `make login` — run the OAuth flow and report who you are signed in as.
- `make lint` — `go vet ./... && staticcheck ./...`
- `make cross` — build every supported platform. Run it after touching anything
  that talks to the terminal or the filesystem.

## Hard rules

1. **`internal/ui` never imports `github.com/zmb3/spotify`.** It works exclusively
   through the `player.Player` interface. Breaking this rule kills both the mock
   backend and any future audio backend.
2. **No network or blocking call inside `Update()`.** All I/O goes through a
   `tea.Cmd` and returns as a `tea.Msg`.
3. **No `panic` and no `log.Fatal` outside `main()`.** Errors are messages the UI
   displays.
4. **No POSIX-only syscall outside a `_unix.go` file.** Windows is a supported
   target; `make cross` is what proves it.
5. **`View()` is a pure function.** It mutates nothing, writes nothing, calls nothing
   remote.
6. **Do not write code for features outside the current milestone.** No placeholders,
   no TODO scaffolding, no speculative abstraction.

## Style

- Small files, one type per file. Split anything over 300 lines.
- Every Lipgloss style lives in `internal/ui/style`. No hex codes in `View()`.
- Wrap errors: `fmt.Errorf("fetch player state: %w", err)`.
- Tests only where the logic is non-trivial: progress arithmetic, PKCE, halfblock
  encoding. No UI tests in this phase.
- Commit messages: short, imperative, lowercase. `add device picker overlay`.
