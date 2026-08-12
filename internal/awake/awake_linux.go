package awake

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/godbus/dbus/v5"
)

// Linux needs two holds, not one, and they are held by different things.
//
// systemd-inhibit stops logind acting on an idle session — suspending the
// machine. It does not stop the screen going dark, because on a desktop that is
// not logind's doing at all: the session's screensaver does it, and the
// screensaver listens on D-Bus. Hold only the first and a party screen stays
// powered on and shows nothing, which is the failure this package exists to
// prevent rather than a lesser version of it.
//
// So both are taken, and either may be missing: a headless box has no session
// bus, a machine without systemd has no inhibitor. One is better than none, so a
// hold succeeds if either does — and says which, because "the screen went dark
// anyway" needs an answer other than a shrug.
func holdMachine() (func(), string, error) {
	var (
		stops []func()
		has   []string
		bad   []error
	)
	for _, try := range []struct {
		what string
		take func() (func(), error)
	}{
		{"sleep", holdSleep},
		{"screen", holdScreen},
	} {
		stop, err := try.take()
		if err != nil {
			bad = append(bad, err)
			continue
		}
		stops = append(stops, stop)
		has = append(has, try.what)
	}
	if len(stops) == 0 {
		return nil, "", errors.Join(bad...)
	}

	held := strings.Join(has, " and ")
	if len(has) == 1 {
		held += " only"
	}
	return func() {
		for _, stop := range stops {
			stop()
		}
	}, held, nil
}

// holdSleep keeps logind from suspending an idle session.
//
// systemd-inhibit holds its lock only while the command it was given runs, so
// the command it is given watches this process and ends the moment we do. Same
// shape as caffeinate's -w on macOS, same reason: an inhibitor that outlives its
// reason is worse than none, because nobody will connect a machine that stopped
// sleeping with a music player they closed yesterday.
//
// The watcher is a shell loop on kill -0 rather than the obvious
// `tail --pid=N -f /dev/null`, because --pid is a GNU coreutils extension and
// BusyBox has no such flag. There, tail would fail at once, systemd-inhibit
// would exit with it, and Start would still have returned no error — leaving us
// believing we held a machine we had let go of.
func holdSleep() (func(), error) {
	if _, err := exec.LookPath("systemd-inhibit"); err != nil {
		return nil, fmt.Errorf("no systemd-inhibit on this machine: %w", err)
	}
	watch := fmt.Sprintf("while kill -0 %d 2>/dev/null; do sleep 5; done", os.Getpid())
	cmd := exec.Command("systemd-inhibit",
		"--what=idle:sleep",
		"--who=spindle",
		"--why=showing the picture",
		"--mode=block",
		"sh", "-c", watch,
	)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("systemd-inhibit: %w", err)
	}
	waited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(waited)
	}()
	return func() {
		_ = cmd.Process.Kill()
		<-waited
	}, nil
}

// screensavers are the names to ask, in order. The freedesktop one is what KDE,
// XFCE, Cinnamon and most others answer to; GNOME answers its own as well, and
// has been known to drop the shared one, so it is worth asking twice.
var screensavers = []struct{ name, path, iface string }{
	{"org.freedesktop.ScreenSaver", "/org/freedesktop/ScreenSaver", "org.freedesktop.ScreenSaver"},
	{"org.gnome.SessionManager", "/org/gnome/SessionManager", "org.gnome.SessionManager"},
}

// holdScreen asks the session's screensaver to leave the screen alone.
//
// The connection is a private one rather than the shared session bus, because
// the inhibit lasts exactly as long as the connection that took it. On the
// shared bus there would be nothing to close, and closing it would take the rest
// of the program's D-Bus with it. This way, letting go is closing our own — and
// so is dying, which is what covers a crash.
func holdScreen() (func(), error) {
	conn, err := dbus.SessionBusPrivate()
	if err != nil {
		return nil, fmt.Errorf("no session bus: %w", err)
	}
	if err := conn.Auth(nil); err != nil {
		conn.Close() //nolint:errcheck // already failing
		return nil, fmt.Errorf("session bus auth: %w", err)
	}
	if err := conn.Hello(); err != nil {
		conn.Close() //nolint:errcheck // already failing
		return nil, fmt.Errorf("session bus hello: %w", err)
	}

	for _, s := range screensavers {
		var cookie uint32
		var call *dbus.Call
		obj := conn.Object(s.name, dbus.ObjectPath(s.path))
		if s.iface == "org.gnome.SessionManager" {
			// GNOME's takes an application, a window to hang it on (none), a
			// reason, and flags — 8 being "idle", which is the one that stops
			// the screen blanking.
			call = obj.Call(s.iface+".Inhibit", 0, "spindle", uint32(0), "showing the picture", uint32(8))
		} else {
			call = obj.Call(s.iface+".Inhibit", 0, "spindle", "showing the picture")
		}
		if call.Err != nil {
			continue
		}
		if err := call.Store(&cookie); err != nil {
			continue
		}
		return func() {
			obj.Call(s.iface+".UnInhibit", 0, cookie) //nolint:errcheck // closing next anyway
			conn.Close()                              //nolint:errcheck // letting go
		}, nil
	}

	conn.Close() //nolint:errcheck // nothing took
	return nil, errors.New("no screensaver on the session bus answered")
}
