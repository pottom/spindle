package ui

import "strings"

// The screens are made of blocks, and this is where a block is defined: once,
// by name, with how it fills a size. A screen then says which blocks it holds
// and where, and nothing else needs to know how any of them is drawn.
//
// The point is what it costs to move one. Before this, a block was built where
// it was placed, so putting the up-next glance on another tab meant carrying its
// composition, its width arithmetic and its name across with it — three things
// to get right in a fourth place. Now it is one entry: the tab that wants it
// names it, the tab that had it stops naming it.
//
// It also puts the name and the geometry in one place. They were written twice
// on the same line in places, which is how a block ends up outlined at a width
// it is not drawn at.

// block is one rectangle of a screen: what it is called, and how it fills a
// given width and height.
type block struct {
	name string
	rows func(w, h int) []string
}

// place draws a block at a size, and puts its border round it while the outlines
// are up. Everything that draws a block goes through here, so no block can be
// drawn at one size and measured at another.
func (m Model) place(b block, w, h int) []string {
	return m.outline(b.rows(w, h), w, b.name)
}

// artBlock is the cover. It is the one block drawn two ways: centred on the
// player, where it is the subject and has a caption beside it, and hung from the
// top on a list, where it heads the rows below.
func (m Model) artBlock(centred bool) block {
	return block{"art", func(w, h int) []string {
		cells := strings.Split(m.artworkCells(), "\n")
		if centred {
			return center(cells, w, h)
		}
		return alignTop(cells, w, h)
	}}
}

// traceBlock is the waveform, the bars or the ladder — whichever the trace is
// set to. It appears under the picture on the player and beside it on a list,
// and it is the same block in both.
func (m Model) traceBlock() block {
	return block{"trace", func(w, h int) []string {
		return stack(m.scopeRender(w, h), w, h)
	}}
}

// playerBlock is the track, the clock and the transport: what somebody means by
// "the player part".
func (m Model) playerBlock() block {
	return block{"player", func(w, h int) []string {
		return stack(m.infoBlock(w), w, h)
	}}
}

// wordsBlock is the lyric window.
func (m Model) wordsBlock() block {
	return block{"lyrics", func(w, h int) []string {
		return m.lyricsBlock(w, h)
	}}
}

// upNextBlock is the glance at what follows: the first few of the queue, set
// like the tables on the list screens. It is on the player today; it is one line
// to put it anywhere else.
//
// The same table, because it is the same thing: a list of tracks in columns. It
// had columns of its own before and no names on them, which is the state every
// other list was in until they were named — and this is the list somebody reads
// out of the corner of an eye, where a column nobody can name is worth least.
//
// Its name stands where the title's would be, in the column the titles are in.
// The heading row was a row of its own with one word on it and nothing else, and
// a glance is four rows tall: a whole row for a label, above a list that can only
// afford four, is the label costing a quarter of what it labels.
func (m Model) upNextBlock() block {
	return block{"up next", func(w, h int) []string {
		// The rows are drawn flush: there is no cursor on this list, so the
		// column one would stand in is dead space.
		glance := m
		glance.rowsAreFlush = true
		// And every row of it is in the queue already, which is what takes the
		// column of queued dots off the front of the titles. See queuedColumn.
		glance.rowsAreTheQueue = true

		name := m.styles.Title.Render("Up next")
		if len(m.queue) == 0 {
			return append([]string{spread(name, m.styles.Empty.Render(peekSubtitle(0)), w)},
				blankRows(w, h-1)...)
		}

		out := []string{glance.columnHead(w, name, rowCells{
			secondary: "artist",
			album:     "album",
			stars:     "stars",
			liked:     "liked",
			tempo:     "tempo",
			trailing:  "time",
		})}
		out = append(out, fit(m.styles.Rule.Render(strings.Repeat(pointerH, w)), w))

		for i := range h - len(out) {
			row := strings.Repeat(" ", w)
			if i < len(m.queue) {
				row = glance.trackRow(m.queue[i], w, false, i+1)
			}
			out = append(out, row)
		}
		return out
	}}
}

// blankRows is n rows of nothing, for a block with nothing to put in them.
func blankRows(w, n int) []string {
	out := make([]string, max(n, 0))
	for i := range out {
		out[i] = strings.Repeat(" ", w)
	}
	return out
}

// nowBlock is the small cover and caption of what is playing, for the screens
// whose subject is a list rather than a track.
func (m Model) nowBlock(l layout, foot int) block {
	return block{"now", func(w, h int) []string {
		return m.nowPanel(l, w, h, foot)
	}}
}

// devicesBlock is the picker, which stands in for whatever it is opened over.
func (m Model) devicesBlock() block {
	return block{"devices", func(w, h int) []string {
		return m.devicePicker(w, h)
	}}
}
