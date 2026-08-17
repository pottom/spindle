package ui

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/xdg"
)

// What the application spindle authenticates as is allowed to do, remembered.
//
// The answer costs a request, and it is the same answer every time for weeks:
// what a registration may do is decided when it is registered. So it is asked
// once and written down beside the client id it was asked about — a different
// application is a different answer, and swapping one in has to change what the
// screen offers rather than what it remembers.
//
// It is asked at all only where the listener has brought their own application.
// The one spindle ships with is known to be allowed everything, and spending a
// request every week to hear it again would be a request spent on nothing. See
// player.Allows and docs/SPOTIFY-API.md.

// allowsHeld is what was asked, about which application, and when.
type allowsHeld struct {
	ClientID string            `json:"granted_to"`
	Allows   player.Allowances `json:"allows"`
	At       time.Time         `json:"at"`
}

// allowsFor is how long an answer stands. A week: Spotify has moved this line
// before and may again, and a week is short enough that a listener who is given
// their endpoints back does not have to know why they came back.
const allowsFor = 7 * 24 * time.Hour

// allowsTook is an answer arriving.
type allowsTook struct {
	allows player.Allowances
	err    error
}

// askAllows finds out what this application may do, unless it is already known.
func (m Model) askAllows() tea.Cmd {
	if m.clientID == "" || !m.asksAllows {
		return nil
	}
	if held, ok := readAllows(m.clientID); ok {
		allows := held
		return func() tea.Msg { return allowsTook{allows: allows} }
	}

	p, ok := m.player.(player.Allower)
	if !ok {
		return nil
	}
	id := m.clientID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		allows, err := p.Allows(ctx)
		if err == nil {
			keepAllows(id, allows)
		}
		return allowsTook{allows: allows, err: err}
	}
}

// tookAllows takes the answer up. A failure leaves the narrow set in place:
// offering a key that cannot work is worse than offering one key too few, and
// the next run asks again.
func (m *Model) tookAllows(message allowsTook) {
	if message.err != nil {
		return
	}
	m.allows = message.allows
}

// allowsPath is where the answer is written down.
func allowsPath() string {
	dir, err := xdg.StateDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "allows.json")
}

// readAllows returns what was written down about this application, where it is
// still worth trusting.
func readAllows(clientID string) (player.Allowances, bool) {
	path := allowsPath()
	if path == "" {
		return player.Allowances{}, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return player.Allowances{}, false
	}

	var held allowsHeld
	if json.Unmarshal(data, &held) != nil {
		return player.Allowances{}, false
	}
	if held.ClientID != clientID || time.Since(held.At) >= allowsFor {
		return player.Allowances{}, false
	}
	return held.Allows, true
}

// keepAllows writes an answer down. A failure to write is not worth reporting:
// the answer is in hand, and the cost of losing it is one request next time.
func keepAllows(clientID string, allows player.Allowances) {
	path := allowsPath()
	if path == "" {
		return
	}

	data, err := json.Marshal(allowsHeld{ClientID: clientID, Allows: allows, At: time.Now()})
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
}

// refuseOpen takes up a refusal of the page that is open: the screen says why
// instead of saying "nothing here", and nothing offers that ability again this
// run.
//
// Which ability it was is not asked of the error. What was refused is whatever
// was being read, and the only thing that reads a list somebody else owns is a
// playlist opened from the library or a search — so the page that was waiting
// is the one that has been refused.
func (m *Model) refuseOpen() {
	if page := m.openMut(); page != nil {
		page.refused = true
		page.pages.loading = false
	}
	if m.allows == nil {
		m.allows = player.Allowances{}
	}
	m.allows[player.Elsewhere] = false
}
