package player

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/zmb3/spotify/v2"
)

// Saving a track, and why it is not part of Player.
//
// Spotify refuses the whole family of library writes to an application
// registered after its 2024 clampdown — and refuses the question "is this
// saved" with it, so a program on such a registration cannot even draw the
// state of a heart, let alone change it. Which is why this is an interface of
// its own that a backend may not implement, and why a refusal at run time has a
// name of its own: the key can then stop being offered rather than failing every
// time it is pressed. See docs/SPOTIFY-API.md.

// ErrNotPermitted reports that Spotify refused the application rather than the
// account. Nothing the listener can do about it, and nothing that will be
// different next time: the registration is what is not allowed.
var ErrNotPermitted = errors.New("this Spotify application may not do that")

// Collector is a backend that can change what the account has saved.
//
// Optional. A backend that does not implement it simply has no like key, in the
// same way that one with no device of its own has nothing to take over.
type Collector interface {
	// Save puts a track in the account's saved tracks, and Unsave takes it out.
	// Both are idempotent: saving what is already saved is not an error, which
	// is what makes a toggle safe to press twice.
	Save(ctx context.Context, trackID string) error
	Unsave(ctx context.Context, trackID string) error
}

func (s *Spotify) Save(ctx context.Context, trackID string) error {
	return permitted("save track", s.client.AddTracksToLibrary(ctx, spotify.ID(trackID)))
}

func (s *Spotify) Unsave(ctx context.Context, trackID string) error {
	return permitted("unsave track", s.client.RemoveTracksFromLibrary(ctx, spotify.ID(trackID)))
}

// permitted turns Spotify's refusal of the application into ErrNotPermitted, and
// leaves everything else to the usual classification.
//
// A 403 here cannot be about the subscription — saving a track is not playback,
// and Spotify refuses it to free accounts no more than to paying ones — so on
// these calls it is about the registration, which is a thing to say once rather
// than every time a key is pressed.
func permitted(action string, err error) error {
	if err == nil {
		return nil
	}

	var apiErr spotify.Error
	if errors.As(err, &apiErr) && apiErr.Status == http.StatusForbidden {
		return fmt.Errorf("%s: %w", action, ErrNotPermitted)
	}
	return classify(action, err)
}
