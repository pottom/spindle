package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pottom/spindle/internal/player"
)

func TestZZZProbe(t *testing.T) {
	m := searched(t, "queen", player.Results{
		Tracks:  page(player.Track{ID: "t1", Title: "Bohemian Rhapsody", Artists: []string{"Queen"}}),
		Artists: page(player.Artist{ID: "a1", Name: "Queen"}),
	})
	m.width, m.height = 140, 40
	m.resize()
	l := m.layout()
	block := m.searchPaneView(l, l.bodyHeight)
	band := m.listBandRows(l)
	for i, row := range block[:min(len(block), band+8)] {
		if i >= band {
			fmt.Printf("row %d (band+%d): %q\n", i, i-band, strings.TrimRight(plain(row), " ")[:min(60, len(strings.TrimRight(plain(row), " ")))])
		}
	}
	fmt.Printf("band=%d head=%d spans=%v leftMargin=%d\n", band, m.searchHeadRows(), m.searchViewSpans(), leftMargin)
	full := strings.Split(m.render(), "\n")
	for i, line := range full {
		if strings.Contains(plain(line), "all 2") {
			fmt.Printf("screen row %d: %q at col %d\n", i, plain(line)[:40], strings.Index(plain(line), "all 2"))
		}
	}
}
