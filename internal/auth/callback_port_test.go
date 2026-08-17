package auth

import (
	"strconv"
	"strings"
	"testing"
)

// Where the browser is sent back to is a setting, not a number in the source.
//
// The right value is a property of the machine rather than of spindle: it has to
// be a port nothing else there wants, and which port that is nobody here can
// know. It was 8888, and 8888 is one of the most contested numbers there is — a
// SOCKS proxy sitting on it is enough to make logging in fail with nothing to
// look at but a bind error.
func TestWhereTheBrowserComesBackToIsASetting(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// Nothing said yet: the default, beside the daemon's own port.
	if got := CallbackPort(); got != defaultCallbackPort {
		t.Errorf("with nothing set, the port is %d, want %d", got, defaultCallbackPort)
	}
	if uri := RedirectURI(); !strings.Contains(uri, strconv.Itoa(defaultCallbackPort)) {
		t.Errorf("the address does not carry the port: %q", uri)
	}
	if strings.Contains(RedirectURI(), "8888") {
		t.Error("still going back to 8888")
	}

	// And once it is set, it is remembered.
	if err := SetCallbackPort(45123); err != nil {
		t.Fatalf("setting the port: %v", err)
	}
	if got := CallbackPort(); got != 45123 {
		t.Errorf("the port was not remembered: %d", got)
	}
	// The path belongs to whichever registration is asking — see CallbackPath —
	// and this test is about the port, so it names its own.
	t.Setenv(clientIDEnv, "1c227ccd43c64c89918ce162bfc38c7b")
	if want := "http://127.0.0.1:45123/callback"; RedirectURI() != want {
		t.Errorf("the address is %q, want %q", RedirectURI(), want)
	}
	// The listener and the address Spotify is told must agree, or the browser
	// comes back to a door nobody is behind.
	if !strings.HasSuffix(RedirectURI(), callbackAddr()+CallbackPath()) {
		t.Errorf("the address %q and the listener %q have come apart", RedirectURI(), callbackAddr())
	}

	// Something that is not a port is refused rather than remembered.
	for _, bad := range []int{0, -1, 70000} {
		if err := SetCallbackPort(bad); err == nil {
			t.Errorf("%d was accepted as a port", bad)
		}
	}
	if got := CallbackPort(); got != 45123 {
		t.Errorf("a refused port changed the setting to %d", got)
	}
}
