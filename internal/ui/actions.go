package ui

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

// actionsPane is the menu of what can be done with whatever the cursor is on.
//
// It exists because a player accumulates verbs faster than it has keys for
// them, and a key that works on only one screen is a key nobody finds. One menu
// listing exactly what is possible for the thing under the cursor is easier to
// learn than six bindings, and it is the only place a verb can explain itself.
type actionsPane struct {
	open  bool
	verbs []verb
	state listState

	// title and subtitle are what the menu was opened over, as it is named at
	// the head of the menu. Held rather than looked up again: the list
	// underneath can be replaced by a poll while the menu is up, and a menu
	// must go on describing what was chosen, not whatever has since slid under
	// the cursor. The verbs hold their own subject for the same reason.
	title, subtitle string
}

// verb is one line of the menu.
type verb struct {
	// key is the shortcut that does the same thing without the menu, shown so
	// the menu teaches its own way out. Empty where there is none.
	key   string
	label string
	do    func(m *Model) tea.Cmd
}

// actionsFor is the menu for a track, on the screen it was opened from. The
// list differs by screen because the verbs do: a track waiting in the queue can
// be taken out of it, and one found by searching cannot.
func (m Model) actionsFor(t player.Track) []verb {
	id := t.ID
	verbs := []verb{{
		key:   "o",
		label: "Play now, keeping the queue",
		do: func(m *Model) tea.Cmd {
			return m.startPlay(playRequest{
				action: "play track",
				track:  t,
				call:   func(ctx context.Context, p player.Player) error { return p.PlayNow(ctx, id) },
			})
		},
	}}

	if m.tab == tabQueue {
		verbs = append(verbs, verb{
			key:   "x",
			label: "Remove from the queue",
			do:    func(m *Model) tea.Cmd { return m.dropQueued() },
		})
	} else {
		verbs = append(verbs, verb{
			key:   "a",
			label: "Add to the queue",
			do:    func(m *Model) tea.Cmd { return m.enqueue(id) },
		})
	}

	// Where the track came from, if the backend said. The daemon's own queue
	// answers with names and no ids, so the verb appears where it can act and
	// stays away where it cannot.
	if t.AlbumID != "" {
		album := player.Album{ID: t.AlbumID, Name: t.Album, Artists: t.Artists, CoverURL: t.CoverURL, Released: t.Released}
		verbs = append(verbs, verb{
			label: "Go to the album",
			do:    func(m *Model) tea.Cmd { return m.push(openedAlbum(album)) },
		})
	}
	if len(t.ArtistIDs) > 0 {
		artist := player.Artist{ID: t.ArtistIDs[0], Name: t.Artists[0]}
		verbs = append(verbs, verb{
			label: "Go to " + artist.Name,
			do:    func(m *Model) tea.Cmd { return m.push(openedArtist(artist)) },
		})
	}

	return append(verbs, verb{
		label: "Copy the Spotify link",
		do:    func(m *Model) tea.Cmd { return m.copyLink(trackLink(id)) },
	})
}

// actionsForAlbum is the menu for a record: what a whole record can be.
func (m Model) actionsForAlbum(a player.Album) []verb {
	album, id := a, a.ID
	verbs := []verb{{
		key:   "enter",
		label: "Open it",
		do:    func(m *Model) tea.Cmd { return m.push(openedAlbum(album)) },
	}, {
		label: "Play it from the top",
		do: func(m *Model) tea.Cmd {
			return m.playContext("play album", player.AlbumURI(id), player.Track{
				Title: album.Name, Artists: album.Artists, CoverURL: album.CoverURL,
			})
		},
	}, {
		key:   "a",
		label: "Add all of it to the queue",
		do:    func(m *Model) tea.Cmd { return m.enqueueList(openAlbum, id, album.Name) },
	}}

	return append(verbs, verb{
		label: "Copy the Spotify link",
		do:    func(m *Model) tea.Cmd { return m.copyLink(albumLink(id)) },
	})
}

// actionsForArtist is the menu for an artist. Their records are what spindle
// can list — what they are best known for is an endpoint closed to this
// application — so opening them means their releases, and playing them is left
// to Spotify's own idea of the artist.
func (m Model) actionsForArtist(a player.Artist) []verb {
	artist, id := a, a.ID
	return []verb{{
		key:   "enter",
		label: "Open their records",
		do:    func(m *Model) tea.Cmd { return m.push(openedArtist(artist)) },
	}, {
		label: "Play what Spotify puts first",
		do: func(m *Model) tea.Cmd {
			return m.playContext("play artist", player.ArtistURI(id), player.Track{
				Title: artist.Name, CoverURL: artist.ImageURL,
			})
		},
	}, {
		label: "Copy the Spotify link",
		do:    func(m *Model) tea.Cmd { return m.copyLink(artistLink(id)) },
	}}
}

// actionsForPlaylist is the menu for a whole playlist, which is what the cursor
// is on at the library's top level and among the search's playlists.
//
// A playlist has no verb in common with a track: what can be done to it is done
// to all of it at once, and that is exactly why it wants a menu — nothing else
// on the screen says that the list under the cursor can be started without
// being opened first.
func (m Model) actionsForPlaylist(pl player.Playlist) []verb {
	id, name := pl.ID, pl.Name
	verbs := []verb{{
		label: "Play it from the top",
		do: func(m *Model) tea.Cmd {
			return m.startPlay(playRequest{
				action: "play " + strings.ToLower(name),
				call: func(ctx context.Context, p player.Player) error {
					if !isLiked(id) {
						return p.PlayPlaylist(ctx, id, 0)
					}
					// Nothing to name, so the list is read and handed over.
					ids, err := listTrackIDs(ctx, p, openPlaylist, id)
					if err != nil {
						return err
					}
					return p.PlayTracks(ctx, ids, 0)
				},
			})
		},
	}, {
		key:   "a",
		label: "Add all of it to the queue",
		do:    func(m *Model) tea.Cmd { return m.enqueueList(openPlaylist, id, name) },
	}}

	// The saved tracks are the account's own and have no address anybody else
	// can open, so there is nothing to copy.
	if !isLiked(id) {
		verbs = append(verbs, verb{
			label: "Copy the Spotify link",
			do:    func(m *Model) tea.Cmd { return m.copyLink(playlistLink(id)) },
		})
	}
	return verbs
}

// openActions raises the menu over whatever the cursor is on, and reports
// whether there was anything to raise it over.
func (m *Model) openActions() bool {
	if a := m.cursorAlbum(); a != nil {
		m.actions = actionsPane{open: true, title: a.Name, subtitle: albumLine(*a)}
		m.actions.verbs = m.actionsForAlbum(*a)
		return len(m.actions.verbs) > 0
	}

	if a := m.cursorArtist(); a != nil {
		m.actions = actionsPane{open: true, title: a.Name, subtitle: artistLine(*a)}
		m.actions.verbs = m.actionsForArtist(*a)
		return len(m.actions.verbs) > 0
	}

	if pl := m.cursorPlaylist(); pl != nil {
		m.actions = actionsPane{open: true, title: pl.Name, subtitle: playlistOwner(*pl)}
		m.actions.verbs = m.actionsForPlaylist(*pl)
		return len(m.actions.verbs) > 0
	}

	t := m.cursorTrack()
	if t == nil {
		return false
	}

	m.actions = actionsPane{open: true, title: t.Title, subtitle: strings.Join(t.Artists, ", ")}
	m.actions.verbs = m.actionsFor(*t)
	return len(m.actions.verbs) > 0
}

// actionsKey drives the menu while it is up. It answers every key, so nothing
// underneath can act on a screen that is not the one being looked at.
func (m *Model) actionsKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	if !m.actions.open {
		return nil, false
	}

	switch {
	case key.Matches(k, m.keys.Back), key.Matches(k, m.keys.Actions):
		m.actions.open = false
		return nil, true

	case key.Matches(k, m.keys.Enter):
		chosen := m.actions.state.cursor
		m.actions.open = false
		if chosen < 0 || chosen >= len(m.actions.verbs) {
			return nil, true
		}
		return m.actions.verbs[chosen].do(m), true
	}

	m.listKey(k, &m.actions.state, len(m.actions.verbs), true)
	return nil, true
}

// actionsBlock draws the menu in place of the list it was opened over, rather
// than floating over it: this screen has no boxes anywhere, and a panel on top
// of a list would need one to be legible.
func (m Model) actionsBlock(w, rows int) []string {
	lines := []string{
		fit(m.styles.Title.Render(m.actions.title), w),
		fit(m.styles.Artist.Render(m.actions.subtitle), w),
		"",
	}

	for i, v := range m.actions.verbs {
		style, gutter := m.styles.RowPrimary, "  "
		if i == m.actions.state.cursor {
			style, gutter = m.styles.RowSelected, m.styles.Cursor.Render(rowCursor)+" "
		}

		shortcut := "  "
		if v.key != "" {
			shortcut = m.styles.FactLabel.Render(v.key) + " "
		}
		lines = append(lines, fit(gutter+shortcut+style.Render(v.label), w))
	}
	return stack(lines, w, rows)
}

// playlistOwner is what the menu says under a playlist's name: whose it is, and
// how much of it there is, because "add all of it" is worth being able to price.
func playlistOwner(p player.Playlist) string {
	if p.Tracks > 0 {
		return fmt.Sprintf("%s · %d tracks", p.Owner, p.Tracks)
	}
	return p.Owner
}

// trackLink and playlistLink are the addresses a thing has outside Spotify's
// own applications.
func trackLink(id string) string    { return "https://open.spotify.com/track/" + id }
func playlistLink(id string) string { return "https://open.spotify.com/playlist/" + id }
func albumLink(id string) string    { return "https://open.spotify.com/album/" + id }
func artistLink(id string) string   { return "https://open.spotify.com/artist/" + id }

// copyLink puts text on the clipboard through the terminal itself.
//
// OSC 52 rather than a clipboard library: spindle may be running over ssh or
// inside tmux, where the clipboard that matters belongs to the terminal at the
// far end and nothing on this machine can reach it. Terminals that do not
// support the sequence ignore it, which is why the line that follows says what
// was copied rather than that it worked.
func (m *Model) copyLink(text string) tea.Cmd {
	m.said, m.saidAt = "Copied "+text, time.Now()
	return func() tea.Msg {
		fmt.Fprintf(os.Stdout, "\x1b]52;c;%s\x07", base64.StdEncoding.EncodeToString([]byte(text)))
		return nil
	}
}
