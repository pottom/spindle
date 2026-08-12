package awake

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// holdMachine runs caffeinate for as long as we live.
//
//	-d  keep the display lit, which is the half audio playback never covers
//	-i  no idle sleep
//	-w  exit when this process does
//
// The -w carries the weight. Without it, a spindle that crashes leaves
// caffeinate behind holding the machine awake until somebody reboots, and
// nobody would think to blame that on a music player. With it, the assertion
// cannot outlive the reason for it, however badly we go.
func holdMachine() (func(), error) {
	cmd := exec.Command("caffeinate", "-d", "-i", "-w", strconv.Itoa(os.Getpid()))
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("caffeinate: %w", err)
	}
	// Reaped in the background so killing it later leaves no zombie.
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
