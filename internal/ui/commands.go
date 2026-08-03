package ui

import (
	"context"
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
		if err != nil {
			return msg.Error{Err: fmt.Errorf("fetch player state: %w", err)}
		}
		return msg.StateFetched{State: st}
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

// controlCmd runs a playback control call off the update loop. A successful call
// produces no message: the next poll is what confirms it.
func controlCmd(action string, call func(context.Context) error) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		if err := call(ctx); err != nil {
			return msg.Error{Err: fmt.Errorf("%s: %w", action, err)}
		}
		return nil
	}
}
