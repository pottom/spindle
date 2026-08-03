package player

import (
	"testing"

	"github.com/zmb3/spotify/v2"
)

func images(widths ...int) []spotify.Image {
	out := make([]spotify.Image, 0, len(widths))
	for _, w := range widths {
		out = append(out, spotify.Image{
			Width:  spotify.Numeric(w),
			Height: spotify.Numeric(w),
			URL:    urlFor(w),
		})
	}
	return out
}

func urlFor(w int) string {
	return "https://img/" + itoa(w)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func TestBestImage(t *testing.T) {
	cases := []struct {
		name string
		in   []spotify.Image
		want string
	}{
		{"largest within the cap", images(64, 300, 640), urlFor(640)},
		{"order does not matter", images(640, 300, 64), urlFor(640)},
		{"skips the oversized", images(1000, 640, 300), urlFor(640)},
		{"exactly at the cap counts", images(640), urlFor(640)},
		// Rather than nothing at all, take the least oversized on offer.
		{"all oversized falls back to the smallest", images(3000, 1000), urlFor(1000)},
		{"no images", nil, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := bestImage(c.in); got != c.want {
				t.Errorf("bestImage = %q, want %q", got, c.want)
			}
		})
	}
}

func TestRepeatFromSpotify(t *testing.T) {
	cases := map[string]string{
		"off":       RepeatOff,
		"context":   RepeatContext,
		"track":     RepeatTrack,
		"":          RepeatOff,
		"something": RepeatOff,
	}
	for in, want := range cases {
		if got := repeatFromSpotify(in); got != want {
			t.Errorf("repeatFromSpotify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOwnerNameFallsBackToID(t *testing.T) {
	if got := ownerName(spotify.User{DisplayName: "Anna", ID: "u1"}); got != "Anna" {
		t.Errorf("ownerName = %q, want Anna", got)
	}
	if got := ownerName(spotify.User{ID: "u1"}); got != "u1" {
		t.Errorf("ownerName = %q, want u1", got)
	}
}
