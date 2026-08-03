package daemon

import "fmt"

// Quality names the audio spindle asks Spotify for. Spotify offers Vorbis at
// three rates; lossless exists but needs a decryption plugin go-librespot
// deliberately ships only an interface for, so it is not on offer here.
type Quality string

const (
	QualityLow    Quality = "low"
	QualityMiddle Quality = "middle"
	QualityHigh   Quality = "high"
)

// DefaultQuality is the best on offer. spindle runs on a desktop, where neither
// bandwidth nor disk is the scarce thing.
const DefaultQuality = QualityHigh

// Bitrate is what the quality means in kbps.
func (q Quality) Bitrate() int {
	switch q {
	case QualityLow:
		return 96
	case QualityMiddle:
		return 160
	default:
		return 320
	}
}

// ParseQuality validates a configured name.
func ParseQuality(s string) (Quality, error) {
	switch Quality(s) {
	case QualityLow, QualityMiddle, QualityHigh:
		return Quality(s), nil
	case "":
		return DefaultQuality, nil
	default:
		return "", fmt.Errorf("unknown quality %q: want low, middle or high", s)
	}
}
