package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/auth"
	"github.com/pottom/spindle/internal/daemon"
	"github.com/pottom/spindle/internal/notes"
	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"

	"github.com/pottom/spindle/internal/ui/cover"
)

// The settings, on a screen rather than in a command.
//
// Everything here was already settable — spindle quality, spindle crossfade,
// spindle notify — which is fine for something set once and never again, and
// wrong for anything somebody might want to try. A screen makes them visible:
// what is on offer, what it is set to now, and which of them the device has to
// be restarted to hear.
type settingsPane struct {
	cursor listState

	// What the settings file says, as it was last read or written. The screen
	// draws these rather than asking the disk on every frame.
	quality   daemon.Quality
	crossfade time.Duration
	notify    bool

	// changed marks a setting the running daemon has not picked up. The device
	// takes its audio settings when it starts, so this is the difference
	// between what is written down and what can be heard.
	changed bool

	// restarting is a device on its way back, so the key cannot be pressed
	// twice into two daemons.
	restarting bool

	// loaded says the file has been read. Until it has, the screen shows what
	// it has rather than defaults it would then correct.
	loaded bool
}

// settingsMsg carries the file's contents back into the model.
type settingsMsg struct {
	quality   daemon.Quality
	crossfade time.Duration
	notify    bool
}

// loadSettingsCmd reads what has been chosen before. A file that cannot be read
// is not worth a complaint on this screen: it means nothing has been chosen,
// and the defaults are what is in force anyway.
func loadSettingsCmd() tea.Cmd {
	return func() tea.Msg {
		out := settingsMsg{quality: daemon.DefaultQuality}

		if name, err := auth.Quality(); err == nil {
			if q, err := daemon.ParseQuality(name); err == nil {
				out.quality = q
			}
		}
		if value, err := auth.Crossfade(); err == nil {
			if d, err := daemon.ParseCrossfade(value); err == nil {
				out.crossfade = d
			}
		}
		if on, err := auth.Notify(); err == nil {
			out.notify = on
		}
		return out
	}
}

// The settings, in the order they matter to somebody listening.
const (
	settingQuality = iota
	settingCrossfade
	settingNotify
	settingArtwork
	settingNotes

	// Which application spindle authenticates as. Like the two above it, this
	// one reports rather than turns: it is changed by a command, and it is here
	// because it decides what the other screens may offer.
	settingApplication

	settingsCount
)

// settingsKey drives the screen. Left and right change the setting under the
// cursor; enter does the same, because on a screen of switches it is what the
// hand reaches for.
func (m *Model) settingsKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	if m.listKey(k, &m.settings.cursor, settingsCount, true) {
		return nil, true
	}

	switch {
	case m.pressed(k, m.keys.SeekFwd), m.pressed(k, m.keys.Enter):
		return m.turnSetting(1), true
	case m.pressed(k, m.keys.SeekBack):
		return m.turnSetting(-1), true

	case m.pressed(k, m.keys.Restart):
		return m.restartDevice(), true
	}
	return nil, false
}

// restartDevice stops the daemon and starts it again, so a setting it only
// reads at start-up can be heard without leaving for a shell.
//
// The music stops for a moment, which is the honest cost and is why it is a key
// of its own rather than something a change does by itself: changing three
// settings would otherwise restart the device three times.
func (m *Model) restartDevice() tea.Cmd {
	if m.settings.restarting {
		return nil
	}

	m.settings.restarting = true
	m.said, m.saidAt = "Restarting the device…", time.Now()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), restartTimeout)
		defer cancel()

		if err := daemon.Restart(ctx); err != nil {
			return msg.Error{Err: fmt.Errorf("restart the device: %w", err)}
		}
		return deviceRestarted{}
	}
}

// deviceRestarted says the device is back.
type deviceRestarted struct{}

// restartTimeout bounds the wait. Starting a daemon means authorising with
// Spotify and registering a device, which is seconds rather than moments.
const restartTimeout = 30 * time.Second

// turnSetting moves the setting under the cursor one step, and writes it down.
func (m *Model) turnSetting(delta int) tea.Cmd {
	switch m.settings.cursor.cursor {
	case settingQuality:
		m.settings.quality = turnQuality(m.settings.quality, delta)
		m.settings.changed = true
		return saveSettingCmd(func() error { return auth.SaveQuality(string(m.settings.quality)) })

	case settingCrossfade:
		m.settings.crossfade = turnCrossfade(m.settings.crossfade, delta)
		m.settings.changed = true
		return saveSettingCmd(func() error { return auth.SaveCrossfade(crossfadeValue(m.settings.crossfade)) })

	case settingNotify:
		m.settings.notify = !m.settings.notify
		m.settings.changed = true
		on := m.settings.notify
		return saveSettingCmd(func() error { return auth.SaveNotify(on) })
	}

	// The artwork row is a fact about the terminal, not a choice: it is here
	// because it explains the picture, and there is nothing to turn.
	return nil
}

// saveSettingCmd writes one setting away from the update loop. A failure
// arrives as any other error does, on the notice line.
func saveSettingCmd(save func() error) tea.Cmd {
	return func() tea.Msg {
		if err := save(); err != nil {
			return msg.Error{Err: err}
		}
		return nil
	}
}

// turnQuality steps through what Spotify offers, in the order it sounds.
func turnQuality(q daemon.Quality, delta int) daemon.Quality {
	order := []daemon.Quality{daemon.QualityLow, daemon.QualityMiddle, daemon.QualityHigh}
	at := len(order) - 1
	for i, one := range order {
		if one == q {
			at = i
		}
	}
	return order[((at+delta)%len(order)+len(order))%len(order)]
}

// turnCrossfade steps the overlap by a second at a time, from gapless to the
// longest Spotify's own clients offer.
func turnCrossfade(d time.Duration, delta int) time.Duration {
	step := time.Duration(delta) * time.Second
	next := d + step
	switch {
	case next < 0:
		return daemon.MaxCrossfade
	case next > daemon.MaxCrossfade:
		return 0
	default:
		return next
	}
}

// crossfadeValue is how the setting is written down: seconds, or the word for
// none.
func crossfadeValue(d time.Duration) string {
	if d <= 0 {
		return "off"
	}
	return fmt.Sprintf("%d", int(d.Seconds()))
}

// settingRow is one line of the screen.
type settingRow struct {
	name  string
	value string

	// says is what the setting is for, shown under the cursor. One sentence:
	// somebody is here to change something, not to read a manual.
	says string

	// live says the change is heard at once. The rest wait for the device to
	// be started again, and saying which is which is the whole reason this
	// screen can be trusted.
	live bool
}

// settingRows is the screen's contents, from what has been chosen.
func (m Model) settingRows() []settingRow {
	artwork, why := "half blocks", artworkSays(cover.Graphics{})
	if m.covers != nil && m.covers.Renderer() != nil {
		artwork, why = m.covers.Renderer().Name(), artworkSays(m.covers.Graphics())
	}

	return []settingRow{{
		name:  "Sound quality",
		value: string(m.settings.quality),
		says:  "What to ask Spotify for. High is 320 kbps, and needs Premium.",
	}, {
		name:  "Crossfade",
		value: describeOverlap(m.settings.crossfade),
		says:  "How long one track overlaps the next as it ends.",
	}, {
		name:  "Track notifications",
		value: onOff(m.settings.notify),
		says:  "Announce each new track to the desktop, for when the window is not in front of you.",
	}, {
		name:  "Artwork",
		value: artwork,
		says:  why,
		live:  true,
	}, {
		name:  "Artist notes",
		value: m.notesSays(),
		says:  notesSays(m.notes),
		live:  true,
	}, {
		name:  "Spotify application",
		value: m.applicationSays(),
		says:  m.applicationWhy(),
		live:  true,
	}}
}

// applicationSays names which registration spindle is authenticating as, and
// what that costs.
//
// It is on this screen because it is the one setting that decides what other
// screens may offer: an application registered since Spotify's 2024 clampdown
// cannot like a track or open somebody else's playlist, and a listener who has
// put their own id in deserves to be told which keys went missing rather than
// left wondering. See player.Abilities and docs/SPOTIFY-API.md.
func (m Model) applicationSays() string {
	if !m.asksAllows {
		return "the one spindle ships with"
	}

	var lost []string
	for _, info := range player.Abilities {
		if !m.allows.Has(info.Ability) {
			lost = append(lost, info.Name)
		}
	}
	if len(lost) == 0 {
		return "your own · everything allowed"
	}
	return "your own · without " + strings.ToLower(strings.Join(lost, ", "))
}

// applicationWhy is the sentence under it.
func (m Model) applicationWhy() string {
	if !m.asksAllows {
		return "Registered before Spotify's 2024 changes, so nothing here is refused. Your own: spindle login <client id>"
	}
	for _, info := range player.Abilities {
		if !m.allows.Has(info.Ability) {
			// The first thing missing, said in full. A list of everything lost
			// would not fit on a line, and one example is what makes the
			// trade-off real to somebody reading it.
			return "Spotify refuses this application " + info.Lost + ". The one spindle ships with is not refused."
		}
	}
	return "Your own registration, and Spotify allows it everything."
}

// notesSays names the databases spindle is asking about an artist, and
// notesSays what a key would add where there is none.
//
// A row that reports rather than turns, like the artwork above it: what is in
// the chain is decided by what has been configured, and configuring it is a
// command rather than a key. It is here because the alternative is a feature
// nobody knows exists — and because somebody who does not want it should see
// that spindle is not asking anybody anything on their behalf.
func (m Model) notesSays() string {
	if m.notes == nil {
		return "off"
	}
	return strings.Join(m.notes.Names(), " · ")
}

func notesSays(held *notes.Cached) string {
	if held == nil {
		return "Nothing is asked about the artists you play."
	}
	if held.Has("Last.fm") {
		return "Who an artist is, who else listens to them, and how many."
	}
	return "Who an artist is. A last.fm key adds the ones no other database has heard of — spindle lastfm"
}

// artworkSays explains the cover, and where it would be better.
//
// The row already said which of the two was in use. What it did not say was
// why, or that there is anything better — so somebody running spindle in a
// terminal without the protocol saw a blurry approximation of the one thing the
// program is built around and had no way of knowing a sharp version exists.
//
// Here rather than in a banner. A banner on every start is nagging about
// something that is not going to change, and a line buried in the help is
// invisible; this is the screen somebody is already on when they wonder why the
// cover looks like that.
func artworkSays(g cover.Graphics) string {
	const better = " kitty and Ghostty draw it as a picture."
	switch {
	case g.Backend() == "kitty":
		return "How the cover is drawn. This terminal takes the picture itself."
	case g.Kitty:
		// It speaks the protocol and not the part spindle draws in, which is
		// worth saying: "no graphics protocol" would be untrue and would send
		// somebody looking for a setting that would not help.
		return "How the cover is drawn. " + g.Name +
			" speaks the graphics protocol but not the placeholders spindle uses, so the cover is coloured blocks." + better
	default:
		return "How the cover is drawn. This terminal has no graphics protocol, so the cover is coloured blocks." + better
	}
}

func onOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

// describeOverlap says a crossfade the way somebody would.
func describeOverlap(d time.Duration) string {
	if d <= 0 {
		return "off"
	}
	return fmt.Sprintf("%d seconds", int(d.Seconds()))
}

// settingsPanel draws the screen.
func (m Model) settingsPanel(l layout, rows int) []string {
	w := l.interior - leftMargin - rightMargin
	s := m.styles

	lines := []string{
		fit(s.Title.Render("Settings"), w),
		fit(s.Album.Render("what spindle keeps between runs"), w),
		strings.Repeat(" ", w),
	}

	items := m.settingRows()
	for i, row := range items {
		style, gutter := s.RowPrimary, "  "
		if i == m.settings.cursor.cursor {
			style, gutter = s.RowSelected, s.Cursor.Render(rowCursor)+" "
		}

		// The value is set against the name rather than out at the edge: they
		// are one statement — this is called that — and a field of dots between
		// them would be a form.
		name := fit(style.Render(row.name), settingsNameCols)
		value := s.Artist.Render(row.value)
		lines = append(lines, fit(gutter+name+value, w))
	}

	// What the cursor is on, explained, and what it will take to hear it. Both
	// under the list rather than beside it: the sentence is long and the values
	// are short, and a column that fits one badly fits the other worse.
	lines = append(lines, strings.Repeat(" ", w))
	if at := m.settings.cursor.cursor; at >= 0 && at < len(items) {
		lines = append(lines, fit(s.Detail.Render(items[at].says), w))
		if !items[at].live {
			lines = append(lines, fit(s.Empty.Render("The device takes this when it starts."), w))
		}
	}

	if m.settings.changed {
		what := warnGlyph + " The device has not heard this yet — press " + keyRestart + " to restart it"
		if m.settings.restarting {
			what = warnGlyph + " Restarting the device…"
		}
		lines = append(lines, strings.Repeat(" ", w), fit(s.Warning.Render(what), w))
	}

	for len(lines) < rows {
		lines = append(lines, strings.Repeat(" ", w))
	}
	return lines[:rows]
}

// settingsChrome is what the screen spends above the switches: its name, the
// line under it, and the blank. Named because the pointer counts down from the
// top of the body to find which switch it is over, and a fourth line added here
// would leave every click a row out. See settingsSpot in mouse.go.
const settingsChrome = 3

// settingsNameCols is the column the names are set in, so the values line up
// under each other.
const settingsNameCols = 26
