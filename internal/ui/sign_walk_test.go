package ui

import "testing"

// The signer faces the way he travels, and carries the placard in front of him.
//
// He crosses right to left, and he is drawn walking left: in walk3 the left leg
// is out in front with the heel down and the right is behind pushing off. He was
// mirrored on the assumption that a sheet is drawn facing right, which is true
// of the dancers and was never true of him — and mirroring keeps the travel
// while reversing the stride, so what crossed the screen was a moonwalk.
//
// Nothing in his face would have caught it: it is a circle with two eyes looking
// straight out, the same either way round. The placard is the only part of him
// that says which way is forward, so that is what this measures — it has to sit
// on the leading side, which going leftwards means left of his middle.
func TestTheSignerWalksTheWayHeGoes(t *testing.T) {
	who, ok := figureFor(figureSigner)
	if !ok {
		t.Fatal("no signer figure")
	}
	for _, tall := range []int{62, 100, 140} {
		for _, frame := range []string{"walk0", "walk1", "walk2", "walk3"} {
			pose, ok := who.at(tall, frame)
			if !ok {
				t.Fatalf("%s at %d: no pose", frame, tall)
			}
			slot := signSlotFor(pose)
			if slot.wide <= 0 {
				t.Fatalf("%s at %d: no blank found", frame, tall)
			}
			sign := slot.left + slot.wide/2
			him := pose.wide / 2
			if sign >= him {
				t.Errorf("%s at %d: the placard is centred at %d of %d, behind him — he is walking backwards",
					frame, tall, sign, pose.wide)
			}
		}
	}
}
