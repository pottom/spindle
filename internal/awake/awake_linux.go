package awake

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// holdMachine takes a systemd inhibitor for as long as we live.
//
// systemd-inhibit holds its lock only while the command it was given is
// running, so the command it is given is one that watches this process:
// `tail --pid=N -f /dev/null` ends the moment we do. That is the same shape as
// caffeinate's -w on macOS and it is there for the same reason — an inhibitor
// that outlives its reason is worse than none, because nobody will connect a
// machine that stopped sleeping with a music player they closed yesterday.
//
// A machine without systemd gets an error and no inhibitor. That is honest: the
// alternative is a screen that dims with nothing said about why.
func holdMachine() (func(), error) {
	if _, err := exec.LookPath("systemd-inhibit"); err != nil {
		return nil, fmt.Errorf("no systemd-inhibit on this machine: %w", err)
	}
	cmd := exec.Command("systemd-inhibit",
		"--what=idle:sleep",
		"--who=spindle",
		"--why=showing the picture",
		"--mode=block",
		"tail", "--pid="+strconv.Itoa(os.Getpid()), "-f", "/dev/null",
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
