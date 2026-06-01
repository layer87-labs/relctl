package versioning

// ReleaseContext holds the information available to a VersionScheme
// when calculating the next version. It is intentionally provider-agnostic:
// all data derives from local Git state.
type ReleaseContext struct {
	// RepoPath is the path to the local Git repository root.
	RepoPath string
}

// VersionScheme is the interface every versioning strategy must implement.
// Implementations must be stateless and side-effect-free – they only read
// local Git state through the supplied ReleaseContext.
type VersionScheme interface {
	// NextVersion calculates and returns the next version string.
	NextVersion(ctx ReleaseContext) (string, error)

	// Validate reports whether the given version string is a valid version
	// for this scheme.
	Validate(version string) bool
}
