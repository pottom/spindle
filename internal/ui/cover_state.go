package ui

// coverState is what the view needs to know about the artwork box: which URL it
// belongs to, the rendered art once it exists, and whether the attempt failed.
type coverState struct {
	url    string
	art    string
	failed bool
}

// loading reports whether the spinner belongs in the artwork box.
func (c coverState) loading() bool {
	return c.url != "" && c.art == "" && !c.failed
}
