package ui

import (
	"encoding/json"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/xdg"
)

// prefsFile is where the screens' switches are kept. It is state rather than
// configuration: it is written by pressing keys, not by editing it, and nobody
// would miss it if it were deleted.
const prefsFile = "view.json"

// prefs is what the screens remember between runs. The visualiser is recorded
// per tab because each screen keeps its own; the words and the glance at what
// is next belong to the player alone.
type prefs struct {
	Scope  []scopeMode `json:"scope"`
	Lyrics bool        `json:"lyrics"`
	Peek   bool        `json:"peek"`
}

// prefsMsg carries the file's contents back into the model.
type prefsMsg prefs

func prefsPath() (string, error) {
	dir, err := xdg.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, prefsFile), nil
}

// loadPrefsCmd reads the switches back. A missing or unreadable file is not an
// error worth showing: the screen simply comes up the way it does for someone
// running spindle for the first time.
func loadPrefsCmd() tea.Cmd {
	return func() tea.Msg {
		path, err := prefsPath()
		if err != nil {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var p prefs
		if err := json.Unmarshal(data, &p); err != nil {
			return nil
		}
		return prefsMsg(p)
	}
}

// savePrefs records the switches as they now stand. It runs off the update loop
// because it touches the disk, and reports nothing: a preference that could not
// be written is not worth interrupting the music for.
func (m Model) savePrefs() tea.Cmd {
	p := prefs{
		Scope:  append([]scopeMode(nil), m.scope.modes[:]...),
		Lyrics: m.lyrics.on,
		Peek:   m.peek.on,
	}
	return func() tea.Msg {
		path, err := prefsPath()
		if err != nil {
			return nil
		}
		data, err := json.Marshal(p)
		if err != nil {
			return nil
		}
		_ = os.WriteFile(path, data, 0o600)
		return nil
	}
}

// applyPrefs puts a loaded file into effect. Anything the file does not cover,
// including a mode from a future version, is left at its default.
func (m *Model) applyPrefs(p prefs) {
	for i, mode := range p.Scope {
		if i < len(m.scope.modes) && mode >= scopeOff && mode < scopeModes {
			m.scope.modes[i] = mode
		}
	}
	m.lyrics.on = p.Lyrics
	m.peek.on = p.Peek
}
