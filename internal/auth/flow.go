package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/oauth2"

	"github.com/pottom/spindle/internal/browser"
)

// Login runs the PKCE flow from end to end and returns a fresh token. Progress
// is written to w, which is the terminal for the login subcommand.
func Login(ctx context.Context, w io.Writer) (*oauth2.Token, error) {
	clientID, err := ClientID()
	if err != nil {
		return nil, err
	}

	server, err := listenForCallback()
	if err != nil {
		return nil, err
	}
	defer server.close()

	state, err := randomState()
	if err != nil {
		return nil, err
	}

	cfg := oauthConfig(clientID)
	verifier := oauth2.GenerateVerifier()
	url := cfg.AuthCodeURL(state,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("show_dialog", "false"),
	)

	// Always print the URL, even when the browser opened. Over SSH the launcher
	// fails silently and this line is the only way through.
	if browser.Open(url) {
		fmt.Fprintln(w, "Opening your browser to sign in to Spotify.")
	} else {
		fmt.Fprintln(w, "Could not open a browser.")
	}
	fmt.Fprintf(w, "If nothing happens, visit:\n\n  %s\n\nWaiting for the redirect…\n", url)

	res, err := server.wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("wait for authorisation: %w", err)
	}
	if res.state != state {
		return nil, fmt.Errorf("authorisation state mismatch: the redirect did not come from this login")
	}

	tok, err := cfg.Exchange(ctx, res.code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("exchange authorisation code: %w", err)
	}
	return tok, nil
}

// randomState is the CSRF guard tying the redirect back to this login.
func randomState() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
