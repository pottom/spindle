package player

import (
	"context"
	"errors"
	"time"
)

// ErrNoActiveDevice reports that nothing is currently playing anywhere. It is a
// normal condition, not a failure: the UI answers it with a device list.
var ErrNoActiveDevice = errors.New("no active playback device")

// Player is the only playback interface the UI is allowed to depend on.
type Player interface {
	State(ctx context.Context) (*State, error)
	Play(ctx context.Context) error
	Pause(ctx context.Context) error
	Next(ctx context.Context) error
	Previous(ctx context.Context) error
	Seek(ctx context.Context, pos time.Duration) error
	SetVolume(ctx context.Context, pct int) error
	SetShuffle(ctx context.Context, on bool) error
	SetRepeat(ctx context.Context, mode string) error
	Devices(ctx context.Context) ([]Device, error)
	TransferTo(ctx context.Context, deviceID string) error
}
