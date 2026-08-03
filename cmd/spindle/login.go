package main

import (
	"bufio"
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
// place the browser flow is meant to be triggered by hand, and the one place
// that asks for the client id.
func runLogin(ctx context.Context) error {
	if err := ensureClientID(os.Stdin, os.Stdout); err != nil {
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

// ensureClientID asks for the application's client id if there is none saved.
// It is asked once and remembered; a malformed answer is rejected on the spot
// rather than turning into a confusing rejection halfway through the browser.
func ensureClientID(in io.Reader, out io.Writer) error {
	if _, err := auth.ClientID(); !errors.Is(err, auth.ErrNoClientID) {
		return err
	}

	fmt.Fprintln(out, auth.SetupHelp())
	fmt.Fprint(out, "\nClient id: ")

	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return fmt.Errorf("no client id given")
	}
	id := strings.TrimSpace(scanner.Text())

	if err := auth.SaveClientID(id); err != nil {
		return err
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
