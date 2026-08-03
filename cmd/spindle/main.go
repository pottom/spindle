// Command spindle is a Spotify Connect remote control for the terminal.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui"
	"github.com/pottom/spindle/internal/ui/cover"
)

func main() {
	mock := flag.Bool("mock", false, "run against the offline mock backend, without auth or network")
	backend := flag.String("cover", "auto", "artwork backend: auto, kitty or halfblock")
	flag.Parse()

	if !*mock {
		fmt.Fprintln(os.Stderr, "spindle: the live Spotify backend is not wired up yet; run with --mock")
		os.Exit(1)
	}

	renderer, err := coverRenderer(*backend)
	if err != nil {
		fmt.Fprintln(os.Stderr, "spindle:", err)
		os.Exit(1)
	}
	loader := cover.NewLoader(renderer, &http.Client{Timeout: 15 * time.Second})

	if _, err := tea.NewProgram(ui.New(player.NewMock(), loader)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "spindle:", err)
		os.Exit(1)
	}
}

// coverRenderer picks the artwork backend. The terminal is probed before Bubble
// Tea claims it, so the query and its reply cannot collide with the event loop's
// own input handling.
func coverRenderer(backend string) (cover.Renderer, error) {
	cell := cover.DetectCellSize(os.Stdout)

	switch backend {
	case "halfblock":
		return cover.NewHalfblock(cell), nil
	case "kitty":
		return cover.NewKitty(os.Stdout, cell), nil
	case "auto":
		if cover.SupportsKitty(os.Stdout, os.Stdin) {
			return cover.NewKitty(os.Stdout, cell), nil
		}
		return cover.NewHalfblock(cell), nil
	default:
		return nil, fmt.Errorf("unknown cover backend %q: want auto, kitty or halfblock", backend)
	}
}
