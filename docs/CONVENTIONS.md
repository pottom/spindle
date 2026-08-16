# spindle — conventions

A Spotify Connect remote control TUI. Currently in **prototype phase** — the goal is
resolving the four risks listed in `DESIGN.md`, not production readiness.

## Commands

- `make run-mock` — mock backend, no auth and no network. Use this during development.
- `make run` — live Spotify API.
- `make login` — run the OAuth flow and report who you are signed in as.
- `spindle daemon` — start the Connect device in the background.
  `spindle daemon --foreground` keeps it in the terminal, where Ctrl-C stops it.
- `make build` — the binary, at `./spindle` and **nowhere else**. It prints the
  commit it was built from, which is the same thing `spindle version`, the help
  header and the debug bar say. Builds under other names cost a morning once:
  a picture was fixed and went on looking broken because an older one was still
  being run, and nothing on screen could tell them apart.
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
7. **Every screen uses the room it is given, and adding anything to one is a
   change to that.** Shrinking the font is how somebody asks for more of a
   screen, and every part of a screen has to answer. Nothing may be a fixed
   height on a tall terminal or a fixed width on a wide one: whatever the other
   parts do not take, something must. In a list that means mandatory columns and
   optional ones that come in, in one fixed order, as the row earns them — and
   never go away again on a wider row.

   This is checked rather than remembered: `internal/ui/responsive_test.go` runs
   every tab from the smallest terminal that draws to a very large one. **Run it
   after any change to what a tab holds**, because a new caption or column is
   exactly what takes the room back without anybody noticing until a screenshot
   arrives.

8. **Every Web API request goes through `gate`, and nothing asks on an event
   without a floor under it.** The gate — `internal/player/errors.go` — counts
   each request into `api.log`, keeps the catalogue answers that cannot go stale,
   and types a 429 while `Retry-After` is still attached. A second http client
   that talks to `api.spotify.com` would be a request nobody can account for.

   The floor is the other half. There is a **daily** quota and reaching it is a
   lockout, not a throttle, so anything that asks in answer to something else —
   a device event, a keystroke, an answer of "nothing is playing" — must have a
   shortest interval of its own. A device coming up said so a dozen times in
   three seconds and spent nineteen requests on it; typing six letters spent six.
   Both were found by reading `api.log`, which is what to do rather than counting
   intervals in the source:

       sort -k3 ~/.local/state/spindle/api.log | uniq -cf 2 | sort -rn | head

   What each registration is allowed to ask, and what an open window costs, is
   measured in [SPOTIFY-API.md](SPOTIFY-API.md).
9. **What the pointer can touch is found from the same arithmetic that draws it.**
   Never from a table filled in while drawing, and never from a second copy of
   the sums. Where the two would diverge, give them one function to share —
   `stackTop`, `bandDetailWidth`, `trackDetailAt` all exist for that reason. A
   bar the eye can see and the pointer cannot find is worse than no bar, and it
   is invisible to every test that does not click.
10. **A renderer slot belongs to the thing, not to the place it sits in.** A slot
    is one picture to the terminal, so two things holding one slot is one picture
    drawn twice. Slots counted from a position break the moment the list shifts
    underneath — the row of saved tracks arrives after the playlists and moves
    every one of them along — and the wall then draws each cover under its
    neighbour's name. Hand them out per thing and keep them until it leaves the
    screen. See `freeSlot` in `internal/ui/library_grid.go`.

## Style

- Small files, one type per file. Split anything over 300 lines.
- Every Lipgloss style lives in `internal/ui/style`. No hex codes in `View()`.
- Wrap errors: `fmt.Errorf("fetch player state: %w", err)`.
- A key is named once, in `internal/ui/keynames.go`, and never spelled again.
  What a hint bar advertises comes from the binding itself — `terse` and `tight`
  in `keys.go` — because somebody reading the bar presses what it says, and the
  bar once said `t` for a screen that is on `f`.
- Tests only where the logic is non-trivial: progress arithmetic, PKCE, halfblock
  encoding. No UI tests in this phase.
- Commit messages: short, imperative, lowercase. `add device picker overlay`.
