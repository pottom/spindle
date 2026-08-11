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
3. The cells go under `assets/marks/<set>/`, with a `mark.json` listing them.
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
cut, because it becomes part of the mark.

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
  and clear empty space between every cell. Nothing touches or overlaps.
- Where a column is one subject and the rows are its phases, the phases are the
  SAME drawing at the SAME size in the SAME position within the cell, with ONLY
  the moving part changed. Everything else must be identical.
- Every subject on the sheet is drawn by one hand, sharing one build and one
  pen, like a single icon family.

Style reference: clean pictogram line icons, the weight and simplicity of
Tabler Icons.
```

## Sheet 1 — instruments, animated

The row that has the clearest reason to exist: each mark already rides its own
band, so a kick at the left kicks on the kick and a hi-hat at the right opens on
the cymbals. Replaces a set assembled out of an icon library.

```
Grid: 8 columns by 3 rows. Each column is one instrument, the three rows are
three phases of it.

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
its head turned towards the viewer. The three rows are three phases.

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
phases of it.

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
Grid: 6 columns by 3 rows. Every mouth is drawn at the same width and in the
same position in its cell. Mouths are outlines only: no teeth, no tongue, no
lips, no face around them.

ROW 1, six mouth shapes for miming a song, one per cell:
   a closed straight line / a small round o / a tall narrow o / a wide flat
   lens shape / a large wide-open oval / a smile curving upwards.

ROW 2, blowing a kiss: six phases of ONE mouth, left to right:
   mouth relaxed / mouth pursed and small / pursed with a tiny heart touching
   it / the heart just clear of the mouth, small / the heart further away and
   larger / the heart largest at the edge of the cell with two short motion
   lines behind it.

ROW 3, six drawings of the same cat, one per cell:
   curled up asleep, seen from the side / the same with one Z above it / with
   two Z's, the second larger / with three Z's rising / stretching with the
   front legs forward / sitting up awake with the ears up.
```

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
