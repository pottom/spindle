package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/pottom/spindle/internal/xdg"
)

// tap is one press: where the playhead was, and what it was aimed at. A word of
// -1 is a line's end rather than one of its words.
type tap struct {
	at    time.Duration
	line  int
	word  int
	words string
}

// done is how many lines a pass has collected, which is what the goal is
// counted against.
//
// Each pass counts what it is actually asking for: a start is a line for the
// starts pass, a line is done when its end is marked, and a words pass has
// covered a line when anything at all was tapped on it.
func done(kind string, taps []tap) int {
	switch kind {
	case passEnds:
		n := 0
		for _, t := range taps {
			if t.word == -1 {
				n++
			}
		}
		return n
	case passWords:
		var lines []int
		for _, t := range taps {
			if !slices.Contains(lines, t.line) {
				lines = append(lines, t.line)
			}
		}
		return len(lines)
	default:
		return len(taps)
	}
}

// save writes a pass down, and returns where it went.
//
// One file per track and pass, overwritten: a pass run again is a pass done
// better, and two versions of the same attempt would only raise the question of
// which one to believe.
func save(w work, kind string, tempo float64, ly lyrics, taps []tap) (string, error) {
	dir, err := xdg.ConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "spike", "taps")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("%s-%s.tsv", w.slug(), kind))

	var b strings.Builder
	fmt.Fprintf(&b, "# %s — %s\n# spotify:track:%s\t%.1f bpm\t%s\n", w.artist, w.title, w.id, tempo, kind)
	fmt.Fprintf(&b, "tap_ms\tline\tword\tline_stamp_ms\twords\n")
	for _, t := range taps {
		stamp := int64(-1)
		if t.line >= 0 && t.line < len(ly.Lines) {
			stamp = ly.Lines[t.line].At
		}
		fmt.Fprintf(&b, "%d\t%d\t%d\t%d\t%s\n", t.at.Milliseconds(), t.line, t.word, stamp, t.words)
	}
	return path, os.WriteFile(path, []byte(b.String()), 0o644)
}

// lag is what a starts pass measured: every tap against the stamp it was aimed
// at. The median is what comes off the later passes, and the quartiles say how
// much of the answer is the hand rather than the singer — so how fine a
// measurement is worth claiming.
func lag(ly lyrics, taps []tap) (median, low, high int64, n int) {
	var lags []int64
	for _, t := range taps {
		best := int64(1 << 62)
		for _, l := range ly.Lines {
			d := t.at.Milliseconds() - l.At
			if d > -2000 && d < 2000 && abs64(d) < abs64(best) {
				best = d
			}
		}
		if best != 1<<62 {
			lags = append(lags, best)
		}
	}
	if len(lags) < 4 {
		return 0, 0, 0, len(lags)
	}
	slices.Sort(lags)
	return lags[len(lags)/2], lags[len(lags)/4], lags[3*len(lags)/4], len(lags)
}

// share is what an ends pass measured: how much of each line's window was sung,
// from the line's own stamp to where the singing stopped.
//
// Against the stamp rather than against the tap that opened the line: the stamp
// carries no lag, and this is the number the model has to predict.
func share(ly lyrics, taps []tap) (median, low, high float64, n int) {
	var shares []float64
	for _, t := range taps {
		if t.word != -1 || t.line < 0 || t.line+1 >= len(ly.Lines) {
			continue
		}
		start := ly.Lines[t.line].At
		window := ly.Lines[t.line+1].At - start
		sung := t.at.Milliseconds() - start
		if window <= 0 || sung <= 0 {
			continue
		}
		shares = append(shares, float64(sung)/float64(window))
	}
	if len(shares) < 4 {
		return 0, 0, 0, len(shares)
	}
	slices.Sort(shares)
	return shares[len(shares)/2], shares[len(shares)/4], shares[3*len(shares)/4], len(shares)
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func fields(s string) []string { return strings.Fields(s) }
