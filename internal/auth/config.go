package auth

import (
	"fmt"
	"os"

	"golang.org/x/oauth2"
)

const (
	// RedirectURI must match the Spotify app's configuration character for
	// character. Spotify no longer accepts "localhost" here, only the literal
	// loopback address.
	//
	// The port and path are the ones registered for DefaultClientID; a client
	// id of your own needs its own redirect URI registered, and this is the one
	// to add.
	RedirectURI = "http://127.0.0.1:8989/login"

	callbackAddr = "127.0.0.1:8989"
	callbackPath = "/login"

	clientIDEnv = "SPINDLE_CLIENT_ID"

	authURL  = "https://accounts.spotify.com/authorize"
	tokenURL = "https://accounts.spotify.com/api/token"
)

// Scopes are the permissions spindle asks for. Search needs none of its own.
//
// Asking for more than is used would be rude, and asking for less means a
// second trip through the browser later: every scope here is one a screen or a
// key already depends on, or is about to.
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

// DefaultClientID is the application spindle authenticates as when nothing else
// is configured. It is a long-standing one, registered before Spotify closed
// several endpoints to new applications.
//
// This is not a nicety. Measured on 2026-08-04 against the same account, an
// application registered today is refused the tracks of any playlist (403), the
// batch track lookup (403), related artists (403) and recommendations (404),
// while this one is answered 200 for every one of them. The difference is the
// registration, not the code — and a player that cannot list a playlist is not
// a player.
//
// Anyone who would rather use their own can: SPINDLE_CLIENT_ID overrides this,
// as does an id saved by "spindle login". They will need to register
// RedirectURI against it, and to accept what Spotify refuses new applications.
const DefaultClientID = "65b708073fc0480ea92a077233ca87bd"

// ClientID returns the application's client id.
//
// The environment wins so a different application can be tried without
// disturbing what is saved; then whatever "spindle login <id>" saved; then the
// one spindle ships with.
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
		return DefaultClientID, nil
	}
	return id, nil
}

// SetupHelp is what to tell someone bringing their own application.
func SetupHelp() string {
	return fmt.Sprintf(`Registering an application of your own:

  1. Open https://developer.spotify.com/dashboard and create an app.
  2. Add this exact redirect URI:

       %s

     Spotify rejects "localhost"; it has to be the numeric form.
  3. Run "spindle login <client id>".

Setting %s overrides what is saved. "spindle login default" goes back to the
application spindle ships with, which several endpoints still answer that
Spotify has closed to new registrations.

The client secret is not needed: spindle authenticates with PKCE.`, RedirectURI, clientIDEnv)
}

// oauthConfig builds the OAuth client for a given application.
func oauthConfig(clientID string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:    clientID,
		RedirectURL: RedirectURI,
		Scopes:      Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:   authURL,
			TokenURL:  tokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}
