package config

// Version is set at build time via:
//
//	go build -ldflags "-X github.com/brixen96/video-storage-ai/internal/config.Version=x.y.z"
//
// Falls back to "dev" when building without the flag.
var Version = "dev"
