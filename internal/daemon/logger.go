package daemon

import (
	"fmt"
	"io"

	librespot "github.com/devgianlu/go-librespot"
)

// logger adapts librespot's logging to a plain writer. Trace and debug are
// dropped: they are enormous and say nothing useful about a device that works.
type logger struct {
	out    io.Writer
	fields string
}

func newLogger(out io.Writer) librespot.Logger { return &logger{out: out} }

func (l *logger) write(level, text string) {
	fmt.Fprintf(l.out, "%-5s %s%s\n", level, text, l.fields)
}

func (l *logger) Tracef(string, ...any) {}
func (l *logger) Debugf(string, ...any) {}
func (l *logger) Infof(f string, a ...any)  { l.write("info", fmt.Sprintf(f, a...)) }
func (l *logger) Warnf(f string, a ...any)  { l.write("warn", fmt.Sprintf(f, a...)) }
func (l *logger) Errorf(f string, a ...any) { l.write("error", fmt.Sprintf(f, a...)) }

func (l *logger) Trace(...any) {}
func (l *logger) Debug(...any) {}
func (l *logger) Info(a ...any)  { l.write("info", fmt.Sprint(a...)) }
func (l *logger) Warn(a ...any)  { l.write("warn", fmt.Sprint(a...)) }
func (l *logger) Error(a ...any) { l.write("error", fmt.Sprint(a...)) }

func (l *logger) WithField(key string, value any) librespot.Logger {
	return &logger{out: l.out, fields: fmt.Sprintf("%s %s=%v", l.fields, key, value)}
}

func (l *logger) WithError(err error) librespot.Logger {
	return &logger{out: l.out, fields: fmt.Sprintf("%s error=%q", l.fields, err)}
}
