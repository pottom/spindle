package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// callbackPage is what the browser is left showing. No styling worth the name:
// it exists for two seconds and then the tab gets closed.
const callbackPage = `<!doctype html>
<meta charset="utf-8">
<title>spindle</title>
<body style="font-family: system-ui, sans-serif; margin: 4rem auto; max-width: 26rem">
<h1 style="font-size: 1.1rem; font-weight: 600">%s</h1>
<p style="color: #666">%s</p>
`

// callbackResult is what came back on the redirect.
type callbackResult struct {
	code  string
	state string
	err   error
}

// callbackServer listens for Spotify's redirect on the loopback address.
type callbackServer struct {
	srv     *http.Server
	results chan callbackResult
}

// listenForCallback starts the server. The listener is opened eagerly so a port
// already in use is reported before a browser window is thrown at the user.
func listenForCallback() (*callbackServer, error) {
	addr := callbackAddr()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// Named, because the two causes want different answers: another login in
		// flight wants waiting, and something else on the port wants moving one
		// of the two. 8888 was the old default and is one of the most contested
		// numbers there is, so the second is the likelier of them.
		return nil, fmt.Errorf("listen on %s: %w — another spindle login may be running, "+
			"or something else has that port. Move it with `spindle callback <port>`, "+
			"and add the new address to the Spotify application", addr, err)
	}

	c := &callbackServer{results: make(chan callbackResult, 1)}
	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, c.handle)
	c.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	go c.srv.Serve(ln) //nolint:errcheck // Serve always ends in ErrServerClosed here
	return c, nil
}

func (c *callbackServer) handle(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	res := callbackResult{code: q.Get("code"), state: q.Get("state")}

	if reason := q.Get("error"); reason != "" {
		res.err = fmt.Errorf("authorisation refused: %s", reason)
	} else if res.code == "" {
		res.err = fmt.Errorf("authorisation returned no code")
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if res.err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, callbackPage, "Sign-in failed", res.err.Error())
	} else {
		fmt.Fprintf(w, callbackPage, "Signed in", "You can close this window and go back to the terminal.")
	}

	// Only the first redirect counts; the channel is buffered for exactly one.
	select {
	case c.results <- res:
	default:
	}
}

// wait blocks until the redirect arrives or the context is done.
func (c *callbackServer) wait(ctx context.Context) (callbackResult, error) {
	select {
	case res := <-c.results:
		return res, res.err
	case <-ctx.Done():
		return callbackResult{}, ctx.Err()
	}
}

// close shuts the server down, giving in-flight responses a moment to finish so
// the browser is not left with a dead connection.
func (c *callbackServer) close() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	c.srv.Shutdown(ctx) //nolint:errcheck // nothing useful to do on failure
}
