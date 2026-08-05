package ui

import (
	"strings"
	"testing"
)

// fallModel is a player screen with a waterfall on it and a history to draw.
func fallModel(slices ...[]float32) Model {
	m := scopeModel(100, 44)
	m.scope.modes[tabPlayer] = scopeFall
	for _, s := range slices {
		m.scope.rememberFall(s)
	}
	return m
}

// spike is a spectrum with one band sounding and the rest silent.
func spike(bands, at int) []float32 {
	out := make([]float32, bands)
	out[at] = 1
	return out
}

// Low frequencies along the bottom, high along the top: the same way round as
// the spectrum's own axis, and the way every waterfall has ever been drawn.
func TestWaterfallPutsTheBassAtTheBottom(t *testing.T) {
	w := fallModel().layout().interior - leftMargin - rightMargin

	rowsWith := func(band int) []int {
		slices := make([][]float32, 0, 200)
		for range 200 {
			slices = append(slices, spike(28, band))
		}
		m := fallModel(slices...)

		var lit []int
		for r, line := range m.fallLines(w) {
			if strings.TrimSpace(ansiOff(line)) != "" {
				lit = append(lit, r)
			}
		}
		return lit
	}

	low, high := rowsWith(0), rowsWith(27)
	t.Logf("the lowest band lights rows %v, the highest %v", low, high)

	if len(low) == 0 || len(high) == 0 {
		t.Fatalf("a band that is sounding drew nothing: low %v, high %v", low, high)
	}
	if low[0] <= high[len(high)-1] {
		t.Errorf("the lowest band drew at rows %v and the highest at %v, want the bass lower down", low, high)
	}
}

// A band below the floor draws nothing at all: the dot is on or off, and half
// the picture's character is where that line is drawn.
func TestWaterfallLeavesTheQuietDark(t *testing.T) {
	w := fallModel().layout().interior - leftMargin - rightMargin

	quiet := make([]float32, 28)
	for i := range quiet {
		quiet[i] = fallFloor - 0.05
	}

	slices := make([][]float32, 0, 200)
	for range 200 {
		slices = append(slices, quiet)
	}

	for r, line := range fallModel(slices...).fallLines(w) {
		if got := strings.TrimSpace(ansiOff(line)); got != "" {
			t.Errorf("row %d drew %q for a spectrum below the floor, want nothing", r, got)
		}
	}
}

// Time runs to the right: what is sounding now is at the edge the eye goes to,
// and what was sounding a moment ago has moved left.
func TestWaterfallDrawsTheNewestOnTheRight(t *testing.T) {
	w := fallModel().layout().interior - leftMargin - rightMargin

	quiet := make([]float32, 28)
	slices := make([][]float32, 0, 400)
	for range 400 {
		slices = append(slices, quiet)
	}
	slices = append(slices, spike(28, 0), spike(28, 0))

	var found bool
	for _, line := range fallModel(slices...).fallLines(w) {
		plain := []rune(ansiOff(line))
		if len(plain) != w {
			t.Fatalf("a row came out %d cells wide, want %d", len(plain), w)
		}
		if strings.TrimSpace(string(plain)) == "" {
			continue
		}
		found = true
		if plain[len(plain)-1] == ' ' {
			t.Errorf("the newest slice drew at %q, want it against the right edge", strings.TrimRight(string(plain), " "))
		}
	}
	if !found {
		t.Error("the newest slice drew nothing")
	}
}

// The history is bounded: a track is longer than any screen, and keeping every
// slice of it would be a leak with a picture on top.
func TestWaterfallForgetsTheOldest(t *testing.T) {
	var s scopeState
	for range fallHistory * 2 {
		s.rememberFall(spike(28, 3))
	}
	if len(s.fall) > fallHistory {
		t.Errorf("the waterfall kept %d slices, want at most %d", len(s.fall), fallHistory)
	}
}
