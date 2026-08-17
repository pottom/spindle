package player

import (
	"context"
	"errors"
	"net/http"

	"github.com/zmb3/spotify/v2"
)

// What the application spindle authenticates as is allowed to do.
//
// The Web API is not one API: what it answers depends on which application is
// asking. spindle ships with a registration that predates Spotify's 2024
// clampdown and may do everything; an application registered since is refused a
// family of endpoints outright, whoever owns it. See docs/SPOTIFY-API.md.
//
// So a feature that needs one of those is turned off rather than offered and
// left to fail — and whether it is allowed is asked rather than assumed, because
// the client id cannot say on its own: somebody may hold a grandfathered
// registration, and Spotify has moved this line before.
//
// One list, and one request each. To add a feature that needs Spotify's
// permission, add a line to Abilities: the probe decides it, the settings screen
// lists it, and whatever draws the feature asks Allowances.Has for it.

// Ability is one thing an application may be allowed to do beyond the plain
// reading and playing every application can.
type Ability string

const (
	// Collecting is the account's saved tracks: liking, unliking, and asking
	// whether one track is saved.
	Collecting Ability = "collecting"

	// Elsewhere is what somebody else owns — reading a playlist another account
	// made, however public it is.
	Elsewhere Ability = "elsewhere"

	// Suggesting is what Spotify thinks goes with what: related artists,
	// recommendations, and an artist's top tracks.
	Suggesting Ability = "suggesting"
)

// AbilityInfo is one ability, what it is called where somebody has to be told
// about it, and the request that settles whether this application has it.
type AbilityInfo struct {
	Ability Ability

	// Name is what the settings screen calls it, and Lost is what a listener
	// does without where it is refused — written for somebody deciding whether
	// to bring their own application.
	Name string
	Lost string

	// probe asks Spotify. It answers nil where the application may, a 403 where
	// it may not, and anything else where the question could not be put.
	probe func(ctx context.Context, s *Spotify) error
}

// Abilities is every one there is, in the order a screen should list them.
var Abilities = []AbilityInfo{
	{
		Ability: Collecting,
		Name:    "Liking tracks",
		Lost:    "the heart on every list, and the key that fills it",
		probe: func(ctx context.Context, s *Spotify) error {
			_, err := s.client.UserHasTracks(ctx, spotify.ID(probeTrack))
			return err
		},
	},
	{
		Ability: Elsewhere,
		Name:    "Other people's playlists",
		Lost:    "a shared playlist cannot be opened at all",
		probe: func(ctx context.Context, s *Spotify) error {
			_, err := s.PlaylistTracksPage(ctx, probePlaylist, 0)
			return err
		},
	},
	{
		Ability: Suggesting,
		Name:    "Related artists and suggestions",
		Lost:    "who else sounds like this, and what might come next",
		probe: func(ctx context.Context, s *Spotify) error {
			_, err := s.client.GetRelatedArtists(ctx, spotify.ID(probeArtist))
			return err
		},
	},
}

// The things asked about. The answers are about the application rather than
// about these, so what they are hardly matters — only that they have been in
// the catalogue a long time and are unlikely to leave it.
const (
	probeTrack    = "6dGnYIeXmHdcikdzNNDMm2" // The Beatles, "Here Comes The Sun"
	probeArtist   = "3WrFJ7ztbogyGnTHbHJFl2" // The Beatles
	probePlaylist = "37i9dQZF1DXcBWIGoYBM5M" // Spotify's own "Today's Top Hits"
)

// Allowances is what this application may do. A missing entry is a no: a
// program that has not found out yet offers nothing it might not honour.
type Allowances map[Ability]bool

// Has reports whether an ability is allowed.
func (a Allowances) Has(ability Ability) bool { return a[ability] }

// Everything is what a backend with nobody to ask hands back — the mock, and
// anything else that answers from a catalogue of its own.
func Everything() Allowances {
	out := make(Allowances, len(Abilities))
	for _, info := range Abilities {
		out[info.Ability] = true
	}
	return out
}

// Allower is a backend that can say what it is allowed to do.
type Allower interface {
	Allows(ctx context.Context) (Allowances, error)
}

// Allows asks Spotify, one question per ability.
//
// A refusal is an answer, not a failure: it means a narrow registration, which
// is a perfectly ordinary way to run. Anything else — no network, a rate limit,
// an expired token — is left as an error against that one ability, because a
// passing trouble must not be written down as a permanent lack.
func (s *Spotify) Allows(ctx context.Context) (Allowances, error) {
	out := make(Allowances, len(Abilities))

	var failed error
	for _, info := range Abilities {
		err := info.probe(ctx, s)
		switch {
		case err == nil:
			out[info.Ability] = true
		case refusedApplication(err):
			out[info.Ability] = false
		default:
			failed = classify("ask what this application may do", err)
		}
	}
	return out, failed
}

// refusedApplication reports that Spotify refused the application rather than
// failing to answer. 403 is the clampdown's refusal; 404 is what
// /recommendations answers a restricted application, which is the same no in
// different words.
func refusedApplication(err error) bool {
	var apiErr spotify.Error
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Status == http.StatusForbidden || apiErr.Status == http.StatusNotFound
}

// RelatedArtists is who else sounds like this one, as Spotify hears it.
//
// Refused to an application registered since the clampdown — it is what the
// Suggesting ability is probed with — so the caller has to be ready for nothing.
func (s *Spotify) RelatedArtists(ctx context.Context, artistID string) ([]Artist, error) {
	found, err := s.client.GetRelatedArtists(ctx, spotify.ID(artistID))
	if err != nil {
		return nil, permitted("fetch related artists", err)
	}

	out := make([]Artist, 0, len(found))
	for i := range found {
		out = append(out, artistFromFull(&found[i]))
	}
	return out, nil
}

// Suggester is a backend that can say what might go with a track.
//
// Optional for the same reason as the rest: Spotify answers /recommendations
// with 404 to an application it has clamped down on, which is the same no as a
// 403 in different words.
type Suggester interface {
	// Suggestions is what Spotify thinks goes with this track. What comes back
	// is other people's records rather than anything in the queue — the use of
	// it here is the artists it names. See the ui's fit.go.
	Suggestions(ctx context.Context, seedTrackID string, limit int) ([]Track, error)
}

// Suggestions asks Spotify what goes with a track.
func (s *Spotify) Suggestions(ctx context.Context, seedTrackID string, limit int) ([]Track, error) {
	seeds := spotify.Seeds{Tracks: []spotify.ID{spotify.ID(seedTrackID)}}

	found, err := s.client.GetRecommendations(ctx, seeds, nil, spotify.Limit(limit))
	if err != nil {
		return nil, permitted("fetch suggestions", err)
	}
	if found == nil {
		return nil, nil
	}

	out := make([]Track, 0, len(found.Tracks))
	for i := range found.Tracks {
		out = append(out, trackFromSimple(&found.Tracks[i]))
	}
	return out, nil
}

// Suggestions on the daemon is the Web API's answer: what goes with what is not
// something a playback device knows.
func (l *Local) Suggestions(ctx context.Context, seedTrackID string, limit int) ([]Track, error) {
	return l.web.Suggestions(ctx, seedTrackID, limit)
}

// RelatedArtists on the daemon is the Web API's answer: who sounds like whom is
// not something a playback device knows.
func (l *Local) RelatedArtists(ctx context.Context, artistID string) ([]Artist, error) {
	return l.web.RelatedArtists(ctx, artistID)
}

// Allows on the daemon is the Web API's answer: what a device may play has
// nothing to do with what an application may read.
func (l *Local) Allows(ctx context.Context) (Allowances, error) { return l.web.Allows(ctx) }

// Allows on the mock is everything. It answers from a catalogue of its own and
// has no registration to be refused.
func (m *Mock) Allows(context.Context) (Allowances, error) { return Everything(), nil }
