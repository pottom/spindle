package player

// Watcher is an optional capability: a backend that knows when something has
// changed instead of waiting to be asked.
//
// It is deliberately not part of Player. Only the local daemon can do this —
// the Web API has no push of any kind — and making every backend pretend to
// would mean the mock and the Web API growing channels that never fire.
type Watcher interface {
	// Changes yields a value whenever the playback state may have moved. It is
	// a signal, not the state: the reader calls State to find out what changed.
	// Sends are dropped rather than blocked on, so a slow reader falls behind
	// by coalescing rather than by holding the backend up.
	Changes() <-chan struct{}
}

// Owner is implemented by a backend that has a playback device of its own —
// the local daemon, and nothing else. The interface uses it to take a device it
// started itself, rather than making somebody pick their own machine out of a
// list of speakers.
type Owner interface {
	// OwnDevice is the Connect device this backend is, or empty before it has
	// registered with Spotify. It takes a few seconds after the daemon starts.
	OwnDevice() string
}
