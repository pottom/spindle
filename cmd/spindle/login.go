package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/zmb3/spotify/v2"

	"github.com/pottom/spindle/internal/auth"
)

// runLogin authorises spindle and reports who it authorised as. It is the one
// place the browser flow is meant to be triggered by hand.
func runLogin(ctx context.Context) error {
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
