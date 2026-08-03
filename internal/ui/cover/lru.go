package cover

import "image"

// imageLRU keeps a handful of decoded images around. Decoding a 640px JPEG is
// the most expensive step of the pipeline, and a four-track rotation revisits
// the same covers constantly.
type imageLRU struct {
	limit int
	keys  []string // most recently used last
	items map[string]image.Image
}

func newImageLRU(limit int) *imageLRU {
	return &imageLRU{limit: limit, items: make(map[string]image.Image, limit)}
}

func (c *imageLRU) get(key string) (image.Image, bool) {
	img, ok := c.items[key]
	if ok {
		c.touch(key)
	}
	return img, ok
}

func (c *imageLRU) put(key string, img image.Image) {
	if _, ok := c.items[key]; !ok {
		c.keys = append(c.keys, key)
	} else {
		c.touch(key)
	}
	c.items[key] = img

	for len(c.keys) > c.limit {
		oldest := c.keys[0]
		c.keys = c.keys[1:]
		delete(c.items, oldest)
	}
}

func (c *imageLRU) touch(key string) {
	for i, k := range c.keys {
		if k == key {
			c.keys = append(append(c.keys[:i:i], c.keys[i+1:]...), key)
			return
		}
	}
}
