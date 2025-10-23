package version

// These variables are set at build time using -ldflags
var (
	// GitCommit is the short git commit hash (set via -ldflags "-X ...")
	GitCommit = "dev"

	// Version is the semantic version (can be set later if needed)
	Version = "1.0.0"
)

// Info returns version information.
func Info() map[string]string {
	return map[string]string{
		"version":    Version,
		"git_commit": GitCommit,
	}
}
