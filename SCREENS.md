# spindle — screen designs

The ASCII drawings are indicative. **The dimension table in section 3 is
authoritative.** Where they disagree, the table wins.

## 1. Principles

**The terminal background stays the user's background.** Never paint a full-screen
background — it looks wrong in every theme and defeats the user's transparency
settings. Set foreground colours and borders only.

**One accent colour.** Green means "active / happening now". Nothing else is green.
If three things are green, none of them is emphasised.

**Icons need an ASCII fallback.** `⏸ ⏭ ⏮` are missing from many fonts. Under
`--ascii`, use `||`, `>>`, `<<`.

**No emoji.** Their width varies by terminal and shifts the layout.

## 2. Palette

Use Lipgloss `AdaptiveColor` so light terminals stay readable.

| Role | Dark | Light | Used for |
|------|------|-------|----------|
| `Accent` | `#1DB954` | `#128A3E` | active device dot, artist name, filled progress, focused element |
| `AccentDim` | `#14833B` | `#1DB954` | unfocused accent |
| `Text` | `#E8E8E8` | `#1A1A1A` | track title, volume value |
| `Muted` | `#8B949E` | `#57606A` | album name, timestamps |
| `Faint` | `#484F58` | `#8C959F` | help bar, disabled shuffle/repeat |
| `Border` | `#30363D` | `#D0D7DE` | frames, separators, empty progress |
| `BorderFocus` | `#1DB954` | `#128A3E` | focused overlay frame |
| `Error` | `#F85149` | `#CF222E` | error banner |
| `Warning` | `#D29922` | `#9A6700` | rate limit, offline banner |

All of this lives in `internal/ui/style/style.go` as a `Theme` struct. No hex codes
appear in `View()`.

## 3. Dimensions

| Element | Value |
|---------|-------|
| Minimum terminal | 64 × 20 |
| Outer frame | full width, capped at 100 columns, centred above that |
| Artwork | 20 × 10 cells |
| Left margin | 2 |
| Gap between artwork and info | 3 |
| Info column | remainder, minimum 28 |
| Help bar | 1 line, separator above |

Above 100 columns the layout must not stretch. A full-width TUI on an ultrawide
monitor is unreadable.

## 4. Screens

### 4.1 Player — the main screen

```
┌─ spindle ────────────────────────────── ● MacBook Pro ─┐
│                                                        │
│  ┌──────────────────┐   Bohemian Rhapsody              │
│  │                  │   Queen                          │
│  │                  │   A Night at the Opera           │
│  │   ALBUM ART      │                                  │
│  │   20 × 10        │   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━   │
│  │                  │   2:14                     5:55  │
│  │                  │                                  │
│  └──────────────────┘   ⏮  ⏸  ⏭ │ shuf off  rep off   │
│                                          vol 72        │
│                                                        │
├────────────────────────────────────────────────────────┤
│ space play/pause · n/p track · ←→ seek · d devices · ? │
└────────────────────────────────────────────────────────┘
```

- Title in `Text`, artist in `Accent`, album in `Muted`. This three-level hierarchy
  carries the rhythm of the whole screen.
- Long titles truncate with `…`. **Never wrap** — wrapping changes height, and
  height changes make the layout jump.
- Filled progress is `Accent`, empty is `Border`.
- **When paused, the progress bar turns `Muted`.** State should be readable at a
  glance, not only from a 3-character icon.
- Shuffle and repeat labels are `Faint` when off, `Accent` when on.
- Device name sits in the top border, preceded by `●`: `Accent` when playing,
  `Muted` when paused.

### 4.2 Artwork loading

```
│  ┌──────────────────┐   Bohemian Rhapsody              │
│  │                  │   Queen                          │
│  │        ⠋         │   A Night at the Opera           │
```

Spinner centred in the artwork box. **The box does not change size** — if the
placeholder were smaller than the image, the layout would jump when loading finishes.

If the download fails, replace the spinner with `♪` in `Faint`. Do not show an error
message; a missing cover is not important enough to interrupt use.

### 4.3 No active device

This is **not an error screen**. It is the most common entry point.

```
┌─ spindle ──────────────────────────────────────────────┐
│                                                        │
│   No active playback device.                           │
│                                                        │
│   Start Spotify on one of your devices, or pick one    │
│   from the list below:                                 │
│                                                        │
│   > MacBook Pro              computer                  │
│     iPhone                   smartphone                │
│     Kitchen speaker          speaker                   │
│                                                        │
│   If the list is empty, open the Spotify app and play  │
│   something for a moment — Spotify only reports        │
│   devices that were recently active.                   │
│                                                        │
├────────────────────────────────────────────────────────┤
│ ↑↓ select · enter activate · r refresh · q quit        │
└────────────────────────────────────────────────────────┘
```

The closing sentence matters. Spotify's Web API only returns recently active
devices, so a freshly launched client often does not appear until something has
played on it. Without that explanation this looks exactly like a bug.

### 4.4 Device picker overlay

Opened with `d`, drawn over the player. The background stays visible but dimmed to
`Faint`.

```
┌─ spindle ────────────────────────────── ● MacBook Pro ─┐
│                                                        │
│  ┌───────────────┐  ╔══════════════════════════════╗   │
│  │               │  ║  Devices                     ║   │
│  │               │  ║                              ║   │
│  │               │  ║  > ● MacBook Pro    computer ║   │
│  │               │  ║      iPhone       smartphone ║   │
│  │               │  ║      Kitchen         speaker ║   │
│  └───────────────┘  ║                              ║   │
│                     ║  ↑↓ · enter · esc            ║   │
│                     ╚══════════════════════════════╝   │
├────────────────────────────────────────────────────────┤
```

- Overlay frame in `BorderFocus`.
- Active device prefixed with `●` in `Accent`.
- Cursor `>` in `Accent`.
- On selection the overlay closes immediately and the header shows a transferring
  indicator until the next fetch confirms.

### 4.5 Login

```
┌─ spindle ──────────────────────────────────────────────┐
│                                                        │
│   Sign in to Spotify                                   │
│                                                        │
│   Your browser should have opened. If it did not,      │
│   visit:                                               │
│                                                        │
│   https://accounts.spotify.com/authorize?client_id=…   │
│                                                        │
│   ⠋ Waiting for redirect…                              │
│                                                        │
├────────────────────────────────────────────────────────┤
│ ctrl+c cancel                                          │
└────────────────────────────────────────────────────────┘
```

Always print the URL, even when the browser launched successfully. Over SSH this is
the only path that works, and `xdg-open` fails silently there.

### 4.6 Status banner

Not a separate screen: one line above the help bar, player still visible beneath.

```
├────────────────────────────────────────────────────────┤
│ ⚠ Offline — retrying in 8s                             │
├────────────────────────────────────────────────────────┤
│ space play/pause · n/p track · ←→ seek · d devices · ? │
└────────────────────────────────────────────────────────┘
```

| Condition | Colour | Text |
|-----------|--------|------|
| Network down | `Warning` | `⚠ Offline — retrying in Ns` |
| Rate limited | `Warning` | `⚠ Rate limited — pausing for Ns` |
| No Premium | `Error` | `✕ Playback control requires Spotify Premium` |
| Token refresh in flight | — | not shown, silent |
| Device disappeared | `Warning` | `⚠ Device is gone — press d to switch` |

The banner **clears itself** when the cause resolves. It never needs acknowledging.

### 4.7 Terminal too small

```
┌──────────────────────────┐
│                          │
│  Window too small        │
│                          │
│  current:  52 × 16       │
│  needed:   64 × 20       │
│                          │
└──────────────────────────┘
```

Updates live while resizing. Below 26 columns, drop the frame and show only
`need 64x20`.

### 4.8 Expanded help

`?` expands the help bar into a full table; the player compresses above it.

```
├────────────────────────────────────────────────────────┤
│  space   play / pause      s      shuffle              │
│  n / p   next / previous   r      cycle repeat         │
│  ← / →   seek ∓5s          d      devices              │
│  ↑ / ↓   volume ±5         ?      close help           │
│                            q      quit                 │
└────────────────────────────────────────────────────────┘
```

The `bubbles/help` component does this natively with a `ShortHelp` / `FullHelp`
pair. Do not hand-roll it.

## 5. Transitions

| Event | Behaviour |
|-------|-----------|
| Track change | Text swaps immediately, artwork when loaded. **No fade.** |
| Play / pause | Icon and progress colour change immediately (optimistic). |
| Seek | Progress jumps immediately; `localProgress` is reset. |
| Volume | Value changes immediately; requests debounced 800 ms, otherwise a held key fires 20 API calls. |
| Overlay open / close | Instant, no animation. |

**No animation apart from the spinner.** Every animation means a redraw, and redraws
are the primary enemy of kitty artwork placement.

## 6. Implementation checklist

- [ ] Every colour readable in a light terminal
- [ ] Layout holds at 64×20; at 63×20 the "too small" screen appears
- [ ] Long track titles truncate, never wrap
- [ ] Usable with `--ascii` in a font without box-drawing glyphs
- [ ] Artwork box is the same size while loading
- [ ] Paused state readable at a glance, not only from the icon
- [ ] The no-device screen explains why the list may be empty
- [ ] The status banner clears itself
