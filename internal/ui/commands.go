package ui

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// callTimeout bounds every backend call so a hung request cannot wedge the loop.
const callTimeout = 10 * time.Second

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
