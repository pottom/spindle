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

Paste this above every sheet, unchanged. The three rules in capitals are the
ones that were broken the first time this was asked, and each of them makes the
sheet unusable rather than imperfect.

```
RETURN ONE SHEET ONLY. Draw exactly the one sheet described below. Do not add
other sheets, do not combine several sheets into one image, do not include
extra rows of subjects that were not asked for. One sheet, one image.

NO TEXT ANYWHERE ON THE IMAGE. No titles, no headings, no captions, no column
names, no row labels, no numbering of the frames, no legend, no signature, no
watermark. Not one letter and not one digit anywhere on the sheet, including
the margins. Every mark on the image is part of a drawing.

AN ANIMATION IS A MOVING PART, NOT MOTION LINES. Between two frames, some part
of the subject must actually change position or shape: a beater falls, a mouth
opens, cymbals part, a tail bends, legs pass each other. Drawing the subject
unchanged and adding short radiating lines around it is NOT an animation and
will be rejected. Short decorative lines are too fine to survive at all: they
disappear when the drawing is reduced. If a subject cannot be animated by
moving one of its own parts, choose a different pose for it, but never fall
back on motion lines.

STYLE
- Pure black background (#000000). All drawing in pure white (#FFFFFF).
- No grey, no fills, no shading, no gradients, no glow, no colour.
- Outline strokes ONLY. Nothing is filled in. No solid shapes, no silhouettes.
- ONE constant stroke weight everywhere, rounded caps and rounded joins.
- Simple enough to survive being reduced to a 24 by 24 grid of dots: no fine
  detail, no hatching, no small features, no textures, no thin decoration.
  Any feature smaller than a tenth of the subject's height will be lost.
- Flat and front-on. No perspective, no 3/4 view, no depth, no shadows.

SIZE
- Return ONE image at the largest resolution available, and no smaller than
  2000 pixels on its long side.
- Each subject is drawn LARGE: it fills at least three quarters of the height
  of its own cell. A sheet where the drawings are small in the middle of empty
  cells is a sheet that cannot be used.

LAYOUT
- An invisible grid, evenly spaced, with a wide empty margin around the sheet
  and clear empty black space between every cell.
- NOTHING MAY OVERLAP OR TOUCH ANYTHING ELSE. Every drawing stays entirely
  inside its own cell. No drawing crosses into a neighbouring cell, no subject
  spans two cells or two rows, no two drawings share a line, no ground line or
  horizon runs across the sheet, nothing is cropped by the edge of the sheet.
  Each cell must be liftable out of the sheet as a plain rectangle containing
  one complete drawing and nothing else.
- A column is one subject and the rows under it are that subject's animation
  frames: the SAME drawing at the SAME size in the SAME position within its
  cell, with ONLY the moving part changed. Nothing that is not moving may shift
  by even a little between frames.
- Every subject on the sheet is drawn by one hand, sharing one build and one
  pen, like a single icon family.

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

## Sheet 1 — the kit, animated

The row that has the clearest reason to exist: every mark rides its own slice of
the spectrum, so a kick at the left kicks on the kick and a cymbal at the right
rings on the cymbals. A drum kit spread across the row is that order without
anybody being told it — this is the set the first attempt invented, kept, with
every column now moving a part of itself instead of growing motion lines.

```
Grid: 8 columns by 3 rows. Each column is one drum or one pair of hands, the
three rows are three frames of it. All eight are STRUCK: they return to where
they started, so three frames is the whole animation.

Every column must move a part. In each of the eight, name the part that moves:

1. Kick drum, seen from the front, with its pedal and beater to one side.
   Beater standing up, away from the head / beater halfway down /
   beater flat against the head, the head bowed slightly inwards.
2. Snare drum, seen from the side, with one stick above it.
   Stick raised well above the drum / stick halfway down /
   stick touching the head, the head bowed slightly downwards.
3. Two hands clapping, seen from the front, fingers up.
   Hands well apart / hands nearly touching / hands flat together as one shape.
4. Hi-hat cymbals on a stand, seen from the side, with the pedal at the bottom.
   Cymbals together and the pedal down / cymbals apart by a little, pedal
   halfway / cymbals wide apart and the pedal up.
5. Open hi-hat, the two cymbals already apart, with a stick above the top one.
   Stick raised / stick halfway down / stick on the top cymbal, which is tilted
   by the blow.
6. Rim shot: a shallow drum seen from above as an ellipse, one stick lying
   across the rim.
   Stick raised above the rim / stick halfway / stick across the rim, the
   ellipse pushed slightly out of round by the blow.
7. Floor tom, seen from the side on its legs, with one stick above.
   Stick raised / stick halfway / stick on the head, the head bowed downwards.
8. Ride cymbal on a stand, seen from the side, with one stick above it.
   Stick raised, cymbal flat / stick on the cymbal, cymbal tilted one way /
   stick lifting away, cymbal tilted the other way.
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
