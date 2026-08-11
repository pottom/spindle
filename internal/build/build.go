// Package build says which spindle this is.
//
// It exists because of a morning spent arguing with a screen. A picture was
// wrong, it was fixed, and it went on being wrong — because the binary being
// run had been built before the fix, and nothing on screen or on the command
// line could tell one build from another. A version that names the commit
// settles that in a second.
//
// Nothing has to be passed at build time. The toolchain stamps the revision and
// whether the tree was dirty into every binary built from a repository, and
// that is read back here.
package build

import (
	"runtime/debug"
	"strings"
)

// release is the version this source is, without regard to what was built from
// it. Bumped by hand, and only meaningful next to the revision.
const release = "v0.0.1"

// Version is what to print: the release, the commit it was built from, and a
// mark when the tree had uncommitted changes in it.
//
// Built outside a repository — from a module cache, or with the stamping turned
// off — there is no revision to have, and the release alone comes back rather
// than a lie about which commit it is.
func Version() string {
	rev, dirty := revision()
	if rev == "" {
		return release
	}
	out := release + "+" + rev
	if dirty {
		out += "-dirty"
	}
	return out
}

func revision() (rev string, dirty bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	// Short, because the whole hash is forty characters of nothing anybody
	// reads, and eight is more than this repository will ever need to be
	// unambiguous.
	if len(rev) > 8 {
		rev = rev[:8]
	}
	return strings.TrimSpace(rev), dirty
}
