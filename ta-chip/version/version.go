package version

// Version is injected at build time via:
//   go build -ldflags="-X ta-chip/version.Version=v1.2.3"
var Version = "dev"
