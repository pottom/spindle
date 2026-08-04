package daemon

import (
	"testing"
	"time"
)

func TestParseCrossfade(t *testing.T) {
	cases := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"", 0, false},
		{"off", 0, false},
		{"0", 0, false},
		{"6", 6 * time.Second, false},
		{"2.5", 2500 * time.Millisecond, false},
		{"12", MaxCrossfade, false},
		{"13", 0, true},
		{"-1", 0, true},
		{"soon", 0, true},
	}
	for _, c := range cases {
		got, err := ParseCrossfade(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseCrossfade(%q) = %v, want an error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseCrossfade(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseCrossfade(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
