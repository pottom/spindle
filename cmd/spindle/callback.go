package main

import (
	"fmt"
	"strconv"

	"github.com/pottom/spindle/internal/auth"
)

// runCallback reports or sets the port the browser is sent back to when logging
// in.
//
// It is a setting rather than a number in the source because the right value is
// a property of the machine: it has to be a port nothing else there wants, and
// which port that is nobody here can know. It was 8888, which is one of the most
// contested numbers there is — a SOCKS proxy sitting on it was enough to make
// logging in fail with nothing to look at but a bind error.
func runCallback(args []string) error {
	if len(args) == 0 {
		fmt.Printf("port:     %d\n", auth.CallbackPort())
		fmt.Printf("address:  %s\n", auth.RedirectURI())
		fmt.Println()
		fmt.Println("This address must be listed in the Spotify application, character for")
		fmt.Println("character. An application may list several, so an old one can stay.")
		return nil
	}
	if len(args) > 1 {
		return fmt.Errorf("spindle callback takes one argument, got %d", len(args))
	}

	port, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("%q is not a port", args[0])
	}
	if err := auth.SetCallbackPort(port); err != nil {
		return err
	}

	fmt.Printf("spindle: logging in will use %s\n", auth.RedirectURI())
	fmt.Println("spindle: add that address to the Spotify application before logging in again")
	return nil
}
