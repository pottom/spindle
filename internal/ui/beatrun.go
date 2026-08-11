package ui

import "time"

// Counting the record's beats, forwards.
//
// Everything that moves on the beat needs to know how many have gone by. The
// obvious way is to divide the time elapsed by the beat's period, and it is
// wrong for a reason that only shows up on a long record: the period is a
// measurement, it moves, and dividing by the newest one renumbers every beat
// that has already happened.
//
// Measured on one three minute record, 337 readings from the running screen:
//
//   - the period wandered between 348 and 352 ms, which is 1.1% — enough on its
//     own to move the count by six beats by the end of the track
//   - and six times it flipped octave, reading 86 bpm for a stretch where it
//     had been reading 171. The music did not halve; a beat finder cannot tell
//     a beat from every other beat, and this is its classic mistake. Each flip
//     moved the count by up to two hundred beats at once.
//
// Marks turn every two to six beats, so a jump of two hundred re-rolls every
// one of them in the same frame. What that looks like is not a faster or slower
// dance — it is the row rearranging itself for no reason, twice a minute, which
// is exactly what "they move like robots" turned out to mean.
//
// So the count is kept rather than derived. Each frame the phase of the current
// beat is looked at, and when it wraps past the top the count goes up by one.
// A period that changes then changes what happens next and nothing that has
// already happened.

// beatRun is the running count of the record's beats and where in the current
// one the screen is.
type beatRun struct {
	// count is how many beats have gone by since the record started, and phase
	// where the last frame found itself inside the current one.
	count int
	phase float32

	// forTrack is the record being counted, so a new one starts at nought
	// rather than carrying somebody else's total.
	forTrack string
}

// beatRunFlow moves the count on by however many beats have landed since the
// last frame.
func (m *Model) beatRunFlow() {
	track := ""
	if m.ps != nil {
		track = m.ps.TrackID
	}
	if m.run.forTrack != track {
		m.run = beatRun{forTrack: track}
	}

	phase, ok := m.beatPhase()
	if !ok {
		// Nothing to count. The phase is left where it was so that a beat
		// coming back does not read as a wrap.
		return
	}

	// A wrap is the phase falling rather than rising. Between two frames at
	// sixty a second the phase moves about a twentieth of a beat, so a fall is
	// unambiguous — and if frames were missed and a whole beat went by
	// unseen, one is still better than none: the count is what the row dances
	// to, not what the record is billed at.
	if phase < m.run.phase {
		m.run.count++
	}
	m.run.phase = phase
}

// beatOctave is how close a new reading has to be to half or double the one
// being kept before it is taken for the same beat counted the other way.
//
// A twelfth. The music does not halve its tempo and then double it back six
// times in three minutes; a beat finder that cannot tell a beat from every
// other beat does, and it did — measured, 42 readings out of 337 came back at
// 86 bpm on a record the other 295 called 171. A twelfth is wide enough to
// catch every one of those and far narrower than any real change of tempo,
// which does not arrive as an exact factor of two.
const beatOctave = 1.0 / 12

// beatSteady is the reading to keep, given the one held and the one that has
// just arrived.
//
// The new one, unless it is the same beat counted the other way — then the held
// one stays, because a picture that halves its dancing for eight seconds and
// doubles it back is not answering anything in the music.
func beatSteady(held, fresh time.Duration) time.Duration {
	if held <= 0 || fresh <= 0 {
		return fresh
	}
	near := func(a, b time.Duration) bool {
		d := float64(a-b) / float64(b)
		return d > -beatOctave && d < beatOctave
	}
	if near(fresh, held*2) || near(fresh*2, held) {
		return held
	}
	return fresh
}

// beatsRun is how many beats have gone by, and whether there is a count worth
// having. It replaces dividing the elapsed time by the period — see the top of
// this file for what that cost.
func (m Model) beatsRun() (int, bool) {
	if !m.beatKeeping() {
		return 0, false
	}
	if m.ps == nil || m.run.forTrack != m.ps.TrackID {
		return 0, false
	}
	return m.run.count, true
}
