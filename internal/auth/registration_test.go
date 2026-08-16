package auth

import (
	"strings"
	"testing"
)

// A client id and the address the browser comes back to are a pair. Spotify
// checks the address against what the application registered, and it does the
// checking after the login rather than before it — so a mismatch shows up as a
// refusal a long way from its cause.
func TestTheCallbackPathBelongsToTheClientID(t *testing.T) {
	t.Setenv(clientIDEnv, "1c227ccd43c64c89918ce162bfc38c7b")
	if got := CallbackPath(); got != ownCallbackPath {
		t.Errorf("our own registration sends the browser to %q, want %q", got, ownCallbackPath)
	}

	// Another player's registration, which is registered in extended quota mode
	// and is the reason anyone would set it. Its own source sends the browser to
	// a loopback port of any number with /login on the end.
	t.Setenv(clientIDEnv, ncspotClientID)
	if got := CallbackPath(); got != ncspotCallbackPath {
		t.Errorf("ncspot's registration sends the browser to %q, want %q", got, ncspotCallbackPath)
	}
	if uri := RedirectURI(); !strings.HasSuffix(uri, ncspotCallbackPath) {
		t.Errorf("the address offered to Spotify is %q, which that application would refuse", uri)
	}
}
