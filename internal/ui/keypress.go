package ui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Which key was pressed, on a keyboard that is not the one the keys were named
// for.
//
// Every letter in this program is a letter on a US keyboard: c folds the queue,
// v turns the picture, f puts the big screen up. On a Hungarian layout the key
// where z sits sends y and the key where y sits sends z, so half the pairs
// somebody learned are the wrong way round — and a binding written as "z" is
// pressed by a key with a different letter on its cap.
//
// The terminal can say which key it really was. The kitty keyboard protocol
// reports, beside the character, the key as a standard US keyboard would have
// had it, and Bubble Tea hands that over as Key.BaseCode. So a binding is asked
// twice: once about the letter that arrived, and once about the key it came
// from.
//
// Both, rather than only the second. Somebody who has remapped their keyboard on
// purpose means the letters they typed; and where nothing reports a base key —
// a terminal without the protocol, an older Windows console — the second
// question cannot be asked at all and this is the ordinary match and nothing
// else.

// pressed reports whether a press is one of these bindings, by the letter that
// arrived or by the key it came from.
func (m Model) pressed(k tea.KeyPressMsg, b ...key.Binding) bool {
	if key.Matches(k, b...) {
		return true
	}
	base, ok := baseKey(k)
	return ok && key.Matches(base, b...)
}

// baseKey is the press as a US keyboard would have sent it, and whether the
// terminal said anything about that at all.
//
// The text goes with the code. Bubble Tea matches a printable key by the text it
// produced, so a press whose code was moved and whose text was not would be
// matched as the letter that arrived — which is the question already asked.
func baseKey(k tea.KeyPressMsg) (tea.KeyPressMsg, bool) {
	if k.BaseCode == 0 || k.BaseCode == k.Code {
		return k, false
	}
	base := k
	base.Code = k.BaseCode
	if k.Text != "" {
		base.Text = string(k.BaseCode)
	}
	return base, true
}
