package player

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Lyrics asks the daemon for the words of a track.
func (l *Local) Lyrics(ctx context.Context, trackID string) (*Lyrics, error) {
	if trackID == "" {
		return nil, nil
	}

	target := l.addr + "/player/lyrics?uri=" + url.QueryEscape(trackURI(trackID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("build lyrics request: %w", err)
	}

	resp, err := l.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch lyrics: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only request

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch lyrics: unexpected status %s", resp.Status)
	}

	var out struct {
		Synced   bool   `json:"synced"`
		Language string `json:"language"`
		Lines    []struct {
			At    int64  `json:"at"`
			Words string `json:"words"`
		} `json:"lines"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode lyrics: %w", err)
	}
	if len(out.Lines) == 0 {
		return nil, nil
	}

	lyrics := &Lyrics{Synced: out.Synced, Language: out.Language, Lines: make([]Lyric, 0, len(out.Lines))}
	for _, line := range out.Lines {
		lyrics.Lines = append(lyrics.Lines, Lyric{At: line.At, Words: line.Words})
	}
	return lyrics, nil
}
