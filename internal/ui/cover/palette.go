package cover

import (
	"image"
	"image/color"
	"math"
)

// hueBuckets is how finely the hue circle is divided when looking for the colour
// an album "reads as". Twelve is coarse enough that a gradient still lands in one
// bucket, fine enough to keep red apart from orange.
const hueBuckets = 12

// sampleStride keeps the scan cheap: every 8th pixel in each direction is plenty
// for a dominant colour and turns a 640² image into a 80² sample.
const sampleStride = 8

// DominantColor picks the colour an album cover is "about": the most saturated
// hue family present, ignoring the near-black, near-white and washed-out pixels
// that dominate most artwork by area but carry no character.
//
// It returns ok=false when the image has no usable colour at all — a black and
// white sleeve, say — so the caller can keep its default accent.
func DominantColor(img image.Image) (color.RGBA, bool) {
	type bucket struct {
		weight           float64
		sumR, sumG, sumB float64
	}
	var buckets [hueBuckets]bucket

	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += sampleStride {
		for x := b.Min.X; x < b.Max.X; x += sampleStride {
			r16, g16, b16, a16 := img.At(x, y).RGBA()
			if a16 == 0 {
				continue
			}
			r, g, bl := float64(r16>>8)/255, float64(g16>>8)/255, float64(b16>>8)/255

			h, s, l := rgbToHSL(r, g, bl)
			// Anything this dark, this pale or this grey says nothing about the
			// artwork's character.
			if s < 0.25 || l < 0.15 || l > 0.9 {
				continue
			}

			// Weight by saturation so a small vivid area outvotes a large dull one.
			w := s * s
			i := int(h * hueBuckets)
			if i >= hueBuckets {
				i = hueBuckets - 1
			}
			buckets[i].weight += w
			buckets[i].sumR += r * w
			buckets[i].sumG += g * w
			buckets[i].sumB += bl * w
		}
	}

	best := -1
	for i := range buckets {
		if best < 0 || buckets[i].weight > buckets[best].weight {
			best = i
		}
	}
	if best < 0 || buckets[best].weight == 0 {
		return color.RGBA{}, false
	}

	w := buckets[best].weight
	return toRGBA(buckets[best].sumR/w, buckets[best].sumG/w, buckets[best].sumB/w), true
}

// Readable pushes a colour to a lightness that stands out against the terminal
// background without becoming neon, and gives it a saturation floor so a muddy
// cover still yields a usable accent.
func Readable(c color.RGBA, onDark bool) color.RGBA {
	h, s, l := rgbToHSL(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255)

	s = max(s, 0.45)
	if onDark {
		l = min(max(l, 0.55), 0.72)
	} else {
		l = min(max(l, 0.28), 0.42)
	}
	return toRGBA(hslToRGB(h, s, l))
}

func toRGBA(r, g, b float64) color.RGBA {
	clamp := func(v float64) uint8 { return uint8(math.Round(min(max(v, 0), 1) * 255)) }
	return color.RGBA{R: clamp(r), G: clamp(g), B: clamp(b), A: 255}
}

func rgbToHSL(r, g, b float64) (h, s, l float64) {
	hi, lo := max(r, g, b), min(r, g, b)
	l = (hi + lo) / 2
	if hi == lo {
		return 0, 0, l
	}

	d := hi - lo
	if l > 0.5 {
		s = d / (2 - hi - lo)
	} else {
		s = d / (hi + lo)
	}

	switch hi {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h / 6, s, l
}

func hslToRGB(h, s, l float64) (r, g, b float64) {
	if s == 0 {
		return l, l, l
	}

	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q

	return hueToRGB(p, q, h+1.0/3), hueToRGB(p, q, h), hueToRGB(p, q, h-1.0/3)
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6:
		return p + (q-p)*6*t
	case t < 1.0/2:
		return q
	case t < 2.0/3:
		return p + (q-p)*(2.0/3-t)*6
	default:
		return p
	}
}
