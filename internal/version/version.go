package version

// Version is set at build time via ldflags:
//
//	-X github.com/blindly/ops/internal/version.Version=v0.2.0
var Version = "dev"
