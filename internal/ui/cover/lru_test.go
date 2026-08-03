package cover

import (
	"image"
	"image/color"
	"testing"
)

func TestImageLRUEvictsLeastRecentlyUsed(t *testing.T) {
	c := newImageLRU(2)
	a, b, d := solid(1, 1, color.Black), solid(1, 1, color.White), solid(1, 1, color.Opaque)

	c.put("a", a)
	c.put("b", b)

	// Touching "a" should make "b" the eviction candidate.
	if _, ok := c.get("a"); !ok {
		t.Fatal("a went missing")
	}
	c.put("d", d)

	if _, ok := c.get("b"); ok {
		t.Error("b should have been evicted")
	}
	for _, key := range []string{"a", "d"} {
		if _, ok := c.get(key); !ok {
			t.Errorf("%s should still be cached", key)
		}
	}
}

func TestImageLRUOverwriteDoesNotGrow(t *testing.T) {
	c := newImageLRU(2)
	var img image.Image = solid(1, 1, color.Black)

	c.put("a", img)
	c.put("a", img)
	c.put("b", img)

	if len(c.keys) != 2 {
		t.Errorf("got %d keys, want 2", len(c.keys))
	}
	if _, ok := c.get("a"); !ok {
		t.Error("a should not have been evicted by its own overwrite")
	}
}
