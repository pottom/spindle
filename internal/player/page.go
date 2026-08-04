package player

// Page is one slice of a list too long to read in a single request, together
// with the only thing a caller wants to know once it has read it: whether
// asking again would bring anything.
//
// It carries that rather than a total on purpose. A total is a number none of
// the backends can give honestly — the daemon's /player/context answers with
// the tracks it was asked for and nothing else, and the Web API's totals shift
// under a list while it is being scrolled — whereas "is there more" is
// answerable everywhere and is what the decision to fetch again is made on. A
// caller that wants a count has the one number that is always true: how many
// items it has actually been given.
type Page[T any] struct {
	Items []T

	// More reports that there is at least one further item behind this page. It
	// can be true for a page that came back short: a backend that drops what it
	// cannot play still has whatever follows.
	More bool

	// Next is the offset to ask for to get that further page, and is meaningless
	// when More is false.
	//
	// It is here because the caller cannot work it out. Advancing by the number
	// of items received is right until a page arrives with a podcast episode or
	// a track unavailable in this market in it — both come back as blanks and
	// are dropped — and from there on every request would overlap the one
	// before, re-reading the same tracks and never reaching the end.
	Next int
}
