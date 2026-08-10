package player

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Bands is how the local device's output is spread across the frequency range,
// and where the beats of it are. Polled like the waveform: the daemon has no
// reason to send frames nobody is drawing.
//
// The two come back together because they belong to the same moment. Asked for
// separately they would arrive a frame apart, which is a fortieth of a beat at
// 120 — the whole precision the beat was measured to.
func (l *Local) Bands() ([]float32, Beat) {
	if l.idle() {
		return nil, Beat{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), waveformTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.addr+"/player/spectrum", nil)
	if err != nil {
		return nil, Beat{}
	}

	resp, err := l.http.Do(req)
	if err != nil {
		return nil, Beat{}
	}
	defer resp.Body.Close() //nolint:errcheck // read-only request

	if resp.StatusCode != http.StatusOK {
		return nil, Beat{}
	}

	var out struct {
		Bands []float32 `json:"bands"`
		Beat  float64   `json:"beat_ms"`
		Since float64   `json:"beat_since_ms"`
		Loud  float64   `json:"loud_db"`
		Notes []float32 `json:"notes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, Beat{}
	}

	return out.Bands, Beat{
		Period: time.Duration(out.Beat * float64(time.Millisecond)),
		Since:  time.Duration(out.Since * float64(time.Millisecond)),
		Loud:   out.Loud,
		Notes:  out.Notes,
	}
}
