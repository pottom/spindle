package auth

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestCallbackCapturesTheCode(t *testing.T) {
	server, err := listenForCallback()
	if err != nil {
		t.Skipf("cannot bind %s: %v", callbackAddr, err)
	}
	defer server.close()

	resp, err := http.Get("http://" + callbackAddr + callbackPath + "?code=abc123&state=xyz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := server.wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.code != "abc123" || res.state != "xyz" {
		t.Errorf("got code %q state %q, want abc123 / xyz", res.code, res.state)
	}
}

func TestCallbackReportsRefusal(t *testing.T) {
	server, err := listenForCallback()
	if err != nil {
		t.Skipf("cannot bind %s: %v", callbackAddr, err)
	}
	defer server.close()

	resp, err := http.Get("http://" + callbackAddr + callbackPath + "?error=access_denied&state=xyz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := server.wait(ctx); err == nil {
		t.Fatal("expected an error for a refused authorisation")
	}
}

// Two logins at once would fight over the port, so the second must say so rather
// than silently never hearing a redirect.
func TestCallbackPortIsExclusive(t *testing.T) {
	first, err := listenForCallback()
	if err != nil {
		t.Skipf("cannot bind %s: %v", callbackAddr, err)
	}
	defer first.close()

	if second, err := listenForCallback(); err == nil {
		second.close()
		t.Error("a second callback server started on the same port")
	}
}
