package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	s, err := NewStore()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStoreRoundTrip(t *testing.T) {
	s := tempStore(t)

	if tok, err := s.Load(); err != nil || tok != nil {
		t.Fatalf("Load on an empty store = %v, %v; want nil, nil", tok, err)
	}

	want := &oauth2.Token{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour).Round(time.Second),
	}
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}

	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != want.AccessToken || got.RefreshToken != want.RefreshToken {
		t.Errorf("loaded %+v, want %+v", got, want)
	}
	if !got.Expiry.Equal(want.Expiry) {
		t.Errorf("expiry = %v, want %v", got.Expiry, want.Expiry)
	}
}

// The token is a credential: nobody else on the machine gets to read it.
func TestStoreIsPrivate(t *testing.T) {
	s := tempStore(t)
	if err := s.Save(&oauth2.Token{RefreshToken: "refresh"}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("token mode = %#o, want 0600", perm)
	}

	dir, err := os.Stat(filepath.Dir(s.Path()))
	if err != nil {
		t.Fatal(err)
	}
	if perm := dir.Mode().Perm(); perm != 0o700 {
		t.Errorf("config directory mode = %#o, want 0700", perm)
	}
}

// A truncated or hand-edited file should send us back through the browser, not
// crash and not half-load.
func TestStoreTreatsUnusableTokenAsAbsent(t *testing.T) {
	for _, content := range []string{"", "{", `{"access_token":"a"}`} {
		s := tempStore(t)
		if err := os.WriteFile(s.Path(), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		tok, err := s.Load()
		if err != nil {
			t.Errorf("Load(%q) returned %v, want no error", content, err)
		}
		if tok != nil {
			t.Errorf("Load(%q) returned %+v, want nil", content, tok)
		}
	}
}

func TestStoreDeleteIsIdempotent(t *testing.T) {
	s := tempStore(t)
	if err := s.Save(&oauth2.Token{RefreshToken: "refresh"}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.Delete(); err != nil {
			t.Fatalf("Delete: %v", err)
		}
	}
	if tok, _ := s.Load(); tok != nil {
		t.Error("token survived deletion")
	}
}

// A token says nothing about which application asked for it or what it was
// allowed to do, and both change. Keeping one that no longer matches turns into
// a refusal somewhere far away from the cause.
func TestATokenGrantedElsewhereIsNotReused(t *testing.T) {
	tempConfig(t)
	store, err := NewStore()
	if err != nil {
		t.Fatal(err)
	}

	tok := &oauth2.Token{AccessToken: "a", RefreshToken: "r", Expiry: time.Now().Add(time.Hour)}
	if err := store.Save(tok); err != nil {
		t.Fatal(err)
	}
	if got, err := store.Load(); err != nil || got == nil {
		t.Fatalf("Load right after Save = %v, %v — want the token back", got, err)
	}

	// The same token, now that the application has changed.
	if err := SaveClientID("1c227ccd43c64c89918ce162bfc38c7b"); err != nil {
		t.Fatal(err)
	}
	if got, err := store.Load(); err != nil || got != nil {
		t.Errorf("Load after the client id changed = %v, want nothing", got)
	}
}

// The same when a screen starts needing a permission the last authorisation
// never asked for.
func TestATokenMissingAScopeIsNotReused(t *testing.T) {
	tempConfig(t)
	store, err := NewStore()
	if err != nil {
		t.Fatal(err)
	}

	id, err := ClientID()
	if err != nil {
		t.Fatal(err)
	}
	short := grant{
		Token:    &oauth2.Token{AccessToken: "a", RefreshToken: "r", Expiry: time.Now().Add(time.Hour)},
		ClientID: id,
		Scopes:   Scopes[:len(Scopes)-1],
	}
	data, err := json.MarshalIndent(short, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.Path(), data, 0o600); err != nil {
		t.Fatal(err)
	}

	if got, err := store.Load(); err != nil || got != nil {
		t.Errorf("Load of a grant short one scope = %v, want nothing", got)
	}
}
