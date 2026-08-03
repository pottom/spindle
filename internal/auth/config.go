package auth

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/oauth2"
)

const (
	// RedirectURI must match the Spotify app's configuration character for
	// character. Spotify no longer accepts "localhost" here, only the literal
	// loopback address.
	RedirectURI = "http://127.0.0.1:8888/callback"

	callbackAddr = "127.0.0.1:8888"
	callbackPath = "/callback"

	clientIDEnv = "SPINDLE_CLIENT_ID"

	authURL  = "https://accounts.spotify.com/authorize"
	tokenURL = "https://accounts.spotify.com/api/token"
)

// Scopes are the permissions spindle asks for. Reading playback state and
// controlling it cover the player; the playlist scopes cover the library tab.
// Search needs no scope of its own.
var Scopes = []string{
	"user-read-playback-state",
	"user-modify-playback-state",
	"user-read-currently-playing",
	"playlist-read-private",
	"playlist-read-collaborative",
}

// ErrNoClientID reports that the application is not configured yet. Callers are
// expected to answer it with SetupHelp rather than a bare error line.
var ErrNoClientID = errors.New("no Spotify client id")

// ClientID reads the application's client id from the environment.
func ClientID() (string, error) {
	id := os.Getenv(clientIDEnv)
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
  3. Copy the client id from the app's settings and export it:

       export %s=<client id>

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
