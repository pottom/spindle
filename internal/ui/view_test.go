package ui

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{-time.Second, "0:00"},
		{0, "0:00"},
		{900 * time.Millisecond, "0:01"},
		{time.Second, "0:01"},
		{9 * time.Second, "0:09"},
		{time.Minute, "1:00"},
		{5*time.Minute + 55*time.Second, "5:55"},
		{12*time.Minute + 3*time.Second, "12:03"},
		{time.Hour, "1:00:00"},
		{time.Hour + 2*time.Minute + 4*time.Second, "1:02:04"},
	}

	for _, c := range cases {
		if got := formatDuration(c.in); got != c.want {
			t.Errorf("formatDuration(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
