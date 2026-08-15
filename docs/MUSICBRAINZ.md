# MusicBrainz for spindle — a research report

Measured on 2026-08-15 against the live public API, User-Agent
`spindle-research/0.1 ( basipottom@gmail.com )`, about one request a second.
Everything marked **[M]** was measured with a real request. Everything else is
from the documentation and is labelled as such. Where the docs and the
measurement disagreed, the measurement won.

Nothing here is built. This is what is possible and what it costs.

---

## 0. The one-paragraph answer

MusicBrainz is a *relationship* database, not a *description* database. It has
no artist biography — none, not one field. What it does have, and what nothing
in the Spotify API gives you, is: **who played what on the recording that is
playing right now**, **who wrote it**, **which band members were in which
years**, **when the song first came out as opposed to when this reissue did**,
and **a stable public identifier you can pivot to Wikipedia, ListenBrainz and
the Cover Art Archive with**. ISRC gets you in the door with a **[M] 91% hit
rate** and one request. Credits are present on **[M] 69%** of the tracks you
will actually play. That is enough to build two or three genuinely good things
and a lot of things I would not build.

---

## 1. What MusicBrainz actually serves

### Entities

Thirteen core entities: `area, artist, event, genre, instrument, label, place,
recording, release, release-group, series, work, url`. Plus non-MBID lookup
resources `discid`, `isrc`, `iswc`, and non-core `rating`, `tag`, `collection`.

The three that matter for a player:

- **recording** — a specific audio master. This is what a Spotify track maps to.
  Carries `title`, `length` (ms), `first-release-date`, `disambiguation`,
  `isrcs`, `video`.
- **release** — a specific pressing or edition. Carries `barcode`, `date`,
  `country`, `status`, `packaging`, `label-info`.
- **release-group** — the album as an abstract thing, across all its editions.
  Carries `first-release-date`, `primary-type` (Album/Single/EP/Broadcast/Other),
  `secondary-types` (Compilation/Live/Remix/Soundtrack/…). **This is the entity
  that answers "when did this album really come out".**

Plus **work** (the composition — composers, lyricists, ISWC, and the link to
every recording of it) and **artist** (`type` Person/Group, `gender`, `area`,
`life-span.begin`/`.end`, `isnis`, `ipis`).

### The three endpoint shapes

```
lookup:  /ws/2/<entity>/<MBID>?inc=<...>&fmt=json
browse:  /ws/2/<result-entity>?<browse-entity>=<MBID>&limit=&offset=&inc=
search:  /ws/2/<entity>?query=<lucene>&limit=&offset=&fmt=json
```

`fmt=json` or `Accept: application/json`; `fmt=` wins if both are given. XML is
the default — always send `fmt=json`.

### `inc=` — what is real

Universal on every entity: `aliases`, `annotation`, `tags`, `genres`, `ratings`
(plus `user-*` variants, which need auth).

Relationship includes, on everything except `genre`: `artist-rels`, `url-rels`,
`work-rels`, `recording-rels`, `release-rels`, `release-group-rels`,
`label-rels`, `area-rels`, `event-rels`, `instrument-rels`, `place-rels`,
`series-rels`, plus the cascading `recording-level-rels`,
`release-group-level-rels`, `work-level-rels`.

Subqueries: `artist` → `recordings|releases|release-groups|works`; `recording` →
`releases|release-groups`; `release` →
`recordings|labels|release-groups|collections`; `release-group` → `releases`.
Modifiers: `artist-credits`, `media`, `isrcs`, `discids`.

**[M] Trap: the `/isrc/` resource takes a much narrower `inc` set than
`/recording/`.** `inc=genres` and `inc=release-groups` are both rejected there
with a 400. Practically: `/isrc/` is a cheap MBID resolver only; do the fat call
against `/recording/<mbid>`.

### Example 1 — ISRC → recording (the entry point)

```
GET https://musicbrainz.org/ws/2/isrc/USUM71703861?inc=artist-credits+releases&fmt=json
```

**[M]** trimmed, about 380 bytes in total:

```json
{"isrc":"USUM71703861","recordings":[{
  "id":"2463ba13-6f03-4d0a-a0e9-e15939d828bc",
  "title":"Cut to the Feeling","length":207959,
  "first-release-date":"2017-05-26","disambiguation":"","video":false,
  "artist-credit":[{"name":"Carly Rae Jepsen","joinphrase":"",
    "artist":{"id":"09887aa7-226e-4ecc-9a0c-02d2ae5777e1","name":"Carly Rae Jepsen",
              "sort-name":"Jepsen, Carly Rae","type":"Person","country":"CA"}}]}]}
```

One request; you now have the recording MBID *and* the artist MBID.

### Example 2 — recording → credits (the payoff call)

```
GET https://musicbrainz.org/ws/2/recording/<MBID>?inc=artist-rels+work-rels+genres+ratings&fmt=json
```

**[M] 1,817 bytes** for a lightly-credited pop track. Relations look like:

```json
{"target-type":"artist","type":"instrument","direction":"backward",
 "attributes":["guitar","slide guitar"],
 "artist":{"id":"...","name":"David Gilmour","sort-name":"Gilmour, David"}}
```

`type` is the credit kind, `attributes` is the instrument list. **[M]** Across 49
real tracks the credit types seen were: `instrument` ×189, `producer` ×64,
`vocal` ×39, `recording` ×36, `engineer` ×28, `mix` ×16, `programming` ×9,
`instrument arranger` ×8, `arranger` ×5, `conductor` ×5, then a tail of
`editor`, `performer`, `mastering`, `chorus master`, `video director`.

**Do not add `inc=releases+release-groups` casually.** **[M]** The same recording
with everything on is **43,312 bytes** — 24× larger — because it enumerates all
14 releases the track appears on.

### Example 3 — artist → members (the other payoff call)

```
GET https://musicbrainz.org/ws/2/artist/83d91898-7763-47d7-b03b-b92132375c47?inc=artist-rels+url-rels&fmt=json
```

**[M]** Pink Floyd:

```
Group | area England | life-span 1965 → 2014 (ended) | disambiguation "English Rock Band"
member of band | Syd Barrett    | [guitar, lead vocals, original] | 1965 – 1968-04
member of band | Richard Wright | [keyboard, lead vocals, original] | 1965 – 1981
member of band | Roger Waters   | [bass guitar, lead vocals, original] | 1965 – 1985
member of band | Nick Mason     | [drums (drum set), original, percussion] | 1965 – now
member of band | David Gilmour  | [guitar, lead vocals, slide guitar] | 1968-02-18 – now
instrumental supporting musician | Guy Pratt | [bass guitar] | 1987 – now
```

**[M] But this response is 86,070 bytes** — Pink Floyd has **116** artist
relations, most of them supporting musicians and one-off collaborations. You must
filter to `type == "member of band"` and you must be ready for a big payload.

---

## 2. Matching Spotify to MusicBrainz, ranked by reliability

### Rank 1 — ISRC → recording. **[M] 91%. Use this.**

54 real ISRCs from an independent source (Deezer's chart plus a spread of
catalogue albums — jazz, classical, rock, electronic) so the test was not
circular.

**[M] 49/54 hits (91%).** One of the five misses was a 503 from the rate limiter,
so the true rate is about 93%. The genuine misses were one brand-new French rap
track (2026-07-28) and three movements from a classical box set. **Classical and
very new indie are the weak spots; mainstream and catalogue are near-total.**

Ignore the frequently-quoted "only 15.3% of MusicBrainz recordings have an ISRC"
**[M]** (6,075,536 of 39,835,028). That statistic has the wrong denominator —
MusicBrainz is full of live bootlegs and obscure recordings nobody streams. From
the Spotify side, which is the side spindle is on, it is 91%.

**[M] The real hazard is cardinality, not coverage.** Histogram of
recordings-per-ISRC over the sample: **41 → one recording, 9 → two recordings,
4 → zero.** So **about 18% of hits are ambiguous**, and taking `recordings[0]`
blindly is wrong:

```
ISRC GBAYE9701274 (Radiohead — Airbag) maps to TWO recordings:
  4a7fea2e… | Airbag | first-release-date 1997-05-21 | length 284400
  9861822b… | Airbag | first-release-date 2008-05-28 | length 287880
ISRC USSM15900113 (Miles Davis — So What) maps to TWO:
  23db7f43… | "mono"  | 1959-08-17 | 545426
  272dbf93… | "studio, 1959-03-02: Columbia Studios, NY" | 1969 | 564000
```

The research first "discovered" that MusicBrainz's first-release dates were wrong
for a third of the sample. They were not — the wrong recording of the pair had
been picked. **Disambiguate by `length` against Spotify's `duration_ms` (±2 s),
then by the earliest `first-release-date`.** This is a correctness bug waiting to
happen and it would have shipped.

### Rank 2 — Spotify artist URL → artist. Exact, no false positives.

MusicBrainz stores Spotify links as URL relations of type **`free streaming`**,
and the reverse lookup works:

```
GET https://musicbrainz.org/ws/2/url?resource=https://open.spotify.com/artist/6sFIWsNpZYqfjUpaCgueju&inc=artist-rels&fmt=json
```

**[M]**

```json
{"id":"1042ada2-…","resource":"https://open.spotify.com/artist/6sFIWsNpZYqfjUpaCgueju",
 "relations":[{"type":"free streaming","direction":"backward","target-type":"artist",
   "artist":{"id":"09887aa7-…","name":"Carly Rae Jepsen","type":"Person","country":"CA"}}]}
```

One request, exact string match, **zero possibility of a false positive**. It
either hits or 404s. That property is worth a lot.

Coverage **[M]**: 33/34 (97%) of chart artists had a Spotify URL relation; on a
deliberately long-tail sample of 40 groups, **31/40 (78%)**. Call it **about 80%
in practice, about 95% for anyone famous.**

It requires the *exact* stored string. Strip Spotify's `?si=` tracking
parameters. Try both with and without a trailing slash. There is no wildcard:
**[M]** `?query=url:"open.spotify.com/artist/…"` returned `count: 0` — the URL
search index does not help.

The same works for albums: **[M]** MusicBrainz releases carry
`free streaming → https://open.spotify.com/album/…`, and recordings carry
`free streaming → https://open.spotify.com/track/…`. Track-level coverage is
thin, and ISRC already beats it, so do not bother at track level.

### Rank 3 — Barcode/UPC → release. **[M] 57%. Weaker than you would hope.**

```
GET https://musicbrainz.org/ws/2/release?query=barcode:00602478820656&fmt=json
```

Exact when it hits. **[M] 20/35 (57%)** on real UPCs from real albums.

**[M] Amy Winehouse's *Back to Black* missed.** So did Twenty One Pilots and The
Growlers. The reason is structural: **the UPC on a streaming album is the
*digital distribution* barcode, which is often territory- and reissue-specific,
and MusicBrainz's barcodes are predominantly from physical pressings.** Same
album, different number. This is not a data-quality gap that can be waited out.

Better album path: **get the release and release-group for free out of the
recording lookup already made.** `inc=releases+release-groups` on the recording
gives every release the track sits on, each with its release-group and
`first-release-date`. Costs bytes (43 KB), not requests.

### Rank 4 — Text search. **Only with field scoping, and never trust the score.**

MusicBrainz search scores are **normalised to the best hit in the result set**,
so score 100 means "the best of what I found", not "correct":

**[M]**

```
?query=Comfortably+Numb+Taylor+Swift        →  count: 82096
    score 100 | Comfortably Numb | The Big Wu
    score 100 | Comfortably Numb | Van Morrison
    score 100 | Comfortably Numb | Dominia
    score 100 | Comfortably Numb | Dar Williams
```

Four confidently wrong matches at maximum score. A client that took `results[0]`
would put Van Morrison's name on a Pink Floyd track.

The scoped, quoted form is safe:

**[M]**

```
?query=recording:"Comfortably Numb" AND artist:"Taylor Swift"  →  count: 0
?query=recording:"Zqxjw Plarbon" AND artist:"Ghfjdk Meepson"   →  count: 0
?query=recording:"Cut to the Feeling" AND artist:"Carly Rae Jepsen"
    100 Cut to the Feeling
     95 Cut to the Feeling (instrumental)
     89 Cut to the Feeling (Kid Froopy remix)
```

Field-scoped `AND` with quoted phrases returns **0 rather than garbage** when it
does not know. That is exactly the behaviour to want.

Add a duration guard: **[M]** `dur:[350000 TO 360000]` works and narrows
Bohemian Rhapsody to 26 hits, all real.

**Useful search fields** — recording: `isrc, arid, artist, recording, dur, qdur,
reid, rgid, firstreleasedate, primarytype, secondarytype, status, country, tag,
video, comment`. Release: `barcode, catno, laid, label, asin, releasestatus,
format, country, date, script, lang`. Artist: `arid, artist, sortname, alias,
primary_alias, type, country, area, begin, end, ended, gender, ipi, isni, tag`.

Escape Lucene specials (`+ - && || ! ( ) { } [ ] ^ " ~ * ? : \ /`) or the result
is 400s and false negatives. Titles containing `:` or `-` are extremely common.

### Rank 5 — ListenBrainz `acr-lookup` as a fuzzy fallback (see §6)

Better than MusicBrainz's own search for name-based matching, tokenless, and it
returns *canonical* MBIDs.

### How to tell a bad match from no match

1. **Prefer paths that can only 404.** ISRC lookup, barcode search and URL lookup
   are exact-key. They return the right thing or nothing. There is no middle.
2. **Never accept a search hit on score alone.** Require field-scoped quoted
   terms *and* a duration check *and* an artist-MBID check where there is one.
3. **Treat multi-result ISRC hits as a decision, not a list.** Disambiguate on
   duration; if two candidates are both within tolerance, take the earlier
   `first-release-date` and note that it was a guess.
4. **A wrong credit line is much worse than no credit line.** On a screen this
   small, showing nothing is a fine outcome. Fail closed.

---

## 3. What is genuinely there, and what is not

### Artist prose: there is some, and this section had it wrong

**Settled by measurement, 2026-08-15.** He said he had read a short description
of an artist on MusicBrainz with his own eyes, against a report that said no such
field exists. He was right, and the flat claim below is what needed correcting:
**the `annotation` field does sometimes hold prose about the artist.**

Measured on the artists in this library:

```
Darude       annotation: "His performance name came from his DJ days of playing
                          one track, rather a lot. …"          ← prose, about him
Queen        annotation: "'''Note:''' Please add any 2011 Remaster releases to
                          either ''Universal Island R…"        ← housekeeping
Miles Davis  annotation: none
Two Steps
  From Hell  annotation: "…is both a [label:…] and and [artist:…]"  ← half prose
Mike Mana    annotation: none
```

So it is neither "there is a description" nor "there is none": **the field is
there, and what is in it varies from a sentence about the artist to a note
between editors about how to file releases.** It cannot be shown unread. What it
can be is a fallback — where it exists and does not start with `'''Note:'''` or
a `[label:…]` link, it is a line worth having.

The rest of this section stands: there is no *biography* field, and the reliable
prose is Wikipedia's, reached through the Wikidata relation below. What exists in
MusicBrainz itself:

- **`disambiguation`** — a short editorial tiebreaker, e.g. **[M]** Pink Floyd →
  `"English Rock Band"`. One line, present on most notable artists, genuinely
  useful as a subtitle. Free: it is in every artist response, no `inc` needed.
- **`annotation`** (`inc=annotation`) — free text, but it is a wiki-ish editor
  note about *data*, not prose about the artist. **[M] Pink Floyd's annotation is
  `null`.** Do not build on this.

**The biography lives behind a Wikidata relation.** Measured cost: three
requests, only the first of which is rate-limited.

```
1. GET musicbrainz.org/ws/2/artist/<mbid>?inc=url-rels      → wikidata → Q2306   [1 req/s limit]
2. GET wikidata.org/w/rest.php/wikibase/v1/entities/items/Q2306/sitelinks/enwiki
3. GET en.wikipedia.org/api/rest_v1/page/summary/Pink_Floyd
```

**[M]** Step 2 returns
`{"title":"Pink Floyd","url":"https://en.wikipedia.org/wiki/Pink_Floyd"}`;
step 3 returns:

```json
{"description":"English rock band",
 "extract":"Pink Floyd are an English rock band formed in London in 1965 by Syd Barrett,
   Nick Mason (drums), Roger Waters and Richard Wright, with David Gilmour joining at the
   end of 1967. Gaining an early underground following as one of the first British
   psychedelic groups, they were distinguished by their extended compositions, sonic
   experiments, philosophical lyrics, and elaborate live performances…",
 "thumbnail":{"source":"…"}, "lang":"en"}
```

Steps 2 and 3 hit Wikimedia, which is *not* on MusicBrainz's 1/s budget and is
comfortable at far higher rates. The `extract` is a clean two-to-four sentence
lead paragraph — exactly the shape a panel wants — and `description` is a
one-line subtitle.

Wikidata coverage **[M]**: 28/34 (82%) of chart artists, 28/40 (70%) long tail.

Because the Q-id can be cached on the MBID, and the MBID on the Spotify artist
id, this collapses to one Wikipedia request per artist per session after the
first time.

### Relationships — the crown jewels

**Recording-level credits.** **[M] 34/49 (69%)** of real tracks had at least one
artist relation. The distribution is strongly bimodal — per-track counts were:

```
0×15, then 2,2,2,2,2,2, 3,3, 5,5,5, 7, 8,8, 9, 10,10, 12, 13,13,13,13,
14, 15,15,15,15, 17, 21,21, 30,30,30, 36
```

**Either you get nothing, or you get a proper session sheet.** 15 tracks had
zero; 19 had ten or more. There is no useful middle. That shape is good for
design: the panel is either empty or worth a screen.

This contradicts nothing about Spotify — the existing note that this music has no
instrument credits was measured against Spotify, and it holds there.
**MusicBrainz has instrument credits, and `instrument` was the single most common
relation type in the sample (189 of about 400).** Worth re-testing against the
actual library rather than a chart-weighted one.

**Band members.** As shown in §1, `member of band` relations carry instruments as
`attributes` and **`begin`/`end` dates**, so a line can read "Roger Waters —
bass, vocals, 1965–1985". **[M] 68%** of chart artists and **88%** of long-tail
groups had artist-artist relations of some kind.

**Work relationships (covers, samples, originals).** The recording → `work` →
composers chain works:

**[M]**

```
work "Comfortably Numb" | type Song | ISWC T-011.289.191-4 | languages [eng]
  composer  | David Gilmour
  composer  | Roger Waters
  lyricist  | Roger Waters
  based on     | "Don't Push"
  revision of  | "The Doctor (Comfortably Numb)"
  other version| "Comfortably Dumb" [parody]
```

**[M] 73%** of tracks had a work relation, so composer credits are broadly
available — and for pop, where the performer did not write it, that is the most
interesting line on the screen.

**"This is a cover of…" is where it falls apart.** **[M]** That work has **404
recording relations**, unsorted, with no artist credits attached and no date sort
available. Identifying the original means
`browse recording?work=<mbid>&inc=artist-credits` across **5 pages at limit=100 —
five seconds of rate-limited requests** — then sorting client-side. And the
answer is often junk: **[M]** the first page of that browse contained
`"A History of Post-War Britain, Part 3 (Includes: Hey You / Comfortably Numb /
…)"`. See §5.

### Release groups and first-release dates — this is real and it works

```
recording "Airbag" (ISRC GBAYE9701274, the 1997 one)
  RG "OK Computer"                       first-release-date 1997-05-21
  RG "Airbag / How Am I Driving?"        1998-04-21
  RG "Album Box Set"                     2007-12-10
  RG "Radiohead Box"                     2008-01-01
recording "So What" (ISRC USSM15900113, mono)
  RG "Kind of Blue"                      first-release-date 1959-08-17
  RG "The Original Mono Recordings"      2013-11-11
```

Take the **minimum** `release-group.first-release-date` across the recording's
releases. That is the "originally released" line, and it is correct — provided
the ISRC ambiguity was resolved first (§2, Rank 1).

`secondary-types` (`["Compilation"]`, `["Live"]`, `["Remix"]`, `["Soundtrack"]`)
is a cheap, reliable signal that a Spotify "album" is actually a compilation.

### Labels — thin

**[M]** `inc=labels` gives
`label-info: [{catalog-number, label:{id, name, sort-name, label-code, type}}]`.
But **[M]** the *E•MO•TION* tenth-anniversary edition's label came back as
`"Silent Records IGA"` with a null catalogue number — a distributor artefact, not
a meaningful imprint. Digital releases have poor label data. Not worth screen
space.

### Area, begin and end dates

**[M]** Solid and free (no `inc` needed): `area: "England"`,
`life-span: {begin: "1965", end: "2014", ended: true}`, `type: "Group"`,
`gender`, `country: "CA"`. A one-liner like `Group · England · 1965–2014` for
essentially every notable artist, at zero marginal cost.

### Tags and genres — present, but licensing-encumbered

**[M]** Carly Rae Jepsen's tags: `pop (15), electropop (5), synth-pop (4),
folk (3), dance (2), dance-pop (2), disco (2), pop rock (2), synthpop (2),
2010s (1), acoustic (1), canadian (1), chillwave (1), female vocalists (1),
teen pop (1)`. `inc=genres` returns the subset that maps to MusicBrainz's
curated genre vocabulary, with MBIDs and counts.

Two problems. They are noisy folksonomy — `female vocalists` and `2010s` sitting
next to `chillwave` — and **the genre and tag *associations* are CC BY-NC-SA, not
CC0** (see §4). Spotify already gives artist genres. Skip.

### Ratings — worthless here

**[M]** Carly Rae Jepsen: `{"value": 4.65, "votes-count": 8}`. *Cut to the
Feeling*: `{"value": 4.5, "votes-count": 2}`. **Two votes.** These are not
statistics, they are anecdotes. Do not display them.

### Cover Art Archive — not needed

**[M]** Works fine (`coverartarchive.org/release-group/<mbid>` → JSON index;
`/front-250` → 307 → 302 → archive.org datanode → 200, 13,933 bytes). No rate
limit — **[M]** 15 concurrent requests all succeeded, versus MusicBrainz search
which 503'd on the same burst.

But **[M] 66.2%** release coverage against Spotify's effective 100%, three
redirect hops to a per-item archive.org datanode with no CDN and no cache
headers, and `http://` URLs inside the JSON index that need rewriting.
**Spotify's cover art is better in every operational dimension.** The only
reasons to touch the archive are back covers, booklets and higher-resolution
scans — none of which the now-playing tab is asking for.

If ever used: `-500` (35 KB **[M]**), because it is guaranteed JPEG whereas the
full-size image can be PNG or even a PDF booklet.

---

## 4. Rules of the road

**Application key or auth: none needed.** **[M]** Every read request in this
report was unauthenticated and returned 200. OAuth and digest are only for
*submitting* data and reading user-specific data.

**User-Agent: mandatory, and it protects you.** Format
`AppName/version ( url-or-email )`. The docs explicitly name
`python-musicbrainz/0.7.3`, `headphones` and `beets` as being throttled harder as
a group — a generic or blank UA lands in a shared, punished bucket. A distinctive
UA is genuinely faster.

**Rate limit: one request a second per IP, on average.** The measured behaviour
is more nuanced and more dangerous than that sentence suggests:

- **[M]** Serial requests at 2–8 req/s did *not* 503. It is a moving average, not
  a hard gate.
- **[M] 15 concurrent requests → 15 × 503.** Concurrency is what kills you, not
  rate. Serialise every MusicBrainz call through one worker.
- **[M]** A 503 still arrived at 1.1 s spacing when requests bunched. Budget
  conservatively.
- **[M]** Response headers carry `X-RateLimit-Limit: 1200`,
  `X-RateLimit-Remaining`, `X-RateLimit-Reset` — **but they are undocumented and
  `Remaining` is not a request counter.** It fell by 385 across one 0.48 s gap
  and by 64 across a 0.12 s gap. It is an estimator output, not a budget.
- **[M] The 503 response strips all three `X-RateLimit-*` headers** and returns
  only `Retry-After` (observed: `7`) and `X-RateLimit-Zone: per-ip`. **Honour
  `Retry-After`. It is the only reliable signal, and it appears exactly when it
  is needed.**

Given spindle's architecture — `player.Player` off the UI thread, background
refresh, 30 s list cache — the right shape is **one dedicated MusicBrainz
goroutine with a 1.1 s ticker and a small priority queue** (now-playing first,
then visible rows, then nothing). Never call MusicBrainz from a render path.
Never fan out.

**Licensing.** The split is per-table, and it matters:

- **CC0, public domain, no attribution required:** artists, releases,
  release-groups, recordings, works, labels, areas, events, instruments, places,
  series, **and all relationships and URLs**. That is everything in §3 worth
  building.
- **CC BY-NC-SA 3.0 US:** tags, **genre associations**, ratings, annotations,
  derived statistics, search indexes, edit history.

So **credits, members, composers, first-release dates and Wikidata links are CC0
and carry no obligations whatsoever.** Tags, genres and ratings pull you into a
non-commercial share-alike licence. Skipping both keeps spindle entirely in CC0
territory. Note the genre *vocabulary* is CC0 while "this artist is trip hop" is
NC — same API, two licences.

Attribution is requested nowhere for the CC0 part. Crediting MusicBrainz on the
help screen is good manners, not a requirement.

Commercial tiers exist ($100–$2000/month) but trigger on "a company with a
current or expected revenue stream". A free open-source player is squarely in the
free tier with no paperwork.

**Bulk and replicated options.** All real, all far too heavy:

| Option | Size | Licence | Verdict |
|---|---|---|---|
| Full Postgres export `mbdump.tar.bz2` | **[M] 6.93 GiB** compressed, 100–350 GB on disk | CC0 | No |
| JSON dumps | `release.tar.xz` **21 GB** | CC0 | No — and `recording.tar.xz` is **[M]** about 0.5% complete by design (standalone recordings only) |
| Live data feed (hourly packets) | incremental | **BY-NC-SA only** | No; needs a free token |
| Canonical metadata dump | **[M] 2.17 GiB** (7.03 GiB unpacked) | CC0 | The only interesting one — an `artist+release+recording → MBID` lookup table with canonical-MBID redirects. Still 100× too big for a TUI |
| musicbrainz-docker | 16 GB RAM / 350 GB disk | — | No |

**[M] Dead path to avoid:**
`data.metabrainz.org/pub/musicbrainz/data/replication/` still returns 200 but its
newest packet is dated **18 May 2015**. Any documentation pointing there is
eleven years stale.

**Conclusion: use the live API with an aggressive local cache.** Cache by Spotify
ID → MBID → payload, persist to disk, and treat entries as good for weeks.
MusicBrainz data barely changes.

---

## 5. Where it would earn its place in spindle — ranked

Costs assume the ISRC is already resolved (1 request, cached forever per track).

### ★★★ 1. Credits panel on now playing — build this

Who played what on this exact recording.

```
producer   Sir Nolan
mix        Serge Courtois
guitar     David Gilmour
drums      Nick Mason
```

**Cost:** 1 request (`/recording/<mbid>?inc=artist-rels+work-rels`),
**[M] 1.8 KB**. Screen: it should share the lyrics panel's real estate — a second
page behind the same key, or a collapsed section.

**Why it wins:** information Spotify structurally cannot give, already recorded as
a dead end from the Spotify side, and a music-nerd terminal player is exactly the
right home for it. **[M] 69% availability**, and when it is there it is often
10–36 lines.

**Design note that follows from the measurement:** because the distribution is
bimodal (**[M]** 15 tracks with zero, 19 with ten or more), do not design a
fixed-height box. Either it is absent, or it is a scrollable list. There is no
"two credits" case to design around.

**And the caveat that decides whether to build it at all.** The 69% above is a
chart-and-catalogue sample. An earlier measurement, on 2026-08-10, took six
records from *this* library and got nothing: Two Steps from Hell "Emerald
Princess", Mike Mana "Never The Same" and MIRBRO "G LOVE" were **not in
MusicBrainz at all**, and Darude "Sandstorm", Queen "Bohemian Rhapsody" and
Miles Davis "So What" were present with **zero instrument credits**. Both
measurements are sound and they are about different music. Six records of the
library that actually plays here outweigh forty-nine of somebody else's:
**measure this again against a real sample of what gets played before building
the panel.**

### ★★★ 2. Composer and lyricist line — build this, it is nearly free

One line, `Written by Gilmour / Waters`. **[M] 73%** availability.

**Cost:** the `work` relation comes back in the same call as the credits, and one
more lookup on the work gets the composers — so one extra request, cached per
work, heavily shared across a discography.

**Why:** for pop and hip-hop, the majority of what people stream, the writer is
different from the performer and genuinely interesting. The cheapest real insight
in this report.

### ★★★ 3. Artist panel: disambiguation, area, years, Wikipedia lead — build this

```
Pink Floyd
English Rock Band · Group · England · 1965–2014

Pink Floyd are an English rock band formed in London in 1965 by Syd Barrett,
Nick Mason, Roger Waters and Richard Wright, with David Gilmour joining at
the end of 1967…
```

**Cost:** 1 MusicBrainz request (artist, `inc=url-rels` — and if the match came
via `?resource=`, the artist object is already in hand) plus 2 Wikimedia requests
**which are not on MusicBrainz's rate budget**. Cache the Q-id forever. **[M]**
The first two lines cost *nothing* extra — `disambiguation`, `area`, `life-span`
and `type` are in every artist response with no `inc` at all.

**Honest split:** the top two lines are ★★★ and free. The Wikipedia paragraph is
★★☆ — **[M]** 70–82% available, a third-party chain, and the one thing here that
is not MusicBrainz. But it is also the only artist prose that exists anywhere.

### ★★☆ 4. "Originally released 1959" on now playing and album views

When Spotify says *Kind of Blue, 1959* it is fine, but when it says *Michael
Jackson's This Is It, 2009* for Billie Jean, or a 2011 remaster date for a 1979
album, MusicBrainz corrects it.

**Cost:** **[M] 43 KB** on the recording lookup
(`inc=releases+release-groups` inflates it 24×), or a separate release-group
lookup. One line of screen.

**Why only ★★☆:** it depends entirely on resolving the ISRC ambiguity correctly
first. That went wrong during this research and produced a confident, entirely
bogus finding. If built, build the duration-based disambiguation first and show
nothing when the candidates disagree.

**Cheaper variant:** show it only when it differs from Spotify's date by more
than a year. Then it is a rare, high-value flag rather than a permanently
redundant line.

### ★★☆ 5. Band members in the artist view

The lineup with instruments and years, from **[M]** `member of band` relations.
**Cost:** 1 request, but **[M] up to 86 KB** for a heavily documented band,
because all 116 relations come back and are filtered client-side.
**Availability [M]:** 68% (chart) / 88% (long-tail groups) have *some* artist
relation; the `member of band` subset is smaller.

**Why only ★★☆:** lovely for Pink Floyd and meaningless for a solo pop artist or
a producer alias, which is most of what streams. A feature for the artist page,
not the player. And the raw list is mostly `instrumental supporting musician`
noise.

### ★☆☆ 6. Related artists

MusicBrainz has **no similar-artist feature.** What it has is `collaboration`,
`member of band`, `is person`, `supporting musician` — genealogy, not similarity.
Real similarity lives on ListenBrainz labs (§6).

### ✗ 7. "This is a cover of…" — do not build

**[M]** The work for *Comfortably Numb* has **404 recording relations**, without
artist credits and in no useful order. Five seconds of the entire request budget,
per track, for an answer wrong often enough to embarrass, about a fact that
applies to maybe 5% of what plays. The tempting shortcut — "composer ≠ performer
therefore cover" — is flatly wrong for all of pop.

The cheap 80% version: **surface the work's `based on`, `other version` and
`revision of` relations when they exist** (**[M]** free, in the work lookup
already made for composers). A curated editorial fact rather than an inference.
Zero extra cost, no false positives, fires rarely.

### ✗ 8. Tags, genres, ratings, labels, Cover Art Archive — do not build

- **Tags and genres:** noisy, duplicated by Spotify, and the one part of
  MusicBrainz that is not CC0.
- **Ratings: [M] two votes** on a Carly Rae Jepsen single. Displaying that as a
  rating is a lie.
- **Labels: [M]** digital releases give distributor artefacts like
  `"Silent Records IGA"` with null catalogue numbers.
- **Cover Art Archive: [M] 66.2%** coverage, three redirect hops, no CDN, no
  cache headers, `http://` URLs in the JSON, versus Spotify's ~100% from a real
  CDN. A strict downgrade, into the one pipeline that was hardest to get right.

### ✗ 9. Anything in the queue table

Enriching a 30-row queue costs 30-plus rate-limited requests for one refresh,
against a list that refreshes every 30 s. The arithmetic does not work, and the
queue table has no free columns anyway. Now playing is the only screen where the
request budget and the screen space both exist.

### Request cost per newly played track

```
ISRC → recording MBID                        1 req   (cached forever per Spotify track)
recording → credits + work                   1 req   1.8 KB
work → composers                             1 req   (cached per work, shared across discography)
artist (only on artist-view open)            1 req   + 2 free Wikimedia
─────────────────────────────────────────────────
steady state on track change:                2–3 req  ≈ 3 s of the 1/s budget
```

Comfortably inside budget for a player where tracks last three minutes.
Everything after the first play of a track is free.

---

## 6. Related sources worth knowing about

**Cover Art Archive** — see §3 and §5. No rate limit **[M]** (15 concurrent all
succeeded), which makes it operationally the *opposite* of MusicBrainz. Still not
worth adopting over Spotify's art.

**Wikidata → Wikipedia** — see §3. The only source of artist prose in the entire
ecosystem. Three requests, two of them off MusicBrainz's budget, cacheable
indefinitely. **[M] 70–82%** coverage.

**ListenBrainz — the underrated one.** Three things it does that MusicBrainz
cannot:

1. **A better fuzzy matcher than MusicBrainz's own search, tokenless.**
   `POST https://labs.api.listenbrainz.org/acr-lookup/json` with
   `[{"artist_credit_name":…,"recording_name":…}]` returns canonical MBIDs.
   **[M] 7/10 hit on raw Spotify titles, 10/10 after stripping
   `" - 2008 Remaster"` and `"(feat. …)"`.** For the same inputs, MusicBrainz's
   Lucene search returned `count: 0` on the dirty titles and non-canonical
   duplicates on the clean ones. **If a name-based fallback is ever built, build
   it against acr-lookup, not MusicBrainz search.** Results come back out of
   order keyed by `index`, and misses are simply absent from the array.
2. **Real listen counts.** `POST /1/popularity/recording` — **[M]** Portishead
   25.1M listens / 273K users; *Strangers* 1.01M / 131K. A meaningful popularity
   number, unlike MusicBrainz's two-vote ratings. (The per-artist GET variants are
   currently 500ing under load.)
3. **Genuine similarity.** **[M]**
   `POST labs.api.listenbrainz.org/similar-artists/json` for Portishead → Massive
   Attack, Radiohead, Björk, Air, Tricky, Morcheeba, Zero 7, Thievery
   Corporation, Röyksopp. And
   `GET /1/lb-radio/artist/<mbid>?mode=easy&pop_begin=30&pop_end=70` returns
   similar artists *plus* their popular recordings *plus* a popularity band
   filter in one call — which, given the queue thinking already written down, is
   the single most interesting endpoint outside MusicBrainz proper.

Rate limit **[M]: 30 requests per 10 s** on the main API, and far more burst
tolerant; labs returns no rate-limit headers at all. Read access needs no token;
only `/1/metadata/lookup` and scrobbling do — and labs `acr-lookup` does the
mapper's job tokenless. Go clients exist (`hirigaray/go-listenbrainz`, plus
in-tree implementations in Navidrome and gonic), none official.

**Note:** `/1/similarity/...` does **not exist** — **[M]** 404 on every shape,
confirmed against the blueprint registry. This is a common wrong assumption in
blog posts.

**AcousticBrainz — discontinued, and not the replacement for Spotify's audio
features.**

Announced dead in February 2022. **[M] The shutdown never actually happened** —
`acousticbrainz.org` returns 200 today and the API still serves `low-level` and
`high-level` for 7.56M recordings. But the data is frozen at 2022-07-06.

Three reasons not to build on it, in increasing order of severity:

1. **Coverage. [M]** 7.56M of MusicBrainz's 39.8M recordings = 19%. Sampled
   against 8 artists × 12 recordings: **32/96 = 33% hit rate**, and **Billie
   Eilish scored 0/12** — the post-2022 catalogue is permanently absent.
2. **It does not have the fields that were lost.** It has `bpm`,
   `key_key`/`key_scale`, `danceability`, `average_loudness`,
   `dynamic_complexity`. **There is no `energy` and no `valence` field. Neither
   has ever existed anywhere outside Spotify.**
3. **MetaBrainz killed it because the key and tempo were wrong.** Their own
   words: the data "simply isn't of high enough quality to be useful for much at
   all". **[M]** The very first record sampled reported `key_strength: 0.403` —
   the extractor saying it does not believe its own answer.

The dumps are still there for a static table: `rhythm` and `tonal` CSVs are about
2 GB to download, dedupable to about 200 MB stored (MBID + bpm + danceability +
key + scale). No network, no rate limit at playback. But that is 200 MB shipped
for a 33% hit rate on data its own authors disowned.

**What actually replaces audio features: local analysis, and spindle is unusually
well placed for it.** The program already decodes PCM and already measures tempo
from the stream. The realistic options:

- **`streaming_extractor_music`** (Essentia's CLI, **[M] 5 MB, still
  downloadable**) spawned on a decoded WAV. Its JSON is schema-identical to the
  AcousticBrainz API, so one parser serves both routes. Gets bpm, key and
  danceability immediately. Downsides: it is the beta-era extractor whose quality
  was disowned, and there is no macOS-arm64 build.
- **Essentia plus TensorFlow models** via a Python sidecar. Full quality, plus
  valence and arousal heads (`deam`, `emomusic`, `muse`) which are the only
  source of a valence-like number anywhere. **[M]** Models are small (18 MB
  embedding plus about 100 KB per head). But it is a Python runtime in the
  deployment, on top of the two Linux containers already needed for cgo, ALSA and
  libFLAC.
- **Licensing is the real blocker:** Essentia is **AGPLv3** (viral across a
  linked binary) and its models are **CC BY-NC-ND 4.0** — non-commercial *and*
  no-derivatives. There are no Go bindings, and cgo static linking against AGPL
  is the worst outcome available.
- **aubio** is GPLv3 and cgo-friendly but does onset, pitch and tempo only — no
  key, no danceability, no valence. It would re-solve the one problem spindle has
  already solved.

**And there is a timing problem worth deciding before any code:** the tempo and
model extractors need a *whole* track, but spindle only receives PCM as it plays.
Pre-computing for the queue means pre-fetching audio, which runs straight into
the audio-key rate limit already recorded. **The honest design is
analyse-as-you-play and cache by track ID — which means the feature only exists
on second encounter.** That is a product decision, not an implementation detail,
and it should be made first.

**Deezer** — free, keyless, and has `bpm`. **[M]** But 3/7 sampled tracks
returned `bpm: 0` as a silent null sentinel, including a famous Daft Punk single.
No key, energy, danceability or valence. Its one genuinely useful trick is
`https://api.deezer.com/track/isrc:<ISRC>` — an ISRC bridge, for a second opinion
on a match.

**TheAudioDB** — schema mirrors Spotify's exactly (`intTempo`, `strKey`,
`intDanceability`, `intValence`, `intEnergy`) but **[M]** 1/4 tracks had tempo,
and `intEnergy`/`intValence` were `null` in every case. Abandoned scraped data.

**Discogs** — **[M]** tracklist entries carry only
`{duration, position, title, type_}`. No audio features. Dead end for this
purpose. Its pressing and credit data is good, but it is rate-limited, needs a
token, and duplicates what MusicBrainz gives for free.

**Last.fm** — listeners, playcount, tags, wiki. No tempo, key, energy,
danceability or valence. Its per-track wiki text is another prose source if
Wikipedia misses, but the quality is poor and it needs an API key.

---

## Measured against this library, 2026-08-15

Everything above is a chart-and-catalogue sample. This is the same set of
questions asked of six records that actually play here — three of them the sort
of thing no database has heard of, three of them famous. It changes which of the
recommendations survive.

**Credits — 0 of 6.** Two Steps From Hell "Emerald Princess", Mike Mana "Never
The Same" and MIRBRO "G LOVE" are **not in MusicBrainz at all**. Darude
"Sandstorm", Queen "Bohemian Rhapsody" and Miles Davis "So What" are there, each
with **zero** artist relations. The 69% above does not transfer, and §5.1 — the
whole case for MusicBrainz as the report tells it — **does not apply to this
music**. It reproduces the 2026-08-10 measurement exactly.

**Composers — 3 of 3 of the tracks that exist.** Every one of them carries a
work relation, and every work carries its writers with an ISWC:

```
Darude — Sandstorm        → composer Ville Virtanen; arrangers Jaakko Salovaara, Ville Virtanen
Queen — Bohemian Rhapsody → composer and lyricist Freddie Mercury
Miles Davis — So What     → composer Miles Davis
```

So §5.2, the cheapest of the recommendations, is the one that survives contact
with this library.

**Artist prose — 4 of 6.** Darude (annotation *and* Wikipedia), Queen
(Wikipedia), Miles Davis (Wikipedia), Two Steps From Hell (Wikipedia: "American
trailer music company … founded in 2006 by Thomas Bergersen and Nick Phoenix").
Mike Mana is in MusicBrainz with nothing attached; MIRBRO is not in it at all.

**Artists — 5 of 6 found**, and the sixth is genuinely absent rather than badly
matched.

**And the limiter is real.** One artist lookup in that run came back 503 at
1.2 s spacing and needed a retry — §4 is not being pessimistic.

## The three things worth doing

1. ~~**ISRC → recording → credits**~~ — **struck out.** Measured twice against
   this library and it found nothing both times: three records absent from the
   database, three present with no credits at all. Keep the ISRC lookup, because
   everything else hangs off it; drop the panel it was for.

   What is left of this one is **the composer line**, which measured 3 of 3 —
   see above, and §5.2.
2. **`disambiguation`, area and years on the artist, free; the Wikipedia lead
   paragraph behind it, three cached requests.** The only artist prose that
   exists.
3. **A single serialised MusicBrainz worker at 1.1 s with a disk cache keyed on
   Spotify ID.** **[M]** Concurrency is what triggers 503s, not rate; honour
   `Retry-After`, ignore `X-RateLimit-Remaining`.

And the two to skip loudest: **"this is a cover of…"** (five rate-limited seconds
per track for an often-wrong answer about a rare case), and **anything that
replaces Spotify cover art with the Cover Art Archive** (**[M]** 66% coverage and
three redirect hops versus about 100% and a CDN).
