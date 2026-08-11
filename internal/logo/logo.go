// Package logo carries the program's own picture.
//
// It lives beside the code rather than in assets/ with the drawings, because
// an embed directive cannot reach outside its own package's directory, and the
// alternative — a generated Go file with nine hundred kilobytes of base64 in it —
// is worse to read, worse to diff and worse to review.
//
// See internal/ui/splash.go for what it is for.
package logo

import _ "embed"

// PNG is the picture, 1024 by 682.
//
// Carried in the binary rather than fetched. It is shown while the playback
// device is being waited for, and what that wait usually is, is the network not
// answering — measured on this machine, tens of seconds of failed name lookups
// after the machine has been asleep. A logo downloaded on demand would be
// missing exactly when it was wanted.
//
//go:embed spindle.png
var PNG []byte
