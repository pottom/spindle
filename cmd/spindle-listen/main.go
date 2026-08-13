// Command spindle-listen writes down what the music is made of, frame by frame.
//
//	go run ./cmd/spindle-listen
//
// It asks the daemon for the spectrum thirty times a second and records it
// against the playhead, until the track ends or it is stopped. Nothing to press
// and nothing to hear: put a record on, start this, walk away.
//
// # What it is for
//
// The words on the lyric screen are placed by a model — a line is sung for 85%
// of its window, its syllables share that out — and the model is as good as the
// singer is predictable, which is about a fifth of a second at the median and
// half a second at the tail. That is the ceiling on guessing. Under it there is
// only measuring: the voice itself says when it starts and stops, and it says so
// in the spectrum, thirty times a second, for nothing.
//
// It was written to answer whether the singing can be told from the band in
// what the daemon already measures. It cannot — see FINDINGS.md, where two
// recordings taken with this settle it, along with what a patched analyser said
// when it was asked the same question with better instruments. The tool stays
// because that answer is worth being able to check again, and because a
// recording of the spectrum against the playhead is the only way to argue about
// anything drawn from it.
//
// # Why it cannot disturb what it measures
//
// The spectrum endpoint answers from the analyser rather than from the playback
// loop — see ApiLive in the fork — so this asks nothing of the goroutine that is
// making the sound. The playhead does come from the loop, so it is read twice a
// second and carried forward on the local clock between times, the same trick
// spindle-tap uses and for the same reason.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/pottom/spindle/internal/daemon"
	"github.com/pottom/spindle/internal/xdg"
)

const (
	// frameEvery is how often the spectrum is taken. The analyser's own window
	// is 46 ms, so thirty a second reads everything there is to read.
	frameEvery = 33 * time.Millisecond

	// anchorEvery is how often the playhead is asked for. Twice a second: it is
	// the one request here that goes to the playback loop.
	anchorEvery = 500 * time.Millisecond
)

var base = fmt.Sprintf("http://%s:%d", daemon.DefaultHost, daemon.DefaultPort)

var client = &http.Client{Timeout: 2 * time.Second}

type status struct {
	Paused  bool `json:"paused"`
	Stopped bool `json:"stopped"`
	Track   struct {
		URI      string   `json:"uri"`
		Name     string   `json:"name"`
		Artists  []string `json:"artist_names"`
		Position int64    `json:"position"`
		Duration int64    `json:"duration"`
	} `json:"track"`
}

// The names are the daemon's own: it answers loud_db, beat_ms and
// beat_since_ms, which is worth writing down because getting one wrong costs a
// recording that looks fine and carries nothing.
type spectrum struct {
	Bands []float32 `json:"bands"`
	Loud  float64   `json:"loud_db"`
	Beat  float64   `json:"beat_ms"`
	Since float64   `json:"beat_since_ms"`
}

func main() {
	st, err := get[status](base + "/status")
	if err != nil {
		die("no daemon at %s: %v", base, err)
	}
	if st.Track.URI == "" || st.Stopped {
		die("nothing is playing — put a record on first")
	}

	id := st.Track.URI[strings.LastIndex(st.Track.URI, ":")+1:]
	dir, err := xdg.ConfigDir()
	if err != nil {
		die("no config directory: %v", err)
	}
	dir = filepath.Join(dir, "spike", "ears")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		die("make %s: %v", dir, err)
	}
	path := filepath.Join(dir, id+".tsv")

	f, err := os.Create(path)
	if err != nil {
		die("write %s: %v", path, err)
	}
	defer f.Close() //nolint:errcheck // flushed below
	out := bufio.NewWriter(f)

	fmt.Fprintf(out, "# %s — %s\n# %s\t%d ms\n", strings.Join(st.Track.Artists, ", "), st.Track.Name,
		st.Track.URI, st.Track.Duration)
	fmt.Fprintf(out, "at_ms\tloud\tbeat\tsince\tbands\n")

	fmt.Printf("%s — %s\n", strings.Join(st.Track.Artists, ", "), st.Track.Name)
	fmt.Printf("writing to %s\nctrl+c stops it; it stops itself when the record does.\n\n", path)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	var (
		at        time.Duration
		when      time.Time
		playing   bool
		frames    int
		lastPrint time.Time
	)
	anchor := time.NewTicker(anchorEvery)
	defer anchor.Stop()
	frame := time.NewTicker(frameEvery)
	defer frame.Stop()

	for {
		select {
		case <-stop:
			finish(out, f, frames, path)
			return

		case <-anchor.C:
			now, err := get[status](base + "/status")
			if err != nil {
				continue
			}
			if now.Stopped || now.Track.URI != st.Track.URI {
				fmt.Println("\nthe record ended.")
				finish(out, f, frames, path)
				return
			}
			at, when, playing = time.Duration(now.Track.Position)*time.Millisecond, time.Now(), !now.Paused

		case <-frame.C:
			if !playing || when.IsZero() {
				continue
			}
			s, err := get[spectrum](base + "/player/spectrum")
			if err != nil || len(s.Bands) == 0 {
				continue
			}
			pos := at + time.Since(when)

			var b strings.Builder
			for i, v := range s.Bands {
				if i > 0 {
					b.WriteByte(' ')
				}
				fmt.Fprintf(&b, "%.4f", v)
			}
			fmt.Fprintf(out, "%d\t%.1f\t%.0f\t%.0f\t%s\n", pos.Milliseconds(), s.Loud, s.Beat, s.Since, b.String())
			frames++

			if time.Since(lastPrint) > time.Second {
				lastPrint = time.Now()
				fmt.Printf("\r  %5.1fs   %d frames", pos.Seconds(), frames)
			}
		}
	}
}

func finish(out *bufio.Writer, f *os.File, frames int, path string) {
	_ = out.Flush()
	_ = f.Close()
	fmt.Printf("\n%d frames -> %s\n", frames, path)
}

func get[T any](url string) (T, error) {
	var out T
	resp, err := client.Get(url) //nolint:noctx // loopback, and a spike
	if err != nil {
		return out, err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("answered %s", resp.Status)
	}
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "spindle-listen: "+format+"\n", args...)
	os.Exit(1)
}
