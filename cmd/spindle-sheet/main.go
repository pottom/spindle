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

	// Whose is what, decided once over the whole sheet rather than cell by cell.
	//
	// A drawing near a cut is the question this tool exists to get right, and it
	// cannot be answered inside one cell: a hand beside the line looks the same
	// whether it is this character reaching out or the next one reaching in.
	// Followed across the sheet it is never in doubt — the hand is joined to a
	// body, and the body is in one cell or the other. So every run of connected
	// ink is found once, and it goes to whichever cell holds most of it.
	//
	// Parts that are drawn loose — eyes, a mouth, the bars of an equaliser — go
	// by the same rule and land where they were drawn, because they are nowhere
	// near a cut.
	owner := blobs(ink, w, h)

	os.MkdirAll(want, 0o755)
	os.MkdirAll(want, 0o755)
	for c := 0; c+1 < len(xs); c++ {
		for r := 0; r+1 < len(ys); r++ {
			x0, x1, y0, y1 := xs[c], xs[c+1], ys[r], ys[r+1]

			// The whole cut region, not the ink in it. What is thrown out next
			// is whatever runs into the cut line, and in a picture already
			// cropped to its ink everything runs into an edge — the leftmost
			// stroke of every drawing touches the left of its own box. Cropping
			// is ownOnly's last act instead.
			mine := func(x, y int) bool {
				if !ink(x, y) {
					return false
				}
				id := owner.at[y*w+x]
				return owner.cx[id] >= x0 && owner.cx[id] < x1 &&
					owner.cy[id] >= y0 && owner.cy[id] < y1
			}

			l, t, rr, bb := w, h, -1, -1
			for y := range h {
				for x := range w {
					if mine(x, y) {
						l, t = min(l, x), min(t, y)
						rr, bb = max(rr, x), max(bb, y)
					}
				}
			}
			if rr < l {
				fmt.Printf("  col %d frame %d: EMPTY\n", c+1, r+1)
				continue
			}
			out := image.NewGray(image.Rect(0, 0, rr-l+1, bb-t+1))
			var on int
			for y := t; y <= bb; y++ {
				for x := l; x <= rr; x++ {
					if mine(x, y) {
						out.SetGray(x-l, y-t, color.Gray{Y: 255})
						on++
					}
				}
			}
			name := filepath.Join(want, fmt.Sprintf("c%d-f%d.png", c+1, r+1))
			g, _ := os.Create(name)
			png.Encode(g, out)
			g.Close()
			b := out.Bounds()
			fmt.Printf("  col %d frame %d: %dx%d px, ink %.1f%%\n",
				c+1, r+1, b.Dx(), b.Dy(), float64(on)/float64(max(b.Dx()*b.Dy(), 1))*100)
		}
	}
}

// held is every run of connected ink on the sheet, and where each one's weight
// lies: cx and cy are the middle of a blob by its own pixels, so a hand joined
// to a body is placed by the body rather than by itself.
type held struct {
	at     []int
	cx, cy []int
}

func blobs(ink func(int, int) bool, w, h int) held {
	out := held{at: make([]int, w*h)}
	for i := range out.at {
		out.at[i] = -1
	}
	var sumX, sumY, n []int
	for y := range h {
		for x := range w {
			if !ink(x, y) || out.at[y*w+x] >= 0 {
				continue
			}
			id := len(n)
			sumX, sumY, n = append(sumX, 0), append(sumY, 0), append(n, 0)
			stack := [][2]int{{x, y}}
			out.at[y*w+x] = id
			for len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				sumX[id], sumY[id], n[id] = sumX[id]+p[0], sumY[id]+p[1], n[id]+1
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						nx, ny := p[0]+dx, p[1]+dy
						if nx < 0 || ny < 0 || nx >= w || ny >= h {
							continue
						}
						if ink(nx, ny) && out.at[ny*w+nx] < 0 {
							out.at[ny*w+nx] = id
							stack = append(stack, [2]int{nx, ny})
						}
					}
				}
			}
		}
	}
	out.cx, out.cy = make([]int, len(n)), make([]int, len(n))
	for i := range n {
		out.cx[i], out.cy[i] = sumX[i]/n[i], sumY[i]/n[i]
	}
	return out
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
