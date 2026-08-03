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
	info := flag.Bool("cover-info", false, "report what the terminal supports and exit")
	flag.Parse()

	if *info {
		reportCoverSupport()
		return
	}

	if !*mock {
		fmt.Fprintln(os.Stderr, "spindle: the live Spotify backend is not wired up yet; run with --mock")
		os.Exit(1)
	}

	cell := cover.DetectCellSize(os.Stdout)
	renderer, err := coverRenderer(*backend, cell)
	if err != nil {
		fmt.Fprintln(os.Stderr, "spindle:", err)
		os.Exit(1)
	}
	loader := cover.NewLoader(renderer, &http.Client{Timeout: 15 * time.Second})

	if _, err := tea.NewProgram(ui.New(player.NewMock(), loader, cell)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "spindle:", err)
		os.Exit(1)
	}
}

// coverRenderer picks the artwork backend. The terminal is probed before Bubble
// Tea claims it, so the query and its reply cannot collide with the event loop's
// own input handling.
func coverRenderer(backend string, cell cover.CellSize) (cover.Renderer, error) {
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

// reportCoverSupport prints what the terminal was found to support, so a fallback
// to halfblock can be told apart from a kitty backend that is simply not drawing.
func reportCoverSupport() {
	cell := cover.DetectCellSize(os.Stdout)
	source := "measured via TIOCGWINSZ"
	if !cell.Measured {
		source = "assumed; the terminal reported no pixel size"
	}

	kitty := cover.SupportsKitty(os.Stdout, os.Stdin)
	backend := "halfblock"
	if kitty {
		backend = "kitty"
	}

	fmt.Printf("terminal:   TERM=%s TERM_PROGRAM=%s\n", os.Getenv("TERM"), os.Getenv("TERM_PROGRAM"))
	fmt.Printf("cell size:  %d × %d px (%s)\n", cell.Width, cell.Height, source)
	fmt.Printf("kitty:      %v\n", kitty)
	fmt.Printf("backend:    %s\n", backend)
	fmt.Printf("artwork:    %d × %d cells = %d × %d px\n",
		20, 10, 20*cell.Width, 10*cell.Height)
}
