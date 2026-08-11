// Command spindle-sheet cuts a sheet of drawings into one file per drawing.
//
//	go run ./cmd/spindle-sheet sheet.png out/ 8 1
//
// A sheet is what comes back when eight characters are asked for in one picture,
// which is how they are drawn — asked for one at a time they do not match each
// other. The columns and rows are given because the picture cannot be trusted to
// say: see below.
//
// It writes out/cN-fM.png, cropped to each drawing's own ink and thresholded to
// black and white, which is what cmd/spindle-marks bakes from.
//
// # Where the cut goes, and the two ways of getting it wrong
//
// Both of these have happened, and each cost an afternoon, because a badly cut
// drawing looks like a drawing until you know what it should have been.
//
// Cutting on the widest runs of nothing splits a character that has a gap inside
// it. An equaliser is a stack of bars with air between them, and it came out as
// two halves that were then baked, shipped and looked at on screen for a day.
//
// Cutting on an even grid regardless takes the arm off whoever is reaching
// across it. One of the faces lost the hand it was throwing horns with, and one
// cut fell so far off that a piano and a note ended up in the same file and were
// drawn as one character half again as wide as anybody else.
//
// So the grid says roughly where and the ink says exactly where: within half a
// cell of each even division, the emptiest column wins. Where two figures reach
// across the gap even the emptiest column has something in it, so what is left
// of the neighbour is thrown away afterwards — see ownOnly.
//
// # The polarity
//
// A sheet may be black on white or white on black, and both have come back from
// the same brief on the same day. The corner decides, so nothing has to be said
// about it.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

const lit = 128

func main() {
	src, want, wantCols, wantRows := os.Args[1], os.Args[2], atoi(os.Args[3]), atoi(os.Args[4])
	f, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	img, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	// A sheet that ruled its own grid has a line through every cell's edge, and a
	// cell cut to its ink then takes in the line and comes out mostly empty. So
	// the lines are struck out of the picture itself rather than only out of the
	// search: anything that is ink nearly all the way across a row or column is
	// a rule and not a drawing.
	// Which way round the sheet is, asked of the corners rather than assumed:
	// some come back white on black and some black on white, and a cutter that
	// assumes one takes the background for the drawing and finds one cell.
	var bright int
	for _, c := range [][2]int{{0, 0}, {w - 1, 0}, {0, h - 1}, {w - 1, h - 1}} {
		r, g, bl, _ := img.At(b.Min.X+c[0], b.Min.Y+c[1]).RGBA()
		if (r+g+bl)/3>>8 >= lit {
			bright++
		}
	}
	dark := bright >= 3

	ruledRow := map[int]bool{}
	ruledCol := map[int]bool{}
	ink := func(x, y int) bool {
		if ruledRow[y] || ruledCol[x] {
			return false
		}
		r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
		if dark {
			return (r+g+bl)/3>>8 < lit
		}
		return (r+g+bl)/3>>8 >= lit
	}

	cols, rows := make([]int, w), make([]int, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if ink(x, y) {
				cols[x]++
				rows[y]++
			}
		}
	}
	// A sheet that drew its own grid lines has no gap without ink in it, so the
	// separators are taken out first: a row or column that is ink nearly all the
	// way across is a ruled line and not a drawing.
	for y := 0; y < h; y++ {
		if rows[y]*10 >= w*9 {
			ruledRow[y] = true
			for x := 0; x < w; x++ {
				cols[x]--
			}
			rows[y] = 0
		}
	}
	for x := 0; x < w; x++ {
		if cols[x]*10 >= h*9 {
			ruledCol[x] = true
			cols[x] = 0
		}
	}
	for i := range cols {
		if cols[i] < 0 {
			cols[i] = 0
		}
	}

	// A drawing with a gap down the middle of it — an equaliser is a stack of
	// bars with air between them — offers the cutter a wider run of nothing
	// than the space beside its neighbour does, and the cut lands inside the
	// character. So a split has to be near where an even division would put it:
	// the sheets are drawn as a regular row, and the grid is the truth about
	// where one drawing ends.
	xs := nearEven(cols, wantCols, w)
	ys := splits(rows, wantRows-1)
	fmt.Printf("%dx%d  %d columns, %d rows\n", w, h, len(xs)-1, len(ys)-1)
	if len(xs)-1 != wantCols || len(ys)-1 != wantRows {
		fmt.Println("!! the grid did not come out as asked")
	}

	os.MkdirAll(want, 0o755)
	for c := 0; c+1 < len(xs); c++ {
		for r := 0; r+1 < len(ys); r++ {
			x0, x1, y0, y1 := xs[c], xs[c+1], ys[r], ys[r+1]
			l, t, rr, bb := x1, y1, x0-1, y0-1
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					if ink(x, y) {
						l, t = min(l, x), min(t, y)
						rr, bb = max(rr, x), max(bb, y)
					}
				}
			}
			if rr < l {
				fmt.Printf("  col %d frame %d: EMPTY\n", c+1, r+1)
				continue
			}
			cell := image.NewGray(image.Rect(0, 0, rr-l+1, bb-t+1))
			for y := t; y <= bb; y++ {
				for x := l; x <= rr; x++ {
					if ink(x, y) {
						cell.SetGray(x-l, y-t, color.Gray{Y: 255})
					}
				}
			}
			out, on := ownOnly(cell)
			name := filepath.Join(want, fmt.Sprintf("c%d-f%d.png", c+1, r+1))
			g, _ := os.Create(name)
			png.Encode(g, out)
			g.Close()
			fmt.Printf("  col %d frame %d: %dx%d px, ink %.1f%%\n",
				c+1, r+1, rr-l+1, bb-t+1, float64(on)/float64((rr-l+1)*(bb-t+1))*100)
		}
	}
}

// ownOnly throws away whatever leant in from next door, and crops to what is
// left.
//
// A cut that avoids taking anybody's arm off has to fall in the emptiest column
// rather than an empty one, and where two figures reach across the gap that
// column still has something in it — so a fingertip or a shoe belonging to the
// neighbour comes with the cell. It is always small and it always touches the
// side it came in from, which is what tells it apart from a face's eyes or an
// equaliser's bars: those are small too, but they sit inside.
//
// One in twenty of the cell's ink is the line. Measured across the sheets here,
// a leant-in fingertip is under a fiftieth and the smallest thing anybody owns
// separately — an eye — is a twentieth and nowhere near an edge.
func ownOnly(pic *image.Gray) (*image.Gray, int) {
	b := pic.Bounds()
	w, h := b.Dx(), b.Dy()
	lit := func(x, y int) bool {
		return x >= 0 && y >= 0 && x < w && y < h && pic.GrayAt(x, y).Y > 128
	}

	// Every blob, by flood fill, with where it reaches.
	blob := make([]int, w*h)
	for i := range blob {
		blob[i] = -1
	}
	var size []int
	var atSide []bool
	var total int
	for y := range h {
		for x := range w {
			if !lit(x, y) || blob[y*w+x] >= 0 {
				continue
			}
			id := len(size)
			size = append(size, 0)
			atSide = append(atSide, false)
			stack := [][2]int{{x, y}}
			blob[y*w+x] = id
			for len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				size[id]++
				total++
				if p[0] == 0 || p[0] == w-1 {
					atSide[id] = true
				}
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						nx, ny := p[0]+dx, p[1]+dy
						if lit(nx, ny) && blob[ny*w+nx] < 0 {
							blob[ny*w+nx] = id
							stack = append(stack, [2]int{nx, ny})
						}
					}
				}
			}
		}
	}

	keep := make([]bool, len(size))
	for i := range size {
		keep[i] = !atSide[i] || size[i]*20 >= total
	}

	// What is left, cropped to itself.
	l, t, r, bt := w, h, -1, -1
	for y := range h {
		for x := range w {
			if id := blob[y*w+x]; id >= 0 && keep[id] {
				l, t = min(l, x), min(t, y)
				r, bt = max(r, x), max(bt, y)
			}
		}
	}
	if r < l {
		return pic, 0
	}
	out := image.NewGray(image.Rect(0, 0, r-l+1, bt-t+1))
	var on int
	for y := t; y <= bt; y++ {
		for x := l; x <= r; x++ {
			if id := blob[y*w+x]; id >= 0 && keep[id] {
				out.SetGray(x-l, y-t, color.Gray{Y: 255})
				on++
			}
		}
	}
	return out, on
}

// splits picks the n widest gaps that have ink on both sides, and returns the
// cut points with the outer edges added.
func splits(on []int, n int) []int {
	type run struct{ from, to int }
	var runs []run
	start := -1
	for i, v := range on {
		if v == 0 {
			if start < 0 {
				start = i
			}
			continue
		}
		if start > 0 { // start of 0 is the sheet's own margin
			runs = append(runs, run{start, i})
		}
		start = -1
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].to-runs[i].from > runs[j].to-runs[j].from })

	// Widest first, but never two cuts inside what must be one gap: a ruled line
	// with a break in it leaves two runs either side of it, and taking both
	// makes a cell out of the line itself — two pixels wide and all ink.
	apart := len(on) / (n + 1) / 2
	var at []int
	for _, r := range runs {
		if len(at) == n {
			break
		}
		mid := (r.from + r.to) / 2
		near := false
		for _, was := range at {
			if mid-was < apart && was-mid < apart {
				near = true
			}
		}
		if !near {
			at = append(at, mid)
		}
	}
	at = append(at, 0)
	at = append(at, len(on))
	sort.Ints(at)
	return at
}

// nearEven keeps the gap nearest each division of an even grid.
// nearEven puts each cut where an even division would put it, moved to the
// nearest column the drawings leave alone.
//
// Two ways of getting this wrong have both happened. Cutting on the widest runs
// of nothing splits a drawing that has a gap inside it — an equaliser is a stack
// of bars with air between them, and it came out as two half characters. Cutting
// on the even grid regardless takes the arm off whoever is reaching across it:
// one of the faces lost the hand it was throwing horns with, and nothing said
// so, because a cut limb looks like a drawing until you know what it should be.
//
// So: the grid says roughly where, and the ink says exactly where. Within half a
// cell of each division, the emptiest column wins — nothing at all if there is
// such a column, and the least ink if there is not, because a cut through one
// stroke is better than a cut through an arm and a leg.
func nearEven(ink []int, want, span int) []int {
	if want <= 1 {
		return []int{0, span}
	}
	out := []int{0}
	for i := 1; i < want; i++ {
		mark := span * i / want
		reach := span / want / 2

		best, least := mark, 1<<30
		for x := max(mark-reach, 1); x < min(mark+reach, span); x++ {
			n := ink[x]
			// Ties go to the column nearest the division, so an even sheet is
			// cut evenly.
			if n < least || (n == least && abs(x-mark) < abs(best-mark)) {
				best, least = x, n
			}
		}
		out = append(out, best)
	}
	return append(out, span)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }
