# Licence

spindle is distributed under the GNU General Public License, version 3. The full
text is in `LICENSE`.

## Why the GPL, and not something shorter

Not a preference. spindle plays audio itself rather than only remote-controlling
another player, and it does that by compiling
[go-librespot](https://github.com/devgianlu/go-librespot) into its own binary.
go-librespot is licensed under the GPL-3.0, and a binary that contains it can
only be distributed under the same terms.

This is worth stating plainly because the obvious comparison invites the wrong
conclusion: other terminal Spotify clients are MIT-licensed, which is possible
for them because the Rust librespot they use is itself MIT. The difference is in
the library, not in the ambition.

The alternative would be to drop the embedded device and go back to being a
remote control for someone else's player — which would take with it the
waveform, the spectrum, the measured tempo and the editable queue, all of which
exist only because the audio passes through this process.

## What spindle carries

| project | licence | what it does here |
|---|---|---|
| [go-librespot](https://github.com/devgianlu/go-librespot) (forked) | GPL-3.0 | the Spotify Connect device: session, decoding, playback |
| [Bubble Tea, Bubbles, Lip Gloss](https://charm.sh) | MIT | the terminal interface |
| [zmb3/spotify](https://github.com/zmb3/spotify) | MIT | the Spotify Web API client |
| [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2) | BSD-3-Clause | the PKCE authorisation flow |
| [gofrs/flock](https://github.com/gofrs/flock) | BSD-3-Clause | one daemon at a time |

Spotify is a trademark of Spotify AB, which has nothing to do with this project.
A Premium account is required, and spindle asks Spotify for permission the same
way any other application does.
