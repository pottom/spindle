# One sheet, one request

An image generator returns one image per request, so twelve sheets is twelve
requests. These files exist so that none of them needs assembling: each one is
the whole brief for one sheet — the style block and that sheet's subjects — and
is pasted as it stands, with nothing added and nothing said before or after it.

They are generated from `docs/DRAWINGS.md`, which is where any change goes. To
rebuild them after editing that file, see the script in the same commit as this
note, or copy the style block and one sheet's fenced block together by hand.

Three things about the asking, learned by doing it wrong:

- **A fresh conversation for every sheet.** Asked one after another in one
  conversation, it starts drawing what it drew before — the second request came
  back with the first sheet's subjects in it.
- **Nothing but the file.** No "please", no "as we discussed", no reminder of
  what came before. Anything conversational is read as a subject to draw.
- **Two or three goes at the same sheet, then pick.** The pen is the same
  across runs of the same brief, so the best column of each can be taken and
  they will still look like one hand. A set does not have to come off one sheet;
  it has to come off one brief.
