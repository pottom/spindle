# spindle — screen designs

The ASCII drawings are indicative. **The dimension table in section 3 is
authoritative.** Where they disagree, the table wins.

## 1. Principles

**The terminal background stays the user's background.** Never paint a full-screen
background — it looks wrong in every theme and defeats the user's transparency
settings. Set foreground colours only.

**No frames.** Hierarchy comes from colour, weight and whitespace. Box drawing is
reserved for things that are genuinely meters: the progress line and the volume
bar. A terminal full of `┌──┐` reads as a form, not as a player.

**The accent colour comes from the album.** Each cover is sampled for the hue it
reads as, and that colour drives the artist name, the played part of the progress
line, the active toggles and the device dot. The screen changes character with the
music. Nothing else is coloured, so the accent always means "this, now".

**Icons need an ASCII fallback.** `⏸ ⏭ ⏮` are missing from many fonts. Under
`--ascii`, use `||`, `>>`, `<<`.

**No emoji.** Their width varies by terminal and shifts the layout.

**One window, three tabs.** The player, the playlists and the search are tabs of
the same window rather than separate screens, so `tab` never loses your place.
The artwork stays anchored to the left on all three, which keeps the eye still
across a switch and keeps the kitty placement from jumping around the screen.
Each tab decides for itself what belongs in that slot — see 4.9 and 4.10.

## 2. Palette

The base palette is fixed; the accent is not. `internal/ui/style` resolves both
against the terminal background, which Bubble Tea reports as `BackgroundColorMsg`.

| Role | Dark | Light | Used for |
|------|------|-------|----------|
| `Accent` | from the artwork, else `#1DB954` | from the artwork, else `#128A3E` | artist name, played progress, active toggles, device dot |
| `Text` | `#E8E8E8` | `#1A1A1A` | track title |
| `Muted` | `#8B949E` | `#57606A` | album name, timestamps, volume |
| `Faint` | `#484F58` | `#8C959F` | help bar, inactive toggles, missing cover |
| `Border` | `#30363D` | `#D0D7DE` | unplayed progress, empty volume |
| `Error` | `#F85149` | `#CF222E` | error line |
| `Warning` | `#D29922` | `#9A6700` | rate limit, offline |

An artwork accent is pushed to a lightness that stands out against the background
and given a saturation floor, so a muddy sleeve still yields something readable. A
cover with no usable colour at all keeps the default green.

No hex codes appear in `View()`.

## 3. Dimensions

| Element | Value |
|---------|-------|
| Minimum terminal | 64 × 20 |
| Content width | full width, capped at 100 columns, centred above that |
| Artwork | scales with the window: as tall as the body allows, never wider than half the content, always square on screen |
| Left margin | 3 |
| Right margin | 2 |
| Gap between artwork and info | 3 |
| Info column | remainder, minimum 32 |
| Tab header | 2 lines plus a blank: the labels and a rule under the active one |
| Status line | 1 line at the foot of the body |
| Help bar | 1 line, or 4 when expanded |
| Artwork share | half the content width on the player, two fifths on the browsing tabs |

Above 100 columns the layout must not stretch. A full-width TUI on an ultrawide
monitor is unreadable.

Squareness needs the pixel size of a cell, which is why `cover.DetectCellSize`
feeds the layout: a 41px-tall cell needs roughly 2.4 columns per row to look square.

## 4. Screens

### 4.1 Player — the main screen

```

   ████████████████████████████████████
   ████████████████████████████████████
   ████████████████████████████████████
   ████████████████████████████████████   Bohemian Rhapsody
   ████████████████████████████████████   Queen
   ████████████████████████████████████   A Night at the Opera
   ████████████████████████████████████
   ████████████████████████████████████
   ████████████████████████████████████   ━━━━━━●───────────────────────
   ████████████████████████████████████   2:14                     5:55
   ████████████████████████████████████
   ████████████████████████████████████
   ████████████████████████████████████   ⏮   ⏸   ⏭    ⇄   ↻   ━━━━━── 72
   ████████████████████████████████████
   ████████████████████████████████████

   ● MacBook Pro
   space play/pause · n/p track · ←→ seek · ? help
```

- Title in `Text` and bold, artist in the album accent, album in `Muted`. That
  three-level hierarchy carries the whole screen; nothing else competes.
- The text block is centred against the artwork rather than stretched to its
  height. Two columns anchored at their extremes with a hole between them reads as
  a bug; a centred block reads as a decision.
- Long titles truncate with `…`. **Never wrap** — wrapping changes height, and
  height changes make the layout jump.
- The progress line is a rule with the playhead `●` riding on it: played part in
  the accent, the rest in `Border`. **Paused, the whole line goes `Muted`,**
  playhead included. State has to be readable at a glance, not only from an icon.
- Shuffle `⇄` and repeat `↻` sit in fixed two-cell slots, `Faint` when off and
  accent when on, so turning one on cannot nudge the row sideways. Repeat-one is
  `↻1`.
- Volume is a short meter in `Muted` over `Border`, with the number beside it.
- The device sits on the status line at the foot: `●` in the accent while playing,
  `Faint` when paused.
- The window title carries `Title · Artist`, so the app stays identifiable without
  spending a row on its own name.

### 4.2 Artwork loading

Spinner centred in the artwork area. **The area does not change size** — if the
placeholder were smaller than the image, the layout would jump when loading
finishes.

If the download fails, replace the spinner with `♪` in `Faint`. Do not show an
error message; a missing cover is not important enough to interrupt use.

The artwork is re-rendered whenever the area changes size, which means on a window
resize and when the expanded help pushes the body shorter. Until the new render
arrives the area shows the spinner again, and the accent from the outgoing cover is
kept so the palette does not flash back to its default mid-swap.

### 4.3 No active device

This is **not an error screen**. It is the most common entry point. It replaces
the player tab only — the drawings in 4.4 to 4.6 are still from the framed era and
are kept for their content, not their chrome.

```
   now playing   playlists   search
   ━━━━━━━━━━━


   No active playback device

   Start Spotify on one of your devices, or pick one below.

   ● MacBook Pro                               computer
     iPhone                                  smartphone
     Kitchen speaker                            speaker


   Missing one? Spotify only lists devices that were recently
   active — open the app on it and play something for a moment.

   r refresh · tab switch · q quit
```

- The tabs stay live. Playlists and search need no active device, and taking them
  away because a speaker went to sleep would be gratuitous.
- The help bar drops the transport keys: with nothing playing they would be a lie.
  `r` re-asks for the device list.
- `●` marks whichever device Spotify still considers active, if any.
- Selecting one to transfer to arrives in M4; for now the list informs.
- The closing sentence matters. Spotify's Web API only returns recently active
  devices, so a freshly launched client often does not appear until something has
  played on it. Without that explanation this looks exactly like a bug.

### 4.4 Device picker overlay

Not built yet — M4. Opened with `d`, drawn over the player. The background stays visible but dimmed to
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

Not a separate screen: one line between the player and the help bar, player still
visible above it.

```
   ● MacBook Pro
   ⚠ Offline — retrying in 8s
   space play/pause · n/p track · ←→ seek · ? help
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

Centred in the window, nothing else on screen.

```
                    Window too small

                    current   52 × 16
                    needed    64 × 20
```

Updates live while resizing. Below 26 columns there is no room for the table, so
show only `need 64x20`.

### 4.8 Expanded help

`?` expands the help bar into a full table; the body compresses above it, which
also shrinks the artwork and triggers a re-render at the new size.

```
   ● MacBook Pro
   space play / pause       s shuffle
   n / p next / previous    r cycle repeat
   ← / → seek ∓5s           ? help
   ↑ / ↓ volume ±5          q quit
```

The `bubbles/help` component does this natively with a `ShortHelp` / `FullHelp`
pair. Do not hand-roll it.

### 4.9 Playlists

Two levels in one tab. `enter` opens a playlist, `esc` comes back out.

```
   now playing   playlists   search
                 ━━━━━━━━━

   ████████████████   Bowie Essentials
   ████████████████   Spotify · 6 tracks · 27m
   ████████████████
   ████████████████   ▸  1  Ashes to Ashes      David Bowie        4:23
   ████████████████      2  Life on Mars?       David Bowie        3:53
   ████████████████   ♪  3  Heroes              David Bowie        6:11
   ████████████████      4  Let's Dance         David Bowie        4:08
   ████████████████      5  Rebel Rebel         David Bowie        4:30
                         6  Under Pressure      Queen, David…      4:08

   ● MacBook Pro
   ↑↓ select · enter play · esc back · tab switch · ? help
```

- The artwork is the **playlist's**, not the playing track's — that already has a
  tab of its own, and a cover that changes under you while you read a list is
  noise. At the top level it follows the cursor; inside a playlist it stays put as
  the thing you are looking at.
- The artwork is top-aligned here, not centred: it starts on the same row as the
  heading, so the two columns read as one block.
- The track playing right now is marked `♪` and coloured in the accent, wherever
  it turns up in a list.
- Rows are cursor gutter, title, artist, duration. The artist column is dropped
  when the pane is too narrow to carry it, rather than squeezing all three into
  uselessness. A list longer than the pane ends in `… N more`.

### 4.10 Search

```
   now playing   playlists   search
                             ━━━━━━

   ████████████████   ⌕ bowie
   ████████████████
   ████████████████   ▸ Under Pressure         Queen, David…      4:08
   ████████████████     Ashes to Ashes         David Bowie        4:23
   ████████████████     Life on Mars?          David Bowie        3:53
   ████████████████     Heroes                 David Bowie        6:11
   ████████████████     Let's Dance            David Bowie        4:08
   ████████████████     Rebel Rebel            David Bowie        4:30

   ● MacBook Pro
   type to search · ↑↓ select · enter play · tab switch
```

- The field takes focus the moment the tab does: arriving at a search screen and
  having to press a key before typing is a small insult.
- Results follow every keystroke. Each query carries a sequence number, so a slow
  one landing after a newer one is discarded rather than flashing stale hits.
- The artwork previews **the highlighted result**, and the accent colour comes with
  it — browsing recolours the whole screen. The load waits 250 ms for the cursor to
  settle, so holding a cursor key does not queue an upload per row.
- `esc` clears the query; on an empty query it returns to the player.
- `q` types a q here, so quitting is `ctrl+c`. The help bar says so.

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

- [x] Every colour readable in a light terminal
- [x] Layout holds at 64×20; at 63×20 the "too small" screen appears
- [x] Long track titles truncate, never wrap
- [ ] Usable with `--ascii` in a font without box-drawing glyphs
- [x] Artwork area is the same size while loading
- [x] Paused state readable at a glance, not only from the icon
- [x] An artwork accent stays readable on both light and dark backgrounds
- [x] The artwork stays square whatever the cell aspect ratio
- [ ] The no-device screen explains why the list may be empty
- [ ] The status banner clears itself
- [x] Every tab keeps its own cursor across a switch
- [x] A list too long for the pane says so rather than silently ending
- [x] Browsing a list does not queue an artwork upload per row
