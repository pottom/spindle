package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pottom/spindle/internal/xdg"
)

// Keeping what every lyric sheet says about itself, so that the one number left
// guessed can be read off a pile of records rather than off one.
//
// What is guessed is how long a line is sung for. A sheet says when each line
// starts and nothing else; the model takes 85% of the way to the next line, at
// most three seconds, and both numbers were fitted by ear on three sections of
// two records. A fourth turned up where that is half a second too long by the
// end of every line — fifteen lines in under three minutes, so every window is
// long, the share never binds, and the ceiling is the whole of the answer.
//
// The ceiling cannot be a constant: the same measurement says the ballad's lines
// really are sung for three and a half seconds. What separates the two is what a
// line has in it, and the sheet knows that — it is the only thing about the
// singing the sheet does know for certain.
//
// So every sheet that arrives writes down its own numbers: where each line
// starts, how many syllables it carries, and how many words. A record played for
// a few seconds leaves its sheet here, and a rule can be fitted across all of
// them and checked against the ones that were tapped by ear.
//
// One line per record, written once when the sheet lands. Nothing is sent
// anywhere: this is the same state directory the frame recording uses.
const sheetsFile = "sheets.jsonl"

// sheetsSeen is the records whose sheets are already down, so a re-fetch does
// not write the same sheet twice in a sitting.
var sheetsSeen = map[string]bool{}

// sheetLine is one line of a sheet: when it starts, and what it has to sing.
type sheetLine struct {
	At    int64 `json:"at_ms"`
	Syl   int   `json:"syl"`
	Words int   `json:"words"`
}

// keepSheet writes a sheet's own numbers down, if it is timed and new.
func (m Model) keepSheet() {
	if !m.lyrics.synced || len(m.lyrics.lines) == 0 || m.ps == nil {
		return
	}
	if sheetsSeen[m.ps.TrackID] {
		return
	}
	sheetsSeen[m.ps.TrackID] = true

	lines := make([]sheetLine, 0, len(m.lyrics.lines))
	for _, line := range m.lyrics.lines {
		lines = append(lines, sheetLine{
			At:    line.At,
			Syl:   lyricsSyllables(line.Words, m.lyrics.language),
			Words: len(wordsPieces(line.Words)),
		})
	}

	raw, err := json.Marshal(struct {
		At       string      `json:"at"`
		Track    string      `json:"track"`
		Name     string      `json:"name"`
		Artist   string      `json:"artist"`
		Length   int64       `json:"length_ms"`
		Language string      `json:"language"`
		Lines    []sheetLine `json:"lines"`
	}{
		At:       time.Now().Format(time.RFC3339),
		Track:    m.ps.TrackID,
		Name:     m.ps.Title,
		Artist:   strings.Join(m.ps.Artists, ", "),
		Length:   m.ps.Duration.Milliseconds(),
		Language: m.lyrics.language,
		Lines:    lines,
	})
	if err != nil {
		return
	}

	dir, err := xdg.StateDir()
	if err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, sheetsFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(raw, '\n'))
}
