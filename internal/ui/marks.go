package ui

// A set of marks, drawn rather than set from a face. See cmd/spindle-marks.
type markSet struct {
	from, licence string
	sizes         []markSize
}

// markSize is the whole row at one dot height.
type markSize struct {
	tall  int
	marks []markDots
}

// markDots is one drawing: its own size in dots, and a bit per dot.
type markDots struct {
	name       string
	wide, tall int
	bits       string
}
