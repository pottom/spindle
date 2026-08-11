package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/build"
	"github.com/pottom/spindle/internal/xdg"
)

// The numbers the screen decides from, on a key.
//
// Built after a bug that took a day and five wrong fixes: a chorus arrived the
// same way every time it came round, and every measurement made of it was of the
// path a chorus does not take. What found it in the end was printing one number
// on the screen and reading it off — the deal was 4 for every occurrence and 5
// for the line after, which said in one glance what an afternoon of instrumented
// tests had not. It found a second one before it was even finished: the count of
// how many times the record had turned over was written down and never
// incremented, so the colour that travels round the palette with it had been
// standing still since the day it was written.
//
// So it holds everything a picture is dealt from, and nothing a picture is made
// of. The deal, the clocks it is dealt against, the measurements coming in from
// the daemon, and the thresholds those measurements are compared with — a
// reading with no line beside it cannot be judged, which is why the join's
// novelty is printed next to what it has to beat rather than on its own.
//
// Two depths, because the two questions are different. One line answers "what is
// it doing"; the block answers "why". The key walks through off, the line, the
// block, and back to off.
//
// It costs nothing while it is off, and it is not a feature — nobody is meant to
// run with it on.
type debugState struct{ level int }

const (
	debugOff = iota
	debugTerse
	debugFull
	debugDepths
)

// debugFile is where the bar writes itself down, under the state directory. It
// is removed as spindle closes — see ForgetDebug.
const debugFile = "debug.jsonl"

// debugKey takes the toggle. Ctrl and shift together, because ctrl alone is
// spoken for: ctrl+d is half a page down in the lists.
func (m *Model) debugKey(k string) bool {
	if k != "ctrl+shift+d" {
		return false
	}
	m.debug.level = (m.debug.level + 1) % debugDepths
	return true
}

// debugOver writes the bar across the top of whatever was drawn.
//
// Over rather than above: rows of its own would move everything under them, and
// a picture looked at with the debug on would not be the picture drawn with it
// off. The block costs the top few rows of the screen, which is the price of
// reading the numbers off the running interface rather than off a log.
func (m Model) debugOver(screen string) string {
	rows := m.debugRows()
	if len(rows) == 0 {
		return screen
	}

	lines := strings.Split(screen, "\n")
	for i, row := range rows {
		if i >= len(lines) {
			break
		}
		// Reset either side: the row underneath was drawn in whatever the
		// picture is wearing, and half of a style left open runs on.
		lines[i] = "\x1b[0m" + row + "\x1b[0m"
	}
	return strings.Join(lines, "\n")
}

// debugGroup is one subject's worth of readings: the rows of the block, and the
// fields of a line in the file. One source for both, so what is read off the
// disk afterwards is what was on the screen at the time.
type debugGroup struct {
	name   string
	fields []debugField
}

// debugGroups is the whole block, in the order it is drawn.
func (m Model) debugGroups() []debugGroup {
	return []debugGroup{
		{"scr", m.debugSelf()},
		{"trk", m.debugTrack()},
		{"snd", m.debugSound()},
		{"bt", m.debugBeat()},
		{"jn", m.debugJoins()},
		{"wrd", m.debugWords()},
		{"lyr", m.debugSheet()},
		{"fig", m.debugFigure()},
	}
}

// debugRows is the bar itself: one line, the block, or nothing at all.
func (m Model) debugRows() []string {
	if m.debug.level == debugOff || m.width < 40 || m.height < 2 {
		return nil
	}
	if m.debug.level == debugTerse {
		return []string{m.debugFit("", m.debugQuick())}
	}

	var rows []string
	for _, g := range m.debugGroups() {
		rows = append(rows, m.debugFit(g.name, g.fields))
	}
	// Never the whole screen. A block taller than the terminal is a block with
	// nothing left to debug underneath it.
	return rows[:min(len(rows), max(m.height-4, 1))]
}

// debugQuick is the one line: what it is doing, in the order it is asked.
func (m Model) debugQuick() []debugField {
	var b debugPad
	b.put("", "%s/%s", debugScreen(m), debugMode(m.frameMode()))
	if m.ps != nil {
		b.put("", "%s %s/%s", debugShort(m.ps.Title, 14), debugClock(m.elapsed()), debugClock(m.ps.Duration))
	} else {
		b.put("", "no track")
	}
	b.add(debugBPM(m))
	b.add(debugDeal(m))
	b.put("jn", "%d %s", m.joinsTurns(), debugAgo(m.elapsed()-m.joinsAt()))
	b.put("swl", "%.2f %.0fdB", m.swell(), m.scope.beat.Loud)

	t := slowNow()
	b.put("fps", "%.0f", t.fps)
	if t.missed > 0 {
		b.put("late", "%d", t.missed)
	}
	return b.out
}

// debugSelf is this program: where it is, how big, and how fast it is going
// round. The frame timing is the half of it nothing else can report — see
// slow.go.
func (m Model) debugSelf() []debugField {
	var b debugPad
	b.put("", "%s/%s", debugScreen(m), debugMode(m.frameMode()))
	b.put("", "%dx%d", m.width, m.height)
	b.put("", "%s", map[bool]string{true: "dark", false: "light"}[m.isDark])
	b.put("tab", "%s", m.tab)
	// Which build is drawing this. A picture that is wrong and a build old
	// enough to still draw it that way are indistinguishable by eye, and an
	// hour has gone on that already.
	b.put("", "%s", build.Version())

	t := slowNow()
	b.put("fps", "%.1f", t.fps)
	b.put("late", "%d/%d", t.missed, t.frames)
	b.put("worst", "%s", debugMs(t.worst))
	b.put("upd", "%s", debugMs(t.update))
	b.put("drw", "%s", debugMs(t.render))
	b.put("api", "%s", debugMs(t.asked))
	return b.out
}

// debugTrack is what the daemon last said about the record, including the two
// things only it can say: the stream it chose and the tempo it measured.
func (m Model) debugTrack() []debugField {
	var b debugPad
	if m.ps == nil {
		b.put("", "no state")
		return b.out
	}

	b.put("", "%s", debugShort(m.ps.Title, 20))
	b.put("", "%s", debugShort(strings.Join(m.ps.Artists, ", "), 16))
	b.put("", "%s/%s", debugClock(m.elapsed()), debugClock(m.ps.Duration))
	b.put("", "%s", map[bool]string{true: "play", false: "paused"}[m.ps.Playing])
	b.put("vol", "%d", m.ps.Volume)
	if m.ps.Shuffle {
		b.put("", "shuf")
	}
	if m.ps.Repeat != "" && m.ps.Repeat != "off" {
		b.put("rep", "%s", m.ps.Repeat)
	}
	if m.ps.Bitrate > 0 {
		b.put("", "%dk", m.ps.Bitrate)
	}
	if m.ps.Tempo > 0 {
		b.put("meas", "%.0f", m.ps.Tempo)
	}
	b.put("id", "%s", debugShort(m.ps.TrackID, 8))
	b.put("dev", "%s", debugShort(m.ps.DeviceName, 10))
	if m.ps.Unplayable != "" {
		b.put("refused", "%s", debugShort(m.ps.Unplayable, 8))
	}
	return b.out
}

// debugSound is the measurement itself, and the two ranges everything drawn from
// it is read against: how loud the record has been lately, and how hard the low
// end is hitting.
func (m Model) debugSound() []debugField {
	var b debugPad
	b.put("loud", "%.1fdB", m.scope.beat.Loud)
	b.put("swl", "%.2f", m.swell())
	b.put("rng", "%.0f..%.0f", m.words.swellLow, m.words.swellHigh)
	b.put("env", "%.2f", m.scope.envelope)
	b.put("drive", "%.2f", m.words.drive)
	if at, ok := m.swayAt(); ok {
		b.put("sway", "%.1fb", at)
	} else {
		b.put("sway", "-")
	}
	b.put("bands", "%d", len(m.scope.bands))
	b.put("notes", "%s", debugNotes(m.scope.beat.Notes))
	return b.out
}

// debugBeat is the beat as the picture has it: not what the daemon reported but
// what that report has been carried forward to, which is what everything on
// screen actually moves on.
func (m Model) debugBeat() []debugField {
	var b debugPad
	beat := m.scope.beat
	if !beat.Found() {
		b.put("", "none")
	} else {
		b.put("", "%.0fbpm", 60000/float64(beat.Period.Milliseconds()+1))
		b.put("per", "%s", debugMs(beat.Period))
		b.put("since", "%s", debugMs(beat.Since))
	}
	if phase, ok := m.beatPhase(); ok {
		b.put("phase", "%.2f", phase)
	} else {
		b.put("phase", "-")
	}
	b.put("pulse", "%.2f", m.beatPulse())
	b.put("", "%s", map[bool]string{true: "kept", false: "loose"}[m.beatKeeping()])
	if !m.stage.loose {
		b.put("", "key off")
	}
	if !m.scope.beatAt.IsZero() {
		b.put("age", "%s", debugMs(time.Since(m.scope.beatAt)))
	}
	return b.out
}

// debugJoins is where the record turns over, with the line it has to cross
// printed beside the reading. See joins.go.
func (m Model) debugJoins() []debugField {
	var b debugPad
	j := m.joins
	b.put("turns", "%d", m.joinsTurns())
	b.put("at", "%s", debugClock(m.joinsAt()))
	b.put("ago", "%s", debugAgo(m.elapsed()-m.joinsAt()))
	b.put("nov", "%.3f", j.nov)
	b.put("typ", "%.3f", j.watch)
	b.put("needs", "%.3f", j.watch*joinEdge)
	b.put("fill", "%d/%d", j.fill, len(j.ring))
	if left := joinApart - (j.heard - j.begins); left > 0 {
		b.put("held", "%s", debugAgo(left))
	}
	switch {
	case m.ps != nil && j.forTrack != m.ps.TrackID:
		// Everything above is then a reading of the record before this one, and
		// the two turns it reports belong to nothing on screen.
		b.put("", "another record")
	case j.heard < joinWarm:
		b.put("", "warming")
	}
	return b.out
}

// debugWords is the deal: which line is up, how it came in, how far through
// coming in it is, and what the colour is doing to it. The row this whole bar
// was built for.
func (m Model) debugWords() []debugField {
	var b debugPad
	w := m.words
	if w.text == "" {
		b.put("", "nothing set")
		return b.out
	}

	kind := "line"
	switch {
	case w.telling:
		kind = "card"
	case w.beats:
		kind = "marks"
	}
	b.put("", "%s", kind)
	// Which company is up, and whether the room is silenced — the two readings
	// that answer "why are those the marks on screen". Empty is the notes the
	// face carries, which is the deal's own empty hand.
	if w.cast == "" {
		b.put("cast", "notes")
	} else {
		b.put("cast", "%s", w.cast)
	}
	if m.muted() {
		b.put("", "MUTED")
	}
	b.add(debugDeal(m))
	b.put("leave", "%s", debugMoveName(w.leave))
	b.put("sung", "%s", debugClock(time.Duration(w.starts)*time.Millisecond))
	b.put("ends", "%s", debugClock(time.Duration(w.ends)*time.Millisecond))
	b.put("words", "%d", w.where.Count)
	b.put("dots", "%dx%d", w.have.DotsX, w.have.DotsY)
	if w.cast != "" {
		b.put("cast", "%s", w.cast)
	}
	b.put("roll", "%.2f→%.2f", w.roll, m.wordsRollAt())
	hue, level := m.wordsColourNow()
	b.put("col", "%d/%d", hue, level)
	b.put("band", "%.1f/%.1f", w.band, w.head)
	if w.beats {
		// How many of the row are facing the other way, and whether any of them
		// may: a set without a front never turns, and neither does a row with no
		// beat to turn on. See wordsTurning.
		if turned := m.wordsTurning(w.where.Count); turned != nil {
			var n int
			for _, one := range turned {
				if one {
					n++
				}
			}
			b.put("turn", "%d/%d", n, len(turned))
		} else {
			b.put("turn", "-")
		}
	}
	if n := len(m.scope.sparks); n > 0 {
		b.put("sparks", "%d", n)
	}
	if w.asked != "" && w.asked != w.text {
		b.put("asked", "%s", debugShort(w.asked, 10))
	}
	return b.out
}

// debugSheet is what the lyric sheet says, which is half of what goes wrong on
// the screen it feeds.
func (m Model) debugSheet() []debugField {
	var b debugPad
	switch {
	case m.ps == nil:
		b.put("", "no track")
	case m.lyrics.forTrack != m.ps.TrackID:
		b.put("", "waiting")
	case m.lyrics.missing:
		b.put("", "none")
	case !m.lyrics.synced:
		b.put("", "untimed")
	default:
		b.put("", "synced")
	}
	b.put("lines", "%d", len(m.lyrics.lines))
	if m.lyrics.synced {
		b.put("at", "%d", m.wordsAt())
	}
	if m.lyrics.fetching != "" {
		b.put("fetch", "%s", debugShort(m.lyrics.fetching, 8))
	}
	b.put("screen", "%s", map[bool]string{true: "on", false: "off"}[m.lyrics.on])
	if m.wordsSilent() {
		b.put("", "silent")
	}
	spell, every := m.wordsSpells()
	b.put("spell", "%d/%s", spell, debugAgo(every))
	return b.out
}

// debugFigure is who is on, what they are in the middle of, and the colour of
// the record coming after this one.
func (m Model) debugFigure() []debugField {
	var b debugPad
	if m.faceUp() {
		b.put("", "up")
		b.put("who", "%s", debugShort(m.faceWho(), 12))
		b.put("doing", "%s", debugDoingName(m.face.doing))
		if m.face.act != "" {
			b.put("act", "%s", m.face.act)
		}
		b.put("walk", "%.2f", m.faceWalk())
		b.put("turns", "%d", m.face.turns)
		b.put("mouth", "%.2f", m.face.mouth)
	} else {
		b.put("", "off")
	}
	if at, ok := m.tideAt(); ok {
		b.put("tide", "%.2f", at)
	} else {
		b.put("tide", "-")
	}
	if next := m.tideComing(); next != "" {
		b.put("next", "%s", debugShort(next, 8))
	}
	if m.cover.hasAccent {
		r, g, bl, _ := m.cover.accent.RGBA()
		b.put("accent", "#%02x%02x%02x", r>>8, g>>8, bl>>8)
	}
	return b.out
}

// debugNote writes the block down as well as drawing it.
//
// The bar is on the screen, and the screen is in front of whoever is listening —
// which leaves everybody else in the conversation, this program's author
// included, being told what the numbers said rather than reading them. So while
// the bar is up the same readings go to a file, in plain text, and the record of
// what the screen was deciding from can be read back afterwards.
//
// Not every frame. A line goes down when something that matters has changed —
// the record, the deal, a join, which line of the sheet is being sung — and once
// a second regardless, so a stretch where nothing changes is still visible as a
// stretch rather than as a gap. Thirty lines a second would bury the change that
// is being looked for in the frames either side of it.
//
// It lives only as long as the session does. See ForgetDebug.
func (m Model) debugNote() {
	if m.debug.level == debugOff {
		return
	}

	debugPen.mu.Lock()
	defer debugPen.mu.Unlock()
	if debugPen.off {
		return
	}

	// What counts as something having happened. The deal is in it because that
	// is the reading this whole bar was built for; the rest is anything that
	// changes which picture is on screen.
	why := "tick"
	key := fmt.Sprintf("%s|%d|%d|%d|%v|%s|%d", m.words.forTrack, m.words.move, m.words.leave,
		m.joinsTurns(), m.words.beats, debugScreen(m), m.wordsAt())
	switch {
	case key != debugPen.was:
		why = "change"
	case time.Since(debugPen.at) < time.Second:
		return
	}
	debugPen.was, debugPen.at = key, time.Now()

	said := map[string]string{}
	for _, g := range m.debugGroups() {
		said[g.name] = debugPlain(g.fields)
	}
	raw, err := json.Marshal(struct {
		At     string `json:"at"`
		Clock  string `json:"clock"`
		Why    string `json:"why"`
		Screen string `json:"screen"`
		Track  string `json:"track"`
		Sound  string `json:"sound"`
		Beat   string `json:"beat"`
		Joins  string `json:"joins"`
		Words  string `json:"words"`
		Sheet  string `json:"sheet"`
		Figure string `json:"figure"`
	}{
		At:     time.Now().Format(time.RFC3339Nano),
		Clock:  debugClock(m.elapsed()),
		Why:    why,
		Screen: said["scr"],
		Track:  said["trk"],
		Sound:  said["snd"],
		Beat:   said["bt"],
		Joins:  said["jn"],
		Words:  said["wrd"],
		Sheet:  said["lyr"],
		Figure: said["fig"],
	})
	if err != nil {
		return
	}

	path, err := debugPath()
	if err != nil {
		debugPen.off = true
		return
	}
	// The first line of a run starts the file again. A session's worth is what
	// anybody wants to read; two sessions interleaved is what nobody does.
	flags := os.O_APPEND | os.O_CREATE | os.O_WRONLY
	if !debugPen.began {
		flags = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
		debugPen.began = true
	}
	f, err := os.OpenFile(path, flags, 0o600)
	if err != nil {
		debugPen.off = true
		return
	}
	defer f.Close()
	_, _ = f.Write(append(raw, '\n'))
}

// debugPen is what the writing above remembers between frames.
var debugPen struct {
	mu    sync.Mutex
	at    time.Time
	was   string
	began bool
	off   bool
}

func debugPath() (string, error) {
	dir, err := xdg.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, debugFile), nil
}

// ForgetDebug removes what the bar wrote down.
//
// Called as spindle closes. The file is for the session it was written in and
// for nothing else: it is a page of working, not a log, and a page of working
// left lying about is read months later as though it still meant something.
func ForgetDebug() {
	debugPen.mu.Lock()
	defer debugPen.mu.Unlock()

	debugPen.began, debugPen.was = false, ""
	path, err := debugPath()
	if err != nil {
		return
	}
	_ = os.Remove(path)
}

// debugPlain is a row as the file wants it: no colour, nothing cut off.
func debugPlain(fields []debugField) string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, f.plain())
	}
	return strings.Join(out, "  ")
}

// debugField is one reading: a dim tag and a value, or a bare value where the
// tag would only repeat what the value says.
//
// The two are kept apart rather than pasted together at the point they are
// written, because the same reading goes two places — the screen, in colour, and
// the file, in plain text. Styled at the point of writing, the file would be
// full of escape sequences.
type debugField struct{ tag, val string }

func (f debugField) plain() string {
	if f.tag == "" {
		return f.val
	}
	return f.tag + " " + f.val
}

func (f debugField) styled() string {
	if f.tag == "" {
		return f.val
	}
	return debugTag(f.tag) + " " + f.val
}

// debugPad collects one row's fields.
type debugPad struct{ out []debugField }

func (b *debugPad) put(tag, format string, a ...any) {
	b.out = append(b.out, debugField{tag, fmt.Sprintf(format, a...)})
}

func (b *debugPad) add(f debugField) { b.out = append(b.out, f) }

// debugFit lays a row out and cuts it where the terminal ends, a whole field at
// a time — half a number is worse than no number.
func (m Model) debugFit(name string, fields []debugField) string {
	line := ""
	if name != "" {
		line = debugName(m, name)
	}
	for _, field := range fields {
		f := field.styled()
		sep := "  "
		if line == "" {
			sep = ""
		}
		if lipgloss.Width(line+sep+f) > m.width {
			if lipgloss.Width(line)+1 <= m.width {
				line += "…"
			}
			break
		}
		line += sep + f
	}
	if pad := m.width - lipgloss.Width(line); pad > 0 {
		line += strings.Repeat(" ", pad)
	}
	return line
}

// debugName is the group's tag at the head of its row, in the record's own
// accent — the one place on this bar where the colour is doing something useful,
// which is telling eight rows of numbers apart at a glance.
func debugName(m Model, name string) string {
	if m.styles.Accent == nil {
		return debugTag(name)
	}
	return lipgloss.NewStyle().Foreground(m.styles.Accent).Render(name)
}

func debugTag(tag string) string {
	return lipgloss.NewStyle().Faint(true).Render(tag)
}

// debugDeal is how the line on screen came in and how far in it has got, which
// is the reading this bar exists for.
func debugDeal(m Model) debugField {
	gather := float32(1)
	if since := time.Since(m.words.since); since < wordsGather {
		gather = float32(since) / float32(wordsGather)
	}
	return debugField{"move", fmt.Sprintf("%s %.0f%%", debugMoveName(m.words.move), gather*100)}
}

// debugBPM is the beat in the shortest form worth reading.
func debugBPM(m Model) debugField {
	beat := m.scope.beat
	if !beat.Found() {
		return debugField{val: "no beat"}
	}
	kept := "loose"
	if m.beatKeeping() {
		kept = "kept"
	}
	return debugField{val: fmt.Sprintf("%.0fbpm %s", 60000/float64(beat.Period.Milliseconds()+1), kept)}
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

var debugMoveNames = [wordsMoves]string{
	wordsDrifting:   "drift",
	wordsRising:     "rise",
	wordsFalling:    "fall",
	wordsBursting:   "burst",
	wordsSpreading:  "spread",
	wordsWiping:     "wipe",
	wordsWipingBack: "wipeback",
	wordsBlurring:   "blur",
	wordsPopping:    "pop",
}

func debugMoveName(w wordsMove) string {
	if w < 0 || int(w) >= len(debugMoveNames) {
		return fmt.Sprintf("?%d", int(w))
	}
	return fmt.Sprintf("%d:%s", int(w), debugMoveNames[w])
}

var debugDoingNames = [faceDoings]string{
	faceStill:    "still",
	faceBlinking: "blink",
	faceWinking:  "wink",
	faceBrowing:  "brow",
	faceGaping:   "gape",
	faceLooking:  "look",
	faceGrinning: "grin",
	faceWaving:   "wave",
	faceSinging:  "sing",
	faceKissing:  "kiss",
	faceStunned:  "stun",
	faceNodding:  "nod",
}

func debugDoingName(d faceDoing) string {
	if d < 0 || int(d) >= len(debugDoingNames) {
		return fmt.Sprintf("?%d", int(d))
	}
	return debugDoingNames[d]
}

// debugNotes names the loudest of the twelve, which is the only part of a chroma
// reading a bar has room for.
func debugNotes(notes []float32) string {
	if len(notes) < 12 {
		return "-"
	}
	names := [12]string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	top, at := float32(0), -1
	for i, v := range notes {
		if v > top {
			top, at = v, i
		}
	}
	if at < 0 {
		return "-"
	}
	return fmt.Sprintf("%s %.2f", names[at], top)
}

func debugClock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	return fmt.Sprintf("%d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}

// debugAgo is a stretch in the largest unit that still says something.
func debugAgo(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	return debugClock(d)
}

func debugMs(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000)
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}

func debugShort(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}
