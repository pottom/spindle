// Command spindle-prompts splits the drawing brief into one file per request.
//
// An image generator answers one image to a request, so the twelve sheets in
// docs/DRAWINGS.md are twelve requests. Each one has to carry the whole brief
// with it — the style block and that sheet's subjects — because there is no
// conversation to carry it: every sheet is asked for in a fresh one, or the
// generator starts redrawing what it drew before.
//
// So the files under docs/prompts are assembled rather than written. Run this
// after editing the brief:
//
//	go run ./cmd/spindle-prompts
//
// Anything else in that directory is left alone.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	brief = "docs/DRAWINGS.md"
	into  = "docs/prompts"
)

var (
	sheetAt = regexp.MustCompile(`(?m)^## Sheet (\d+) — (.+)$`)
	notWord = regexp.MustCompile(`[^a-z0-9]+`)
)

func main() {
	raw, err := os.ReadFile(brief)
	if err != nil {
		fail(err)
	}
	doc := string(raw)

	style, err := fenced(doc, "## The style block")
	if err != nil {
		fail(fmt.Errorf("the style block: %w", err))
	}

	found := sheetAt.FindAllStringSubmatchIndex(doc, -1)
	if len(found) == 0 {
		fail(fmt.Errorf("%s: no sheets in it", brief))
	}
	if err := os.MkdirAll(into, 0o755); err != nil {
		fail(err)
	}

	for i, at := range found {
		n, title := doc[at[2]:at[3]], doc[at[4]:at[5]]
		end := len(doc)
		if i+1 < len(found) {
			end = found[i+1][0]
		}
		// The sheet's own block, which is everything the generator is told about
		// what to draw. The prose above it is for whoever is reading the brief.
		body, err := fencedIn(doc[at[1]:end])
		if err != nil {
			fail(fmt.Errorf("sheet %s: %w", n, err))
		}

		name := fmt.Sprintf("sheet-%02s-%s.txt", n, notWord.ReplaceAllString(strings.ToLower(title), "-"))
		name = strings.TrimSuffix(name, "-.txt") + ""
		out := style + "\n\nTHE SHEET TO DRAW\n\n" + body + "\n"
		if err := os.WriteFile(filepath.Join(into, name), []byte(out), 0o644); err != nil {
			fail(err)
		}
		fmt.Printf("wrote %s (%d words)\n", filepath.Join(into, name), len(strings.Fields(out)))
	}
}

// fenced is the first fenced block after a heading.
func fenced(doc, heading string) (string, error) {
	at := strings.Index(doc, heading)
	if at < 0 {
		return "", fmt.Errorf("no %q", heading)
	}
	return fencedIn(doc[at+len(heading):])
}

func fencedIn(s string) (string, error) {
	open := strings.Index(s, "```")
	if open < 0 {
		return "", fmt.Errorf("no fenced block")
	}
	rest := s[open+3:]
	shut := strings.Index(rest, "```")
	if shut < 0 {
		return "", fmt.Errorf("a fenced block was not closed")
	}
	return strings.Trim(rest[:shut], "\n"), nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "spindle-prompts:", err)
	os.Exit(1)
}
