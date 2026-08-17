package auth

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
)

const (
	// defaultCallbackPort is used until the settings say otherwise. Beside the
	// daemon's 3678, so the two read as a pair. See settings.CallbackPort for
	// why the number lives there rather than here.
	defaultCallbackPort = 3679

	// ncspotClientID is what spindle authenticates as unless it is told
	// otherwise: another terminal player's registration, public in ncspot's own
	// source, which spotify-player ships as its default for the same reason.
	//
	// It is registered in Spotify's extended quota mode and predates the 2024
	// changes to the Web API. An application registered today is not: it gets a
	// daily cap a window left open can reach, and a family of endpoints refused
	// outright — a playlist somebody else owns cannot be listed, and a track
	// cannot be liked or even asked about. Which registration is asking decides
	// that, not spindle. See docs/SPOTIFY-API.md.
	//
	// Anyone can use their own instead, and spindle then turns off what that
	// registration is not allowed to do rather than offering keys that fail. An
	// id and the address its registration sends the browser back to are a pair:
	// this one takes a loopback port of any number with /login on the end.
	ncspotClientID = "d420a117a32841c2b3474932e49fb54b"

	ownCallbackPath    = "/callback"
	ncspotCallbackPath = "/login"

	clientIDEnv = "SPINDLE_CLIENT_ID"
	lastFMEnv   = "SPINDLE_LASTFM_KEY"

	authURL  = "https://accounts.spotify.com/authorize"
	tokenURL = "https://accounts.spotify.com/api/token"
)

// RedirectURI is where Spotify sends the browser back to. It must match the
// application's own configuration character for character; Spotify no longer
// accepts "localhost" here, only the literal loopback address.
func RedirectURI() string { return "http://" + callbackAddr() + CallbackPath() }

// CallbackPath is the part of the address Spotify checks against what the
// application registered. It belongs to the client id rather than to spindle:
// an application somebody else registered sends the browser wherever they said
// it would go, and a mismatched path is refused after the login rather than
// before it — which is a long way from the cause.
func CallbackPath() string {
	if id, err := ClientID(); err == nil && id == ncspotClientID {
		return ncspotCallbackPath
	}
	return ownCallbackPath
}

// callbackAddr is what the callback server listens on.
func callbackAddr() string { return "127.0.0.1:" + strconv.Itoa(CallbackPort()) }

// CallbackPort is the port the browser is sent back to: what the settings say,
// or the default. An unreadable settings file is not worth refusing to log in
// over, so it falls back rather than failing.
func CallbackPort() int {
	if s, err := load(); err == nil && s.CallbackPort > 0 && s.CallbackPort < 65536 {
		return s.CallbackPort
	}
	return defaultCallbackPort
}

// LastFMKey is the key for last.fm, or empty where there is none.
//
// The environment first, as with the client id: a key passed in for one run is a
// key nobody has to write down. An unreadable settings file is not worth
// refusing to start over — the source is simply not in the chain.
func LastFMKey() string {
	if key := strings.TrimSpace(os.Getenv(lastFMEnv)); key != "" {
		return key
	}
	if s, err := load(); err == nil {
		return strings.TrimSpace(s.LastFM)
	}
	return ""
}

// SetLastFMKey remembers it, or forgets it when handed nothing.
func SetLastFMKey(key string) error {
	s, err := load()
	if err != nil {
		return err
	}
	s.LastFM = strings.TrimSpace(key)
	return save(s)
}

// SetCallbackPort remembers where the browser should be sent back to.
func SetCallbackPort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("callback port %d is not a port", port)
	}
	s, err := load()
	if err != nil {
		return err
	}
	s.CallbackPort = port
	return save(s)
}

// Scopes are the permissions spindle asks for. Search needs none of its own.
//
// Asking for more than is used would be rude, and asking for less means a
// second trip through the browser later: every scope here is one a screen or a
// key already depends on, or is about to. They cost nothing that is refused —
// what a new application is refused is endpoints, not permissions.
var Scopes = []string{
	// The player.
	"user-read-playback-state",
	"user-modify-playback-state",
	"user-read-currently-playing",

	// The library tab, and editing what is in it.
	"playlist-read-private",
	"playlist-read-collaborative",
	"playlist-modify-private",
	"playlist-modify-public",

	// Liked songs, and the like key.
	"user-library-read",
	"user-library-modify",

	// Following an artist, and the pages built from listening history.
	"user-follow-read",
	"user-follow-modify",
	"user-top-read",
	"user-read-recently-played",
}

// ClientID returns the application's client id.
//
// The environment wins so a different application can be tried without
// disturbing what is saved; otherwise it comes from the settings file, which is
// where it lands after being asked for once.
func ClientID() (string, error) {
	if id := os.Getenv(clientIDEnv); id != "" {
		if !clientIDPattern.MatchString(id) {
			return "", fmt.Errorf("%w: %s is set but malformed", ErrMalformedClientID, clientIDEnv)
		}
		return id, nil
	}

	id, err := loadClientID()
	if err != nil {
		return "", err
	}
	if id == "" {
		// Nobody has said otherwise, so it is the one spindle ships with.
		// Nothing is written down: a saved copy of the default is a copy that
		// goes stale the day the default changes.
		return ncspotClientID, nil
	}
	return id, nil
}

// DefaultClientID is the registration spindle authenticates as out of the box.
func DefaultClientID() string { return ncspotClientID }

// OwnApplication reports that the listener has put their own registration in
// place of the one spindle ships with. What Spotify allows depends on which is
// asking, so it is the question every optional feature is decided by.
func OwnApplication() bool {
	id, err := ClientID()
	return err == nil && id != ncspotClientID
}

// SetupHelp is what to tell somebody who wants to authenticate as their own
// application rather than as the one spindle ships with.
func SetupHelp() string {
	return fmt.Sprintf(`spindle authenticates as a Spotify application, and ships with one.

To use your own instead:

  1. Open https://developer.spotify.com/dashboard and create an app.
  2. Add this exact redirect URI:

       %s

     Spotify rejects "localhost"; it has to be the numeric form.
  3. Run "spindle login <client id>", or set %s for one run.

Know what it costs. An application registered after November 2024 is in
Spotify's development mode: it has a daily quota, and it is refused a family of
endpoints outright — a playlist somebody else owns cannot be listed, and a track
cannot be liked or even asked about. spindle turns those off rather than
offering keys that fail, and the settings screen says which are gone.

The client secret is not needed: spindle authenticates with PKCE.`,
		RedirectURI(), clientIDEnv)
}

// oauthConfig builds the OAuth client for a given application.
func oauthConfig(clientID string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:    clientID,
		RedirectURL: RedirectURI(),
		Scopes:      Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:   authURL,
			TokenURL:  tokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}
