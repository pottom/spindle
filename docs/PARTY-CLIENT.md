# spindle-party-client

A screen for a room. One window, one picture: the lyric picture spindle already
draws — the line being sung set as type, with the water, the marks and the
dancers around it. It watches what a spindle elsewhere is playing and does
nothing else. It cannot skip a track or move the volume, and should not be able
to.

Written up 2026-08-12, after measuring the three things that could have killed
it. None of them did. Everything below with a number beside it was measured
rather than reasoned about; everything without one is a decision, and the reason
is given.

## What it is not

Not the interface. There is no queue, no library, no search, no settings, no
device picker, no cover, and no keys. Somebody who wants those is sitting at the
machine with the speakers.

Not a second player either. It holds no session and no token, and the daemon
gives it a read-only right so that it could not control playback even if it
tried.

## The artefacts

| | |
|---|---|
| `spindle` | one binary, interface and daemon, macOS and Linux |
| `spindle-party-client` | Windows, macOS, Linux, ARM |
| the picture | not an artefact, but the reason both can exist |

`spindle` stays where it is. cgo and librespot tie it to macOS and Linux — the
FLAC and vorbis decoders are C, and librespot has no audio driver for Windows at
all. That is not worth fighting.

The party client has no audio, no cgo and no login, so it goes wherever Go goes.

The picture has to become a package of its own, used by both. If it does not,
the two drift and in six months the party screen is drawing something the
interface no longer draws.

## One picture, and what it does when there are no words

The lyric picture, always. Never the waveform, the bars or the ladder. So there
is no mode to choose, none to keep in step between screens, and no key to change
it.

It does not go blank on a record with no synced lyric: `wordsNow` puts up the
record and the artist in the same idiom. That is most records — 13 of 22 measured
tracks have no timings — and it answers, for free, the question of how a room
knows what is playing.

## What was measured

**The terminal.** Alacritty on a Windows laptop drew spindle's own stream — 100
frames of 200×50 coloured braille, byte for byte the same file used on macOS — at
**296 frames a second**. That is 3.4 ms a frame against a 16.7 ms budget, five
times over, and in the same band as the three measured on macOS: Alacritty 501,
Ghostty 471, kitty 380, WezTerm 190.

**The network.** 600 spectrum requests from that laptop to the Mac, through a
SOCKS proxy: median **4.47 ms**, p95 5.85, p99 6.8, **spread 2.32 ms**. The bar
was a tenth of a beat, which is 50 ms at 120 bpm, because the picture keeps time
from the beat in the answer. One outlier at 87.8 ms in 600 delays a spectrum
sample rather than the beat, since the picture extrapolates from period and
phase.

**The server.** Load against the daemon, each client on its own connection at 60
requests a second:

| clients | requests/s | median | p99 | worst | failed |
|---|---|---|---|---|---|
| 1 | 60 | 0.27 ms | 0.56 | 1.66 | 0 |
| 5 | 300 | 0.24 ms | 0.49 | 1.18 | 0 |
| 20 | 1200 | 0.39 ms | 1.02 | 1.56 | 0 |
| 50 | 3000 | 0.57 ms | 1.22 | 2.51 | 0 |
| 100 | 6000 | 0.82 ms | 1.64 | 5.08 | 0 |

A hundred times the load still answers in under a millisecond. The reason is
that the spectrum and the waveform are answered off the player's loop, from
whatever the analyser last put down — no lock, no queue, nothing to block.

**The server, with music playing and something really drawing.** The three
measurements above are the easy cases: no music means the daemon is not decoding,
and a load generator does not draw. Repeated with a record playing and a real
interface open on the picture, plus four more clients:

| | |
|---|---|
| the daemon — decoding audio *and* serving five | **4.4% CPU** |
| one client drawing the picture at 60fps | **30.1% CPU** |
| answers | 0.17 ms median, p99 0.47, worst 3.52, none failed |
| what the audio path reported | nothing |

**Drawing costs seven times what serving costs.** That is the finding, and it
turns the question round: the server is not the constraint and will not become
one. Each screen needs its own machine's processor, which is exactly what a
screen in a room has.

It also names the one real risk: a client on the same machine as the daemon takes
30% away from the audio. Five screens on five machines is free; five windows on
the Mac would stutter.

## Decisions

**Which actions travel.** All five of the picture's keys write to the local model
and send nothing — the picture walks, keeping time goes off, the record's name is
told, the mark set changes, a company of dancers is called. For a party they
should reach every screen: the host presses a key and the room answers.

The rule that sorts them: **what is "make something happen" travels; what is "how
I want to look at it" stays.** So the record's name told by hand, the dancers, and
the change of mark set become events on the daemon's stream. Keeping time off
does not — it is an A/B switch for judging the picture, and it belongs to whoever
is judging. Which picture no longer arises: there is only one.

This is what lets the party screen have no keys at all and still have everything
happen on it.

**The colour, not the cover.** The big screen draws no artwork — there is not one
reference to it in `stage.go`, and `stageView` says so: "nothing but the picture".
What it needs from the cover is the accent, twice: the palette is built from the
playing track's, and the tide crosses to the next track's over the last twelve
seconds.

So the daemon serves those two colours — a few bytes rather than two JPEGs. The
client then decodes nothing, scales nothing, and needs no internet at all, only
the LAN. The daemon is the right place for it anyway: it already knows both URLs
and fetches once however many screens are watching.

**Reads and commands are different rights.** The daemon listens on
`127.0.0.1:3678` and the CLI deliberately reads only this machine. Opening that
without a lock would let anything on the network skip tracks and empty a queue.
So: an address to listen on, off by default, and a shared secret in a header,
generated on first use and printed once. The party screen is given a read-only
right.

**Whether screens agree is a setting, not a rule.** Nothing in the dealing is
random. Which mark set, how a line arrives, which figure comes by are all worked
out from the record and the moment — `markCastFor(record, starts)`,
`wordsMoveFor(text, starts, …)` — so screens showing the same track deal the same
by construction rather than by being synchronised.

Rather than choosing, the daemon hands each client a **seed** when it connects.
The same seed to everyone and the screens breathe together; a different seed each
and they live their own lives. One number in the handshake, one more input to the
picture, no branch and no "sync mode" anywhere in the drawing. Half the machinery
already takes a seed — `pick`, `markCrowdFor`, `markPicture` — it just comes from
the record today.

    spindle screens together | apart

Changeable while it runs, because a seed takes effect on the next record. Both
can be tried at the party rather than argued about beforehand.

Three things drift whatever the seed says, and all three are harmless: a screen
that joins mid-record has a different "what came before", the hand-called things
are hand-called, and anything answering the sound is answering a different
millisecond. So "together" means the same company, the same arrivals, the same
mark set — the water falls in its own time on each screen, which is better than
perfect mirroring anyway.

## The order of work

**1. Lift the picture out of the Model.** The large piece, and nothing else can
start until it is done. 7,141 lines across 13 files touching 137 Model fields —
but most of those are its own state (`words` 188 uses, `stage` 46, `face` 27,
`volume` 24, `tide` 22, `sign` 19, `run`, `scope`) and move with it. From outside
it takes eight things: the player state, the palette, the lyrics, the terminal
size, the head of the queue, and the crossfade.

    Show.Step(Input)      what Update does now
    Show.Draw(w, rows)    what View does now

There is a check that says when this is done:

    CGO_ENABLED=0 GOOS=windows go build ./internal/<the picture>/...

Today that fails, and the whole reason is one import: `internal/ui/settings.go`
reaches for `internal/daemon`, which drags in librespot and 49 packages behind
it. `internal/player` and `internal/ui/cover` are already clean.

**2. The daemon reachable, with the lock and the seed.**

**3. The accent served as a colour.**

**4. The client.** Everything else it needs is already served: `/status`,
`/player/spectrum`, `/player/waveform`, `/player/lyrics`, `/player/queue`. It runs
unattended for hours, so reconnecting is a requirement rather than a nicety — the
held-answer work behind `Warning: 110` helps, and the rest is a client that keeps
trying.

## Left open

**What the screen does when nothing is playing.** Muted has an answer already —
the row of marks with somebody putting their fingers in their ears, because a room
with no sound coming out of it is worth more than whichever picture was up.
Stopped and run-out have none: the picture stays up and stops moving, because
there is nothing for it to answer. In the interface that is a moment; at a party
it can be five minutes, and a frozen screen reads as a crashed machine.

Deliberately not decided here. It wants looking at rather than arguing about.

**The cover on Windows.** Nothing here shows one, so it does not matter for this
— but for the record: neither kitty nor Ghostty runs on Windows, and of the
terminals that do, only Rio speaks the kitty protocol at all. Whether it does the
Unicode placeholders spindle draws in is unmeasured.
