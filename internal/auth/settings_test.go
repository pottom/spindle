package auth

import (
	"errors"
	"os"
	"testing"
)

func tempConfig(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(clientIDEnv, "")
}

func TestClientIDRoundTrip(t *testing.T) {
	tempConfig(t)

	// A fresh config authenticates as the application spindle ships with, which
	// is the whole point of shipping one.
	if got, err := ClientID(); err != nil || got != DefaultClientID {
		t.Fatalf("ClientID on a fresh config = %q, %v — want the default", got, err)
	}

	const id = "1c227ccd43c64c89918ce162bfc38c7b"
	if err := SaveClientID(id); err != nil {
		t.Fatal(err)
	}

	got, err := ClientID()
	if err != nil {
		t.Fatal(err)
	}
	if got != id {
		t.Errorf("ClientID = %q, want %q", got, id)
	}
}

// Catching the shape here is what turns a pasted secret or a truncated copy
// into an explanation, instead of an opaque rejection halfway through the
// browser flow.
func TestSaveClientIDRejectsNonsense(t *testing.T) {
	tempConfig(t)

	bad := map[string]string{
		"empty":           "",
		"too short":       "1c227ccd43c64c89918ce162bfc38c7",
		"too long":        "1c227ccd43c64c89918ce162bfc38c7bb",
		"uppercase":       "1C227CCD43C64C89918CE162BFC38C7B",
		"a whole url":     "https://developer.spotify.com/dashboard/1c227ccd43c64c89918ce162bfc38c7b",
		"with whitespace": " 1c227ccd43c64c89918ce162bfc38c7b",
		"not hexadecimal": "zc227ccd43c64c89918ce162bfc38c7b",
	}
	for name, id := range bad {
		if err := SaveClientID(id); !errors.Is(err, ErrMalformedClientID) {
			t.Errorf("SaveClientID(%s) = %v, want ErrMalformedClientID", name, err)
		}
	}
}

// The environment wins, so a second application can be tried without disturbing
// what is saved.
func TestEnvironmentOverridesSettings(t *testing.T) {
	tempConfig(t)
	if err := SaveClientID("1c227ccd43c64c89918ce162bfc38c7b"); err != nil {
		t.Fatal(err)
	}

	const other = "aaaabbbbccccddddeeeeffff00001111"
	t.Setenv(clientIDEnv, other)

	got, err := ClientID()
	if err != nil {
		t.Fatal(err)
	}
	if got != other {
		t.Errorf("ClientID = %q, want the environment's %q", got, other)
	}
}

func TestMalformedEnvironmentIsReported(t *testing.T) {
	tempConfig(t)
	t.Setenv(clientIDEnv, "nonsense")

	if _, err := ClientID(); !errors.Is(err, ErrMalformedClientID) {
		t.Errorf("ClientID = %v, want ErrMalformedClientID", err)
	}
}

// A settings file that cannot be read falls back to the shipped application,
// rather than refusing to start over a stray character.
func TestUnreadableSettingsAreTreatedAsAbsent(t *testing.T) {
	tempConfig(t)
	if err := SaveClientID("1c227ccd43c64c89918ce162bfc38c7b"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(SettingsPath(), []byte("{oh dear"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got, err := ClientID(); err != nil || got != DefaultClientID {
		t.Errorf("ClientID = %q, %v — want the default", got, err)
	}
}

func TestSettingsArePrivate(t *testing.T) {
	tempConfig(t)
	if err := SaveClientID("1c227ccd43c64c89918ce162bfc38c7b"); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(SettingsPath())
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("settings mode = %#o, want 0600", perm)
	}
}

// The client id and the quality live in the same file, and neither may erase
// the other: they are set at different moments, by different commands.
func TestSettingsFieldsSurviveEachOther(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	const id = "0123456789abcdef0123456789abcdef"
	if err := SaveClientID(id); err != nil {
		t.Fatalf("SaveClientID: %v", err)
	}
	if err := SaveQuality("middle"); err != nil {
		t.Fatalf("SaveQuality: %v", err)
	}

	got, err := loadClientID()
	if err != nil || got != id {
		t.Errorf("loadClientID() = %q, %v, want the id back", got, err)
	}

	q, err := Quality()
	if err != nil || q != "middle" {
		t.Errorf("Quality() = %q, %v, want middle", q, err)
	}

	// And setting the id again keeps the quality.
	if err := SaveClientID(id); err != nil {
		t.Fatalf("SaveClientID: %v", err)
	}
	if q, _ := Quality(); q != "middle" {
		t.Errorf("Quality() = %q after re-saving the id, want middle", q)
	}
}
