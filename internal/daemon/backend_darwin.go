package daemon

// audioBackend on macOS is AudioToolbox, which is part of the system.
const audioBackend = "audio-toolbox"

// audioDevice is empty because AudioToolbox does not take one: it asks
// CoreAudio for the system's default output device and follows it wherever the
// listener moves it. See backend_unix.go, where the name is not optional.
const audioDevice = ""
