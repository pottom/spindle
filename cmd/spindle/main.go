// Command spindle is a Spotify Connect remote control for the terminal.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui"
)

func main() {
	mock := flag.Bool("mock", false, "run against the offline mock backend, without auth or network")
	flag.Parse()

	if !*mock {
		fmt.Fprintln(os.Stderr, "spindle: the live Spotify backend is not wired up yet; run with --mock")
		os.Exit(1)
	}

	if _, err := tea.NewProgram(ui.New(player.NewMock())).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "spindle:", err)
		os.Exit(1)
	}
}
