package main

// Build metadata injected at release time via -ldflags (GoReleaser).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)
