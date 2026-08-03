package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// ErrReauthRequired reports that the stored grant is no longer good and the
// browser flow has to run again. The token has already been deleted by the time
// this surfaces, so the next Session call starts clean.
//
// It is deliberately not answered by launching a browser from inside a token
// source: by then the TUI owns the terminal, and stealing it would be rude.
var ErrReauthRequired = errors.New("authorisation expired; sign in again")

// Session is an authenticated connection to Spotify.
type Session struct {
	Source oauth2.TokenSource
	store  *Store
}

// NewSession loads the stored grant, running the browser flow only if there is
// none. Progress during that flow is written to w.
func NewSession(ctx context.Context, w io.Writer) (*Session, error) {
	clientID, err := ClientID()
	if err != nil {
		return nil, err
	}

	store, err := NewStore()
	if err != nil {
		return nil, err
	}

	tok, err := store.Load()
	if err != nil {
		return nil, err
	}
	if tok == nil {
		if tok, err = Login(ctx, w); err != nil {
			return nil, err
		}
		if err := store.Save(tok); err != nil {
			return nil, err
		}
	}

	cfg := oauthConfig(clientID)
	return &Session{
		Source: &persistingSource{src: cfg.TokenSource(ctx, tok), store: store, last: tok.AccessToken},
		store:  store,
	}, nil
}

// Client is an HTTP client that attaches and refreshes the access token.
func (s *Session) Client(ctx context.Context) *http.Client {
	return oauth2.NewClient(ctx, s.Source)
}

// TokenPath is where the grant is stored, for diagnostics.
func (s *Session) TokenPath() string { return s.store.Path() }

// persistingSource writes every refreshed token back to disk, so a long-running
// session does not lose its grant when it exits.
type persistingSource struct {
	src   oauth2.TokenSource
	store *Store
	last  string
}

func (p *persistingSource) Token() (*oauth2.Token, error) {
	tok, err := p.src.Token()
	if err != nil {
		if isGrantRejected(err) {
			// A revoked or expired refresh token is not worth keeping around.
			if derr := p.store.Delete(); derr != nil {
				return nil, fmt.Errorf("%w (and %w)", ErrReauthRequired, derr)
			}
			return nil, fmt.Errorf("%w: %w", ErrReauthRequired, err)
		}
		return nil, fmt.Errorf("refresh access token: %w", err)
	}

	if tok.AccessToken != p.last {
		p.last = tok.AccessToken
		if err := p.store.Save(tok); err != nil {
			return nil, err
		}
	}
	return tok, nil
}

// isGrantRejected distinguishes "this grant is dead" from "the network is down".
// Spotify answers a revoked refresh token with 400 and invalid_grant.
func isGrantRejected(err error) bool {
	var re *oauth2.RetrieveError
	if !errors.As(err, &re) {
		return false
	}
	if re.ErrorCode == "invalid_grant" {
		return true
	}
	return re.Response != nil && re.Response.StatusCode == http.StatusBadRequest
}
