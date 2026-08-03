# spindle — design

A Spotify Connect remote control TUI in Go, built on Bubble Tea and Lipgloss.

## 1. Architecture

```
cmd/spindle/main.go       – flag parsing, wiring, tea.NewProgram
internal/auth/            – PKCE flow, token store, refresh
internal/player/          – Player interface + Spotify Web API impl + mock impl
internal/ui/              – Bubble Tea Model/Update/View
internal/ui/msg/          – tea.Msg types
internal/ui/cover/        – CoverRenderer interface + kitty/halfblock backends
internal/ui/style/        – all Lipgloss styles in one place
internal/xdg/             – XDG config and cache directories
```

### Directories

Everything spindle writes goes to XDG paths, not to Go's platform-conventional
`os.UserConfigDir` and `os.UserCacheDir` — on macOS those mean `~/Library`, which
is not where someone who keeps their dotfiles in order looks.

| What | Where |
|------|-------|
| OAuth token | `$XDG_CONFIG_HOME/spindle/token.json`, mode 0600 |
| Cover cache | `$XDG_CACHE_HOME/spindle/covers/` |

### Portability

`linux`, `darwin`, `windows` and the BSDs, on `amd64`, `arm64` and `arm`. Terminal
syscalls live in `_unix.go` / `_windows.go` pairs; `make cross` builds the lot.

### Layering rule

`ui` never calls the Spotify client directly, only `player.Player`. This is the one
rule that keeps the project from collapsing later, and it is what makes the mock
backend possible.

## 2. Dependencies

| Package | Purpose | Note |
|---------|---------|------|
| `charm.land/bubbletea/v2` | TUI runtime | v2 chosen over v1: the `bubbles` components are fully available for it (`charm.land/bubbles/v2`). Consequences: `View()` returns `tea.View`, alt screen and cursor are declarative View fields, and key events arrive as `tea.KeyPressMsg`. |
| `charm.land/lipgloss/v2` | styling and layout | `AdaptiveColor` lives in the impure `compat` subpackage; use `lipgloss.LightDark(isDark)` instead, fed from `tea.BackgroundColorMsg`. |
| `charm.land/bubbles/v2` | `progress`, `spinner`, `key`, `help`, `list` | |
| `github.com/charmbracelet/x/ansi` | ANSI-aware truncation and padding | already under Lipgloss; used to fit styled text to a cell width |
| `github.com/zmb3/spotify/v2` | Spotify Web API client | built on `golang.org/x/oauth2` |
| `golang.org/x/oauth2` | PKCE flow | |
| `golang.org/x/image/draw` | artwork resizing | CatmullRom kernel |
| `github.com/blacktop/go-termimg` | artwork rendering | Optional — evaluate in M5b |

Go 1.22 or later.

## 3. Data model

### 3.1 Application state machine

The player, playlists and search screens are tabs of one window rather than
states: `tab` cycles them and each keeps its own cursor. Only the states below
take the screen over entirely.

```
        ┌────────────┐  no token / expired
        │ stateAuth  │◄────────────────────┐
        └─────┬──────┘                     │
              │ auth succeeded             │ 401
              ▼                            │
        ┌───────────────┐                  │
        │ stateNoDevice │                  │
        └─────┬─────────┘                  │
              │ active device found        │
              ▼                            │
        ┌───────────────┐                  │
        │ statePlayer   │──────────────────┘
        └─────┬─────────┘
              │ 'd' key
              ▼
        ┌───────────────┐
        │ stateDevices  │  (overlay, ESC returns)
        └───────────────┘
```

`stateNoDevice` is **not an error state**. It is the normal entry path: if the user
has no running Spotify client, the Web API returns an empty response. Show a device
list, not an error. This is the single largest source of "it doesn't work for me".

### 3.2 The Player interface

```go
package player

type State struct {
    TrackID    string
    Title      string
    Artists    []string
    Album      string
    CoverURL   string        // largest available, capped at 640px
    Progress   time.Duration
    Duration   time.Duration
    Playing    bool
    Shuffle    bool
    Repeat     string        // "off" | "context" | "track"
    Volume     int           // 0–100
    DeviceID   string
    DeviceName string
}

type Device struct {
    ID     string
    Name   string
    Type   string
    Active bool
}

// Track is a list entry: identity and labels, without playback flags.
type Track struct {
    ID       string
    Title    string
    Artists  []string
    Album    string
    CoverURL string
    Duration time.Duration
}

type Playlist struct {
    ID       string
    Name     string
    Owner    string
    CoverURL string
    Tracks   int
    Duration time.Duration
}

type Player interface {
    State(ctx context.Context) (*State, error)
    Play(ctx context.Context) error
    Pause(ctx context.Context) error
    Next(ctx context.Context) error
    Previous(ctx context.Context) error
    Seek(ctx context.Context, pos time.Duration) error
    SetVolume(ctx context.Context, pct int) error
    SetShuffle(ctx context.Context, on bool) error
    SetRepeat(ctx context.Context, mode string) error
    Devices(ctx context.Context) ([]Device, error)
    TransferTo(ctx context.Context, deviceID string) error

    Search(ctx context.Context, query string) ([]Track, error)
    Playlists(ctx context.Context) ([]Playlist, error)
    PlaylistTracks(ctx context.Context, playlistID string) ([]Track, error)
    PlayTrack(ctx context.Context, trackID string) error
    PlayPlaylist(ctx context.Context, playlistID string, offset int) error
}

var ErrNoActiveDevice = errors.New("no active playback device")
```

Two implementations: `player.NewSpotify(client)` and `player.NewMock()`.

### 3.3 UI model

```go
type Model struct {
    state  AppState
    player player.Player
    ps     *player.State      // last known server state

    localProgress   time.Duration  // ticked locally, not ps.Progress
    optimisticUntil time.Time      // poll must not overwrite before this

    cover     coverState
    devices   []player.Device
    devCursor int

    err           error
    tickCount     int
    width, height int

    keys     keyMap
    help      help.Model
    progress  progress.Model
    spinner   spinner.Model
}
```

## 4. The two non-trivial mechanisms

### 4.1 Polling cadence

Two separate rhythms run concurrently:

- **1 s tick** — advances `localProgress` locally. No network. This is what makes the
  progress bar look continuous.
- **every 5th tick (≈5 s)** — a real `State()` call that resynchronises.

```go
case msg.Tick:
    m.tickCount++
    if m.ps != nil && m.ps.Playing {
        m.localProgress += time.Second
    }
    cmds := []tea.Cmd{tickCmd()}
    if m.tickCount%5 == 0 {
        cmds = append(cmds, fetchStateCmd(m.player))
    }
    return m, tea.Batch(cmds...)
```

Polling every second will hit 429. Five seconds is the point where drift is not yet
visible but the quota is comfortable.

After `Next` or `Previous`, issue one immediate extra fetch (delayed ~400 ms so the
change has propagated server-side), otherwise the wrong track shows for 5 seconds.

### 4.2 Optimistic updates

A key press mutates local state **first**, then dispatches the network command:

```go
case key.Matches(msg, m.keys.PlayPause):
    m.ps.Playing = !m.ps.Playing
    m.optimisticUntil = time.Now().Add(2 * time.Second)
    return m, togglePlaybackCmd(m.player, m.ps.Playing)
```

`optimisticUntil` is the critical part. An in-flight poll may return the *old* state
because Spotify has not propagated the change yet; writing that naively makes the UI
snap back. So:

```go
case msg.StateFetched:
    if time.Now().Before(m.optimisticUntil) {
        // adopt metadata only; playback flags and progress stay local
        m.ps.Title, m.ps.Artists, m.ps.Album = msg.State.Title, msg.State.Artists, msg.State.Album
    } else {
        m.ps = msg.State
        m.localProgress = msg.State.Progress
    }
```

The same applies after seek, volume, shuffle and repeat.

## 5. Album artwork

### 5.1 Renderer interface

```go
package cover

type Renderer interface {
    // Render fits img into a wCells × hCells area and returns a string
    // embeddable in View().
    Render(img image.Image, wCells, hCells int) (string, error)
    Name() string
}

func Detect(w io.Writer, r io.Reader) Renderer  // kitty → halfblock fallback
```

### 5.2 Backends, in build order

**Halfblock first.** Zero risk, works everywhere. The `▀` character with foreground
colour for the upper half-pixel and background colour for the lower gives two pixel
rows per cell. Build the whole pipeline — download, cache, resize, render — against
this backend.

**Kitty second.** This is where the real risk lives. Two approaches:

- **Unicode placeholders (`U+10EEEE`)** — transmit the image once with
  `a=T,U=1,i=<id>`, then have `View()` emit a rectangle of placeholder characters
  with row and column indices encoded as combining diacritics. Because this is
  *text*, Bubble Tea's diff handles it correctly and it stays put when scrolling.
  This is the theoretically correct solution; kitty and Ghostty both support it.
- **Cursor-positioned placement** — `View()` leaves the artwork area blank and an
  `a=p,i=<id>` is emitted after the frame using absolute positioning. Simpler, but
  resize and scroll cleanup are manual.

Try both and record the outcome in `FINDINGS.md`.

### 5.3 Protocol detection

Send a kitty graphics query together with a device attributes query:

```
\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\  followed by  \x1b[c
```

If only the device attributes response comes back, the terminal does not support the
protocol. Timeout 200 ms; the program must never block on this.

### 5.4 Cell pixel size

Aspect-correct scaling needs the pixel size of a cell. In order:

1. `TIOCGWINSZ` ioctl → `ws_xpixel` / `ws_ypixel` (fast, not always populated)
2. `CSI 14 t` (window in pixels) + `CSI 18 t` (window in cells), then divide
3. Fall back to assuming a 2:1 ratio

### 5.5 Caching

Album ID → downloaded JPEG under `$XDG_CACHE_HOME/spindle/covers/`. Download each
album once. Also keep an in-memory LRU of 10 decoded and resized images — the resize
is more expensive than it looks.

## 6. Mock backend

`spindle --mock` runs `player.Mock`: four hardcoded tracks with real-time progress,
no auth and no network. It also accepts `--mock-fail=<case>` to simulate error paths.

This is not a convenience feature, it is the most important tool in the prototype:

- the UI can be iterated without auth (seconds, not minutes, per feedback cycle)
- states are reproducible (track change at exactly 30 seconds)
- error paths are testable on demand
- the program is demonstrable without a Premium account

The mock is built in M0, **before** OAuth.

## 7. Findings

The actual output of the prototype is `FINDINGS.md`, answering:

1. Does the kitty placeholder mode cooperate with Bubble Tea's diff? What hacks were needed?
2. `go-termimg` or a hand-rolled implementation?
3. What is the smallest polling interval that avoids 429? What is the real quota?
4. What is the measured latency on the control endpoints? Is a 2 s optimistic window enough?
5. Do kitty and Ghostty differ in placement behaviour?
6. Binary size, and memory after one hour of running — does the cover cache leak?
