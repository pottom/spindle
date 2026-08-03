// Package daemon runs spindle as a Spotify Connect device: it holds the
// librespot session, decodes and plays audio, and serves a local API the TUI
// drives it through.
//
// It exists because the public Web API cannot edit a queue — it offers only Get
// Queue and Add Item, with no reorder, remove or jump — and cannot push state,
// so a controller built on it has to poll. Speaking Spotify's own protocol buys
// both, at the cost of CGO and of Windows.
package daemon
