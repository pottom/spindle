package ui

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/player"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{-time.Second, "0:00"},
		{0, "0:00"},
		{900 * time.Millisecond, "0:01"},
		{time.Second, "0:01"},
		{9 * time.Second, "0:09"},
		{time.Minute, "1:00"},
		{5*time.Minute + 55*time.Second, "5:55"},
		{12*time.Minute + 3*time.Second, "12:03"},
		{time.Hour, "1:00:00"},
		{time.Hour + 2*time.Minute + 4*time.Second, "1:02:04"},
	}

	for _, c := range cases {
		if got := formatDuration(c.in); got != c.want {
			t.Errorf("formatDuration(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// The bitrate answers "what am I actually hearing", so it belongs next to the
// device — but only while something is arriving.
func TestStatusLineShowsTheBitrateWhilePlaying(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{DeviceName: "spindle", Playing: true, Bitrate: 320}

	if got := m.statusLine(); !strings.Contains(got, "320 kbps") {
		t.Errorf("statusLine() = %q, want the bitrate in it", got)
	}

	m.ps.Playing = false
	if got := m.statusLine(); strings.Contains(got, "kbps") {
		t.Errorf("statusLine() = %q, want no bitrate while paused", got)
	}

	m.ps.Playing, m.ps.Bitrate = true, 0
	if got := m.statusLine(); strings.Contains(got, "kbps") {
		t.Errorf("statusLine() = %q, want no bitrate when unknown", got)
	}
}

// The mark beside the device is the only thing on screen that says sound is
// coming out right now, so it turns while playing and settles when it stops.
func TestDeviceMarkTurnsOnlyWhilePlaying(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{DeviceName: "spindle", Playing: true}

	first := m.statusLine()
	m.device, _ = m.device.Update(m.device.Tick().(spinner.TickMsg))
	if second := m.statusLine(); second == first {
		t.Errorf("statusLine() = %q on two frames, want the mark to have moved", first)
	}

	m.ps.Playing = false
	if got := m.statusLine(); !strings.Contains(got, deviceDot) {
		t.Errorf("statusLine() = %q, want the plain dot once stopped", got)
	}
}

// Nothing may drive the mark while the music is stopped: it would redraw the
// screen eight times a second to say nothing.
func TestDeviceMarkDoesNotTickWhileStopped(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{DeviceName: "spindle", Playing: false}

	if cmd := m.spinDevice(); cmd != nil {
		t.Error("spinDevice() started a tick loop while stopped")
	}

	m.ps.Playing = true
	if cmd := m.spinDevice(); cmd == nil {
		t.Fatal("spinDevice() = nil while playing, want a tick")
	}
	// And a second caller must not start a competing loop.
	if cmd := m.spinDevice(); cmd != nil {
		t.Error("spinDevice() started a second tick loop")
	}
}

// The top row carries both bearings: which machine is making the sound, and
// which screen you are on. The rule has to land under the active tab, or it
// points at the wrong one.
func TestHeaderPutsTheDeviceLeftAndTheTabsRight(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{DeviceName: "spindle", Playing: true, Bitrate: 320}
	m.tab = tabQueue

	rows := m.header(96)
	if len(rows) != 2 {
		t.Fatalf("header() = %d rows, want 2", len(rows))
	}

	top := ansiOff(rows[0])
	if !strings.HasPrefix(strings.TrimLeft(top, " "), "◐") && !strings.Contains(top, "spindle") {
		t.Errorf("header row = %q, want the device on the left", top)
	}
	if !strings.HasSuffix(strings.TrimRight(top, " "), "search") {
		t.Errorf("header row = %q, want the tabs flush right", top)
	}

	// The rule sits under the active tab, which is "queue".
	rule := ansiOff(rows[1])
	if got, want := column(rule, "━"), column(top, "queue"); got != want {
		t.Errorf("rule starts at column %d, want %d — under the active tab", got, want)
	}
}

// column is where a substring begins on screen. Counting bytes would be wrong
// the moment a line carries anything outside ASCII, which every one of them does.
func column(line, want string) int {
	at := strings.Index(line, want)
	if at < 0 {
		return -1
	}
	return lipgloss.Width(line[:at])
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func ansiOff(s string) string { return ansiPattern.ReplaceAllString(s, "") }

// A long device name must not push the tabs off the edge: the name is a detail,
// the tabs are how you move around.
func TestHeaderKeepsTheTabsWhole(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{DeviceName: strings.Repeat("very long name ", 6), Playing: true, Bitrate: 320}

	top := ansiOff(m.header(minWidth - leftMargin - rightMargin)[0])
	if !strings.HasSuffix(strings.TrimRight(top, " "), "search") {
		t.Errorf("header row = %q, want the tabs intact", top)
	}
	if !strings.Contains(top, "…") {
		t.Errorf("header row = %q, want the device name cut instead", top)
	}
}

// The help belongs on the bottom row of the terminal with clear space above it.
// Butted straight against a list it reads as one more entry in it.
func TestHelpSitsOnTheLastRowWithSpaceAbove(t *testing.T) {
	p := player.NewMock()
	st, err := p.State(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	q, err := p.Queue(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	for _, size := range [][2]int{{minWidth, minHeight}, {96, 28}, {120, 44}} {
		for _, tab := range []tabID{tabPlayer, tabQueue, tabLibrary, tabSearch} {
			m := New(p, nil, defaultTestCell)
			m.ps, m.tab, m.queue = st, tab, q.Upcoming
			// A queue longer than the pane, so the list runs to the last row
			// it is given and cannot leave a gap of its own.
			for i := range 40 {
				m.queue = append(m.queue, trackAt(fmt.Sprintf("f%d", i), "filler"))
			}
			m.width, m.height = size[0], size[1]
			m.resize()

			rows := strings.Split(ansiOff(m.render()), "\n")
			if len(rows) != m.height {
				t.Errorf("%dx%d %v: render() = %d rows, want %d", size[0], size[1], tab, len(rows), m.height)
				continue
			}
			if strings.TrimSpace(rows[len(rows)-1]) == "" {
				t.Errorf("%dx%d %v: last row is blank, want the help", size[0], size[1], tab)
			}
			if got := strings.TrimSpace(rows[len(rows)-2]); got != "" {
				t.Errorf("%dx%d %v: row above the help = %q, want it blank", size[0], size[1], tab, got)
			}
		}
	}
}

// The digits go straight to a screen, which is faster than tabbing round to it
// — except on the search tab, where a digit is something you are typing.
func TestDigitsSwitchTabs(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	if got := tm.(Model).tab; got != tabLibrary {
		t.Errorf("tab = %v after 3, want playlists", got)
	}

	tm, _ = tm.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	if got := tm.(Model).tab; got != tabPlayer {
		t.Errorf("tab = %v after 1, want now playing", got)
	}

	// On the search tab the digit is a query, not a destination.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '4', Text: "4"})
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	got := tm.(Model)
	if got.tab != tabSearch {
		t.Errorf("tab = %v, want the digit to stay in the query", got.tab)
	}
	if q := got.search.input.Value(); q != "2" {
		t.Errorf("query = %q, want the digit typed", q)
	}
}

// Both screens draw the same facts from the same place. The queue names them
// because it is read as a table; the player leaves the names off because they
// sit under the title as one caption — and a bare number there has to say what
// it is.
func TestBothScreensDrawTheSameFacts(t *testing.T) {
	pop := 50
	track := player.Track{
		Title: "Holiday", Artists: []string{"Madonna"},
		Album: "The Immaculate Collection", Duration: 4*time.Minute + 2*time.Second,
		Released: "1990-11-09", AlbumType: "compilation", TrackNumber: 1, Popularity: &pop,
	}

	m := New(player.NewMock(), nil, defaultTestCell)
	caption := ansiOff(strings.Join(m.trackCaption(track, 60), "\n"))

	for _, want := range []string{"The Immaculate Collection", "1990", "compilation", "track 1", starFull} {
		if !strings.Contains(caption, want) {
			t.Errorf("trackCaption() = %q, want %q in it", caption, want)
		}
	}
	// The names belong to the queue's table, not here.
	for _, unwanted := range []string{"Album", "Released", "Track", "Popularity"} {
		if strings.Contains(caption, unwanted) {
			t.Errorf("trackCaption() = %q, want no label %q", caption, unwanted)
		}
	}
	// And the length is under the progress bar, where it is read against the
	// elapsed time; repeating it here would be furniture.
	if strings.Contains(caption, "4:02") {
		t.Errorf("trackCaption() = %q, want no duration", caption)
	}
}

// The volume is drawn like the progress bar: what is set in the accent, the
// rest of the way faint, and the marker on the join. One control, one shape —
// a meter of its own beside it read as a different kind of thing entirely.
func TestVolumeIsShapedLikeTheProgressBar(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{Volume: 50, Duration: time.Minute, Progress: 30 * time.Second, Playing: true}

	volume := ansiOff(m.volumeLine(volumeCells))
	if got := len([]rune(volume)); got != volumeCells {
		t.Errorf("the volume bar is %d cells, want %d", got, volumeCells)
	}
	if !strings.Contains(volume, knob) {
		t.Error("the volume bar has no marker")
	}
	if !strings.Contains(volume, meterFull) || !strings.Contains(volume, meterEmpty) {
		t.Errorf("the volume bar = %q, want it filled up to the marker and faint after", volume)
	}

	// The marker moves with the setting, and the accent runs up to it.
	at := func(v int) int {
		m.ps.Volume = v
		return strings.Index(ansiOff(m.volumeLine(volumeCells)), knob)
	}
	if a, b := at(0), at(100); a >= b {
		t.Errorf("the marker sits at %d for silence and %d for full, want it to travel", a, b)
	}

	// It answers the paused state the same way the progress bar does: both go
	// grey, so one control never contradicts the other.
	m.ps.Volume = 50
	for _, playing := range []bool{true, false} {
		m.ps.Playing = playing
		if a, b := colourUsed(m.volumeLine(volumeCells)), colourUsed(m.progressLine(40)); a != b {
			t.Errorf("playing=%v: the volume is %q and the progress bar %q, want the same",
				playing, a, b)
		}
	}
}

// The transport is drawn in the artwork's accent, like the bars beside it.
func TestTransportIsAccent(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{Volume: 50, Duration: time.Minute, Playing: true}

	if got, want := m.styles.Controls.GetForeground(), m.styles.Accent; got != want {
		t.Errorf("the transport is %v, want the accent %v", got, want)
	}
	// The icons are rendered as one run, so the row carries that colour.
	if !strings.Contains(m.transportLine(60), colourUsed(m.styles.Controls.Render("x"))) {
		t.Error("the transport row is not drawn in that colour")
	}
}

// Muting silences the device and remembers what to come back to. The music
// carries on: this is the volume control, not the transport.
func TestMuteRemembersTheLevel(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{Volume: 70, Playing: true, Duration: time.Minute}

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	got := tm.(Model)
	if got.ps.Volume != 0 {
		t.Errorf("volume = %d after muting, want 0", got.ps.Volume)
	}
	if !got.ps.Playing {
		t.Error("muting stopped the music")
	}

	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if v := tm.(Model).ps.Volume; v != 70 {
		t.Errorf("volume = %d after unmuting, want the level it was at", v)
	}
}

// Reaching for the volume is a decision about the level, so there is nothing
// left to come back to.
func TestVolumeKeyClearsTheMute(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{Volume: 70, Playing: true, Duration: time.Minute}

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})

	if v := tm.(Model).ps.Volume; v == 70 {
		t.Error("the old level came back after the volume had been set by hand")
	}
}

// The row is laid out from the right, so a reading that narrows drags the bar
// along with it. It holds three columns whatever it says.
func TestVolumeBarDoesNotMoveWithTheReading(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{Volume: 100, Playing: true, Duration: time.Minute}

	// Where the bar begins is what must not move; the marker inside it travels
	// with the setting by design.
	at := func(v int) int {
		m.ps.Volume = v
		row := ansiOff(m.transportLine(70))
		return min(indexOr(row, meterFull), indexOr(row, knob))
	}
	full, two, one := at(100), at(90), at(5)
	if full != two || two != one {
		t.Errorf("the bar starts at %d, %d and %d for 100, 90 and 5, want it still", full, two, one)
	}
}

// indexOr is strings.Index with a large sentinel for "not there", so a missing
// piece never wins a min().
func indexOr(s, sub string) int {
	if at := strings.Index(s, sub); at >= 0 {
		return at
	}
	return 1 << 20
}
