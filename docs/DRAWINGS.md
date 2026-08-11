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

Paste this above every sheet, unchanged.

```
STYLE, applying to everything on the sheet
- Pure black background (#000000). All drawing in pure white (#FFFFFF).
- No grey, no fills, no shading, no gradients, no glow, no colour.
- Outline strokes ONLY. Nothing is filled in. No solid shapes, no silhouettes.
- ONE constant stroke weight everywhere, about 3% of a cell's height, with
  rounded caps and rounded joins.
- Simple enough to survive being shrunk to a 24 by 24 grid of dots: no fine
  detail, no hatching, no small features, no textures, no thin decorations.
- No text, letters, numbers, labels, captions or watermarks anywhere.
- No boxes, frames, grid lines, separators, drop shadows or ground shadows.
- Flat and front-on. No perspective, no 3/4 view, no depth.
- Draw each subject large, filling its own cell.
- Return ONE image, at the largest resolution available, at least 250 pixels
  per cell.

LAYOUT, applying to everything on the sheet
- An invisible grid, evenly spaced, with a wide empty margin around the sheet
  and clear empty space between every cell.
- NOTHING MAY OVERLAP OR TOUCH ANYTHING ELSE. Every drawing stays entirely
  inside its own cell with clear black space all around it. No drawing crosses
  into a neighbouring cell, no two drawings share a line, no element is common
  to two cells, no ground line or horizon runs across the sheet, no subject is
  cropped by the edge of the sheet, and no line of one drawing crosses a line of
  another. Each cell must be liftable out of the sheet as a plain rectangle with
  one complete drawing in it and nothing else.
- Where a column is one subject and the rows are its animation frames, the
  frames are the SAME drawing at the SAME size in the SAME position within the
  cell, with ONLY the moving part changed. Everything else must be identical.
- Every subject on the sheet is drawn by one hand, sharing one build and one
  pen, like a single icon family.

Style reference: clean pictogram line icons, the weight and simplicity of
Tabler Icons.
```

## Frames, and how they play

A mark that animates is a set of drawings, and how many there are is decided by
how the movement returns to where it started. Say which of these a subject is
when asking for it, because it is what the number of cells means.

**Struck** — three frames: at rest, striking, ringing. A kick, a clap, a piano
key, the woodpecker. Played 1, 2, 3, 2 and back to 1, with the strike landing on
the beat rather than after it. Three drawings, four steps, and it always returns
to rest, which is why three is enough.

**Swung** — three frames: one side, the middle, the other side. A metronome, a
pendulum, a flag, a shaker. Played 1, 2, 3, 2 across two beats, so the swing is
a bar rather than a twitch. Never draw the two ends as mirror images of each
other by hand: draw one and say the other is its mirror, or the row will lean.

**Cycled** — four frames, and it does not return: a walk, a run, wings beating,
a disc turning. Played 1, 2, 3, 4, 1, and frame 4 has to lead back into frame 1
or the loop has a bump in it. This is the one that needs more than three, and
the one to ask for explicitly.

**Once** — as many frames as the movement takes, played front to back and then
stopped. Blowing a kiss, a curtain opening, a page turning, falling asleep. Six
is comfortable; the movement is over when the frames are.

**Why three, in numbers.** The frames advance on the beat, and a beat on the
records this is watched against runs 470 to 710 milliseconds. Three frames
played out and back is four steps in that beat: 120 to 180 milliseconds a frame,
which is six to eight drawings a second. Hand-drawn animation on twos is twelve.
So three is plainly enough for a strike, where the whole point is that it is
sudden and everything after it is a decay, and plainly thin for a walk, where
the eye is following a limb from one place to another. Hence four for a cycle,
and six if a walk still steps once it is baked.

None of this is the whole movement, either. The row leans, sways, rides and
bounces in the code at the full frame rate, so what the drawn frames add is the
shape changing, not the motion — which is why a strike gets away with three.

The default is three. Where a sheet below wants four or six it says so, and the
grid is that many rows or that many cells wide. Asking for more frames costs
nothing in the code — a mark is about a kilobyte baked at all four sizes — and
costs a great deal in the drawing, because every extra cell is another chance
for the subject to drift. Get a set working at three or four first, then ask for
more on that set alone, when its style is already known to bake well.

**What not to draw at all.** The row already leans, sways, rides and bounces:
those are transforms in the code, applied to whatever the mark is, and they take
their timing from the beat. A subject drawn leaning gets the lean twice. So the
frames are only ever the part that changes SHAPE — a mouth opening, a beater
falling, cymbals parting, legs passing each other — and never the whole subject
tilted, shifted, grown or moved across its cell.

Two things about the frames themselves, whatever the count:

- They are the same drawing with one part moved. Not the same subject drawn
  again — the same drawing. Anything that is not moving must not shift by a
  pixel between frames, and the easiest way to get that wrong is to let the
  whole subject drift a little towards the middle of its cell.
- The movement between two frames should be the same size as the movement
  between the next two. Three frames where the first two are nearly identical
  and the third leaps is a stutter, not an animation.

## Sheet 1 — instruments, animated

The row that has the clearest reason to exist: each mark already rides its own
band, so a kick at the left kicks on the kick and a hi-hat at the right opens on
the cymbals. Replaces a set assembled out of an icon library.

```
Grid: 8 columns by 3 rows. Each column is one instrument, the three rows are
three frames of it. Every instrument here is STRUCK or SWUNG: it returns to
where it started, so three frames is the whole animation.

1. Kick drum with pedal and beater, seen from the front.
   Beater pulled back / beater against the drum head / beater on the head with
   two short curved motion lines beside the drum.
2. Bass guitar standing upright, thick strings.
   Plucking hand open above the strings / hand touching the strings / hand
   below, with one string drawn as a shallow wave.
3. Electric guitar standing upright.
   Pick held above the strings / pick crossing the strings / pick below, with
   the strings drawn as shallow waves.
4. A short run of piano keys seen from the front, with a hand above them.
   All keys level / one key pressed down / two keys pressed down with two short
   straight lines rising above them.
5. Microphone on a short stand.
   Bare / one curved arc leaving one side / two curved arcs leaving one side.
6. Two hands clapping, seen from the front.
   Hands apart / hands touching / hands touching with four short straight lines
   radiating outward.
7. A shaker: a closed cylinder with a handle.
   Upright / tilted to the right with two short motion lines / tilted to the
   left with two short motion lines.
8. Hi-hat cymbals on a stand, seen from the side.
   The two cymbals closed / a small gap between them / a wide gap with two
   short curved motion lines.
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

## Sheet 4 — the mouth, the kiss, and being asleep

The mouth mimes the record. The kiss is what the screen does back when somebody
presses a key. The sleeping row is for the quiet: a paused player that keeps
moving is a player that has not noticed.

```
Grid: 6 columns by 3 rows. Row 1 is six separate mouth shapes rather than an
animation. Row 2 is ONE animation of six frames, played once. Row 3 is one
animation of six frames, played once. Every mouth is drawn at the same width and
in the same position in its cell. Mouths are outlines only: no teeth, no tongue, no
lips, no face around them.

ROW 1, six mouth shapes for miming a song, one per cell:
   a closed straight line / a small round o / a tall narrow o / a wide flat
   lens shape / a large wide-open oval / a smile curving upwards.

ROW 2, blowing a kiss: six frames of ONE mouth, played once, left to right:
   mouth relaxed / mouth pursed and small / pursed with a tiny heart touching
   it / the heart just clear of the mouth, small / the heart further away and
   larger / the heart largest at the edge of the cell with two short motion
   lines behind it.

ROW 3, six drawings of the same cat, one per cell:
   curled up asleep, seen from the side / the same with one Z above it / with
   two Z's, the second larger / with three Z's rising / stretching with the
   front legs forward / sitting up awake with the ears up.
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
Grid: 8 columns by 4 rows. Each column is one dancing figure, the four rows are
four frames. CYCLED: frame 4 must lead back into frame 1 without a bump, so a
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

## Ideas without a sheet yet

Kept here so they are not lost, and because several of them need something in
the code before a drawing would help.

**The screen answering the person at it.** The figure blows a kiss when a key is
pressed — the kiss is on sheet 4, the reason for it is not built. Waving as
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

Before anything is cut, look for these. Any one of them is a redraw, because
none of them survives being baked.

- Two drawings in a column that are not the same object in the same place. This
  is the one that fails most often and the one that matters most.
- A stroke weight that drifts between cells, or a subject drawn small in its
  cell while its neighbours fill theirs.
- Anything filled in: a solid head, a black note, a shaded body.
- Detail finer than a tenth of the drawing's height — a face on a figure, teeth,
  fingers, strings drawn as five separate hairlines.
- Grey. There is no grey in the pipeline; it is either ink or it is not.
- A caption, a number, a frame, or one cell's line crossing into another.

Then cut, bake, and look at the row on screen at the size a normal terminal
actually gives it. That is the only judgement that counts.
