# The drawings, and how to ask for them

The wordless screen puts up a row of marks. Each mark rides its own slice of the
spectrum — the lowest at the left, the cymbals at the right — so the row is a
picture of the sound rather than an ornament beside it. The marks are drawings,
baked into dots by `cmd/spindle-marks`, and the drawings are made outside this
repository.

This file is what to ask for. It is written to be pasted: the style block and
each sheet below go into an image generator as they stand, one sheet at a time.

## How a sheet is used

1. Paste the style block and one sheet's brief. Ask for one image.
2. Keep the returned PNG. It is cut into one file per cell.
3. The cells go under `assets/marks/<set>/`, with a `mark.json` listing them:
   the marks in the order they stand in the row, the frames of each, and how
   those frames play — struck, swung, cycled or once. See below.
4. `go run ./cmd/spindle-marks` bakes every mark at every size into
   `internal/ui/marks_gen.go`. Nothing is rasterised at playback time.
5. `go run ./cmd/spindle-marks -show <set>` bakes and then prints the row in the
   terminal, in the same braille the screen draws in. `-size 24` for one height.

The judging happens in dots, not on the sheet. A drawing that reads beautifully
at 300 pixels and turns to porridge at 24 dots is a drawing that failed, and the
only way to know is to bake it and look.

## What the pipeline needs

These are measured against the working sets, not preferences.

**White on black.** Ink is any pixel at or above half brightness; the background
is what is left. The sheet that produced the dancers was white strokes on pure
black and 4% of it was ink.

**Strokes, never fills.** Every other picture on that screen is a stroke two or
three dots wide — the type, the meters, the water. A filled shape among them
reads as a hole punched in the picture.

**One pen for the whole sheet.** The pen is measured off the drawing and then
taken to three dots at every baked size. A sheet drawn with three different
weights bakes into a row that looks like three different sets.

**It is baked as small as 24 dots tall.** The sizes are 24, 36, 54 and 72 dots,
and the largest that fits the whole row across the screen wins — on a normal
terminal that is usually not the largest. So the real question about every
drawing is whether it survives as a 24-dot-tall arrangement of three-dot
strokes. Anything finer than about a tenth of the drawing's height is gone.

**Each mark is cut to its own ink.** Margins on the sheet do not matter and the
row stands on one floor whatever they were. What does matter is that nothing
from a neighbouring cell — a stray line, a caption, a frame — falls inside the
cut, because it becomes part of the mark. The cut is a rectangle, so two
drawings that touch cannot be separated at all: the sheet is thrown away and
asked for again. This is the rule that fails most often and the one to state
loudest.

**No text anywhere.** A label inside the sheet is ink, and ink is a drawing.

**The whole set is the row.** Every mark of a set is shown, in the order the
manifest lists them, and that order is the spectrum. A set is designed as a row
read left to right, not as eight unrelated icons.

**Relative size is set here, not drawn.** Every mark is baked to the same
height, so an animal choir does not get its scale from the sheet. Draw each one
large, filling its cell, for the resolution; the sizes relative to each other go
into `mark.json`.

**Phases must be registered.** A mark that animates is several drawings that
differ only in the part that moves. Same object, same size, same place in the
cell — otherwise switching between them reads as two pictures rather than one
thing moving.

## The style block

Paste this above every sheet, unchanged. The three rules in capitals are the
ones that were broken the first time this was asked, and each of them makes the
sheet unusable rather than imperfect.

```
GENERATE A NEW IMAGE FROM NOTHING. There is no input image and nothing is being
edited: what follows describes a picture to draw from scratch. Draw it.

The picture is one reference sheet of line-art drawings arranged in a grid, of
the subjects listed at the end. Everything between here and there says how it
should look.

ONE SHEET ONLY. Draw the one sheet described below. Do not add other sheets and
do not combine several sheets into one image. One sheet, one image.

DRAW EVERYTHING THAT IS ASKED FOR. Every subject named below must be on the
sheet. If you have a better idea than one of them — a better subject for the
set, a better way to animate it, something the brief did not think of — keep it,
but ADD it rather than swapping it in: draw everything asked for, and put the
extra subjects in extra columns at the right-hand end of the same sheet, in the
same style and to the same rules. Nothing asked for is ever dropped to make
room for something better. A brief is a floor, not a ceiling.

NO TEXT ANYWHERE ON THE IMAGE. No titles, no headings, no captions, no column
names, no row labels, no numbering of the frames, no legend, no signature, no
watermark. Not one letter and not one digit anywhere on the sheet, including
the margins. Every mark on the image is part of a drawing.

AN ANIMATION IS A MOVING PART, NOT MOTION LINES. Between two frames, some part
of the subject actually changes position or shape: a beater falls, a mouth
opens, cymbals part, a tail bends, legs pass each other. Drawing the subject
unchanged and adding short radiating lines around it is not an animation, and
those lines are too fine to survive: they disappear when the drawing is
reduced. Where a subject is hard to animate, choose a pose for it whose parts
can move, rather than falling back on motion lines.

STYLE
- Pure black background (#000000). All drawing in pure white (#FFFFFF).
- No grey, no fills, no shading, no gradients, no glow, no colour.
- Outline strokes ONLY. Nothing is filled in. No solid shapes, no silhouettes.
- ONE constant stroke weight everywhere, rounded caps and rounded joins.
- Simple enough to survive being reduced to a 24 by 24 grid of dots: no fine
  detail, no hatching, no small features, no textures, no thin decoration.
  Any feature smaller than a tenth of the subject's height will be lost.
- Flat and front-on. No perspective, no 3/4 view, no depth, no shadows.

SIZE AND FILE
- Return ONE image, PNG, at the largest resolution available and no smaller
  than 2000 pixels on its long side. Not JPEG.
- Each subject is drawn LARGE: it fills at least three quarters of the height
  of its own cell, and is no wider than three quarters of its own height —
  taller than it is wide, always. Where a thing is naturally wide, turn it: a
  cymbal at a steeper angle, a pedal tucked under rather than out to the side,
  a stick coming in from above rather than lying across.
- Every subject on the sheet is drawn at the same height as every other, even
  where the real things differ in size. A whale and a mosquito are the same
  height here.
- Nothing on the sheet is smaller than a tenth of a subject's height, and no
  enclosed shape is narrower than a fifth of it. Below those, a small circle
  fills in solid and a short line disappears entirely when the drawing is
  reduced.
- No repeated small features: no rows of lugs around a drum, no tension rods, no
  spokes, no studs, no teeth. Eight small things around a small shape merge into
  a band of solid ink.

LAYOUT
- An invisible grid, evenly spaced, with a wide empty margin around the sheet
  and clear empty black space between every cell.
- NOTHING OVERLAPS OR TOUCHES ANYTHING ELSE. Every drawing sits entirely inside
  its own cell. No drawing crosses into a neighbouring cell, no subject spans
  two cells or two rows, no two drawings share a line, no ground line or horizon
  runs across the sheet, and nothing runs off the edge of the sheet. Each cell
  is a plain rectangle holding one complete drawing and nothing else.
- A column is one subject and the rows under it are that subject's animation
  frames: the SAME drawing at the SAME size in the SAME position within its
  cell, with ONLY the moving part changed. Nothing that is not moving may shift
  by even a little between frames.
- Every subject on the sheet is drawn by one hand, sharing one build and one
  pen, like a single icon family. Where the subjects are creatures or figures
  they share one build as well: the same proportions, the same simple body.
- All the subjects stand on the same invisible line across the sheet, at the
  same place within their cells, so that a row of them reads as standing on one
  floor.
- Everything faces the same way: to the right.
- Keep the eight subjects to a similar amount of ink. One dense subject beside
  seven sparse ones makes the row lopsided.
- Keep each subject under about fifteen strokes.
- Nothing depends on a face, on eyes, or on fingers to be recognised.
- Read the frames downwards: the first frame is the top row.

Style reference: clean pictogram line icons, the weight and simplicity of
Tabler Icons.
```

## What the first sheet taught

Asked once, without the three rules above, and this is what came back — worth
recording, because every one of these is a thing an image generator does by
default rather than a mistake it happened to make.

**It bundled five sheets into one image.** 1543 by 1019 pixels for all of them,
which left each drawing about 60 pixels tall where 250 was asked for. Everything
else follows from that: at 60 pixels the pen is two pixels wide and there is no
detail left to lose.

**It labelled everything.** A title over each sheet, a name over each column,
and the frame numbers down the side. All of it is ink, and ink is a drawing.

**It animated with motion lines.** Frame 1 the subject at rest, frame 2 the same
subject with short lines radiating from it, frame 3 the same subject with more
of them. Nothing moved in eleven of the sixteen columns. Those lines are exactly
the detail that vanishes when a drawing is reduced to 24 dots, so those columns
would bake into three identical marks.

**It let a figure span three rows.** The dancers came back as one tall figure per
column crossing all three frame rows, so there were sixteen poses and no
animation at all.

**And it improved on what was asked.** The instruments came back as a drum kit
spread across the row — kick, snare, clap, closed hi-hat, open hi-hat, rim, tom,
cymbal — instead of the mixed band that was asked for, and that is the better
row: it is the spectrum from low to high without anybody having to be told, and
the open and closed hi-hat as neighbours is a gift. It also invented a weather
sheet that nobody asked for, which is the best answer yet to the loudness. Both
are kept below. A brief is a floor, not a ceiling.

## Frames, and how they play

A mark that animates is a set of drawings, and how many there are is decided by
two things: how the movement returns to where it started, and how many drawings
a beat has room for.

### How many a beat has room for

The drawn frames advance on the beat, not on the frame rate — that is what makes
them independent of the machine. What the frame rate decides is how many of them
fit before one goes by unseen.

A beat on the records this is watched against runs 470 to 710 milliseconds. The
picture is drawn at 60 frames a second by default, so a beat is 28 to 43 drawn
frames, and a listener may set it as low as 15. Put N drawings in a beat and each
one is on screen for beat/N:

	N       held for      seen at 60fps     seen at 30fps
	3       160-240ms     10-14 frames      5-7 frames
	4       120-180ms     7-11 frames       4-5 frames
	6        80-120ms     5-7 frames        2-4 frames
	8        60-90ms      4-5 frames        2-3 frames
	12       40-60ms      2-4 frames        1-2 frames

A drawing that is on screen for one rendered frame is a drawing that flickers
rather than one that is seen. So: **eight is the ceiling at sixty, six at
thirty**, and twelve is out of the question at any rate this runs at. Twelve is
also where hand-drawn animation sits — "on twos" — which is worth knowing as the
thing we cannot reach and are not trying to.

The default is **four**. Three was the old default and it was chosen before the
rate was: at sixty a beat has room for four comfortably, and the fourth is the
one that turns a hit into a hit.

The sheets below that still ask for three were written before this, and every one
of them is struck. When one of them is asked for, add the fourth frame and say
where it goes: it is the wind-up, and it always sits between at rest and
contact — the subject gathered, nothing else changed.

### The four kinds

**Struck** — four frames: at rest, wound up, contact, recoiling. A kick, a clap,
a piano key, a beak. The contact frame lands ON the beat and the wound-up frame
is the one before it, so the movement anticipates the beat the way a drummer's
arm does. Played 1, 2, 3, 4 and back to 1.

**Swung** — three frames: one side, the middle, the other side, played 1, 2, 3, 2
across two beats so the swing is a bar rather than a twitch. Draw one side only
and let the code mirror it — a hand-drawn pair of ends never matches, and the row
leans for it.

**Cycled** — six frames, and it does not return: a walk, a run, wings beating, a
disc turning. Played round and round, one frame a beat-eighth, and frame 6 has to
lead back into frame 1 or the loop has a bump in it. Six rather than four because
a cycle is the one movement the eye follows limb by limb.

**Once** — six to eight frames, played front to back and then stopped. Blowing a
kiss, a curtain opening, a page turning, falling asleep. It is not on the beat at
all: it takes the time it takes.

### What is not drawn

The row already leans, sways, rides and bounces, and everything crossing the
screen is moved by the code at the full frame rate — smoothly, at 60 or at 30,
because those motions are worked out per second rather than per frame. A subject
drawn leaning gets its lean twice.

So the frames are only ever the part that changes SHAPE — a mouth opening, a
beater falling, cymbals parting, legs passing each other — and never the whole
subject tilted, shifted, grown, or moved across its cell.

### Two things that ruin a frame, whatever the count

- They must be the same drawing with one part moved. Not the same subject drawn
  again. Anything not moving must not shift by a pixel, and the easiest way to
  get that wrong is to let the whole subject drift towards the middle of its cell.
- The movement between each pair of frames should be the same size. Three frames
  where the first two are nearly identical and the third leaps is a stutter.

Asking for more frames costs nothing in the code — a mark is about a kilobyte
baked at all four sizes — and costs a great deal in the drawing, because every
extra cell is another chance for the subject to drift. Get a set working at four
first, then ask for six on that set alone, once its style is known to bake well.

## What survives being baked

Everything above is about the sheet. This is about what is left of it at the size
it is actually seen, and it is arithmetic rather than taste.

A mark is baked at 24, 36, 54 and 72 dots tall, and the largest that fits the
whole row across the terminal wins — which on a normal window is usually 24 or
36. The pen is taken to three dots at every one of them. So a 24-dot mark is a
drawing about eight pen-widths tall. That is the whole budget.

	drawn feature          at 24 dots         verdict
	the subject itself     24 dots            fine
	a limb, a tail         6-12 dots          fine
	an eye, a hand         2-4 dots           a blob or nothing
	a short motion line    1-2 dots           gone
	hatching, texture      under a dot        gone

**And a mark must be taller than it is wide.** This is the one nobody would
guess, and it decided a whole sheet. The row is fitted across the terminal at the
largest baked size that holds all of it, so the *width* of the marks decides the
*height* they are drawn at. Measured on a 167 by 41 terminal, where the band has
55 dots of height to play with and so 36 would fit:

	set      mean width, as a share of the height     baked at
	dance    0.62                                     36 dots
	band     0.75                                     36 dots
	moods    0.84                                     36 dots
	a kit drawn wide, with pedals and sticks
	         0.95, and one mark at 1.7                24 dots

The kit lost a whole size step — from 36, where it reads, to 24, where it is a
scatter of dots — for being wide. Not for being detailed: for being wide. A kick
drum with its pedal sticking out to one side, a stick lying across a rim, a
cymbal drawn as a wide flat plate; each of them costs the entire row.

So: **no drawing wider than three quarters of its height, and none wider than
its height at all.** Where a thing is naturally wide, turn it: a cymbal seen at
a steeper angle, a pedal tucked under rather than beside, a stick coming in from
above rather than across.

Two more rules come out of the same arithmetic, and they are the ones to check a
sheet against:

- **Nothing smaller than a tenth of the subject's height.** Below that it is not
  simplified, it is absent.
- **No enclosed area narrower than a fifth of the subject's height.** A three-dot
  pen on both sides of a small circle leaves no hole in the middle: the eye of a
  face, the sound hole of a guitar and the gap inside a small loop all fill in
  and become blobs.
- **No repeated small features.** The lugs around a snare, the tension rods on a
  drum, the spokes of a wheel, the teeth of a comb: eight of anything around a
  24-dot shape is eight things a third of a dot apart. They do not simplify —
  they merge into a band of solid ink and the shape underneath is lost.

And one test worth applying to every drawing before it is asked for: would it
still be recognisable as a solid black shape, with all its inside detail thrown
away? If not, it will not read at 24 dots either.

## Sheet 1 — the kit, animated

The row with the clearest reason to exist: every mark rides its own slice of the
spectrum, so a kick at the left kicks on the kick and a cymbal at the right rings
on the cymbals. A drum kit spread across the row is that order without anybody
being told it — this is the set the first attempt invented, kept, with every
column now moving a part of itself instead of growing motion lines.

```
Grid: 8 columns by 4 rows, so 32 drawings. Each column is one drum or one pair
of hands. The four rows are four frames of a single strike, read downwards:

  row 1  at rest
  row 2  wound up: the stick or beater at its highest, nothing else changed
  row 3  contact: the stick or beater against the drum, and the struck surface
         visibly pushed in or bowed by the blow
  row 4  recoiling: the stick or beater lifting away again, the surface still
         slightly out of shape

Nothing on this sheet is wider than three quarters of its own height. A drum kit
is full of things that stick out sideways — pedals, sticks lying flat, cymbals
as wide plates — and each of them costs the whole row a size when it is drawn.
Turn them: pedals under, sticks from above, cymbals at a steeper angle.

The stick or beater is the part that moves. In rows 1, 2 and 4 the drum itself
is drawn identically; only in row 3 does its struck surface change shape, and
only slightly.

1. Kick drum: a large circle seen from the front, standing on two short legs.
   The pedal is tucked UNDER the drum, in front of it, not out to one side, and
   the beater comes down onto the head from above. Drawn with the pedal beside
   it this mark measured 1.17 of its height in width and cost the whole row a
   size; it has to be no wider than the drum itself.
2. Snare drum: a shallow cylinder seen from the side, on a stand, with one
   stick above it.
3. A pair of hands clapping, seen from the front, fingers up. Here the two
   hands are the moving part: apart / wide apart / flat together as one shape /
   parting again.
4. Hi-hat: two cymbals on a stand seen from the side, with a pedal at the
   bottom. Here the moving part is the cymbals and the pedal together: closed
   with the pedal down / open with the pedal up / closed hard, the top cymbal
   pushed against the bottom one / open a little again.
5. Open hi-hat: the same two cymbals already parted, with a stick above the top
   one. The stick moves; the top cymbal tilts when it is struck.
6. Rim shot: a shallow drum seen FROM THE FRONT, as the snare in column 2 is,
   with one stick coming down onto its near rim from above. Not seen from
   above: drawn as a wide ellipse with a stick lying across it, this mark
   measured 1.44 of its height in width, which is the widest thing on the sheet
   and on its own enough to drop the row a size.
7. Floor tom: a deep cylinder seen from the side, standing on three legs, with
   one stick above it.
8. Ride cymbal: a wide shallow cymbal on a stand seen from the side, with one
   stick above it. The cymbal tilts when struck and is still tilted the other
   way as the stick lifts.
```

## Sheet 2 — the animal choir

The row as one picture instead of eight objects: the deepest voice at the left,
the highest at the right, so the order is not a convention to be learned but
something anybody hears. Each animal opens its mouth when its own band sounds.

```
Grid: 8 columns by 3 rows. Each column is one animal, seen from the side with
its head turned towards the viewer. The three rows are three frames: mouth
closed, half open, wide. STRUCK — it returns to closed, so three is enough.

Every animal is drawn LARGE, filling its cell to the same height, whatever the
real animal's size is. Do not draw them to scale with each other.

1. Whale.       Mouth closed / mouth open / mouth wide, with a spout above.
2. Elephant.    Trunk down, mouth closed / trunk raised, mouth open / trunk
                high, mouth wide, two short curved lines above.
3. Bear.        Mouth closed / open / wide, head tilted back.
4. Dog.         Mouth closed / open / howling with the head up and two short
                curved lines.
5. Cat.         Mouth closed / open / wide.
6. Small bird.  Beak closed / beak open / beak wide, wings slightly lifted.
7. Cricket.     Legs down / legs half raised / legs up.
8. Mosquito.    Wings level / wings mid-beat / wings up, with two short motion
                arcs.
```

## Sheet 3 — keeping time, and answering the loudness

Things the screen measures and does not yet draw. The beat is already found and
carried frame by frame; the loudness already has a range the picture is read
against. These are the objects that say so without being read.

```
Grid: 8 columns by 3 rows. Each column is one object, the three rows are three
frames of it. All eight are STRUCK or SWUNG: they return to where they started.

1. Metronome, the classic wedge with a swinging arm and a weight on it.
   Arm leaning left / arm upright / arm leaning right.
2. A ball above a straight floor line.
   Resting on the line / halfway up / at the top with two short motion lines
   under it.
3. Pendulum clock: a plain case with a swinging pendulum.
   Pendulum left / centre / right.
4. Woodpecker on a tree trunk, seen from the side.
   Head back / beak against the trunk / beak on the trunk with three short
   lines radiating from it.
5. Heart, a simple outline.
   Small / larger / largest, with two short curved lines either side.
6. Candle with a flame.
   Flame small and straight / taller and leaning / tall, wide and split at the
   tip with one spark above.
7. Campfire: three logs and flames.
   Low flames / medium flames / tall flames with two sparks above.
8. Flag on a pole, seen from the side.
   Hanging almost straight / half out with one wave / fully out with two waves.
```

## Sheet 4 — the mouth: singing, and blowing a kiss

Three separate things that all happen to be a mouth, so they are drawn by one
hand on one sheet.

The **singing** row is not an animation. It is an alphabet: the shape is chosen
every frame from what is being sung, so any shape may follow any other, and that
is what makes them hard to draw. They have to be interchangeable — the same
width, the same centre, the same weight — or the mouth appears to jump about the
screen while it sings.

The **kiss** is an animation, played once, and it is what the screen does back
when somebody presses the key for it. Nothing else on this screen is addressed
to the person watching.

The **sleeping** row is for a paused player. A picture that carries on moving
while the music has stopped is a picture that has not noticed.

```
Grid: 8 columns by 3 rows, so 24 drawings.

THIS SHEET IS LAID OUT IN ROWS, NOT COLUMNS. Each row is its own set of eight
drawings, read left to right. The columns mean nothing here.

Every drawing on the sheet is an outline only: no teeth, no tongue, no lips
drawn as separate shapes, no shading. No face is drawn around any mouth — the
mouth is the whole drawing.

ROW 1, eight mouth shapes for miming a song. Every one of them is exactly the
same width, centred at exactly the same point in its cell, and drawn with the
same weight, so that any one may be swapped for any other without the mouth
appearing to move. They differ only in shape:
   1. closed: a single straight horizontal line
   2. barely open: a very shallow lens shape
   3. a small round o
   4. a tall narrow oval
   5. a wide flat lens, the corners drawn out sideways
   6. a large wide-open oval, taller than it is wide
   7. an open mouth pushed to one side, wider on the right
   8. a smile: an upward curve with the ends turned up

ROW 2, blowing a kiss: eight frames of ONE mouth, played once, left to right.
The moving part is the mouth's shape and then the heart's position and size.
   1. mouth relaxed, a shallow lens
   2. mouth gathering, narrower
   3. mouth pursed to a small round o
   4. pursed, with a small heart outline touching the mouth
   5. the heart just clear of the mouth, still small
   6. the heart a third of the way across the cell, larger
   7. the heart two thirds across, larger again
   8. the heart at the far edge of the cell, largest
No motion lines behind the heart at any point: the heart moving IS the motion,
and short lines behind it disappear when the drawing is reduced.

ROW 3, falling asleep: eight frames of ONE cat, seen from the side, played once.
The cat is drawn identically in every frame apart from what is named:
   1. sitting up, ears up, eyes open
   2. sitting, eyes half closed
   3. curling, head lowered
   4. curled up, eyes closed
   5. curled up, one small Z above it
   6. curled up, two Z's, the second larger
   7. curled up, three Z's rising and growing
   8. stretching awake, front legs pushed forward, ears up
```

## Sheet 5 — the dancers, animated

The eight already exist and one hand drew them, which is the whole reason they
work: a row assembled out of an icon library is eight strangers standing next to
each other, and the eye reads the differences in build instead of the
differences in pose. What they do not do is move. Each pose becomes three, and
the row dances rather than leaning.

Ask for the same eight poses, and say so: this is a redraw of an existing set,
not a new cast.

```
Grid: 8 columns by 4 rows, so 32 separate drawings. Each cell holds one whole
figure, head to feet, inside that cell alone: no figure is drawn tall enough to
cross into the row above or below it. Asked without this, the figures came back
one per column spanning all three rows, which is sixteen poses and no animation.

Each column is one dancing figure, the four rows are four frames. CYCLED: frame 4 must lead back into frame 1 without a bump, so a
figure that leans left through the four frames has to be on its way back by the
fourth. Where a pose is named with three positions below, the fourth frame is
the second one again on the way back.

All eight figures share one build: the same height, the same proportions, the
same simple body, no faces, no hair, no clothing detail. They are drawn front
on, standing on an invisible floor at the same level.

1. Standing square, arms straight out to the sides.
   Arms level / arms slightly down / arms slightly up.
2. Grooving, knees soft, arms bent at the elbows.
   Weight left / centred / weight right.
3. Stepping to one side.
   Feet together / one foot lifted / foot planted wide.
4. A long stride.
   Legs together / mid-stride / legs at their widest.
5. Reaching to one side, one arm long.
   Arm halfway / arm out / arm out and the body leaning after it.
6. Leaning back from the waist.
   Upright / leaning / leaning further with both arms trailing.
7. Both arms raised above the head.
   Arms at the shoulders / arms halfway up / arms straight up.
8. Jumping, both feet off the floor.
   Crouched / rising with the feet just off the floor / at the top, legs tucked.
```

## Sheet 6 — the faces, animated

The set that is up most often, and the one that is still an icon library. A
smile that widens on the chorus is worth more than a smile.

```
Grid: 8 columns by 3 rows. Each column is one face, the three rows are three
frames of it. STRUCK: it returns to the first, so three is the whole animation.

Every face is the same circle at the same size in the same position. Only the
features change between columns, and only the moving feature changes between
phases. No hair, no ears, no colour, no cheeks, no eyebrows unless named.

1. Happy.      Small smile / wider smile / widest smile with the eyes creased.
2. Delighted.  Smile and open eyes / open mouth / mouth wide with two short
               lines either side of the head.
3. Winking.    Both eyes open / one eye half closed / one eye shut, smile wider.
4. Sad.        Slight frown / deeper frown / frown with one tear below one eye.
5. Cross.      Level brows / brows angled down / brows down hard with two short
               lines above the head.
6. Puzzled.    Level brows, straight mouth / one brow raised / one brow raised
               and the mouth pushed to one side.
7. Bookish.    Round glasses, straight mouth / small smile / smile and the
               glasses pushed up slightly.
8. Asleep.     Eyes shut, straight mouth / one Z above the head / two Z's, the
               second larger.
```

## Sheet 7 — the crowd, animated

A row of people rather than a row of objects: the room the record is playing to.
Same build for all of them, the way the dancers are.

```
Grid: 8 columns by 4 rows. Each column is one figure, four frames each. CYCLED,
like the dancers: frame 4 leads back into frame 1. Walking and running are true
cycles — left foot forward, passing, right foot forward, passing.

All eight share one build, drawn front on, standing on an invisible floor at the
same level, no faces.

1. Waving.        Hand at the shoulder / hand up / hand up and leaning over.
2. Bouncing.      Feet down / feet just off the floor / at the top of the hop.
3. Arms up.       Arms at the chest / arms halfway / arms straight up.
4. Two arm in arm. Standing / leaning together / leaning the other way.
5. Walking.       Feet together / mid-step / feet apart.
6. Running.       Feet together / mid-stride / legs at their widest, leaning.
7. Stretching.    Arms down / arms out / arms up and the back arched.
8. Clapping above the head.
                  Hands apart / hands together / hands together with four short
                  lines radiating out.
```

## Sheet 8 — the record turning over

The picture is dealt at the record's own joins now, so a change of set is the
record changing. These are the objects that say a page has turned, for the one
mark in a row that turns over when it does.

```
Grid: 8 columns by 3 rows. Each column is one object, three frames each. These
are ONCE animations — a page turns and stays turned — so the three frames are
the beginning, the middle and the end of the movement, not a loop.

1. Turntable seen from above: a disc, a spindle, a tonearm.
   Arm parked to one side / arm halfway across / arm on the disc with two short
   curved lines following the groove.
2. A pair of stage curtains.
   Closed / parted a little / open wide.
3. A book or a page, seen from the front.
   Flat / one corner lifted and curling / the page halfway over.
4. A traffic light with three lamps.
   The top lamp filled with a ring around it / the middle one / the bottom one.
   (Only the ring changes: no lamp is ever solid.)
5. A chameleon on a branch, seen from the side.
   Plain body / body with a few spots / body with many spots.
6. A kaleidoscope pattern: a simple six-fold star.
   Upright / turned a little / turned further.
7. A train seen from the side, an engine and two carriages.
   At the left of the cell / centred / at the right, with two short motion lines
   behind it.
8. A door in a frame.
   Shut / ajar / wide open with a straight line of light across the floor.
```

## Sheet 9 — words that change the picture

For the line being sung: a word in the lyric puts something on the screen. These
are the ten worth having, because they are the words that actually turn up.

```
Grid: 5 columns by 6 rows. Each cell is one drawing. Rows 1 and 2 are the three
phases of the objects in row 1's columns; rows 3 and 4 the same for row 3's; and
so on. Read it as three bands of two rows: in each band, the top row names the
object and the row under it continues its phases.

Simpler statement of the same thing: ten objects, each drawn three times as the
frames of a small movement, laid out in a grid five cells wide. All ten are
STRUCK — small, larger, largest, and back — so three frames is the whole of it.

1. Flame.        Small / taller and leaning / tall and split at the tip.
2. Raindrop.     High in the cell / halfway down / at the bottom with a small
                 splash of three short lines.
3. Heart.        Small / larger / largest with two short lines either side.
4. Crescent moon and one star.
                 Star small / star larger / star largest with four points.
5. Sun with rays.
                 Short rays / longer rays / longest rays.
6. Car seen from the side.
                 At the left / centred / at the right with two motion lines.
7. Teardrop below an eye.
                 Just formed / halfway down the cheek / at the bottom.
8. Bird in flight, seen from the side.
                 Wings down / wings level / wings up.
9. House with a door and a roof.
                 Windows plain / one window with a cross of light / smoke
                 curling from the chimney.
10. Snowflake, six arms.
                 Small / larger / largest.
```

## Sheet 10 — under the line: the sea

The meter that runs under everything on that screen draws a line across it, and
the water thrown off the picture falls back through it. Read that line as the
surface of a sea and the screen already has two halves that were always there:
the marks stand on it, the spray comes off it, and underneath it there is room
for something to swim past. Nothing new is introduced — it is the same picture
with something living in it.

Each of these crosses the screen the way the visiting figure does: it comes on
at one side, goes about its business, and leaves. The crossing itself is not
drawn. Only the swimming is: the tail beating, the wings going. Where it is, how
fast, and whether it is there at all comes from the record.

```
Grid: 8 columns by 4 rows. Each column is one creature seen from the side,
swimming or flying on the spot, facing right. Four frames, CYCLED: frame 4 must
lead back into frame 1 without a bump.

The creature does not move within its cell between frames and does not change
size. Only the part that beats — a tail, a fin, a wing — changes shape. Nothing
is drawn moving forwards: the crossing happens elsewhere.

1. Small fish.   Tail straight / tail bent one way / straight / bent the other.
2. Large fish, a deep body and a wide tail.  Same four.
3. Whale, long body, broad flat tail.        Same four, slower and wider.
4. Dolphin, seen from the side.              Same four.
5. Jellyfish, a bell and trailing threads.
   Bell squeezed narrow, threads trailing / half open / wide open, threads
   spread / half closed.
6. Crab, seen from the front, legs either side and two claws.
   Legs down, claws low / legs mid, one claw raised / legs up, both claws
   raised / legs mid, the other claw raised.
7. Seagull in flight, seen from the side.
   Wings down / wings level rising / wings up / wings level falling.
8. Butterfly, seen from the side.            Same four as the gull, sharper.
```

## Sheet 11 — above the line, and passing through

The other half of the same world, and the visitors that are neither fish nor
weather: things that walk across the floor the marks stand on, or fall through
the air above them.

```
Grid: 8 columns by 4 rows. Each column is one subject seen from the side, facing
right, moving on the spot. Four frames, CYCLED: frame 4 leads into frame 1.

Nothing moves forwards within its cell and nothing changes size between frames.
Only the legs, wings or wheels change.

1. Cat walking, seen from the side, tail up.
   Legs together / mid-step, tail curled left / legs apart / mid-step, tail
   curled right.
2. Mouse running, seen from the side.        Four frames of a fast run cycle.
3. Snail, shell on its back, seen from the side.
   Body stretched forward / gathered / stretched / gathered, the two feelers
   swapping which is higher.
4. Spider hanging from a single thread that rises out of the top of the cell.
   Legs gathered / half spread / spread wide / half spread.
5. Bicycle seen from the side, no rider.
   The pedals and the spokes at four positions of one turn.
6. Paper plane seen from the side.
   Level / nose slightly up / level / nose slightly down.
7. Balloon on a string, seen from the side.
   Round / squeezed taller / round / squeezed wider, the string trailing after
   it in each.
8. Rocket seen from the side, nose up, flame beneath.
   Flame short / medium / long / medium, with two sparks in the longest.
```

## Sheet 12 — the weather

Invented by the generator rather than asked for, and the best answer yet to the
one measurement that has nothing drawn from it: how loud the record is against
its own recent range. A row of weather is a row that says quiet and loud without
a number — drizzle in the verse, downpour in the chorus.

```
Grid: 8 columns by 3 rows. Each column is one kind of weather, the three rows
are three frames of it: least, middle, most. STRUCK — it returns to the least.

Between frames the drawing itself must change, not just gain decoration: more
drops, longer rays, a taller cloud, a longer bolt. Nothing here is a subject
sitting still with lines added around it.

1. Sun with straight rays around it.
   Short rays / longer rays / longest rays, and the disc a little larger.
2. Cloud alone.
   One small cloud / the same cloud taller / two clouds overlapping into one
   larger shape.
3. Rain: a cloud with drops beneath it.
   Three short drops / six drops, longer / ten drops, longest, reaching the
   bottom of the cell.
4. Lightning: a cloud with a bolt beneath it.
   A short bolt with two turns / a longer bolt with three turns / a long bolt
   with three turns and a second, shorter bolt beside it.
5. Snow: a cloud with flakes beneath it.
   Three flakes / six flakes / ten flakes, the lowest ones larger.
6. Wind: three horizontal curling lines.
   Three short lines, gently curled / three longer lines, more curled / four
   long lines with tighter curls at their ends.
7. Rainbow: concentric arcs rising from a flat base.
   Two arcs, low / three arcs, higher / four arcs, a full half circle.
8. Fog: horizontal broken lines stacked up the cell.
   Three lines, short / five lines, longer / seven lines filling the cell, the
   breaks in different places.
```

## Ideas without a sheet yet

Kept here so they are not lost, and because several of them need something in
the code before a drawing would help.

**What the sea is for.** The creatures on sheets 10 and 11 need a reason to
arrive, and the reasons are already measured. A shoal whose number follows how
busy the top of the spectrum is. A whale that only comes through after a long
quiet stretch, and only once. Birds that cross at a join. A fish that jumps the
row of marks on the loudest moment of a record. A snail on the slowest track
this machine has measured, and a mouse on the fastest. A balloon that rises with
the swell and is gone on the drop. None of it is drawn — it is where and when,
which is code, and it is the part worth arguing about once the drawings exist.

**The screen answering the person at it.** The kiss is drawn on sheet 4; what is
not built is the key that asks for it, and the rule that the mouth is only ever
up on a record with no words of its own. Waving as
spindle closes. Leaving footprints across the floor it walks over. Carrying the
record's title card on rather than having it appear.

**The twelve notes.** A piano keyboard whose pressed keys are the notes actually
sounding, a guitar neck with the fingering, a harp with the struck strings
moving, a choir of twelve mouths where whoever is sounding sings. All of them
wait on the chroma being worth drawing from, which has not been measured yet —
see the note in `internal/ui/joins.go` about what the harmony was worth for
finding sections, which is not the same question.

**The low end and the top of the range.** A speaker cone pushed out, a spring
compressed, a jelly wobbling, a mole out of its hole; rain, sparks, glitter, a
shooting star, a wind chime, insects. These are per-band answers, and the marks
row already maps the bands, so they would be a set rather than an addition —
worth drawing only if one of them is better than the instruments at saying the
same thing.

**A whole row that is one landscape.** A skyline whose windows light up by band,
a forest from oak to reed with the wind as the loudness, a mountain range, an
ocean cross-section, a stack of speaker cabinets from bins to tweeters. Same
idea as the animal choir: the row read as one picture rather than eight objects.

**Instruments as players.** The band as people rather than as objects — a
drummer, a bassist, a guitarist, a singer — each playing on their own band. It
is the same row as sheet 1 with a different subject, and it would want the
dancers' build so the two sets read as one company.

## Judging what comes back

Before anything is cut. Any one of these is a redraw, because none of them
survives being baked, and asking again costs one image where cutting a bad sheet
costs an afternoon.

**The three that make a sheet unusable**

- More than one sheet in the image, or subjects nobody asked for.
- Any text at all: a title, a column name, a frame number, a signature.
- An animation made of motion lines rather than a moving part. Look along each
  column: if the subject is the same in all frames and only the decoration around
  it changed, that column is three identical marks.

**The ones that cost the sheet quietly**

- Two frames of a column that are not the same object in the same place. Lay
  them over each other in your head: everything not moving must sit still.
- A subject that spans two cells or two rows, or is cropped by the edge.
- Two drawings that touch. The cut is a rectangle: they cannot be separated.
- A stroke weight that drifts between cells, or a subject drawn small while its
  neighbours fill theirs.
- Anything filled in: a solid head, a black note, a shaded body.
- Detail finer than a tenth of the subject's height, or an enclosed shape
  narrower than a fifth of it — the second one is the sneaky one, because it
  looks fine on the sheet and bakes into a blob.
- Grey. There is no grey in the pipeline: it is ink or it is not.
- Subjects drawn at their real relative sizes rather than all to one height.

Then cut the cells, put them under `assets/marks/<set>/` with a manifest, and:

	go run ./cmd/spindle-marks -show <set>
	go run ./cmd/spindle-marks -show <set> -size 24

which bakes them and prints the row in the terminal, in the braille the screen
itself draws in. That is the only judgement that counts, and the size to judge
at is the one a normal terminal actually gives the row — usually 24 or 36 dots,
not the 72 it would flatter the drawing to be seen at.
