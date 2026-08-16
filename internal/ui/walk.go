package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// Reading a list to its end.
//
// It used to be the cursor that pulled the next page in: come within ten rows of
// what was loaded and another fifty were sent for. That is the right shape for a
// catalogue nobody will ever reach the end of, and the wrong one for a library —
// searching a list that is a tenth loaded searches a tenth of it and says
// nothing about the rest, and every list that is worth having is worth having
// whole.
//
// So a list read once is now read to its end: the page that lands sends for the
// next one, and so on until there is no next one. What that costs is arithmetic
// rather than opinion — a three thousand track playlist is sixty requests at
// fifty a page — and against the daily quota that is nothing. What is not
// nothing is doing it again: the danger measured on 2026-08-15 was sixty
// thousand requests a day from cadences nobody had added up, not a burst of
// sixty.
//
// So it is once. A list already walked is not walked again, however many times
// it is opened, until spindle is restarted or the list is replaced by something
// else. The refresh that catches an edit made on a phone still re-reads the
// first page, and that is all it re-reads.

const (
	// walkGap is the pause between one page and the next.
	//
	// Not nothing: fifteen requests at once was what Spotify answered with a
	// wall of 503s when the databases were being measured, and concurrency is
	// what trips these limits rather than rate. One at a time with a quarter of
	// a second between them is four a second at worst, and it reads a long
	// playlist in about fifteen seconds.
	//
	// A number to change from a measurement rather than from this comment. It
	// has not been measured against a live account, because the account was
	// locked out for the day when it was written.
	walkGap = 250 * time.Millisecond

	// walkMost is how many pages one list may be read in. Far past anything
	// real — a hundred pages is five thousand tracks at fifty a page — and there
	// so that a backend answering "there is more" for ever cannot spend a day's
	// quota in a minute.
	walkMost = 100
)

// walking reports whether a list should send for its next page now.
//
// Not where the cursor is any more. What matters is that there is more of it,
// that nothing is already on its way, and that Spotify has not asked to be left
// alone.
func (m Model) walking(p paging, pages int) bool {
	return p.more && !p.loading && pages < walkMost && !m.throttled()
}

// readOn sends for the next page of whatever list is on screen, if there is one.
//
// Called as each page lands, so a list reads itself to the end one page at a
// time. Each list keeps its own count of how many pages it has taken, which is
// both the cap and the answer to "has this been walked already": a list whose
// last page has arrived has more=false and never asks again.
func (m *Model) readOn() tea.Cmd {
	switch {
	case m.open() != nil:
		page := m.openMut()
		if !m.walking(page.pages, page.pages.pages) {
			return nil
		}
		page.pages.loading = true
		page.pages.pages++
		return paced(fetchOpenCmd(m.player, page.kind, page.id, page.pages.next))

	case m.tab == tabLibrary:
		paging := m.library.paging()
		if !m.walking(*paging, paging.pages) {
			return nil
		}
		paging.loading = true
		paging.pages++
		return paced(fetchLibraryCmd(m.player, m.library.kind, paging.next))
	}
	return nil
}

// paced puts a pause in front of a command.
//
// In the command rather than on a timer, because a command already runs off the
// update loop: nothing on screen is waiting for this, and a tick would put a
// message through the model for something the model has no opinion about.
func paced(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		time.Sleep(walkGap)
		return cmd()
	}
}
