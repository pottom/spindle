package player

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// waveformTimeout is short on purpose. A frame that arrives late is worse than
// no frame: the trace would show what was heard a moment ago while the next one
// is already due.
const waveformTimeout = 120 * time.Millisecond

// Waveform is the sound the local device is putting out. It is polled rather
// than pushed: the daemon has no reason to send frames nobody is drawing, and
// the request is a loopback round trip.
func (l *Local) Waveform() []float32 {
	if l.idle() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), waveformTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.addr+"/player/waveform", nil)
	if err != nil {
		return nil
	}

	resp, err := l.http.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close() //nolint:errcheck // read-only request

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var out struct {
		Samples []float32 `json:"samples"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil
	}
	return out.Samples
}
