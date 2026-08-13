# Asking for a dance, one move at a time

A figure that dances to the music needs more than poses: it needs moves that can
be dealt one after another, held for as long as a phrase lasts, and joined
without a cut. That is a different thing from the sheets in `DRAWINGS.md` —
those are one image each, asked for in a fresh conversation, and split out by
`cmd/spindle-prompts`. This is a conversation: one sheet establishes the
character and the vocabulary, and then each move is redrawn on its own, large.

Nothing here is assembled by a tool. It is pasted by hand, in order, into one
conversation with an image generator.

Two blocks at the top are meant to be rewritten: **the character** and **the
dance**. Everything after them is the part that makes a drawing usable, and is
the same whoever is dancing.

## The character

Rewrite this to change who is dancing.

```
A stick figure: a round head with two dot eyes and a small smile, a straight
line for the body, straight stick arms and legs, and small solid blobs for the
hands and feet. No clothes, no hair, no detail. He is cheerful and light.
```

## The dance

Rewrite this to change the style. Seven moves, and the first one is the one he
returns to between the others — it must be the one he can do standing, for ever,
without going anywhere.

```
1. the bounce in place: stepping from foot to foot, arms swinging, upright
2. dropping to the floor onto the hands, one leg sweeping under
3. the floor sweep: on the hands, legs stretched out and sliding on the floor
4. the headstand: up on the head, legs opening and turning in the air
5. the six-step: on the hands, the legs circling under the body
6. the backspin: on the back, turning a full circle
7. side floor work: lying on one side, propped on one arm, kicking
```

## Why it is asked for twice

The first request is a sheet of all seven, a row each, small. It is not what
gets used: it is what settles the character and shows whether the seven moves
are seven different things. Every row of it begins and ends in the same standing
pose, which is what lets one move follow another without a join.

Then each row is asked for again on its own, at five times the size. A move
drawn at ninety pixels a frame is fine to look at and unusable once the limbs
cross each other on the floor, which is most of breakdancing.

## What every move has to have

Three things, and each of them makes the picture unusable rather than imperfect.

**One scale.** The figure is exactly the same size in every cell of the sheet.
Every drawing here is baked into dots at a fixed height, and a frame drawn
larger than its neighbours is a frame where the figure jumps. The head is the
measure: it must be the same diameter in every cell.

**One floor.** Whatever is on the ground — feet, a hand, a shoulder, the head —
sits at the same height in every cell. Without it a figure that lies down on
frame nine is a figure that sinks through the floor and climbs back out.

**A loop in the middle.** The music decides how long a move lasts, not the
drawing. If the middle of the move returns to where it started, it can be held
for two bars or for eight; if it does not, every move is exactly as long as the
number of frames it was drawn in, and the dance cannot follow the record. So a
move is: a standing pose, three frames into it, a loop of twelve, three frames
back out, and the standing pose again.

## The first request

Paste this into a new conversation, with the two blocks above filled in.

```
GENERATE A NEW IMAGE FROM NOTHING. There is no input image and nothing is being
edited: what follows describes a picture to draw from scratch.

Draw one reference sheet of a dancing figure: seven rows, one move to a row,
about seventeen frames across each row, read left to right.

THE FIGURE
[paste "The character" here]

THE MOVES — one to a row, in this order
[paste "The dance" here]

EVERY ROW BEGINS AND ENDS THE SAME WAY. The first cell of every row and the last
cell of every row are the identical neutral standing pose: facing the viewer,
arms slightly out from the body, feet apart. Draw that pose the same way in all
fourteen of those cells, so that any row can follow any other row.

STYLE
- Pure black line art on a pure white background. No grey, no fills, no shading,
  no colour, no shadows.
- Outline strokes only, one even stroke weight everywhere, rounded caps, and
  that weight is heavy: about a fortieth of the cell height.
- No motion lines, spin swooshes, speed arcs, dust, sweat drops or shadows. An
  animation is a moving part, not lines drawn around a still one.
- Flat and front-on. No perspective, no depth.

NO TEXT ANYWHERE ON THE IMAGE. No titles, no labels, no numbering of the frames,
no signature, no watermark. Not one letter and not one digit.

LAYOUT
- An even grid: every cell the same width and the same height.
- The figure is the same size in every cell. The head must be the same diameter
  in all of them. Never zoom in or out between cells.
- An invisible floor line three quarters of the way down every cell. Whatever
  touches the floor rests on it, at the same height, in every cell. Do not draw
  the line.
- Nothing crosses into a neighbouring cell. Keep a clear margin of at least a
  tenth of the cell on all four sides.
- Return ONE image, PNG, at the largest resolution available.
```

## The seven that follow

Then, in the same conversation, one message per move. The first carries the
rules and the rest point back at it.

```
Take row 1 of the sheet you just made — [the first move] — and redraw just that
one move, much larger, as its own sprite sheet.

Same character, same style, same line weight. Do not redesign him.

- A grid of exactly 4 columns x 5 rows = 20 cells on a 2048 x 2560 canvas.
- Every cell exactly 512 x 512 px, one pose per cell, left to right, top to
  bottom. No borders, no numbers, no text, no background.
- The figure is EXACTLY the same size in every cell: the head must be the same
  diameter in all 20 cells. Never zoom in or out between cells.
- An invisible floor line 75% down every cell. Whatever touches the floor rests
  on it, at the same height, in every cell. Do not draw the line.
- No motion lines, spin swooshes, speed arcs, dust, sweat or shadows.
- Pure black strokes on pure white. One even stroke weight, thick, about 1/40 of
  the cell height. No grey, no shading, no fills, no colour.
- Nothing crosses into a neighbouring cell: keep a margin of at least 10% of the
  cell on all four sides.
- All 20 cells are one seamless loop: cell 20 must flow straight back into cell
  1, so it can repeat for ever with no jump.
```

The first move is the one he stands and does, so the whole of it is the loop.
The other six go down to the floor and come back, so they are asked for with an
entry and an exit instead — same rules, different last line:

```
Now row 2 — [the second move]. Same canvas, same grid, same rules. But this one
is not one long loop: cell 1 is the neutral standing pose, cells 2-4 go into the
move, cells 5-16 are the move itself as a seamless loop (cell 16 flows back into
cell 5), cells 17-19 come back up, and cell 20 is the same standing pose as
cell 1.
```

```
Now row 3 — [the third move]. Same as the last one.
```

…and so on to row 7.

## What to check before saving one

Three things, in this order. The first is the only one that cannot be repaired
afterwards, so it is worth asking again rather than accepting.

1. **Is the head the same size in all twenty cells?** If one frame is drawn
   larger, the figure jumps on that frame and nothing downstream can fix it.
2. **Is everything on the floor at the same height?** Feet, hands, shoulders,
   head — one line across the whole sheet.
3. **Are there motion arcs, dust or shadows anywhere?** They were asked against,
   and they come back on the spinning moves in particular.

## Where the results go

Not settled. Seven PNGs at full resolution, one per move, kept together with
whatever manifest ends up describing them — the cutter has to be told the grid
and told which frames are the loop, and neither `assets/marks` nor
`assets/figures` says that today.
