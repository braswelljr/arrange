package common

// AppName is the canonical binary name used for daemon registration and cobra usage lines.
const AppName = "arrange"

var (
	// Version is injected at build time via -ldflags; defaults to "DEV" for local builds.
	Version = "DEV"
)
