package cover

import (
	"bytes"
	"image"
	"image/color"
	"io"
	"os"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// The placeholder grid is the whole reason the kitty backend survives Bubble
// Tea's line diff: it has to measure as ordinary text of exactly the right
// width, and every cell has to carry its own row and column.
func TestPlaceholderGrid(t *testing.T) {
	const cols, rows = 20, 10

	grid := placeholderGrid(testImageID, cols, rows)
	lines := strings.Split(grid, "\n")
	if len(lines) != rows {
		t.Fatalf("got %d lines, want %d", len(lines), rows)
	}

	index := make(map[rune]int, len(rowColumnDiacritics))
	for i, r := range rowColumnDiacritics {
		index[r] = i
	}

	for row, line := range lines {
		if w := lipgloss.Width(line); w != cols {
			t.Errorf("row %d measures %d cells, want %d", row, w, cols)
		}

		cells := splitPlaceholders(line)
		if len(cells) != cols {
			t.Fatalf("row %d has %d placeholder cells, want %d", row, len(cells), cols)
		}
		for col, marks := range cells {
			if len(marks) != 2 {
				t.Fatalf("row %d col %d has %d combining marks, want 2", row, col, len(marks))
			}
			if got, ok := index[marks[0]]; !ok || got != row {
				t.Errorf("row %d col %d encodes row %d", row, col, got)
			}
			if got, ok := index[marks[1]]; !ok || got != col {
				t.Errorf("row %d col %d encodes column %d", row, col, got)
			}
		}
	}
}

func TestPlaceholderGridRefusesOversizedArea(t *testing.T) {
	if got := placeholderGrid(testImageID, len(rowColumnDiacritics)+1, 1); got != "" {
		t.Errorf("got %q, want an empty grid when the area exceeds the diacritic table", got)
	}
}

// splitPlaceholders returns the combining marks attached to each placeholder in
// a rendered row, ignoring the surrounding colour escapes.
func splitPlaceholders(line string) [][]rune {
	var cells [][]rune
	for _, r := range line {
		switch {
		case r == placeholder:
			cells = append(cells, nil)
		case len(cells) > 0 && r >= 0x0300:
			last := len(cells) - 1
			cells[last] = append(cells[last], r)
		}
	}
	return cells
}

// Two covers can be in flight at once — moving the cursor down a list starts one
// per row that settles. They share an image id, so an overtaken load must not
// reach the terminal after the one that replaced it: the picture would be the
// old size while the screen draws the new one, and only a corner of it shows.
func TestOvertakenTransmissionIsDropped(t *testing.T) {
	var out bytes.Buffer
	k := NewKitty(&out, CellSize{Width: 9, Height: 18, Measured: true})
	img := solid(640, 640, color.RGBA{R: 10, G: 20, B: 30, A: 255})

	if _, err := k.Render(img, 14, 7, 2, 0); err != nil {
		t.Fatalf("Render(seq 2): %v", err)
	}
	newest := out.Len()

	// The older load finishes last and must be refused.
	if _, err := k.Render(img, 40, 20, 1, 0); !IsStale(err) {
		t.Errorf("Render(seq 1) = %v, want it reported as overtaken", err)
	}
	if out.Len() != newest {
		t.Error("an overtaken cover was written to the terminal")
	}

	// A newer one still gets through.
	if _, err := k.Render(img, 20, 10, 3, 0); err != nil {
		t.Fatalf("Render(seq 3): %v", err)
	}
	if out.Len() == newest {
		t.Error("the newest cover never reached the terminal")
	}
}

// Every cover goes out under the same image id. Kitty replaces the picture but
// keeps the placement a previous transmission created, so a cover of a new size
// has to arrive with the old placement deleted — otherwise the terminal draws
// into the old rectangle and only the corner that fits is ever seen.
func TestNewGeometryClearsTheOldPlacement(t *testing.T) {
	var out bytes.Buffer
	k := NewKitty(&out, CellSize{Width: 14, Height: 29, Measured: true})
	img := solid(640, 640, color.RGBA{R: 10, G: 20, B: 30, A: 255})

	if _, err := k.Render(img, 49, 23, 1, 0); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if _, err := k.Render(img, 24, 11, 2, 0); err != nil {
		t.Fatal(err)
	}

	sent := out.String()
	del := strings.Index(sent, "a=d,d=I")
	place := strings.Index(sent, "r=11")
	switch {
	case del < 0:
		t.Error("the old placement was never deleted")
	case place < 0:
		t.Error("the new placement was never sent")
	case del > place:
		t.Error("the delete came after the new placement, which undoes it")
	}
}

// testImageID stands in for the per-process id, which the grid only carries.
const testImageID = 1

// Two spindles on one machine must not share an image id, and neither may the
// two pictures one spindle draws at once. The id belongs to the terminal, and
// every cover is preceded by a delete of that id's placements: a shared number
// would take the other picture off the screen with every track change.
func TestEachRendererHasItsOwnImageID(t *testing.T) {
	k := NewKitty(io.Discard, CellSize{Width: 10, Height: 20, Measured: true})

	seen := map[int]bool{}
	for slot, id := range k.imageID {
		if id == 0 {
			t.Errorf("slot %d has image id 0, which the protocol does not accept", slot)
		}
		if id > 0xFFFFFF {
			t.Errorf("slot %d has image id %d, which does not fit the 24 bits a placeholder carries", slot, id)
		}
		if seen[id] {
			t.Errorf("slot %d shares image id %d with another slot", slot, id)
		}
		seen[id] = true
	}
	if k.imageID[0] != (os.Getpid()*slots)&0xFFFFFF|1 {
		t.Errorf("image id = %d, want it derived from the process", k.imageID[0])
	}
}

// The two pictures do not overtake each other: a load in one slot must not
// discard a newer load in the other.
func TestSlotsKeepTheirOwnSequence(t *testing.T) {
	var out strings.Builder
	k := NewKitty(&out, CellSize{Width: 10, Height: 20, Measured: true})
	img := solid(640, 640, color.White)

	if _, err := k.Render(img, 20, 10, 5, 0); err != nil {
		t.Fatalf("Render(slot 0, seq 5): %v", err)
	}
	if _, err := k.Render(img, 12, 6, 1, 1); err != nil {
		t.Errorf("Render(slot 1, seq 1) = %v, want the other slot's sequence to be its own", err)
	}
}

// Every slot the program uses is a slot this renderer has.
//
// It had two — the cursor's cover and the playing one — and two more were added
// above it without anybody counting: the next record's colour, and the picture
// shown while a device is being waited for. Render refuses a slot it does not
// have, quietly, and only where pictures are drawn this way. What that looked
// like was a logo that never appeared and a colour that never crossed, on
// exactly the terminals good enough to show both.
func TestEverySlotTheProgramUsesExists(t *testing.T) {
	k := NewKitty(io.Discard, CellSize{Width: 10, Height: 20, Measured: true})
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for slot := range slots {
		if _, err := k.Render(img, 20, 10, uint64(slot+1), slot); err != nil {
			t.Errorf("slot %d: %v", slot, err)
		}
	}
	if _, err := k.Render(img, 20, 10, 99, slots); err == nil {
		t.Error("a slot past the end was accepted, so running out would be silent")
	}
}
