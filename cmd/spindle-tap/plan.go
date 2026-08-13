package main

import (
	"fmt"
	"time"
)

// What to listen to, in what order, and what to listen for on each.

// The three kinds of pass.
const (
	passStarts = "lines"
	passEnds   = "bounds"
	passWords  = "words"
)

// pass is one thing to listen for, and how much of it is enough.
type pass struct {
	kind  string
	asks  string // what to do, in the imperative, on the panel
	keys  string // the keys that mean anything during it
	goal  int    // lines, after which the pass moves on by itself
	tempo string //nolint:unused // reserved for a pass that needs its own note
}

var (
	starts = pass{passStarts, "SPACE THE MOMENT EACH LINE STARTS", "space  a line starts", 12, ""}
	ends   = pass{passEnds, "SPACE WHEN A LINE STARTS, E WHEN ITS SINGING STOPS",
		"space  a line starts     e  its singing stops", 12, ""}
	every = pass{passWords, "SPACE ON EVERY WORD, IN ORDER",
		"space  this word     enter  skip a word you missed", 8, ""}
)

// work is one record to listen to, and what to listen for on it.
type work struct {
	id     string
	artist string
	title  string
	why    string

	// from is where in the record to start. Zero is the top, and the top is
	// not always where the interesting singing is: the same record can hold a
	// verse at two syllables a second and a rap at eight, and it is the second
	// that says whether the words in a line change how long it takes to sing.
	from   time.Duration
	passes []pass
}

// slug names the file a pass writes to, so two sections of one record do not
// overwrite each other.
func (w work) slug() string {
	if w.from == 0 {
		return w.id
	}
	return fmt.Sprintf("%s@%.0f", w.id, w.from.Seconds())
}

// plan is the running order, worst first.
//
// Worst by measurement rather than by taste: each of these was ranked by how far
// its syllables fall from a musical value and how often the model would overrun
// a line. The two at the end are the controls — the model says they are right,
// and if they turn out wrong then the fault is the model's rather than the
// estimate's. See FINDINGS.md.
var plan = []work{
	{"3Fov51HzsiwqhoyR0ou6yk", "Tony Joe White", "The Other Side",
		"the worst fit there is: 41% off a musical value, and the two lyric sources disagree by 645 ms",
		0, []pass{starts, ends, every}},
	{"0HNXnhNZZtRZFLhP4hfyCP", "DJ Ötzi", "Hey Baby (Radio Mix)",
		"33% off, and the model claims its lines stop earliest of all — 60% of the bar",
		0, []pass{starts, ends}},
	{"4CNCUed2qq47dwKlvkBNRI", "Majka", "Mindenki táncol /90'/ — the rap",
		"the same record as the slow verse already tapped, at three times the syllables: the one comparison that needs no second singer",
		92 * time.Second, []pass{starts, ends}},
	{"3MhdH8PxqH1FuQp3HBptUI", "Sean Paul", "I'm Still in Love with You",
		"a control: exactly half a beat to the syllable, so this one should hold",
		0, []pass{starts, ends}},
	{"7eHc5hK0b2VxQ1USUZAhK9", "Bohemian Betyars", "Ellentétek balladája",
		"41% off, Hungarian, and its tempo will not sit still",
		0, []pass{starts, ends}},
	{"0pMUR7Uvp6vxlbG0qBFvgM", "Alice Deejay", "Better Off Alone",
		"few words and a lot of music: the class the estimate can never get right",
		0, []pass{starts, ends}},
	{"2UB8LkhAtBgL9cCxKY2qwi", "Carson Coma", "Immunissá válunk",
		"the other control: half a beat, and the lowest overrun of the lot",
		0, []pass{starts, ends}},
}
