package daemon

import "testing"

func TestParseQuality(t *testing.T) {
	cases := []struct {
		in      string
		want    Quality
		bitrate int
		wantErr bool
	}{
		{in: "low", want: QualityLow, bitrate: 96},
		{in: "middle", want: QualityMiddle, bitrate: 160},
		{in: "high", want: QualityHigh, bitrate: 320},
		// An unset quality is not a misconfiguration, it is a fresh install.
		{in: "", want: DefaultQuality, bitrate: 320},
		{in: "lossless", wantErr: true},
		{in: "HIGH", wantErr: true},
	}

	for _, c := range cases {
		got, err := ParseQuality(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseQuality(%q) = %q, want an error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseQuality(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseQuality(%q) = %q, want %q", c.in, got, c.want)
		}
		if got.Bitrate() != c.bitrate {
			t.Errorf("%q.Bitrate() = %d, want %d", got, got.Bitrate(), c.bitrate)
		}
	}
}
