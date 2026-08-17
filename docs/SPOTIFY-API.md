# The Spotify Web API, as measured

Everything here was measured against a live account rather than read out of the
documentation, and the dates say when. Spotify changes what a registration may
do without changing a word of its documentation, so anything undated is a guess.

Two separate things authenticate:

- **The daemon** logs in as a Spotify Connect device, over librespot's own
  protocol. It plays the music. Nothing in this document affects it.
- **spindle itself** talks to the Web API over OAuth (PKCE, no client secret) for
  everything that is not playback: the library, search, artists, playlists,
  devices, transferring playback. This document is about that.

## What a registration may ask

The Web API is not one API. What it answers depends on which application is
asking — the client id — and Spotify has three tiers:

- **development mode**, which is what any application registered after November
  2024 gets. A daily quota, and a whole family of endpoints simply refused.
- **extended quota mode**, which has to be applied for and is, in practice, not
  granted to a program like this one.
- **grandfathered**, meaning registered before the clampdown and never demoted.

Measured 2026-08-17, same account, same endpoints, two registrations: spindle's
own (`1c227ccd…`, registered 2026, development mode) and ncspot's
(`d420a117…`, public in ncspot's source, extended quota mode).

| endpoint | own registration | ncspot's |
|---|---|---|
| `GET /search` | 200 | 200 |
| `GET /me/playlists`, `/me/tracks`, `/me/albums` | 200 | 200 |
| `GET /albums/{id}/tracks`, `/artists/{id}/albums` | 200 | 200 |
| `GET /playlists/{id}/items` (own lists) | 200 | 200 |
| `GET /playlists/{id}/items` (anybody else's) | **403** | 200 |
| `GET /playlists/{id}/tracks` | **403** | 200 |
| `GET /tracks?ids=…` (batch) | **403** | 200 |
| `GET /tracks/{id}` (one) | 200 | 200 |
| `GET /me/tracks/contains` | **403** | 200 |
| `GET /artists/{id}/top-tracks` | **403** | 200 |
| `GET /artists/{id}/related-artists` | **403** | 200 |
| `GET /recommendations` | **404** | 200 |
| `GET /audio-features/{id}` | **403** | **403** |

So on our own registration:

- **A playlist somebody else owns cannot be listed at all**, however public it is.
- **The like key is impossible.** The read (`/me/tracks`) is allowed and the write
  (`PUT /me/tracks`) is not, and `contains` is refused too — so even the question
  "is this saved" has no answer.
- **Queue metadata has to come from the daemon**, because the batch track lookup
  is shut. See `internal/player/local.go`.
- Audio features are gone for everybody, so nothing that needs tempo, key or
  energy can be built on this API at all.

## What it costs to ask

There is a **daily** quota as well as a rate, and reaching it is a lockout rather
than a throttle: `429 QUOTA_EXCEEDED` on every request afterwards, with
`Retry-After` measured at 23 hours. A window left open used to reach it.

Every request spindle makes is written down, so this is a matter of reading
rather than arithmetic:

    sort -k3 ~/.local/state/spindle/api.log | uniq -cf 2 | sort -rn | head

One line per request — time, method, path, status. The path carries no query, so
no search term or id is recorded. A line that says `kept` is a request that did
not go out, because the answer was already in hand.

Measured 2026-08-17, with the pacing and caching described below:

| | requests |
|---|---|
| starting up, nothing playing | 8, over 18 seconds, then quiet |
| an open window, nothing playing, three minutes | 0 |
| ten minutes of playing | 1 |
| ninety seconds of browsing | 3 |
| typing a six-letter query | 1 |
| typing it again after clearing the box | 0 |

What keeps it there, all of it in the code rather than in anybody's discipline:

- **The state comes from the daemon** whenever the daemon is the one playing, over
  localhost. The Web API is asked only when nothing is loaded here.
- **The cadence rests.** An answer that repeats — including "nothing is playing
  anywhere" — doubles the wait, up to a minute; any key, any press, or any answer
  that differs puts it back to three seconds.
- **A storm of device events is one request.** A device coming up says something
  changed a dozen times in three seconds; those coalesce into one ask a second.
- **Catalogue answers are kept for an hour** — search, an artist's albums, a
  record's tracks. Nothing under `/me` is ever kept: that is the account, it
  changes while spindle runs, and a stale answer to it is a wrong screen.
- **Lists are written to disk** and read back at once, checked against a
  playlist's `snapshot_id` where there is one. See `internal/ui/listcache.go`.
- **A query is asked once the typing stops**, not once per letter.

All of it passes through one place — `gate` in `internal/player/errors.go` — which
counts, keeps, and turns a 429 into a typed error while `Retry-After` is still
attached.

## The daemon's own token is not a way round it

The daemon answers `POST /token` with its session's access token, which belongs
to **Spotify's own desktop client id** (`65b708073fc0480ea92a077233ca87bd`) —
librespot logs in as that. It is tempting: no registration, no development mode.

Measured 2026-08-17, one call every twenty seconds for ten minutes: `/me`,
`/me/playlists`, `/search`, batch tracks, `contains`, related artists — **429 on
every one**, `Retry-After` between 15 and 55 seconds. Every librespot, ncspot and
spotify-player instance alive shares that id, so it is saturated as a matter of
course. It is not a source and it is not a fallback; it is a lottery.

(A token obtained *as* ncspot in August behaved the same way: minutes after
authenticating, `429` with `Retry-After: 86400`. That was the desktop id, not
ncspot's own registration — the two were conflated at the time, and the table
above is the corrected measurement.)

## What somebody who downloads spindle has to do

Nothing. **Decided 2026-08-17**: spindle ships with ncspot's registration and
authenticates as that unless it is told otherwise, so a fresh install lands in
the right-hand column of the table above — everything works.

Using your own instead is three steps:

1. Register an application at
   [developer.spotify.com/dashboard](https://developer.spotify.com/dashboard).
2. Add the redirect URI exactly: `http://127.0.0.1:3679/callback`. Spotify
   refuses the word `localhost`; it has to be the numeric form.
3. `spindle login <client id>`.

And that lands in the left-hand column — which spindle then **handles rather
than suffers**. See below.

## What the program does about it

Nothing optional is assumed. `player.Abilities` is a list of the things a
registration may or may not be allowed to do, each with the request that settles
it:

| ability | what is lost without it |
|---|---|
| `Collecting` | the heart on every list, and the key that fills it |
| `Elsewhere` | a shared playlist cannot be opened at all |
| `Suggesting` | who else sounds like this, and what might come next |

Adding a feature that needs Spotify's permission is one line in that list: the
probe decides it, the settings screen lists it, and whatever draws the feature
asks `Allowances.Has`.

The asking happens **only where the listener has brought their own application**
— the shipped one is known to be allowed everything, and a request spent hearing
that again is a request wasted. The answer is written down next to the client id
it was asked about (`~/.local/state/spindle/allows.json`, a week), because it is
the same answer for as long as the registration is.

Until the answer arrives, everything optional is off: a key that fails when it
is pressed is worse than a key that is not there. A failure to ask is not an
answer — nothing is written down and nothing is turned on, and the next run asks
again. Behind all of it the runtime refusal still stands: a 403 on the way past
turns the feature off for the rest of the run and says so once.

**The client id and the address the browser comes back to are a pair.** Spotify
checks the address against what the application registered, and it checks it
*after* the login, so a mismatch arrives as a refusal a long way from its cause.
spindle's own registration uses `/callback`; ncspot's takes a loopback port of
any number with `/login` on the end, and `auth.CallbackPath` knows about that one
by name. Anything else has to match whatever its owner registered.

### The choice, as it was decided

**Decided 2026-08-17: the second, with the third's honesty.** spindle ships
ncspot's id and handles a narrower one rather than refusing it. The three ways
it could have gone:

| | what the user configures | what they get | what it costs |
|---|---|---|---|
| **own application** (today) | dashboard, redirect URI, id | a private quota, no surprises | three steps before the first note, and the left-hand column above |
| **ncspot's id as the default** | nothing at all | every endpoint, at once | somebody else's registration: the consent screen says **ncspot**, and so does the "apps with access" list in their Spotify account; the quota is shared with every ncspot and spotify-player user; if it is ever rotated, every spindle stops |
| **asked on first run** | one answer | whichever they chose | one more question, and they can see what they are agreeing to |

spotify-player takes the second, and warns anyone who sets their own id that a
new registration will be refused and rate limited. That warning is the strongest
evidence in this document that the tier, not the code, is what decides how much
of Spotify a terminal player can reach.
