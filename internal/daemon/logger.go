package daemon

import (
	"fmt"
	"io"
	"sync"
	"time"

	librespot "github.com/devgianlu/go-librespot"
)

// The daemon's log, and the time in front of every line of it.
//
// Without one the log answers "what happened" and refuses "when" — and every
// question worth asking a log is about when. An afternoon went on guessing
// whether a refusal in this file was from an hour ago or from a week ago, over a
// file covering thirteen days: the answer was in the file and unreadable.
//
// The time to the second and no date, because the file is read by somebody who
// was there and the date is the same for pages at a time. When it does change —
// a daemon left running overnight, which is the normal way this one lives — the
// new day gets a line of its own rather than a longer stamp on every line after
// it.

// stamp is the time in front of a line, and dayStamp the date on the line that
// announces a new one.
const (
	// Stamp is exported for the one line written before there is a logger to
	// write it. See cmd/spindle.
	Stamp    = "15:04:05"
	dayStamp = "2006-01-02, Monday"
)

// sink is the file, and what is known about what has already been written to
// it. Shared by every logger derived from the first, because the day is a fact
// about the file rather than about whoever is writing.
type sink struct {
	mu  sync.Mutex
	out io.Writer
	day string

	// notice is shown every line as it is written and may answer with a line of
	// its own. It is how the one thing go-librespot writes for a person to read
	// reaches one — see signin.go. Nil means nobody is reading.
	notice func(text string) (string, bool)

	// now is the clock, so a test can hold it still. Nil is time.Now.
	now func() time.Time
}

func (s *sink) at() time.Time {
	if s.now == nil {
		return time.Now()
	}
	return s.now()
}

// write puts one line down, opening a new day where this is the first line of
// one.
func (s *sink) write(level, text, fields string) {
	at := s.at()

	s.mu.Lock()
	defer s.mu.Unlock()

	if day := at.Format(dayStamp); day != s.day {
		s.day = day
		fmt.Fprintf(s.out, "──── %s\n", day)
	}
	fmt.Fprintf(s.out, "%s %-5s %s%s\n", at.Format(Stamp), level, text, fields)
}

// logger adapts librespot's logging to a plain writer. Trace and debug are
// dropped: they are enormous and say nothing useful about a device that works.
type logger struct {
	sink   *sink
	fields string
}

// newLogger writes to out. notice, which may be nil, is shown every line and
// may answer with one to put after it.
func newLogger(out io.Writer, notice func(text string) (string, bool)) librespot.Logger {
	return &logger{sink: &sink{out: out, notice: notice}}
}

// write puts the line down and then lets the reader answer it. The answer goes
// straight to the sink rather than back through here, so a reader that returns
// a line beginning with whatever it was watching for cannot start a loop.
func (l *logger) write(level, text string) {
	l.sink.write(level, text, l.fields)
	if l.sink.notice == nil {
		return
	}
	if say, ok := l.sink.notice(text); ok {
		l.sink.write("info", say, "")
	}
}

func (l *logger) Tracef(string, ...any)     {}
func (l *logger) Debugf(string, ...any)     {}
func (l *logger) Infof(f string, a ...any)  { l.write("info", fmt.Sprintf(f, a...)) }
func (l *logger) Warnf(f string, a ...any)  { l.write("warn", fmt.Sprintf(f, a...)) }
func (l *logger) Errorf(f string, a ...any) { l.write("error", fmt.Sprintf(f, a...)) }

func (l *logger) Trace(...any)   {}
func (l *logger) Debug(...any)   {}
func (l *logger) Info(a ...any)  { l.write("info", fmt.Sprint(a...)) }
func (l *logger) Warn(a ...any)  { l.write("warn", fmt.Sprint(a...)) }
func (l *logger) Error(a ...any) { l.write("error", fmt.Sprint(a...)) }

func (l *logger) WithField(key string, value any) librespot.Logger {
	return &logger{sink: l.sink, fields: fmt.Sprintf("%s %s=%v", l.fields, key, value)}
}

func (l *logger) WithError(err error) librespot.Logger {
	return &logger{sink: l.sink, fields: fmt.Sprintf("%s error=%q", l.fields, err)}
}
