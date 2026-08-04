# spindle

A Spotify player for the terminal that plays the audio itself.

Not a remote control: spindle registers as a Spotify Connect device and decodes
the stream in its own process. That is what lets it show you a waveform of the
music actually leaving the speakers, measure the tempo of what is playing, and
edit the queue as one list rather than asking a server to do it.

    ┌ screenshot: the player screen with a cover, the waveform and lyrics ┐
    └ screenshot: the queue screen mid-reorder                            ┘

Requires a Spotify **Premium** account.

## What it does

**Plays.** Its own Connect device, named `spindle`, at up to 320 kbps, with an
optional crossfade. Quitting the interface leaves the music playing; there is a
separate key for taking it with you.

**Shows the music.** A trigger-synced waveform or a spectrum with peak markers,
drawn in braille and coloured from the album cover's own accent. Album art
through the kitty graphics protocol where the terminal has it, half blocks
everywhere else.

**Knows the tempo.** The beat rate is measured from the audio as it plays and
remembered per track, so the queue can show a bpm column. Spotify's own
audio-features endpoint is closed to applications registered today; this is the
way round it.

**Reads the words.** Synced lyrics, faded either side of the line being sung and
swept across as it goes.

**Lets you rearrange.** One list of what is coming, whether a track was queued
by hand or came from the album. Remove a track, move it a place at a time, or
bring it forward to play now — and everything else keeps its place.

## Keys

`?` shows the full list, and each screen's own keys are in the bar at the
bottom. The ones worth knowing:

| | |
|---|---|
| `space` | play / pause |
| `n` / `p` | next / previous track |
| `←` `→` | seek ∓5s |
| `↑` `↓` | volume ±5 (on the player screen) |
| `m` | mute, remembering the level |
| `s` `r` | shuffle, cycle repeat |
| `v` | waveform → spectrum → off |
| `l` | lyrics |
| `u` | a glance at what is next |
| `d` | devices |
| `tab`, `1`–`4` | move between screens |
| `q` / `Q` | quit / quit and stop the music |

On a list: `enter` plays, `o` plays only that track and keeps the queue, `a`
adds to the queue, `x` removes, `j` / `k` move a queued track.

## Getting it running

    go build -o spindle ./cmd/spindle

spindle needs a Spotify application to authenticate as. Register one at
[developer.spotify.com/dashboard](https://developer.spotify.com/dashboard), add
this exact redirect URI

    http://127.0.0.1:8888/callback

and then

    ./spindle login <client id>
    ./spindle

The client secret is not needed: authorisation is PKCE. Playback needs a second
authorisation, which the daemon asks for on its own the first time.

Settings live in `~/.config/spindle`:

    spindle quality low|middle|high      96, 160 or 320 kbps
    spindle crossfade <seconds>|off      up to 12s
    spindle daemon stop                  stop the playback device
    spindle --cover-info                 what the terminal was found to support

### From a shell

The daemon can be driven and read without opening the interface, which is what a
key binding or a status bar wants:

    spindle play | pause | toggle        resume, stop, or flip between them
    spindle next | prev                  move through the queue
    spindle status                       what is playing, one field per line
    spindle queue                        the playing track and what follows
    spindle volume [0-100]               report the level, or set it
    spindle seek 90 | +30 | -15          to a position, or by an offset

Output is plain text, one thing per line and never coloured; `--json` on any of
them prints the daemon's own answer instead, for `jq`. Nothing starts a daemon:
these talk to the one already running, and say so when there is none.

The exit code says which it was: `0` done, `1` refused, `3` no daemon is
running, `4` the daemon is running but nothing is playing.

## What it needs

A terminal, a Premium account, and Go 1.26 to build. CGO is required — the audio
decoder is C — so it builds for the machine it is built on. macOS and Linux;
Windows is not supported.

Artwork is drawn through the kitty graphics protocol when the terminal reports
it (kitty, Ghostty, WezTerm), and through half blocks otherwise, which works
anywhere with 24-bit colour.

## Shape

Two processes. A **daemon** holds the Spotify session and the audio pipeline and
serves a small API on `127.0.0.1:3678`; the **interface** draws and sends
commands to it. That is why closing the window does not stop the music, and why
the queue can be edited at all — the order lives in the device's own state
machine, and only the process holding it can rewrite it correctly.

The daemon is [go-librespot](https://github.com/devgianlu/go-librespot), forked
to add what a controller needs from it: queue editing, a tap on the audio for
the waveform and the tempo, and lyrics.

## Licence

GPL-3.0 — see [`NOTICE.md`](NOTICE.md) for why, and for what else is in the
binary.
