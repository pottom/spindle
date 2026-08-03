package ui

import (
	"context"
	"errors"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/msg"
)

const (
	// callTimeout bounds every backend call so a hung request cannot wedge the loop.
	callTimeout = 10 * time.Second

	// coverTimeout is more generous: it covers a download as well as the resize.
	coverTimeout = 20 * time.Second

	// coverSettleDelay is how long the browse cursor has to rest before its
	// artwork is loaded. Without it, holding a cursor key would queue an upload
	// per row.
	coverSettleDelay = 250 * time.Millisecond

	// trackChangeDelay is how long to wait before confirming a skip. Spotify
	// still reports the previous track for a moment afterwards.
	trackChangeDelay = 400 * time.Millisecond

	// volumeDebounce collapses a held volume key into one request.
	volumeDebounce = 800 * time.Millisecond
)

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return msg.Tick{Time: t}
	})
}

func fetchStateCmd(p player.Player) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		st, err := p.State(ctx)
		if errors.Is(err, player.ErrNoActiveDevice) {
			return msg.NoActiveDevice{}
		}
		var limited *player.RateLimitedError
		if errors.As(err, &limited) {
			return msg.RateLimited{RetryAfter: limited.RetryAfter}
		}
		if err != nil {
			return msg.Error{Err: fmt.Errorf("fetch player state: %w", err)}
		}
		return msg.StateFetched{State: st}
	}
}

func fetchDevicesCmd(p player.Player) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		devices, err := p.Devices(ctx)
		if err != nil {
			return msg.Error{Err: fmt.Errorf("fetch devices: %w", err)}
		}
		return msg.DevicesFetched{Devices: devices}
	}
}

// coverCmd runs the artwork pipeline off the update loop: cache lookup, download,
// decode, resize and render. A failure is reported as a missing cover, not as an
// error banner.
func coverCmd(loader *cover.Loader, url string, wCells, hCells int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), coverTimeout)
		defer cancel()

		art, err := loader.Load(ctx, url, wCells, hCells)
		if err != nil {
			return msg.CoverFailed{URL: url, Width: wCells, Height: hCells}
		}
		return msg.CoverReady{URL: url, Width: wCells, Height: hCells, Art: art}
	}
}

func fetchPlaylistsCmd(p player.Player) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		items, err := p.Playlists(ctx)
		if err != nil {
			return msg.Error{Err: fmt.Errorf("fetch playlists: %w", err)}
		}
		return msg.PlaylistsFetched{Playlists: items}
	}
}

func fetchPlaylistTracksCmd(p player.Player, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		tracks, err := p.PlaylistTracks(ctx, id)
		if err != nil {
			return msg.Error{Err: fmt.Errorf("fetch playlist tracks: %w", err)}
		}
		return msg.PlaylistTracksFetched{PlaylistID: id, Tracks: tracks}
	}
}

func searchCmd(p player.Player, query string, seq int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		tracks, err := p.Search(ctx, query)
		if err != nil {
			return msg.Error{Err: fmt.Errorf("search: %w", err)}
		}
		return msg.SearchResults{Seq: seq, Tracks: tracks, Query: query, Matched: true}
	}
}

// coverSettleCmd waits out the debounce before an artwork preview is loaded.
func coverSettleCmd(seq int) tea.Cmd {
	return tea.Tick(coverSettleDelay, func(time.Time) tea.Msg {
		return msg.CoverSettled{Seq: seq}
	})
}

// controlCmd runs a playback control call off the update loop. The result is
// classified here so the UI can tell "Spotify is throttling us" and "this account
// cannot do that" apart from an ordinary failure.
func controlCmd(action string, call func(context.Context) error) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		err := call(ctx)
		if err == nil {
			return msg.ControlDone{}
		}

		var limited *player.RateLimitedError
		if errors.As(err, &limited) {
			return msg.RateLimited{RetryAfter: limited.RetryAfter}
		}
		if errors.Is(err, player.ErrNoActiveDevice) {
			return msg.NoActiveDevice{}
		}
		return msg.Error{Err: fmt.Errorf("%s: %w", action, err)}
	}
}

// refetchCmd asks for a fresh state after a delay. Spotify reports the old track
// for a moment after a skip, so an immediate poll would confirm the wrong thing.
func refetchCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return msg.Refetch{} })
}

// volumeSettleCmd waits out the volume debounce.
func volumeSettleCmd(seq int) tea.Cmd {
	return tea.Tick(volumeDebounce, func(time.Time) tea.Msg {
		return msg.VolumeSettled{Seq: seq}
	})
}
