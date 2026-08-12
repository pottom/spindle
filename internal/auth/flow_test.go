package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

// The PKCE challenge is computed by the oauth2 package, but wiring it into the
// authorisation URL is ours to get wrong. Recompute it independently.
func TestAuthURLCarriesS256Challenge(t *testing.T) {
	const verifier = "a-known-verifier-of-sufficient-length-1234567890"

	raw := oauthConfig("client-123").AuthCodeURL("state-abc", oauth2.S256ChallengeOption(verifier))
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()

	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])

	checks := map[string]string{
		"code_challenge":        want,
		"code_challenge_method": "S256",
		"response_type":         "code",
		"client_id":             "client-123",
		"state":                 "state-abc",
		"redirect_uri":          RedirectURI(),
	}
	for key, expected := range checks {
		if got := q.Get(key); got != expected {
			t.Errorf("%s = %q, want %q", key, got, expected)
		}
	}

	// A secret would defeat the point of PKCE for a public client.
	if q.Has("client_secret") {
		t.Error("the authorisation URL must not carry a client secret")
	}
}

func TestAuthURLRequestsTheScopesWeNeed(t *testing.T) {
	raw := oauthConfig("client-123").AuthCodeURL("s")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	scopes := u.Query().Get("scope")
	for _, want := range Scopes {
		if !contains(scopes, want) {
			t.Errorf("scope %q missing from %q", want, scopes)
		}
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestRandomStateIsUnpredictable(t *testing.T) {
	seen := make(map[string]bool, 64)
	for range 64 {
		s, err := randomState()
		if err != nil {
			t.Fatal(err)
		}
		if len(s) < 24 {
			t.Fatalf("state %q is too short to be a useful guard", s)
		}
		if seen[s] {
			t.Fatalf("state %q repeated", s)
		}
		seen[s] = true
	}
}
