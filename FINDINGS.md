# spindle — findings

Answers to the six questions in `DESIGN.md` section 7, filled in as the milestones
that can answer them are completed. Questions 3 and 4 need the live Spotify API and
stay open until M2/M3.

Measurements below were taken on macOS 15 (arm64), Go 1.26.4, in an 80×24 terminal,
against the mock backend.

## 1. Does the kitty placeholder mode cooperate with Bubble Tea's diff?

**Yes, and it needed no hacks in the view layer.** This was the risk the whole
prototype was built to resolve, and it came out better than expected.

The mechanism: the image is transmitted once with `a=T,q=2,f=100,t=d,i=1,U=1,c=20,r=10`,
which creates a *virtual* placement. `View()` then emits a 20×10 rectangle of
`U+10EEEE`, each cell carrying its row and column as two combining diacritics, with
the image id encoded in the run's foreground colour.

What was verified by parsing the bytes Bubble Tea actually wrote to the tty:

| Property | Result |
|---|---|
| Placeholder cells emitted | 200 — exactly 20 × 10 |
| Cells with both combining marks intact | 200 / 200 |
| Row indices decoded | 0–9, twenty cells each |
| Column indices decoded in row order | 0–19, in sequence |
| `lipgloss.Width` of one placeholder row | 20 — the text layer counts one cell per placeholder |
| Grapheme clusters per row (uniseg) | 20 — base plus marks cluster correctly |
| `ansi.Truncate(row, 20)` | passes all 60 runes through untouched |

Redraw behaviour, which is where a naive implementation falls apart:

| Event | Uploads | Placeholder repaints |
|---|---|---|
| First paint | 1 | 1 |
| Six idle seconds (six progress ticks) | 0 | 0 |
| Track change | 1 | 1 |
| Terminal resize 80×24 → 100×30 | 0 | 1 |

The idle row is the important one: the progress bar ticks every second and the diff
leaves the artwork cells alone, so there is nothing to flicker. On resize the image
is *not* re-transmitted, yet it follows the frame to its new position, because the
placeholders are ordinary text that the diff relocates like any other.

Four things that did have to be got right:

- **`q=2` on the transmission.** Without it the terminal replies with an `\x1b_G…`
  acknowledgement, and Bubble Tea reads that reply as keyboard input.
- **The upload happens in a `tea.Cmd`, not in `View()`.** `View()` stays pure and
  returns only text; the escape sequence goes straight to the tty from the pipeline
  goroutine. This is the one place the program writes outside Bubble Tea's renderer.
- **A single, reused image id.** Re-transmitting under `i=1` replaces the image,
  which is exactly the behaviour wanted on a track change, and it keeps the
  terminal's image store from filling up over a long session.
- **Detection cannot use `os.File` read deadlines.** This one cost a release: the
  first implementation timed the reply with `os.Stdin.SetReadDeadline`, which on a
  terminal fails outright with `file type does not support deadline`, because Go's
  runtime poller will not accept a tty. The detector took that error as "no reply"
  and *every* terminal was classified as unsupported, so the kitty path never ran
  and the fallback quietly rendered 20 × 20 pixel artwork. The fix is `poll(2)` on
  the raw descriptor. Detection is now regression-tested from both sides: a pty
  that answers the graphics query yields the kitty backend, one that answers only
  the device attributes query yields halfblock.

Detection runs before `tea.NewProgram`, so the query and its reply cannot collide
with the event loop. With no reply inside 200 ms the program falls back to halfblock;
measured cost of a negative detection is the full 200 ms timeout, once, at startup.

Transmission size is 113–146 KB of base64 per track change, in 4 KB chunks.

**Still to confirm by eye:** that Ghostty and kitty actually *display* the result.
Everything above is a measurement of the byte stream, not of pixels on a screen.
`spindle --cover-info` reports the detected backend and cell size, which is how a
fallback gets told apart from a kitty backend that is simply not drawing.

## 2. `go-termimg` or a hand-rolled implementation?

**Hand-rolled**, and `blacktop/go-termimg` was not evaluated. The hand-rolled
placeholder path is about 120 lines plus a 297-entry generated table, and it works
without hacks (question 1), so there was no remaining problem for a dependency to
solve. This is a deviation from the M5b plan, which asked for a comparison; it is
recorded here rather than silently skipped.

The generated table is `internal/ui/cover/diacritics.go`, derived from kitty's
`gen/rowcolumn-diacritics.txt`.

## 3. What is the smallest polling interval that avoids 429? What is the real quota?

**Open.** Needs the live Web API — M2/M3. The current cadence is the 1 s local tick
with a real `State()` call every fifth tick, per `DESIGN.md` 4.1.

## 4. What is the measured latency on the control endpoints? Is a 2 s optimistic window enough?

**Open for the real API.** Against the mock's 150 ms artificial latency the 2 s
window is ample: a pause survives the next two resync polls without snapping back,
verified both by unit test and by driving the running program.

## 5. Do kitty and Ghostty differ in placement behaviour?

**Open.** Only Ghostty is available on this machine, and the verification so far is
of the emitted byte stream rather than of the rendered picture. `--cover=kitty` and
`--cover=halfblock` force a backend so the two can be compared side by side once a
kitty install is at hand.

## 6. Binary size, and memory after one hour — does the cover cache leak?

**No leak.** Resident memory plateaus once the LRU is full:

| After | RSS |
|---|---|
| first cover | 22.7 MB |
| 40 track changes | 34.5 MB |
| 80 track changes | 35.6 MB |
| 120 track changes | 35.8 MB |
| 200 track changes | 35.8 MB |

The in-memory LRU holds ten decoded images; with four covers in rotation it fills
immediately and then stops growing. The disk cache held exactly four files (440 KB)
after those 200 track changes, so each album is downloaded once, as intended.

Binary size: **10.6 MB** (`go build`, no flags, arm64).

A full hour of continuous running has not been measured; 200 cover swaps in a few
minutes is a harsher test of the cache than an idle hour would be, but it does not
cover slow drift.

## Halfblock backend

Universal fallback, no detection required. At the 20 × 10 cell artwork box it
produces a 20 × 20 pixel image — recognisable but coarse, which is precisely why the
kitty path matters. Verified by decoding the emitted escape sequences back into a
bitmap and comparing against the source.

Scaling is aspect-correct: the cell pixel size comes from `TIOCGWINSZ`, falling back
to a 2:1 assumption when the terminal reports zeroes. The kitty backend supersamples
2× whenever that fallback is in play, so an unreported cell size costs payload rather
than sharpness. A square cover in the 20 × 10 box lands on 20 × 7 cells with a
48 px-tall cell and fills the box exactly with a 2:1 cell; the remainder is padding,
so the box never changes size.
