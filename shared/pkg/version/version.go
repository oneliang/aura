// Package version provides version information for Aura.
package version

// Version information set via ldflags at build time
var (
	Version   string = "v0.0.1-alpha"
	BuildTime string = "unknown"
)

// GetVersion returns the version string
func GetVersion() string {
	if Version != "" {
		return Version
	}
	return "v0.0.1-alpha"
}

// GetBuildTime returns the build time string
func GetBuildTime() string {
	if BuildTime != "" {
		return BuildTime
	}
	return "unknown"
}

// FullVersion returns a formatted version string
func FullVersion() string {
	return GetVersion() + " (built: " + GetBuildTime() + ")"
}
