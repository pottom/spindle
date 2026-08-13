// Package bones is a dance written as joint positions, and the figure drawn
// from them.
//
// A move drawn as pictures has to be cut out of a sheet, and every frame of it
// is drawn in a slightly different place: measured on the seven sheets that came
// first, the figure wandered half a cell between one frame and the next, so his
// head jumped about while he danced. A move written as numbers cannot do that —
// there is nothing to align, because there is nothing that was ever apart.
//
// What arrives is eleven points a frame. What is drawn from them is ours: our
// pen, at any size, at any rate. The frames are keyframes rather than pictures,
// and what goes between them is worked out here — so the smoothness of the
// dance stops being a matter of how many drawings were asked for.
package bones

import (
	"encoding/json"
	"math"
	"sort"
)

// Point is a joint, in the drawing's own thousandths. R is the head's radius and
// nought everywhere else.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	R float64 `json:"r,omitempty"`
}

// Pose is one frame: the eleven points a figure is built from.
type Pose struct {
	Head   Point `json:"head"`
	Neck   Point `json:"neck"`
	Hip    Point `json:"hip"`
	ElbowL Point `json:"elbowL"`
	HandL  Point `json:"handL"`
	ElbowR Point `json:"elbowR"`
	HandR  Point `json:"handR"`
	KneeL  Point `json:"kneeL"`
	FootL  Point `json:"footL"`
	KneeR  Point `json:"kneeR"`
	FootR  Point `json:"footR"`
}

// Dance is a move: a box to measure in, the floor it stands on, and the
// keyframes.
type Dance struct {
	Move   string  `json:"move"`
	Box    float64 `json:"box"`
	Floor  float64 `json:"floor"`
	Frames []Pose  `json:"frames"`
}

// Read takes a move as it arrives.
func Read(raw []byte) (Dance, error) {
	var d Dance
	if err := json.Unmarshal(raw, &d); err != nil {
		return Dance{}, err
	}
	if d.Box <= 0 {
		d.Box = 1000
	}
	if d.Floor <= 0 {
		d.Floor = d.Box * 0.9
	}
	return d, nil
}

// Bone is a pair of joints that keeps its length however the figure moves.
type Bone struct {
	Name string
	From func(*Pose) *Point
	To   func(*Pose) *Point
}

// Bones is every one of them, in the order they hang off each other.
func Bones() []Bone {
	return []Bone{
		{"head-neck", func(p *Pose) *Point { return &p.Head }, func(p *Pose) *Point { return &p.Neck }},
		{"neck-hip", func(p *Pose) *Point { return &p.Neck }, func(p *Pose) *Point { return &p.Hip }},
		{"neck-elbowL", func(p *Pose) *Point { return &p.Neck }, func(p *Pose) *Point { return &p.ElbowL }},
		{"elbowL-handL", func(p *Pose) *Point { return &p.ElbowL }, func(p *Pose) *Point { return &p.HandL }},
		{"neck-elbowR", func(p *Pose) *Point { return &p.Neck }, func(p *Pose) *Point { return &p.ElbowR }},
		{"elbowR-handR", func(p *Pose) *Point { return &p.ElbowR }, func(p *Pose) *Point { return &p.HandR }},
		{"hip-kneeL", func(p *Pose) *Point { return &p.Hip }, func(p *Pose) *Point { return &p.KneeL }},
		{"kneeL-footL", func(p *Pose) *Point { return &p.KneeL }, func(p *Pose) *Point { return &p.FootL }},
		{"hip-kneeR", func(p *Pose) *Point { return &p.Hip }, func(p *Pose) *Point { return &p.KneeR }},
		{"kneeR-footR", func(p *Pose) *Point { return &p.KneeR }, func(p *Pose) *Point { return &p.FootR }},
	}
}

// Length is what a bone measures across the whole move, and how far the worst
// frame strays from it as a share.
func (d Dance) Length(b Bone) (want, stray float64) {
	if len(d.Frames) == 0 {
		return 0, 0
	}
	all := make([]float64, len(d.Frames))
	for i := range d.Frames {
		all[i] = Span(*b.From(&d.Frames[i]), *b.To(&d.Frames[i]))
	}
	want = median(all)
	if want == 0 {
		return 0, 0
	}
	for _, l := range all {
		stray = math.Max(stray, math.Abs(l-want)/want)
	}
	return want, stray
}

// Mend pulls every joint back onto the length the rest of the move agrees on,
// and gives the head one radius. It reports how many it had to move.
//
// Because a hand writing coordinates will quietly stretch a leg to reach a pose,
// and a leg that changes length between frames is the same fault as a head that
// jumps — only harder to see and impossible to fix once it is drawn. Pulled back
// along its own direction, so the pose is kept and only the reach is corrected.
func (d *Dance) Mend() int {
	var moved int
	for _, b := range Bones() {
		want, _ := d.Length(b)
		if want == 0 {
			continue
		}
		for i := range d.Frames {
			from, to := b.From(&d.Frames[i]), b.To(&d.Frames[i])
			at := Span(*from, *to)
			if at == 0 || math.Abs(at-want) < 0.5 {
				continue
			}
			to.X = from.X + (to.X-from.X)*want/at
			to.Y = from.Y + (to.Y-from.Y)*want/at
			moved++
		}
	}

	radii := make([]float64, len(d.Frames))
	for i := range d.Frames {
		radii[i] = d.Frames[i].Head.R
	}
	if want := median(radii); want > 0 {
		for i := range d.Frames {
			if d.Frames[i].Head.R != want {
				d.Frames[i].Head.R = want
				moved++
			}
		}
	}
	return moved
}

// At is the pose this far through the move, in keyframes, looping round.
func (d Dance) At(frame float64) Pose {
	n := len(d.Frames)
	if n == 0 {
		return Pose{}
	}
	if n == 1 {
		return d.Frames[0]
	}
	frame = math.Mod(frame, float64(n))
	if frame < 0 {
		frame += float64(n)
	}
	i := int(frame)
	return Tween(d.Frames[i], d.Frames[(i+1)%n], frame-float64(i))
}

// Tween is the pose part of the way from one keyframe to the next.
//
// Turned rather than slid: a joint moved straight to where it ends up passes
// through a place its own bone cannot reach, so an arm swung through a right
// angle would shorten by a tenth on the way and spring back. Every bone is
// carried round its own joint instead, at its own length, which is what a limb
// does.
func Tween(a, b Pose, at float64) Pose {
	out := a
	slide := func(p, q Point) Point {
		return Point{X: p.X + (q.X-p.X)*at, Y: p.Y + (q.Y-p.Y)*at, R: p.R + (q.R-p.R)*at}
	}
	out.Head = slide(a.Head, b.Head)
	out.Neck = slide(a.Neck, b.Neck)
	out.Hip = slide(a.Hip, b.Hip)

	turn := func(fromA, toA, fromB, toB, now Point) Point {
		one := math.Atan2(toA.Y-fromA.Y, toA.X-fromA.X)
		two := math.Atan2(toB.Y-fromB.Y, toB.X-fromB.X)
		// The short way round, so a limb never takes the long journey home.
		by := math.Mod(two-one+3*math.Pi, 2*math.Pi) - math.Pi
		long := Span(fromA, toA) + (Span(fromB, toB)-Span(fromA, toA))*at
		ang := one + by*at
		return Point{X: now.X + long*math.Cos(ang), Y: now.Y + long*math.Sin(ang)}
	}

	out.ElbowL = turn(a.Neck, a.ElbowL, b.Neck, b.ElbowL, out.Neck)
	out.HandL = turn(a.ElbowL, a.HandL, b.ElbowL, b.HandL, out.ElbowL)
	out.ElbowR = turn(a.Neck, a.ElbowR, b.Neck, b.ElbowR, out.Neck)
	out.HandR = turn(a.ElbowR, a.HandR, b.ElbowR, b.HandR, out.ElbowR)
	out.KneeL = turn(a.Hip, a.KneeL, b.Hip, b.KneeL, out.Hip)
	out.FootL = turn(a.KneeL, a.FootL, b.KneeL, b.FootL, out.KneeL)
	out.KneeR = turn(a.Hip, a.KneeR, b.Hip, b.KneeR, out.Hip)
	out.FootR = turn(a.KneeR, a.FootR, b.KneeR, b.FootR, out.KneeR)
	return out
}

// Pen is how thick the stroke is at a given height in dots, and never thinner
// than one — a line under a dot wide is a line that comes and goes as the figure
// moves, which is the dotted, flickering look the marks were traced away from.
func Pen(tall int) float64 { return math.Max(float64(tall)/40, 1) }

// Draw lights the dots of one pose.
//
// scale is dots to the drawing's own units, left where the box begins across the
// screen and floor the row his feet stand on. Everything is measured up from the
// floor rather than down from the top, because the floor is the one line every
// pose of every move has in common.
func Draw(p Pose, box, floor float64, scale float64, left, ground int, pen float64, light func(x, y int)) {
	at := func(q Point) Point {
		return Point{X: float64(left) + q.X*scale, Y: float64(ground) - (floor-q.Y)*scale}
	}
	head, neck, hip := at(p.Head), at(p.Neck), at(p.Hip)

	dot := func(x, y float64) {
		r := pen / 2
		for dy := -int(r) - 1; dy <= int(r)+1; dy++ {
			for dx := -int(r) - 1; dx <= int(r)+1; dx++ {
				if float64(dx*dx+dy*dy) <= r*r {
					light(int(math.Round(x))+dx, int(math.Round(y))+dy)
				}
			}
		}
	}
	line := func(a, b Point) {
		n := int(math.Hypot(b.X-a.X, b.Y-a.Y)) + 1
		for i := 0; i <= n; i++ {
			t := float64(i) / float64(n)
			dot(a.X+(b.X-a.X)*t, a.Y+(b.Y-a.Y)*t)
		}
	}
	ring := func(cx, cy, rx, ry, turn float64) {
		n := int(2*math.Pi*math.Max(rx, ry)) + 8
		for i := range n {
			a := 2 * math.Pi * float64(i) / float64(n)
			x, y := rx*math.Cos(a), ry*math.Sin(a)
			dot(cx+x*math.Cos(turn)-y*math.Sin(turn), cy+x*math.Sin(turn)+y*math.Cos(turn))
		}
	}
	blob := func(at Point, r float64) {
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				if x*x+y*y <= r*r {
					light(int(math.Round(at.X+x)), int(math.Round(at.Y+y)))
				}
			}
		}
	}

	// The head, and the body it stands on: an oval along the line from the neck
	// to the hip, which is what this character is drawn with.
	ring(head.X, head.Y, p.Head.R*scale, p.Head.R*scale, 0)
	ring((neck.X+hip.X)/2, (neck.Y+hip.Y)/2,
		p.Head.R*scale*0.62, math.Hypot(hip.X-neck.X, hip.Y-neck.Y)/2,
		math.Atan2(hip.Y-neck.Y, hip.X-neck.X)-math.Pi/2)
	line(head, neck)

	for _, limb := range [][3]Point{
		{neck, at(p.ElbowL), at(p.HandL)},
		{neck, at(p.ElbowR), at(p.HandR)},
		{hip, at(p.KneeL), at(p.FootL)},
		{hip, at(p.KneeR), at(p.FootR)},
	} {
		line(limb[0], limb[1])
		line(limb[1], limb[2])
		blob(limb[2], math.Max(pen*0.9, 1.5))
	}
}

// Box is how wide and tall a pose stands, in the drawing's own units, so that
// whoever draws it knows how much room to leave.
func (d Dance) Reach() (left, right, top float64) {
	left, right, top = d.Box, 0, d.Floor
	for i := range d.Frames {
		for _, b := range Bones() {
			for _, q := range []*Point{b.From(&d.Frames[i]), b.To(&d.Frames[i])} {
				r := math.Max(q.R, 0)
				left = math.Min(left, q.X-r)
				right = math.Max(right, q.X+r)
				top = math.Min(top, q.Y-r)
			}
		}
	}
	return left, right, top
}

// Span is how far apart two joints are.
func Span(a, b Point) float64 { return math.Hypot(b.X-a.X, b.Y-a.Y) }

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return s[len(s)/2]
}
