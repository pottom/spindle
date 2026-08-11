package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/msg"
)

const (
	// callTimeout bounds every backend call so a hung request cannot wedge the loop.
	callTimeout = 10 * time.Second

	// recentMost is how much of the listening history is asked for. Spotify
	// keeps about this many and answers no more.
	recentMost = 50

	// bulkTimeout covers a control that has to read a list before it can act:
	// queueing a whole playlist is a page of it per request.
	bulkTimeout = 60 * time.Second

	// coverTimeout is more generous: it covers a download as well as the resize.
	coverTimeout = 20 * time.Second

	// coverSettleDelay is how long the browse cursor has to rest before its
	// artwork is loaded. Without it, holding a cursor key would queue an upload
	// per row.
	coverSettleDelay = 250 * time.Millisecond

	// scopeInterval is the waveform's frame time. Thirty a second is the point
	// where a trace stops looking like a slideshow.
	scopeInterval = 33 * time.Millisecond

	// A track change does not appear in State() the moment the command returns.
	// Measured against a live account it took 466, 530, 564 and 678 ms — so the
	// 400 ms single shot DESIGN.md guessed at would have confirmed the *old*
	// track every time and left the wrong title up until the next five-second
	// poll.
	//
	// Rather than pick a bigger number and still be wrong on a slow day, ask
	// again until the answer changes: first after confirmFirst, then every
	// confirmRetry, giving up after confirmWindow. That resolves around the
	// median rather than the worst case, and it self-corrects when propagation
	// is slower than anything measured here.
	// The first ask is deliberately just under the fastest propagation seen
	// (466 ms): asking sooner is guaranteed to come back with the old track and
	// only spends a request to learn nothing.
	confirmFirst  = 450 * time.Millisecond
	confirmRetry  = 200 * time.Millisecond
	confirmWindow = 4 * time.Second

	// volumeDebounce is the quiet period after a volume request before another
	// may go out. The first press of a run is not delayed by it — see setVolume.
	volumeDebounce = 400 * time.Millisecond

	// orderDebounce is how long the queue waits after a move key before it
	// sends the new order. Reordering is not a request the device can be given
	// twice a second: each one lifts tracks out of the context, and the next
	// press would be describing a list the device has not agreed to yet. So
	// unlike the volume there is no leading request — the run goes out once,
	// when it is clear where the track was meant to end up.
	orderDebounce = 400 * time.Millisecond

	// playFloor is the shortest gap between two track starts, whichever key
	// asked for one. Each start asks Spotify for an audio key, and asking too
	// fast is answered with refusals that outlast the burst — holding a key
	// down is not a reason to lose the next minute of listening.
	// Measured against a live account: twelve starts a second apart are fine
	// twice over, six 400ms apart are refused, and fourteen at that rate reset
	// the connection. One a second is the shape of it; half again on top is the
	// margin for the ones we do not send.
	playFloor = 1500 * time.Millisecond

	// pageAhead is how close to the end of what is loaded the cursor has to come
	// before the next page is sent for. Far enough that the rows are usually
	// there by the time they are reached, near enough that idly opening a list
	// does not fetch the whole of it.
	pageAhead = 10

	// saidWindow is how long a line about something that just happened stays up.
	// Long enough to read, short enough that it is gone before it is furniture.
	saidWindow = 4 * time.Second

	// unplayableWindow is how long to keep saying that a track was skipped. It
	// is news rather than a state: nothing is wrong now, and the line would
	// otherwise sit there over music that is playing perfectly well.
	unplayableWindow = 6 * time.Second

	// ranOutSlack is how near the start of a track counts as never having
	// begun it. The device rewinds when it runs out of list.
	ranOutSlack = 2 * time.Second
)

// lyricsCmd fetches the words of a track. A failure is reported as a track
// without lyrics: nothing can be done about it, and most tracks that come back
// empty genuinely have none.
func lyricsCmd(source player.LyricSource, trackID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		out := msg.LyricsFetched{TrackID: trackID}
		lyrics, err := source.Lyrics(ctx, trackID)
		if err != nil || lyrics == nil {
			return out
		}
		out.Synced, out.Lines = lyrics.Synced, lyrics.Lines
		return out
	}
}

// scopeFrameCmd waits out a frame and then asks the backend what is being
// heard. The wait and the fetch are one command so the trace paces itself: the
// next frame is only asked for once the last one has arrived, which keeps a
// slow backend from queueing up requests nobody will draw.
func scopeFrameCmd(p player.Player, mode scopeMode) tea.Cmd {
	return func() tea.Msg {
		scopeWait()

		// How long the daemon takes to answer is timed, because a frame is the
		// wait plus this: the picture is only as steady as the answers are. See
		// slow.go.
		asked := time.Now()
		defer func() { slowAsked(time.Since(asked)) }()

		// Only what is being drawn is asked for: the two measurements are
		// independent, and fetching both would double the work for a frame
		// that shows one.
		var out msg.WaveformReady
		if mode.wave() {
			if source, ok := p.(player.Waveform); ok {
				out.Samples = source.Waveform()
			}
		}
		if mode.spectrum() {
			if source, ok := p.(player.Spectrum); ok {
				out.Bands, out.Beat = source.Bands()
			}
		}
		return out
	}
}

// scopePace is when the next frame is due. Package state, because the frames
// are a cadence rather than a property of any one model.
var scopePace struct {
	mu  sync.Mutex
	due time.Time
}

// scopeWait sleeps until the next frame is due, rather than for a whole frame.
//
// The command is issued at the end of an update, so a whole frame's sleep starts
// from after the work rather than from where the last frame did: the period
// becomes the frame plus the fetch plus the update plus the draw. Measured off
// the running interface at 28.1 frames a second against the 30 this is set to —
// two a second going missing, steadily, with nothing reporting them late because
// each one was only two milliseconds over. Sleeping to a deadline puts them back
// on the grid.
func scopeWait() {
	if wait := time.Until(scopeDue(time.Now())); wait > 0 {
		time.Sleep(wait)
	}
}

// scopeDue moves the grid on and hands back when this frame should be drawn.
func scopeDue(now time.Time) time.Time {
	scopePace.mu.Lock()
	defer scopePace.mu.Unlock()

	// A frame late by more than a frame is the picture having been away —
	// stopped, off screen, or the machine busy elsewhere. Carrying the old grid
	// through that would fire the frames it missed back to back to catch up,
	// which is a burst of work for pictures nobody saw. It starts again here.
	if scopePace.due.IsZero() || now.After(scopePace.due.Add(scopeInterval)) {
		scopePace.due = now
	}
	scopePace.due = scopePace.due.Add(scopeInterval)
	return scopePace.due
}

// wordsCmd sets a line in dots, off the update loop: it rasterises type and
// scales an image, which is not work for the loop that draws.
func wordsCmd(lines []string, cellsX, cellsY int) tea.Cmd {
	return func() tea.Msg {
		img, layout, ok := wordsImage(lines, cellsX*dotsPerCellX, cellsY*dotsPerCellY)
		if !ok {
			return nil
		}
		return msg.WordsReady{
			Text:   strings.Join(lines, "\n"),
			CellsX: cellsX, CellsY: cellsY,
			Grain: cover.Grind(grayToImage(img), cellsX, cellsY, dotsPerCellX, dotsPerCellY),
			Words: layout,
		}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return msg.Tick{Time: t}
	})
}

func fetchStateCmd(p player.Player) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		st, err := p.State(ctx)
		if errors.Is(err, player.ErrNoActiveDevice) {
			return msg.NoActiveDevice{}
		}
		var limited *player.RateLimitedError
		if errors.As(err, &limited) {
			return msg.RateLimited{RetryAfter: limited.RetryAfter}
		}
		if err != nil {
			return msg.Error{Err: fmt.Errorf("fetch player state: %w", err)}
		}
		return msg.StateFetched{State: st}
	}
}

func fetchDevicesCmd(p player.Player) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		devices, err := p.Devices(ctx)
		if err != nil {
			return msg.Error{Err: fmt.Errorf("fetch devices: %w", err)}
		}
		return msg.DevicesFetched{Devices: devices}
	}
}

// coverCmd runs the artwork pipeline off the update loop: cache lookup, download,
// decode, resize and render. A failure is reported as a missing cover, not as an
// error banner.
func coverCmd(loader *cover.Loader, url string, wCells, hCells, slot int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), coverTimeout)
		defer cancel()

		art, err := loader.Load(ctx, url, wCells, hCells, slot)
		if cover.IsStale(err) {
			// A newer cover is already on screen. Reporting this one would only
			// give the model something to discard.
			return nil
		}
		if err != nil {
			return msg.CoverFailed{URL: url, Width: wCells, Height: hCells, Slot: slot}
		}
		return msg.CoverReady{URL: url, Width: wCells, Height: hCells, Slot: slot, Art: art}
	}
}

// fetchQueueCmd asks what comes next, so a skip can be shown before Spotify has
// caught up with it. A failure is silent: the queue is an optimisation, and the
// confirming fetch still puts the truth on screen.
func fetchQueueCmd(p player.Player) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		q, err := p.Queue(ctx)
		if err != nil {
			return msg.QueueFetched{}
		}
		out := msg.QueueFetched{Tracks: q.Upcoming}
		if q.Current != nil {
			out.Current = []player.Track{*q.Current}
		}
		return out
	}
}

// fetchLibraryCmd asks for one page of one of the library's lists.
func fetchLibraryCmd(p player.Player, kind libraryKind, offset int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		out := msg.LibraryFetched{Kind: int(kind), Offset: offset}
		switch kind {
		case libraryAlbums:
			page, err := p.SavedAlbums(ctx, offset)
			if err != nil {
				return msg.Error{Err: err}
			}
			out.Albums, out.More, out.Next = page.Items, page.More, page.Next

		case libraryArtists:
			page, err := p.FollowedArtists(ctx, offset)
			if err != nil {
				return msg.Error{Err: err}
			}
			out.Artists, out.More, out.Next = page.Items, page.More, page.Next

		case libraryRecent:
			// The history is asked for whole: it is short by design, and there
			// is no offset to walk it by.
			tracks, err := p.RecentlyPlayed(ctx, recentMost)
			if err != nil {
				return msg.Error{Err: err}
			}
			out.Tracks = tracks

		default:
			page, err := p.PlaylistsPage(ctx, offset)
			if err != nil {
				return msg.Error{Err: err}
			}
			out.Playlists, out.More, out.Next = page.Items, page.More, page.Next
		}
		return out
	}
}

// fetchOpenCmd asks for one page of whatever has been opened. Which call that
// is depends on the kind, and this is the only place that has to know.
func fetchOpenCmd(p player.Player, kind openKind, id string, offset int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		if kind == openArtist {
			page, err := p.ArtistAlbums(ctx, id, offset)
			if err != nil {
				return msg.Error{Err: err}
			}
			return msg.OpenedFetched{
				ID: id, Albums: page.Items, Offset: offset, More: page.More, Next: page.Next,
			}
		}

		page, err := trackPage(ctx, p, kind, id, offset)
		if err != nil {
			return msg.Error{Err: err}
		}
		return msg.OpenedFetched{
			ID: id, Tracks: page.Items, Offset: offset, More: page.More, Next: page.Next,
		}
	}
}

// trackPage is one page of the tracks a thing holds: an album's, a playlist's,
// or the account's saved ones. Everything above this reads the three the same
// way, which is the point of carrying liked songs as a playlist at all.
func trackPage(ctx context.Context, p player.Player, kind openKind, id string, offset int) (player.Page[player.Track], error) {
	switch {
	case kind == openAlbum:
		return p.AlbumTracks(ctx, id, offset)
	case isLiked(id):
		return p.LikedTracks(ctx, offset)
	default:
		return p.PlaylistTracksPage(ctx, id, offset)
	}
}

// listTrackIDs reads a playlist or an album through to its end, or to the cap,
// whichever comes first.
func listTrackIDs(ctx context.Context, p player.Player, kind openKind, id string) ([]string, error) {
	var ids []string
	for offset := 0; len(ids) < enqueueMost; {
		page, err := trackPage(ctx, p, kind, id, offset)
		if err != nil {
			return nil, err
		}
		for _, t := range page.Items {
			if len(ids) == enqueueMost {
				break
			}
			ids = append(ids, t.ID)
		}
		if !page.More || page.Next <= offset {
			break
		}
		offset = page.Next
	}
	return ids, nil
}

// searchCmd asks for one kind, or for every kind when the kind is empty — which
// is what a fresh query does, since Spotify answers them all in one request.
func searchCmd(p player.Player, query string, kind player.SearchKind, seq, offset int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		res, err := p.Search(ctx, query, kind, offset)
		if err != nil {
			return msg.Error{Err: err}
		}
		return msg.SearchResults{
			Seq: seq, Query: query, Kind: kind, Matched: true,
			Offset: offset, Results: res,
		}
	}
}

// coverSettleCmd waits out the debounce before an artwork preview is loaded.
func coverSettleCmd(seq int) tea.Cmd {
	return tea.Tick(coverSettleDelay, func(time.Time) tea.Msg {
		return msg.CoverSettled{Seq: seq}
	})
}

// controlCmd runs a playback control call off the update loop. The result is
// classified here so the UI can tell "Spotify is throttling us" and "this account
// cannot do that" apart from an ordinary failure.
// playCmd is controlCmd for a request to start something. It reports when the
// call has been answered, so the next request can go out after it rather than
// alongside it.
func playCmd(action string, call func(context.Context) error) tea.Cmd {
	inner := controlCmd(action, call)
	return func() tea.Msg { return msg.PlayDone{Result: inner()} }
}

func controlCmd(action string, call func(context.Context) error) tea.Cmd {
	return controlWithin(action, callTimeout, call)
}

// controlWithin is the same for the few controls that are not one request: a
// whole playlist has to be read a page at a time before it can be queued, and
// the timeout that suits a play or a pause would cut that off halfway.
func controlWithin(action string, within time.Duration, call func(context.Context) error) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), within)
		defer cancel()

		err := call(ctx)
		if err == nil {
			return msg.ControlDone{}
		}

		var limited *player.RateLimitedError
		if errors.As(err, &limited) {
			return msg.RateLimited{RetryAfter: limited.RetryAfter}
		}
		if errors.Is(err, player.ErrNoActiveDevice) {
			return msg.NoActiveDevice{}
		}
		return msg.Error{Err: fmt.Errorf("%s: %w", action, err)}
	}
}

// watchCmd waits for the backend to report a change. It re-arms itself on every
// wake-up, so the UI follows the daemon without polling it at all.
func watchCmd(w player.Watcher) tea.Cmd {
	return func() tea.Msg {
		<-w.Changes()
		return msg.StateChanged{}
	}
}

// refetchCmd asks for a fresh state after a delay. Spotify reports the old track
// for a moment after a skip, so an immediate poll would confirm the wrong thing.
func refetchCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return msg.Refetch{} })
}

// orderSettleCmd waits out the queue's move debounce.
func orderSettleCmd(seq int) tea.Cmd {
	return tea.Tick(orderDebounce, func(time.Time) tea.Msg {
		return msg.OrderSettled{Seq: seq}
	})
}

// playFloorCmd waits out the gap between two track starts, and then reports as
// an answer would, so the pending request goes out through the same door.
func playFloorCmd() tea.Cmd {
	return tea.Tick(playFloor, func(time.Time) tea.Msg { return msg.PlayDone{} })
}

// volumeSettleCmd waits out the volume debounce.
func volumeSettleCmd(seq int) tea.Cmd {
	return tea.Tick(volumeDebounce, func(time.Time) tea.Msg {
		return msg.VolumeSettled{Seq: seq}
	})
}
