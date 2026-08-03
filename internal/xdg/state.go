package xdg

// StateDir is $XDG_STATE_HOME/spindle, or ~/.local/state/spindle. It is where
// logs go: data that is useful across restarts but that nobody would miss if it
// were deleted, which is neither config nor cache.
func StateDir() (string, error) {
	return appDir("XDG_STATE_HOME", ".local/state", 0o700)
}
