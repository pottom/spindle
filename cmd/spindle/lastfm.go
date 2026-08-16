package main

import (
	"fmt"

	"github.com/pottom/spindle/internal/auth"
)

// runLastFM reports or sets the last.fm key.
//
// A command rather than a field on the settings screen, for the same reason the
// client id is one: a key is pasted out of a browser once and never looked at
// again, and a screen somebody works in is not where a forty-character string
// belongs. The settings screen says whether one is set, which is the only thing
// about it worth seeing twice.
func runLastFM(args []string) error {
	if len(args) == 0 {
		return reportLastFM()
	}
	if len(args) > 1 {
		return fmt.Errorf("spindle lastfm takes one argument, got %d", len(args))
	}

	// "none" rather than an empty argument, which a shell eats.
	key := args[0]
	if key == "none" {
		key = ""
	}
	if err := auth.SetLastFMKey(key); err != nil {
		return err
	}
	return reportLastFM()
}

func reportLastFM() error {
	if key := auth.LastFMKey(); key != "" {
		fmt.Printf("last.fm:  set (%s…)\n", key[:min(len(key), 6)])
		fmt.Println()
		fmt.Println("Artist notes, who else listens to them, and how many.")
		fmt.Println("Remove it with: spindle lastfm none")
		return nil
	}

	fmt.Println("last.fm:  not set")
	fmt.Println()
	fmt.Println("Optional. With a key, spindle can say something about artists")
	fmt.Println("no other free database has heard of, who else people listening")
	fmt.Println("to them listen to, and how many are listening.")
	fmt.Println()
	fmt.Println("A key is free and immediate: https://www.last.fm/api/account/create")
	fmt.Println("Then: spindle lastfm <key>")
	return nil
}
