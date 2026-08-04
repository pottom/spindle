package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/zmb3/spotify/v2"

	"github.com/pottom/spindle/internal/auth"
)

// runLogin authorises spindle and reports who it authorised as. It is the one
// place the browser flow is meant to be triggered by hand.
//
// An argument names the application to authenticate as, for anyone who would
// rather use their own registration than the one spindle ships with.
func runLogin(ctx context.Context, args []string) error {
	if err := useClientID(args, os.Stdout); err != nil {
		return err
	}

	session, err := auth.NewSession(ctx, os.Stdout)
	if err != nil {
		return err
	}

	user, err := currentUser(ctx, session)
	if errors.Is(err, auth.ErrReauthRequired) {
		// The stored grant was dead and has been deleted; earn a new one.
		fmt.Fprintln(os.Stdout, "The saved authorisation is no longer valid. Signing in again.")
		if session, err = auth.NewSession(ctx, os.Stdout); err != nil {
			return err
		}
		user, err = currentUser(ctx, session)
	}
	if err != nil {
		return err
	}

	name := user.DisplayName
	if name == "" {
		name = user.ID
	}
	fmt.Printf("Signed in as %s.\nToken stored at %s\n", name, session.TokenPath())
	return nil
}

// useClientID records the application to authenticate as. "default" goes back
// to the one spindle ships with; a malformed id is refused here rather than
// halfway through the browser.
func useClientID(args []string, out io.Writer) error {
	if len(args) == 0 {
		return nil
	}
	if len(args) > 1 {
		return fmt.Errorf("spindle login takes one client id, got %d arguments", len(args))
	}

	id := strings.TrimSpace(args[0])
	if id == "default" {
		id = ""
	}
	if err := auth.SaveClientID(id); err != nil {
		return err
	}

	if id == "" {
		fmt.Fprint(out, "Using the application spindle ships with.\n\n")
		return nil
	}
	fmt.Fprintf(out, "Saved to %s\n\n", auth.SettingsPath())
	return nil
}

func currentUser(ctx context.Context, session *auth.Session) (*spotify.PrivateUser, error) {
	user, err := spotify.New(session.Client(ctx)).CurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch current user: %w", err)
	}
	return user, nil
}
