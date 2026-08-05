package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// A daemon playing something, and one holding nothing, as the real one reports
// them. The volume counts in 64 steps to prove the rescaling: half of 64 is not
// half of 100.
const (
	playingStatus = `{"device_name":"spindle","stopped":false,"paused":false,"volume":32,"volume_steps":64,` +
		`"track":{"name":"Sultans of Swing","artist_names":["Dire Straits"],"album_name":"Dire Straits",` +
		`"position":94000,"duration":348000}}`
	pausedStatus = `{"device_name":"spindle","stopped":false,"paused":true,"volume":32,"volume_steps":64,` +
		`"track":{"name":"Sultans of Swing","artist_names":["Dire Straits"],"album_name":"Dire Straits",` +
		`"position":94000,"duration":348000}}`
	stoppedStatus = `{"device_name":"spindle","stopped":true,"volume":32,"volume_steps":64}`
	sampleQueue   = `{"current":{"name":"Sultans of Swing","artist_names":["Dire Straits"],"duration":348000},` +
		`"tracks":[{"name":"Romeo and Juliet","artist_names":["Dire Straits"],"duration":360000}]}`
	emptyQueue = `{"current":null,"tracks":[]}`
)

// sent is one command the stub daemon was given.
type sent struct {
	path string
	body map[string]any
}

// stub answers the way the daemon's local API does, and remembers what it was
// asked to do.
type stub struct {
	url    string
	status string
	queue  string
	got    []sent
}

func newStub(t *testing.T, status, queue string) *stub {
	t.Helper()
	s := &stub{status: status, queue: queue}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			switch r.URL.Path {
			case "/status":
				io.WriteString(w, s.status) //nolint:errcheck // test server
			case "/player/queue":
				io.WriteString(w, s.queue) //nolint:errcheck // test server
			default:
				w.WriteHeader(http.StatusNotFound)
			}
			return
		}

		command := sent{path: r.URL.Path}
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			json.Unmarshal(raw, &command.body) //nolint:errcheck // absence is the assertion
		}
		s.got = append(s.got, command)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	s.url = server.URL
	return s
}

// run drives one command line against the stub.
func (s *stub) run(name string, args ...string) (code int, out, errOut string) {
	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	c := &cli{remote: newRemote(s.url), out: stdout, errOut: stderr}
	code = c.run(context.Background(), name, args)
	return code, stdout.String(), stderr.String()
}

func TestParseSeek(t *testing.T) {
	for _, c := range []struct {
		arg      string
		want     time.Duration
		relative bool
		fails    bool
	}{
		{arg: "90", want: 90 * time.Second},
		{arg: "0"},
		{arg: "+30", want: 30 * time.Second, relative: true},
		{arg: "-15", want: -15 * time.Second, relative: true},
		{arg: "1:30", fails: true},
		{arg: "30s", fails: true},
		{arg: "", fails: true},
	} {
		got, relative, err := parseSeek(c.arg)
		if c.fails {
			if err == nil {
				t.Errorf("parseSeek(%q) = %v, want an error", c.arg, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSeek(%q): %v", c.arg, err)
			continue
		}
		if got != c.want || relative != c.relative {
			t.Errorf("parseSeek(%q) = %v, relative %v; want %v, relative %v", c.arg, got, relative, c.want, c.relative)
		}
	}
}

func TestParseVolume(t *testing.T) {
	for _, c := range []struct {
		arg   string
		want  int
		fails bool
	}{
		{arg: "0"},
		{arg: "60", want: 60},
		{arg: "100", want: 100},
		{arg: "101", fails: true},
		{arg: "-1", fails: true},
		{arg: "half", fails: true},
	} {
		got, err := parseVolume(c.arg)
		if c.fails {
			if err == nil {
				t.Errorf("parseVolume(%q) = %d, want an error", c.arg, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("parseVolume(%q) = %d, %v; want %d", c.arg, got, err, c.want)
		}
	}
}

func TestTakeJSONFlagAnywhere(t *testing.T) {
	args, found := takeJSONFlag([]string{"+30", jsonFlag})
	if !found || len(args) != 1 || args[0] != "+30" {
		t.Errorf("takeJSONFlag = %v, %v; want [+30], true", args, found)
	}

	if args, found := takeJSONFlag([]string{"60"}); found || len(args) != 1 {
		t.Errorf("takeJSONFlag = %v, %v; want [60], false", args, found)
	}
}

// A daemon that is not there is its own exit code: a script can start one.
func TestNoDaemonExitsThree(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	c := &cli{remote: newRemote(url), out: stdout, errOut: stderr}

	if code := c.run(context.Background(), "status", nil); code != exitNoDaemon {
		t.Errorf("exit = %d, want %d", code, exitNoDaemon)
	}
	if stdout.String() != "" {
		t.Errorf("stdout = %q, want nothing", stdout)
	}
	if !strings.Contains(stderr.String(), "no daemon") {
		t.Errorf("stderr = %q, want it to mention the missing daemon", stderr)
	}
}

// Nothing playing is not a failure, and not the same failure as no daemon.
func TestIdleExitsFour(t *testing.T) {
	for _, name := range []string{"status", "queue", "play", "pause", "toggle", "next", "seek"} {
		s := newStub(t, stoppedStatus, emptyQueue)

		args := []string{}
		if name == "seek" {
			args = []string{"+30"}
		}

		code, out, errOut := s.run(name, args...)
		if code != exitIdle {
			t.Errorf("%s exit = %d, want %d", name, code, exitIdle)
		}
		if out != "" {
			t.Errorf("%s stdout = %q, want nothing", name, out)
		}
		if !strings.Contains(errOut, "nothing is playing") {
			t.Errorf("%s stderr = %q, want it to say nothing is playing", name, errOut)
		}
		if len(s.got) != 0 {
			t.Errorf("%s sent %v to a stopped daemon", name, s.got)
		}
	}
}

func TestPlayAndPause(t *testing.T) {
	for _, c := range []struct{ name, path, printed string }{
		{"play", "/player/resume", "playing"},
		{"pause", "/player/pause", "paused"},
		{"next", "/player/next", "next"},
		{"prev", "/player/prev", "previous"},
	} {
		s := newStub(t, playingStatus, sampleQueue)

		code, out, errOut := s.run(c.name)
		if code != exitOK {
			t.Errorf("%s exit = %d (%s)", c.name, code, errOut)
		}
		if out != c.printed+"\n" {
			t.Errorf("%s printed %q, want %q", c.name, out, c.printed)
		}
		if len(s.got) != 1 || s.got[0].path != c.path {
			t.Errorf("%s sent %v, want a single %s", c.name, s.got, c.path)
		}
	}
}

// Toggle has to say which way it went, which means reading the state first.
func TestToggleFollowsTheState(t *testing.T) {
	s := newStub(t, pausedStatus, sampleQueue)
	if code, out, _ := s.run("toggle"); code != exitOK || out != "playing\n" {
		t.Errorf("toggle on a paused daemon = %d, %q; want 0, \"playing\"", code, out)
	}
	if len(s.got) != 1 || s.got[0].path != "/player/resume" {
		t.Errorf("toggle sent %v, want /player/resume", s.got)
	}

	s = newStub(t, playingStatus, sampleQueue)
	if code, out, _ := s.run("toggle"); code != exitOK || out != "paused\n" {
		t.Errorf("toggle on a playing daemon = %d, %q; want 0, \"paused\"", code, out)
	}
	if len(s.got) != 1 || s.got[0].path != "/player/pause" {
		t.Errorf("toggle sent %v, want /player/pause", s.got)
	}
}

// The daemon counts volume in steps of its own, so a percentage has to be
// rescaled on the way in and back on the way out.
func TestVolumeRescales(t *testing.T) {
	s := newStub(t, playingStatus, sampleQueue)
	if code, out, _ := s.run("volume"); code != exitOK || out != "50\n" {
		t.Errorf("volume = %d, %q; want 0, \"50\"", code, out)
	}

	s = newStub(t, playingStatus, sampleQueue)
	if code, out, _ := s.run("volume", "25"); code != exitOK || out != "25\n" {
		t.Errorf("volume 25 = %d, %q; want 0, \"25\"", code, out)
	}
	if len(s.got) != 1 || s.got[0].body["volume"] != float64(16) {
		t.Errorf("volume 25 sent %v, want 16 of 64 steps", s.got)
	}
}

// The level belongs to the device, so it can be read and set with nothing
// loaded — unlike every other command here.
func TestVolumeWorksWhileStopped(t *testing.T) {
	s := newStub(t, stoppedStatus, emptyQueue)

	if code, out, errOut := s.run("volume", "75"); code != exitOK || out != "75\n" {
		t.Errorf("volume 75 = %d, %q (%s); want 0, \"75\"", code, out, errOut)
	}
	if len(s.got) != 1 || s.got[0].body["volume"] != float64(48) {
		t.Errorf("volume 75 sent %v, want 48 of 64 steps", s.got)
	}
}

func TestVolumeRefusesOutOfRange(t *testing.T) {
	s := newStub(t, playingStatus, sampleQueue)

	code, out, errOut := s.run("volume", "120")
	if code != exitFailed {
		t.Errorf("exit = %d, want %d", code, exitFailed)
	}
	if out != "" || !strings.Contains(errOut, "0 to 100") {
		t.Errorf("printed %q / %q, want a complaint about the range", out, errOut)
	}
	if len(s.got) != 0 {
		t.Errorf("sent %v, want nothing", s.got)
	}
}

func TestSeekRelativeAndAbsolute(t *testing.T) {
	// The playhead is at 1:34 of 5:48.
	for _, c := range []struct {
		arg      string
		position float64
		relative bool
		printed  string
	}{
		{arg: "90", position: 90000, printed: "1:30"},
		{arg: "+30", position: 30000, relative: true, printed: "2:04"},
		{arg: "-30", position: -30000, relative: true, printed: "1:04"},
		// Past the end lands at the end: the daemon clamps, and so does what
		// gets printed about it.
		{arg: "+600", position: 600000, relative: true, printed: "5:48"},
	} {
		s := newStub(t, playingStatus, sampleQueue)

		code, out, errOut := s.run("seek", c.arg)
		if code != exitOK {
			t.Errorf("seek %s exit = %d (%s)", c.arg, code, errOut)
		}
		if out != c.printed+"\n" {
			t.Errorf("seek %s printed %q, want %q", c.arg, out, c.printed)
		}
		if len(s.got) != 1 || s.got[0].path != "/player/seek" {
			t.Fatalf("seek %s sent %v, want /player/seek", c.arg, s.got)
		}
		if s.got[0].body["position"] != c.position || s.got[0].body["relative"] != c.relative {
			t.Errorf("seek %s sent %v, want position %v relative %v",
				c.arg, s.got[0].body, c.position, c.relative)
		}
	}
}

func TestArgumentsAreRefused(t *testing.T) {
	for _, c := range []struct {
		name string
		args []string
	}{
		{"play", []string{"now"}},
		{"status", []string{"--all"}},
		{"seek", nil},
		{"seek", []string{"+30", "+40"}},
		{"volume", []string{"10", "20"}},
	} {
		s := newStub(t, playingStatus, sampleQueue)

		code, out, errOut := s.run(c.name, c.args...)
		if code != exitFailed {
			t.Errorf("%s %v exit = %d, want %d", c.name, c.args, code, exitFailed)
		}
		if out != "" || errOut == "" {
			t.Errorf("%s %v printed %q / %q, want the complaint on stderr", c.name, c.args, out, errOut)
		}
	}
}

// A status bar wants a sentence, not a field per line: without one it has to
// spawn jq or head every time it redraws.
func TestStatusOnOneLine(t *testing.T) {
	s := newStub(t, playingStatus, sampleQueue)

	code, out, _ := s.run("status", "--line")
	if code != exitOK {
		t.Fatalf("exit = %d, want %d", code, exitOK)
	}
	if got := strings.TrimSpace(out); got != "▶ Sultans of Swing — Dire Straits" {
		t.Errorf("status --line printed %q", got)
	}
	if strings.Count(out, "\n") != 1 {
		t.Errorf("status --line printed %d lines, want one", strings.Count(out, "\n"))
	}
}

// The template is the bar's own: braces because its configuration file is
// already full of dollars and percent signs.
func TestStatusFollowsATemplate(t *testing.T) {
	s := newStub(t, pausedStatus, sampleQueue)

	code, out, _ := s.run("status", "--format", "{state} {title} {position}/{duration} {volume}%")
	if code != exitOK {
		t.Fatalf("exit = %d, want %d", code, exitOK)
	}
	if got := strings.TrimSpace(out); got != "paused Sultans of Swing 1:34/5:48 50%" {
		t.Errorf("status --format printed %q", got)
	}
}

// A field nobody knows is left as it was written: a status bar is not where a
// typo should be discovered, and "{titel}" says where to look.
func TestAnUnknownFieldIsLeftAlone(t *testing.T) {
	s := newStub(t, playingStatus, sampleQueue)

	_, out, _ := s.run("status", "--format", "{titel}")
	if got := strings.TrimSpace(out); got != "{titel}" {
		t.Errorf("status --format printed %q, want the field as it was written", got)
	}
}

// Nothing playing prints nothing and says so in the exit code — and says where
// it looked, because these commands reach this machine and nowhere else.
func TestIdleSaysWhereItLooked(t *testing.T) {
	s := newStub(t, stoppedStatus, emptyQueue)

	code, out, errOut := s.run("status", "--line")
	if code != exitIdle {
		t.Errorf("exit = %d, want %d", code, exitIdle)
	}
	if out != "" {
		t.Errorf("status printed %q, want nothing", out)
	}
	if !strings.Contains(errOut, "on this machine") {
		t.Errorf("the explanation reads %q, want it to say where it looked", strings.TrimSpace(errOut))
	}
}

// The command is looked for past any leading flags: "spindle --json status" is
// how everybody writes it the first time.
func TestTheCommandIsFoundPastTheFlags(t *testing.T) {
	name, rest, ok := controlCommandIn([]string{"--json", "status", "--line"})
	if !ok || name != "status" {
		t.Fatalf("controlCommandIn() = %q, %v, want the status command", name, ok)
	}
	if len(rest) != 2 || rest[0] != "--json" || rest[1] != "--line" {
		t.Errorf("arguments = %v, want both flags kept", rest)
	}
}
