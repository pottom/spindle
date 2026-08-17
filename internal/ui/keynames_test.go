package ui

import (
	"reflect"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
)

// Every key a screen names is a key something reads.
//
// This is the check the constants cannot make on their own. The bar once
// offered t for the full-screen visualiser, which is on f: the letter was
// invented while the list was being written out, and nothing said so. Somebody
// reading a hint bar presses what it says, so a name with nothing behind it is
// the one kind of mistake a hint bar must not make.
//
// Only what is plainly a key is checked: a single character, a control chord,
// or one of the keys spelled out by name. The glyphs (← ↑ ⇞), the caret
// shorthand (^d) and the words that are instructions rather than keys ("type")
// are pictures of keys for a reader, not names this can look up.
func TestEveryKeyNamedIsAKeyRead(t *testing.T) {
	k := newKeyMap()
	read := keysRead(k)

	bars := map[string]tabKeys{
		"no device":    k.forNoDevice(),
		"devices":      k.forDevices(),
		"read-only":    k.forReadOnlyQueue(true),
		"player":       k.forPlayer(true, true, true, true, true, 200),
		"player bare":  k.forPlayer(false, false, false, false, false, 200),
		"open album":   k.forOpen(true, true, true),
		"open list":    k.forOpen(false, false, true),
		"open unliked": k.forOpen(false, false, false),
	}
	for tab := tabID(0); tab < tabCount; tab++ {
		bars["tab "+tab.String()] = k.forTab(tab, true, true, true)
	}

	for where, bar := range bars {
		bindings := append([]key.Binding{}, bar.short...)
		for _, row := range bar.full {
			bindings = append(bindings, row...)
		}
		for _, b := range bindings {
			for _, name := range namedKeys(b.Help().Key) {
				if !read[name] {
					t.Errorf("%s says %q (%q), which nothing reads", where, name, b.Help().Key)
				}
			}
		}
	}

	// The help screen writes its keys out by hand — it is answering "what is
	// this screen for", which the bindings do not know — so it can drift the
	// same way, and is measured the same way.
	for _, g := range helpGroups() {
		for _, row := range g.keys {
			for _, name := range namedKeys(row[0]) {
				if !read[name] {
					t.Errorf("the help's %q says %q, which nothing reads", g.title, name)
				}
			}
		}
	}
}

// keysRead is every key the bindings answer to.
func keysRead(k keyMap) map[string]bool {
	out := map[string]bool{}
	v := reflect.ValueOf(k)
	for i := 0; i < v.NumField(); i++ {
		b, ok := v.Field(i).Interface().(key.Binding)
		if !ok {
			continue
		}
		for _, s := range b.Keys() {
			out[s] = true
		}
	}
	// The two that belong to no screen's key map: the full screen answers any
	// key at all with a way out, and the debug bar is not advertised anywhere.
	out[keyDebug] = true
	return out
}

// namedKeys picks the keys out of the way a hint spells them.
func namedKeys(help string) []string {
	var out []string
	for _, tok := range strings.FieldsFunc(help, func(r rune) bool { return r == '/' || r == ' ' }) {
		if isKeyName(tok) {
			out = append(out, tok)
		}
	}
	return out
}

// spelledOut is the keys a hint names in words rather than by their character.
// Anything else of more than one character — "type", "pgdn", "spindle" — is a
// reader's shorthand and not a name to look up.
var spelledOut = map[string]bool{
	"space": true, "enter": true, "esc": true, "tab": true,
	"home": true, "end": true, "up": true, "down": true,
	"left": true, "right": true, "shift+tab": true,
}

func isKeyName(tok string) bool {
	if spelledOut[tok] || strings.HasPrefix(tok, "ctrl+") {
		return true
	}
	// A single character, and an ASCII one: the arrows and the page glyphs are
	// drawings of keys, not the names of any.
	return len(tok) == 1 && tok[0] > ' ' && tok[0] < 0x7f
}
