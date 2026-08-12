package awake

import (
	"fmt"
	"os"
	"os/exec"
)

// holdMachine takes a systemd inhibitor for as long as we live.
//
// systemd-inhibit holds its lock only while the command it was given is
// running, so the command it is given is one that watches this process and ends
// the moment we do. That is the same shape as caffeinate's -w on macOS and it is
// there for the same reason — an inhibitor that outlives its reason is worse
// than none, because nobody will connect a machine that stopped sleeping with a
// music player they closed yesterday.
//
// The watcher is a shell loop on kill -0 rather than the obvious
// `tail --pid=N -f /dev/null`, because --pid is a GNU coreutils extension and
// BusyBox has no such flag. On a machine without it, tail would fail at once,
// systemd-inhibit would exit with it, and Start would still have returned no
// error — leaving us believing we held a machine we had let go of. A shell loop
// is POSIX and runs anywhere there is a shell at all.
//
// A machine without systemd gets an error and no inhibitor. That is honest: the
// alternative is a screen that dims with nothing said about why.
func holdMachine() (func(), error) {
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
