package ui

import "sync"

// The three buffers a picture is drawn into, borrowed rather than made afresh.
//
// Every frame of the big screen needs one byte of dots, one of brightness and
// one of colour for each cell of the terminal. At the size a wall-mounted screen
// actually is — 352 by 84 in the recording this was measured on — that is 89 kB
// a frame, sixty times a second, and it was being asked of the heap every time.
// Measured with a profile: a frame allocated 280 kB in all, and a fifth of that
// was these three.
//
// It is not the arithmetic that costs; it is the collector. The frames that go
// missing on this screen mostly do not go missing in the drawing — measured, on
// 47 late frames the drawing took 2.6 ms at the median and the time went
// somewhere else entirely on 31 of them, which is what a collector running
// through five megabytes a second looks like from inside the loop that is being
// interrupted.
//
// Borrowed and given back, so the same three live for the life of the program.
// A pool rather than a field on the model because View has a value receiver and
// may not write back — see the rule about it in CONVENTIONS.md — and because two
// pictures are never drawn at once anyway, so the pool holds one.
var framePool sync.Pool

// frameBuf is a picture's three layers, held together so they are borrowed and
// returned as one thing.
type frameBuf struct {
	grid  []uint8
	paint []int8
	hue   []int8
}

// takeFrame borrows the three, sized for w by rows cells and cleared the way a
// fresh set would arrive: no dots, no colour, and the brightness at the "nothing
// has been written here" mark the drawers test against.
func takeFrame(w, rows int) *frameBuf {
	n := max(w*rows, 0)
	b, _ := framePool.Get().(*frameBuf)
	if b == nil {
		b = &frameBuf{}
	}
	if cap(b.grid) < n {
		b.grid, b.paint, b.hue = make([]uint8, n), make([]int8, n), make([]int8, n)
	}
	b.grid, b.paint, b.hue = b.grid[:n], b.paint[:n], b.hue[:n]

	clear(b.grid)
	clear(b.hue)
	for i := range b.paint {
		b.paint[i] = -1
	}
	return b
}

// giveFrame hands them back. Nothing that was drawn keeps a reference: the rows
// leave as strings.
func giveFrame(b *frameBuf) { framePool.Put(b) }
