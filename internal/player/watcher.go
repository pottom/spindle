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
