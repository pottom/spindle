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
| Placeholder cells emitted | one per cell of the artwork area, exactly |
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

**Still needs a live account to answer with numbers**, but the machinery to survive
being wrong about it is in place.

The cadence is unchanged: a 1 s local tick with a real `State()` call every fifth,
per `DESIGN.md` 4.1. What M3 added is what happens when that turns out to be too
often.

Spotify's own client library is no help here: it decodes the error body and throws
the `Retry-After` header away with it, and its built-in retry silently blocks the
calling goroutine instead of telling anyone. So the transport is wrapped, and a 429
becomes a typed `RateLimitedError` carrying the header before the client ever sees
the response. `http.Client` wraps transport errors in `*url.Error`, which
`errors.As` sees straight through.

Polling then stops for exactly as long as Spotify asked. Carrying on regardless is
how a short throttle becomes a long one. The banner counts the wait down and clears
itself.

A header that is missing, zero, negative or not a whole number of seconds falls
back to 5 s rather than to zero — waiting zero would send us straight back into the
limit.

Reducing the request count also helped on its own: a held volume key used to be one
request per keypress and is now one per 800 ms, and a skip now costs one control
call plus one confirming fetch rather than an immediate poll that would have
confirmed the wrong track anyway.

## 4. What is the measured latency on the control endpoints? Is a 2 s optimistic window enough?

**Yes — with about three times the headroom it needs.** Measured against a live
Premium account, spotifyd as the device, from Hungary.

Round-trip time of a call, five samples each:

| Call | min | median | max |
|---|---|---|---|
| `Devices` (read) | 57 ms | 61 ms | 74 ms |
| `SetVolume` | 89 ms | 94 ms | 97 ms |
| `SetRepeat` | 117 ms | 119 ms | 123 ms |
| `SetShuffle` | 121 ms | 125 ms | 139 ms |
| `State` (read) | 112 ms | 155 ms | 404 ms |

The round trip is not the interesting number, though. What the optimistic window
has to survive is **propagation**: how long after a command returns before a poll
agrees with it. That is longer, and it is what actually matters:

| Change | observed |
|---|---|
| pause | 711, 741, 746, 752 ms |
| resume | 467, 474, 521, 659 ms |
| track change | 466, 530, 564, 678 ms |

So the 2 s window holds comfortably — the slowest propagation seen was 752 ms, and
the window is 2.7 times that. It stays as it is.

**The 400 ms confirming fetch after a skip was wrong, and this is what caught it.**
`DESIGN.md` 4.1 guessed 400 ms. Every single measured track change took longer than
that — the fastest was 466 ms. The confirming fetch would have agreed with the
*old* track every time, leaving the wrong title on screen until the next
five-second poll: exactly the symptom the fetch exists to prevent.

It is now 800 ms, above the slowest sample seen with margin. The asymmetry is the
argument: being early costs five seconds of wrong title, being late costs a
fraction of a second.

These numbers are one account, one device, one network. The shape is what to trust,
not the digits.

### What the measurements changed

Timing the round trip was the least useful part. Two things came out of it that
actually altered the program.

**A skip used to take about five seconds to show on screen.** Pressing `enter` on
a search result or a playlist track sent the play command and then asked for
nothing, so the player tab kept the previous track until the next five-second
poll. It looked like the key had done nothing. Both now ask for confirmation.

**Guessing a delay was the wrong shape of answer.** A single fetch at a fixed
offset is either too early (confirms the old track, and you wait for the poll
anyway) or needlessly late. Asking repeatedly until the answer changes — first at
450 ms, then every 200 ms, giving up after 4 s — resolves near the median instead
of the worst case and survives a slow day. Measured on screen, a skip went from
~5 s to 1212 ms with the naive retry, then to ~650 ms once the first ask was
moved just under the fastest propagation seen.

**Then the queue removed the wait entirely.** `/v1/me/player/queue` says what is
coming, so a skip does not have to ask Spotify anything to know what to draw. The
next track's title, artist and artwork go up immediately and the confirming fetch
demotes to a check. Measured on screen: **37 ms**, which is the measurement
harness, not the program.

That required one guard worth recording: inside the optimistic window `adopt` was
taking over metadata from any snapshot that arrived, including one that still
showed the track being left. It would have flashed the old title back a moment
after the skip. A snapshot still reporting the track we are leaving is stale by
definition and is now ignored.

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

## Portability

Everything builds for `linux`, `darwin`, `windows`, `freebsd` and `openbsd`, on
`amd64`, `arm64`, `arm` and `386`. `make cross` keeps it that way.

Only two files ever needed splitting, both of them terminal syscalls:

- **Cell pixel size.** `TIOCGWINSZ` on unix; on Windows the console API reports
  the window in cells and never in pixels, and no Windows terminal answers the
  `CSI 14 t` query either, so the 2:1 assumption stands and the kitty backend
  supersamples to compensate.
- **Graphics protocol detection.** `poll(2)` on unix. Not implemented on Windows:
  reading the reply needs a console-specific wait that cannot be exercised from
  here, and a probe that hangs on stdin would swallow the first keystroke. Windows
  therefore gets halfblock automatically, and `--cover=kitty` remains available for
  the terminals that do support it there.

Nothing else was platform-specific. The artwork pipeline, the OAuth flow and the
whole UI are portable as written.

## Halfblock backend

Universal fallback, no detection required. It yields one pixel per cell across and
two down, so even a large artwork area stays coarse — a 50 × 20 cell area is a
50 × 40 pixel picture. That is precisely why the kitty path matters. Verified by
decoding the emitted escape sequences back into a bitmap and comparing against the
source.

Scaling is aspect-correct: the cell pixel size comes from `TIOCGWINSZ`, falling back
to a 2:1 assumption when the terminal reports zeroes. The kitty backend supersamples
2× whenever that fallback is in play, so an unreported cell size costs payload rather
than sharpness. The artwork area itself is sized from the cell
aspect ratio, so a square cover stays square whatever the terminal's cell shape.

## Do the lyrics ever arrive word by word? (measured 2026-08-13)

**No, and the payload says so twice over.** The question was whether the words
picture could know when each *word* is sung rather than each line. The daemon
reads Spotify's own `color-lyrics/v2`, which is fetched with the session's token
— our Web API token is refused there, `403 RBAC: access denied` — so the answer
had to come from a raw dump through a patched daemon.

Thirty tracks from the account's saved songs, of which 26 answered 200 (the other
four 404 and fall through to LRCLIB), **1327 lines** in all.

| | |
|---|---|
| `syncType` | `LINE_SYNCED` on 23, `UNSYNCED` on 3 |
| `syllables` | present on every one of the 1327 lines, **empty on all of them** |
| `endTimeMs` | present on every line, **`"0"` on all of them** |
| provider | Musixmatch on all 26 |

So the schema keeps a place for word or syllable timings and Spotify never fills
it, and it will not even say when a line *ends* — which would have been the
cheaper prize, since the largest error in guessing a word's moment is assuming a
line runs until the next one begins. Anything word-level has to be worked out
here, from the audio and the text.

The other fields the parser drops, for the record: `syncLyricsUri` (empty on all
26), `alternatives` and `translationLanguages` (empty), `previewLines` (the first
five lines again), `colors` (Spotify's own accents, which spindle takes from the
sleeve instead), `transliteratedWords` (empty), `capStatus` (`NONE`), and
`hasVocalRemoval` — **false on all 26**, so there is no karaoke stem to subtract
either.

### What the line stamps themselves say (measured 2026-08-13)

The same 26 dumps, 23 of them line-synced, 1081 lines with words and a next line
to measure against. The question was whether the words of a line can be placed
inside its window from the text alone.

**The window is not the text's.** A line's window is about two and a half
seconds whether it holds four syllables or fifteen — the median gap by syllable
count runs 2.4 s, 2.6 s, 2.5 s, 2.7 s, 3.3 s across the range, and a straight
line fitted through syllables against window explains nothing (**R² = 0.06**,
1080 lines). The windows are the song's, not the sentence's: fitted to a repeating
unit they sit on it three times better than chance (mean off-grid distance 0.15
against 0.25 for random), and the units come out at 1.2–3.0 s, which is a bar at
80–160 BPM.

**The singing rate is roughly constant.** The lower envelope of window against
syllables converges at **160–180 ms a syllable** for long lines, and the denser
quartile sits at 200–300 ms. That is an eighth note at 90–140 BPM — the same
range the tempo detector reports for these tracks.

**So the naive model is unusable.** Spread a line evenly across its window and
the last syllable lands **1.1 s late at the median and 2.4 s late at p75**. The
error is not in placing the words within the sung part; it is that a line
finishes long before the next one starts, and nothing in the text says when.

### LRCLIB, and what a second opinion is worth (measured 2026-08-13)

100 saved tracks looked up: **93 found, 73 line-synced**, 17 words only, 3
instrumental, 7 absent.

- **No word-level timings anywhere.** Not one of the 73 carries the LRC `<mm:ss.xx>`
  word stamps. Which also settles a latent worry: the parser leaves such stamps in
  the words, and none arrive, so nothing is drawing angle brackets on screen.
- **Blank-line stamps close 4% of lines** (147 of 3406). Where one is present it
  is worth a great deal — the window shrinks by a median of **6.7 s** — so they
  catch the worst case, the verse that runs into an instrumental, and not the
  common one.
- **Where the two sources are independent they agree to ~±40–100 ms** per line,
  after removing a constant per-track offset. Many entries are simply copies of
  each other (0 ms spread on 6 of 16 tracks); the honest outlier disagrees by
  645 ms. The constant offsets themselves reach **810 ms**, which matters to
  anything that mixes the two sources.

### The line window is a bar (measured 2026-08-13)

The daemon caches the tempo it measures from the audio, which is an independent
witness to the stamps a stranger typed. For the 22 dumped tracks that had one, the
repeating unit fitted to their line windows comes out at **four beats — one bar —
on 13 of them**, within a quarter of a beat, and at three beats on three more.
Two are degenerate fits on tiny units.

Allowing the tempo detector to be an octave out is what makes this readable: half
the rap tracks are reported at double time, and the halved reading is the one that
puts the window on a bar. That is a known ambiguity of the detector rather than a
fault in the lyrics.

So the coordinate system for the words is musical, not textual: a line starts on a
bar, and its syllables fall on that bar's eighths. The measured tempo is the one
number that carries both.

### A syllable is half a beat (measured 2026-08-13)

With the window shown to be a bar, the next question is what a syllable is worth.
Taking each track's fastest tenth of lines — the ones with no rest in them to
speak of — and dividing the window by the syllables in it:

**a syllable is 0.54 of a beat at the median**, quartiles 0.43–0.70. An eighth
note. Twelve of the 21 tracks sit nearest an eighth, five nearest a triplet
eighth (the rap and the sea shanty), and three nearest a quarter — those three
are sparse dance records where no line is dense enough for the estimate to have
anything to work with.

Turned into a model and run against every line:

| a syllable takes | lines the model overruns | predicted / window, median |
|---|---|---|
| a fixed eighth | 19% | 0.70 |
| a triplet eighth | 3% | 0.47 |
| the track's own fastest tenth | **8%** | 0.71 |

An overrun is the visible failure — the words would run past the line and into
the next — so the fixed eighth is not good enough on its own. The track's own
rate, estimated once from its lyric and its tempo, does better, and capping the
predicted end at the window makes an overrun impossible at the cost of crowding
the last syllables of a fast line.

So the model to beat: the line starts at its stamp, its syllables take the
track's own note value, it ends when they run out, and the rest of the window is
silence — the median line singing for 71% of its bar, a tenth of them for barely
a third of it.

### When a line actually stops being sung (measured 2026-08-13)

Thirty-six lines, tapped by ear against the daemon's playhead, with a guided
tapper — `cmd/spindle-tap`, removed along with the feature — that measures each
pass's own lag against the line stamps it already knows, which is what makes a
hand a usable instrument here.

| | lines | sung share of the window | ms a syllable |
|---|---|---|---|
| Tony Joe White — The Other Side | 12 | 0.68 | 389 |
| Majka — the slow verse | 12 | 0.76 | 442 |
| Majka — the rap, same record | 12 | 0.82 | **141** |

**A syllable is not worth a fixed number of milliseconds.** The rate varies
three to one inside a single record, at one tempo, sung by one man: 442 ms a
syllable in the verse and 141 in the rap. Anything multiplying syllables by a
constant dies there, and did.

The count is not useless, though, and it is worth writing down what it is worth.
How crowded a line is does predict how much of its window it fills — 1.5
syllables a second fills 0.68 of it, 1.8 fills 0.76, 5.6 fills 0.82, and fitted
across all 36 lines that is share = 0.63 + 0.033 per syllable a second. The
direction is real. What it is not is **useful yet**: line by line it explains a
fifth of the variance (R² = 0.22), and used as the model it does worse than the
flat rule below (288 ms against 211). The reason is this measurement's own noise
— the tapper's lag is subtracted with a ±150 ms uncertainty, which on a
one-and-a-half second rap line is a fifth of the whole thing, and the effect
lives mostly between the three sections rather than within them. A finer
measurement, from the audio rather than from a hand, could bring it back.

What holds instead is the **share of the window**, which moves between 0.68 and
0.82 across all three. Against every line measured:

| a line is sung for | median error | p90 |
|---|---|---|
| the whole window (what is drawn today) | 734 ms | 1926 ms |
| a fixed 2.8 s, capped at the window | 372 ms | 1118 ms |
| syllables × 431 ms, capped | 384 ms | 863 ms |
| **85% of the window, at most 3 s** | **211 ms** | **792 ms** |

So: the line starts at its stamp and stops at 85% of the way to the next one, or
three seconds, whichever comes first. No syllables, no tempo, no audio — three
and a half times better than what is drawn today, from two numbers.

The caveats are worth keeping in view. Both numbers were chosen on these 36
lines, so they flatter themselves; the tapper's lag (350–430 ms) is subtracted
from every one of them, and being 100 ms out about it moves everything; and
nothing here has been checked above 160 bpm, where three seconds may be too long
a ceiling. Where the words fall *inside* the sung part is a separate question,
and only the words pass answers it.

### The head start has to be a share of the line (measured 2026-08-13)

Putting the rule on screen turned up a second thing the same 36 lines could
answer. A line takes the screen before its own stamp — half a second, so it can
be read before it is sung — and the sweep across it has to finish by the time it
gives way. On a ballad that costs nothing. On a rap line of two seconds it is a
quarter of the line, and the light finished a median of **446 ms before the
singer did**.

Made a share of the window instead — fifteen per cent, with the half second as
its ceiling — the whole screen's error falls from **414 ms to 233**, and the fast
lines' from 446 to 162. Ten per cent is better still for them and starts to cost
the slow records, which is where the reading happens.

What does not matter, measured: the share of the window that is sung. At 0.85,
0.90 or 0.95 the answer is identical to the millisecond, because the head start
and the three second ceiling are what actually bind. The share is the number
everybody would tune first, and it is the one doing the least.

### The voice is not in the spectrum we draw (measured 2026-08-13)

The model that places the words is as good as a singer is predictable, and the
way past that is to measure the voice rather than guess at it. The daemon
already computes a spectrum thirty times a second, so the question was whether
the singing can be told from the band in it.

`cmd/spindle-listen` records that spectrum against the playhead — silently, as
it happens, because the analyser is tapped ahead of the volume control. Two
recordings, both over passages that had been tapped by ear so that "singing" and
"not singing" are known: 56 s of a slow blues and 28 s of a rap.

Every feature tried, scored as how far apart the sung and unsung frames sit in
standard deviations:

| | slow blues | rap |
|---|---|---|
| level, bands 16–18 (1.4–2.1 kHz) | 0.02σ | −0.15σ |
| level, bands 19–23 (3–8 kHz) | 0.51σ | −0.59σ |
| level, bands 12–20 | 0.11σ | −0.35σ |
| spectral flux, bands 12–20 | 0.30σ | — |
| 19–23 over 10–16, a ratio | 0.67σ | −0.27σ |
| where the loudest band sits | 0.22σ | — |

**Nothing separates them.** Half a standard deviation on one track and the
opposite sign on the other is not a detector; it is noise with a preference.

The reason is in the shape of the data rather than in the choice of window.
Twenty-eight bands log-spaced from 40 Hz to 16 kHz put three bars across the
whole of 1.4–2.1 kHz, where the guitar, the snare and the voice all live at
once; and every band is scaled against the mix's own loudness over a 40 dB
range, which is precisely the normalisation that flattens a voice into its
backing. The bands are built to be drawn, and they are good at that.

So the audio path is not closed, but it does not start here. What would answer
is a feature built for the question rather than for the picture — the analyser's
own 21 Hz bins carry a sung note's harmonic comb and its vibrato, and a
fundamental that moves between 80 and 400 Hz is a voice in a way that energy in
a band never is. That is a change to the fork, and a real one.

### Teaching the analyser to hear a voice, and what came of it (measured 2026-08-13)

The bands are built to be drawn, so the fork was patched to measure the thing
itself: from the same 2048-point transform, at 21.5 Hz a bin, four features that
have nothing to do with a picture — the strongest harmonic comb between 80 and
500 Hz and how far it stands above its rivals, the unnormalised energy of
200–4000 Hz, how much of that is new since the last window, and how far the
fundamental moved in semitones. Then the stereo image was split as well, mid
against side: a voice is mixed to the middle and cancels out of L−R, which is
the trick every karaoke machine is built on, and the difference between the two
over the vocal range is as close to "is somebody singing" as a mix that nobody
has separated will give.

Scored the same way as before, against passages timed by ear:

| | slow blues | rap |
|---|---|---|
| harmonic comb above its rivals | −0.47σ | −0.31σ |
| the fundamental found | 0.65σ | 0.00σ |
| how far the pitch moved | 0.32σ | 0.17σ |
| new energy, 200–4000 Hz | 0.74σ | 0.30σ |
| **centre against the sides** | **1.15σ** | 0.07σ |

The karaoke trick is the best thing tried by some distance, and on the blues it
genuinely finds the singing. On the rap it finds nothing: the drums, the bass and
the whole nineties production sit in the middle too, and the singing barely stops
long enough to give the measure a silence to compare against.

**One and a bit standard deviations is not a detector.** Frame by frame it would
be wrong a quarter of the time, and it holds on one record out of two. The
harmonic comb was worse than useless — it finds the bass guitar, which is the
loudest periodic thing in the range and is not the singer.

So the audio path is closed with what a mix will give: the voice cannot be told
from the band without separating them, and separating them is a model, not a
measurement. The line model stays. What was learnt is in the fork's `voice.go`
and in this table, and the patch is not merged: the daemon runs the pinned build
again.

### Two screens, two rulers (measured 2026-08-13)

Watched on the wall — "the last word of the line lags the singer, and not only on
this record" — and then found in the code: the two screens divided a line's
singing by two different measures.

The player screen spends it on **syllables**, and punctuation belongs to the word
it hangs off. The big screen spent it on **pieces**, and a piece there is a word
*or* a comma or a full stop of its own, because on that screen a mark has to be
able to move independently of the word it follows. Every piece got an equal
slice of the singing, so a line's marks took time off its words.

Where the two put the start of a line's last word, on a 3.5 s window:

| line | pieces | words | big screen | player screen | apart |
|---|---|---|---|---|---|
| And all along the borderlines, of everything we knew | 10 | 9 | 2677 ms | 2776 ms | −99 ms |
| They say the times are changing, on the other side. | 12 | 10 | 2479 | 2727 | **−247** |
| Pedig már eltelt jó pár év. | 7 | 6 | 2125 | 2603 | **−478** |
| Sose voltunk ilyen büszkék, | 5 | 4 | 1785 | 2231 | **−446** |
| Amit adtunk, abból szépet | 5 | 4 | 2380 | 2231 | +148 |

Both directions, up to half a second, on the same line at the same instant —
which is what the eye was catching: the screens do not agree with each other, so
at least one of them does not agree with the singer.

The pieces are worth keeping: cutting a comma loose is what lets it bounce on its
own. What is not worth keeping is paying it. A piece is now worth its syllables,
a mark is worth none and lights with the word beside it, and the same walk is
used by all four followings — see `wordsSyncShares`. On the same lines the two
screens now light the last word 0–10 ms apart, which is inside a frame.

Worth knowing for later: this was **not** the tapped model being wrong. Against
the 33 hand-tapped line endings, at the moment the last word is actually sung the
model sits at 91% (Tony Joe White) and 92% (Majka) of the line, which is within
one piece of where it should be. The error was between the two screens, not
between the model and the music.

### Why the word-level following was taken out (2026-08-13)

Everything above about where the voice is inside a line was measured, and the
measurements hold. The feature built on them does not, and it is gone: the four
followings on the big screen, the sweep across the line on the player, the
syllable ruler, and the key that let a record be held to a shorter span.

What closed it, in order:

**The line model is right on the records it was fitted on.** Against the 33
tapped line endings, at the moment the last word is actually sung the model sits
at 91% (Tony Joe White) and 92% (Majka) of the line — inside one word of where it
should be.

**It is half a second wrong on a record it was not.** Mike Mana's "Never The
Same" has fifteen lines in two minutes fifty-three; every window is longer than
three and a half seconds, so the share never binds and the ceiling is the whole
answer. Judged by ear in the room, a line there is sung for 1.8 s where the
ceiling gives 3.

**The ceiling cannot be a constant, and syllables do not fix it.** Sweeping a
per-syllable ceiling across the tapped lines and that record together:

| ceiling | tapped, median error | what it gives Mike Mana |
|---|---|---|
| 3 s (what was shipped) | 514 ms | 3.00 s |
| 300 ms a syllable | 1037 ms | 1.80 s ✓ |
| 400 ms a syllable | 694 ms | 2.40 s |
| 500 ms a syllable | 390 ms ✓ | 3.00 s |

Tony Joe White's five-syllable lines really are held for three seconds — 600 ms a
syllable — and Mike Mana's five-syllable lines are done in 1.8. Same count,
double the pace. Whatever is right for one is twice wrong for the other.

**And a sheet cannot say which it is.** Thirty-one sheets were collected as they
arrived and asked the only question a sheet can answer about pace: the tightest
line in it, window over syllables, which the singing must fit inside. The range
across records runs 155 to 640 ms — and the two records above land at 441 and
422, adjacent, with opposite answers. A record whose singing never fills a window
leaves no trace of how fast it sings, because nothing marks where the singing
stopped and the band played on.

The audio was already closed for the same question (see above: 1.15σ at best, and
the harmonic comb finds the bass guitar).

So the position of a voice inside a line is not knowable from what arrives, it
is wrong by half a second on records that are not rare, and the only alternative
was a knob for the room to set per record. A screen that has to be corrected by
hand to stop lying is worse than a screen that only claims what it knows. The
words are lit a line at a time, which is exactly what a sheet says.
