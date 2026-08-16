package auth

import (
	"errors"
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

	// ncspotClientID is another terminal player's registration, and the reason
	// this file knows about anybody else's. It is registered in Spotify's
	// extended quota mode and predates the 2024 changes to the Web API, so it is
	// not held to the daily cap a newly registered application is — which is the
	// cap a window left open for a day can reach. spotify-player ships it as its
	// default for exactly that reason, and warns anyone who sets their own.
	//
	// spindle does not ship it. It is here so that setting it in the environment
	// works rather than failing at the browser: an id and the address its
	// registration sends the browser back to are a pair, and this one's is a
	// loopback port of any number with /login on the end.
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

// ErrNoClientID reports that the application is not configured yet. Callers are
// expected to answer it with SetupHelp rather than a bare error line.
var ErrNoClientID = errors.New("no Spotify client id")

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
		return "", ErrNoClientID
	}
	return id, nil
}

// SetupHelp is what to tell someone who has not registered an application yet.
func SetupHelp() string {
	return fmt.Sprintf(`spindle needs a Spotify application of its own.

  1. Open https://developer.spotify.com/dashboard and create an app.
  2. Add this exact redirect URI:

       %s

     Spotify rejects "localhost"; it has to be the numeric form.
  3. Copy the client id from the app's settings.

Run "spindle login" and it will ask for the id once and remember it. Setting %s
overrides what is saved, which is handy for trying a second application.

The client secret is not needed: spindle authenticates with PKCE.`, RedirectURI(), clientIDEnv)
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
