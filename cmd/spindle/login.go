package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/zmb3/spotify/v2"

	"github.com/pottom/spindle/internal/auth"
)

// runLogin authorises spindle and reports who it authorised as. It is the one
// place the browser flow is meant to be triggered by hand, and the one place
// that asks for the client id.
//
// An argument sets the application to authenticate as, which is easier to hand
// to someone than a prompt, and is what a second application wants.
func runLogin(ctx context.Context, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("spindle login takes one client id, got %d arguments", len(args))
	}
	if len(args) == 1 {
		if err := auth.SaveClientID(strings.TrimSpace(args[0])); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Saved to %s\n\n", auth.SettingsPath())
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

func currentUser(ctx context.Context, session *auth.Session) (*spotify.PrivateUser, error) {
	user, err := spotify.New(session.Client(ctx)).CurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch current user: %w", err)
	}
	return user, nil
}
