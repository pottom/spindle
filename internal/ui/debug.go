package ui

import (
	"fmt"
	"strings"
	"time"
)

// A line across the top saying what the screen is deciding from.
//
// Built after a bug that took a day and five wrong fixes: a chorus arrived the
// same way on every return, and every measurement made of it was of the path a
// chorus does not take. What found it in the end was printing one number on the
// screen and reading it off — the deal was 4 for every occurrence and 5 for the
// line after, which said in one glance what an afternoon of instrumented tests
// had not.
//
// So the numbers this screen decides from live on a key instead of being
// rebuilt each time something is doubted. What belongs here is anything a
// picture is dealt from and nothing a picture is made of: the deal, the clocks
// it is dealt against, and the measurements coming in from the daemon.
//
// It is off unless asked for, it costs nothing while it is off, and it is not a
// feature — nobody is meant to run with it on.
type debugState struct{ on bool }

// debugKey takes the toggle. Ctrl and shift together, because ctrl alone is
// spoken for: ctrl+d is half a page down in the lists.
func (m *Model) debugKey(k string) bool {
	if k != "ctrl+shift+d" {
		return false
	}
	m.debug.on = !m.debug.on
	return true
}

// debugLine is the bar itself, or empty when it is not asked for.
func (m Model) debugLine() string {
	if !m.debug.on || m.width < 40 {
		return ""
	}

	var b strings.Builder
	add := func(f string, a ...any) { fmt.Fprintf(&b, f+"  ", a...) }

	// Where the interface is, and how big.
	add("%s/%s %dx%d", debugScreen(m), debugMode(m.scopeMode()), m.width, m.height)

	// What is playing and where it has got to.
	if m.ps != nil {
		add("%s %s/%s", debugShort(m.ps.Title, 14),
			debugClock(m.elapsed()), debugClock(m.ps.Duration))
	} else {
		add("no track")
	}

	// The deal, which is what the day this was built for was about.
	if m.words.text != "" {
		gather := float32(1)
		if since := time.Since(m.words.since); since < wordsGather {
			gather = float32(since) / float32(wordsGather)
		}
		kind := "line"
		if m.words.beats {
			kind = "marks"
		}
		if m.words.telling {
			kind = "card"
		}
		cast := m.words.cast
		if cast == "" {
			cast = "notes"
		}
		add("%s move %d leave %d gather %2.0f%% starts %s cast %s words %d",
			kind, m.words.move, m.words.leave, gather*100,
			debugClock(time.Duration(m.words.starts)*time.Millisecond), cast, m.words.where.Count)
	} else {
		add("nothing set")
	}

	// The sheet, since half of what goes wrong here is what the sheet says.
	switch {
	case m.ps == nil:
	case m.lyrics.forTrack != m.ps.TrackID:
		add("sheet waiting")
	case m.lyrics.missing:
		add("sheet none")
	case !m.lyrics.synced:
		add("sheet untimed")
	default:
		add("sheet %d lines", len(m.lyrics.lines))
	}

	// The beat, and whether anything is keeping it.
	if beat := m.scope.beat; beat.Found() {
		add("beat %.0f bpm %s", 60000/float64(beat.Period.Milliseconds()+1),
			map[bool]string{true: "kept", false: "loose"}[m.beatKeeping()])
	} else {
		add("beat none")
	}

	// The record's own turns, which is what the picture is dealt on now.
	add("joins %d last %s", m.joinsTurns(), debugClock(m.elapsed()-m.joinsAt()))

	// How hard the record is going against its own range, and how loud it is.
	add("swell %.2f loud %.0fdB", m.swell(), m.scope.beat.Loud)

	// And the frames, from the timing that is still in. See slow.go.
	slowState.mu.Lock()
	frames, missed, worst := slowState.frames, slowState.missed, slowState.worst
	slowState.mu.Unlock()
	if frames > 0 {
		add("frames %d late %d (%.1f%%) worst %dms", frames, missed,
			float64(missed)/float64(frames)*100, worst.Milliseconds())
	}

	line := strings.TrimRight(b.String(), " ")
	if over := []rune(line); len(over) > m.width {
		line = string(over[:m.width])
	}
	return line
}

// debugMode is which picture the big screen is set to, in one word.
func debugMode(s scopeMode) string {
	switch {
	case s.wave():
		return "wave"
	case s.bars():
		return "bars"
	case s.words():
		return "words"
	case s.ladder():
		return "ladder"
	case s.mirror():
		return "mirror"
	}
	return "off"
}

// debugScreen is which of the three is up.
func debugScreen(m Model) string {
	switch {
	case m.stage.on:
		return "stage"
	case m.lyrics.on:
		return "lyrics"
	default:
		return "player"
	}
}

// debugOver writes the bar across the top row of whatever was drawn.
//
// Over rather than above: a bar that took a row of its own would move
// everything under it, and a picture measured with the debug on would not be
// the picture measured with it off.
func (m Model) debugOver(screen string) string {
	line := m.debugLine()
	if line == "" {
		return screen
	}

	rows := strings.SplitN(screen, "\n", 2)
	pad := m.width - len([]rune(line))
	out := "\x1b[0m" + line + strings.Repeat(" ", max(pad, 0)) + "\x1b[0m"
	if len(rows) == 2 {
		return out + "\n" + rows[1]
	}
	return out
}

func debugClock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	return fmt.Sprintf("%d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}

func debugShort(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}
