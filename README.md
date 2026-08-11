<p align="center">
  <img src="internal/logo/spindle.png" alt="spindle" width="620">
</p>

<p align="center">
  <b>The best-looking Spotify player in a terminal — and it plays the music itself.</b>
</p>

<p align="center">
  <img alt="licence GPL-3.0" src="https://img.shields.io/badge/licence-GPL--3.0-blue">
  <img alt="Go 1.26" src="https://img.shields.io/badge/go-1.26-00ADD8">
  <img alt="macOS and Linux" src="https://img.shields.io/badge/macOS%20%C2%B7%20Linux-333">
  <img alt="Spotify Premium required" src="https://img.shields.io/badge/Spotify-Premium-1DB954">
</p>

---

Most terminal Spotify clients are remote controls: they ask a server what is
playing and draw the answer. spindle registers as a Spotify Connect device and
**decodes the stream in its own process**, which changes what it is able to show
you.

|  |  |
|---|---|
| **A real waveform** | Not a bar meter fed by a guess — the audio actually leaving the speakers, trigger-synced so it holds still, drawn in braille and coloured from the album cover's own accent. |
| **The tempo, measured** | Spotify's audio-features endpoint is closed to applications registered today. spindle finds the beat in the audio as it plays, remembers it per track, and shows a bpm column. |
| **A queue you can edit** | One list of what is coming, whether a track was queued by hand or came from the album. Move it, drop it, or bring it forward to play now — and everything else keeps its place. |
| **A full screen worth looking at** | Water that falls on the beat, a row of drawn characters that dance to it, the lyric set as type and swept as it is sung. Press `f`. |

Closing the interface leaves the music playing. There is a separate key for
taking it with you.

## What it does

**Plays.** Its own Connect device, named `spindle`, at up to 320 kbps, with an
optional crossfade.

**Shows the music.** A trigger-synced waveform or a spectrum with peak markers.
Album art through the kitty graphics protocol where the terminal has it, half
blocks everywhere else.

**Reads the words.** Synced lyrics, faded either side of the line being sung and
swept across as it goes. Where a sheet has no timings — which is most of them —
it says so rather than guessing.

**Answers the beat.** On the full screen the picture keeps time with the record:
the marks turn on it, a line of the lyric sparks on it, and `b` turns keeping
time off so the two can be put side by side on the same song.

**Goes where you point it.** Liked songs, albums, artists, recently played, your
playlists and a search across all of them, with vim motions, counts and paging.

## Keys

`?` shows the full list, and each screen's own keys are in the bar at the
bottom. The ones worth knowing:

| | |
|---|---|
| `space` | play / pause |
| `n` / `p` | next / previous track |
| `←` `→` | seek ∓5s |
| `↑` `↓` | volume ±5 |
| `m` | mute, remembering the level |
| `s` `r` | shuffle, cycle repeat |
| `f` | the visualiser, full screen |
| `v` | waveform → spectrum → mirrored, with water → lamps → off |
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
    spindle notify on | off              announce each new track to the desktop
    spindle daemon start | stop          the playback device itself
    spindle daemon restart | status      restart it, or ask whether one is running

Output is plain text, one thing per line and never coloured; `--json` on any of
them prints the daemon's own answer instead, for `jq`. Nothing starts a daemon:
these talk to the one already running, and say so when there is none.

The exit code says which it was: `0` done, `1` refused, `3` no daemon is
running, `4` the daemon is running but nothing is playing. They read the daemon
on this machine and nothing else, so music coming out of a phone is silence as
far as they are concerned — which is what exit `4` means by "on this machine".

### In a status bar

A bar wants a sentence rather than a field per line, and wants it without
spawning `jq` every second:

    spindle status --line                ▶ Sultans of Swing — Dire Straits
    spindle status --format '{title}'    the fields, arranged your way
    spindle status --follow --line       a fresh line whenever something changes

The fields are `{icon}`, `{state}`, `{title}`, `{artist}`, `{album}`,
`{position}`, `{duration}`, `{volume}` and `{device}`; anything else is left as
it was written, so a typo says where to look. `--follow` holds the daemon's
event stream open and prints only when something actually happens, which costs
nothing while nothing does — and prints an empty line, rather than an error,
while the daemon is away.

### From the keyboard

The hardware play, next and previous keys drive spindle while it is the player
you are using: the daemon reads them where they pass, because macOS routes them
to whichever application it thinks is playing, and a program in a terminal is
not one of those.

They stay spindle's for half an hour after the music stops, so play, pause, play
reaches one player rather than starting Apple Music halfway through; after that
they go back to the system. The volume keys are left alone — the music has a
level of its own, separate from the machine's.

macOS asks for permission the first time: allow the terminal under System
Settings › Privacy & Security › Input Monitoring, then start the daemon again.
The keys are answered on macOS only; Linux has MPRIS, which is a different piece
of work.

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

Everything drawn on the full screen — the characters, the marks, the placards —
is a drawing cut from a sheet and baked into dots at build time by the tools in
`cmd/`. Nothing is rendered from a font except the lyric itself.

## Licence

GPL-3.0 — see [`NOTICE.md`](NOTICE.md) for why, and for what else is in the
binary.
