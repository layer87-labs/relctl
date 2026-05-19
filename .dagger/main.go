// Package main is the entry point for the l87-relctl Dagger module.
//
// l87-relctl exposes release-management helpers for Layer87 CI pipelines:
//   - CommitHash: short SHA of HEAD (for build metadata)
//   - Version: latest SemVer git tag (or "dev")
//   - NextVersion: next bumped version derived from the branch-name prefix
//
// Functions are intentionally small and composable so that dagger-pipelines
// (and other callers) can build higher-level workflows on top.
package main

import (
	"context"
	"strings"

	"dagger/l-87-relctl/internal/dagger"
)

// gitImage is the Alpine git image used for all git operations.
const gitImage = "alpine/git:2"

// L87Relctl is the root Dagger module type.
type L87Relctl struct{}

// gitCtr returns a base container with git and the source mounted at /src.
func (m *L87Relctl) gitCtr(source *dagger.Directory) *dagger.Container {
	return dag.Container().
		From(gitImage).
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		// Tell git it's safe to operate on this directory (ownership may differ inside container)
		WithExec([]string{"git", "config", "--global", "--add", "safe.directory", "/src"})
}

// CommitHash returns the short (7-char) git commit hash of HEAD.
//
// The source directory must contain a .git directory (i.e. the root of a
// git repository). Pass the repo root via --source=. on the command line.
//
// Example:
//
//	dagger call commit-hash --source=.
func (m *L87Relctl) CommitHash(ctx context.Context, source *dagger.Directory) (string, error) {
	out, err := m.gitCtr(source).
		WithExec([]string{"git", "rev-parse", "--short", "HEAD"}).
		Stdout(ctx)
	if err != nil {
		return "unknown", nil //nolint:nilerr // best-effort, callers fall back to "unknown"
	}
	return strings.TrimSpace(out), nil
}

// Version returns the latest SemVer git tag reachable from HEAD.
// Returns "dev" when no tag is found or when the repository has no tags yet.
//
// Example:
//
//	dagger call version --source=.
func (m *L87Relctl) Version(ctx context.Context, source *dagger.Directory) (string, error) {
	out, err := m.gitCtr(source).
		WithExec([]string{"sh", "-c", `git describe --tags --abbrev=0 2>/dev/null || echo "dev"`}).
		Stdout(ctx)
	if err != nil {
		return "dev", nil //nolint:nilerr
	}
	v := strings.TrimSpace(out)
	if v == "" {
		return "dev", nil
	}
	return v, nil
}

// NextVersion computes the next SemVer version bump based on the current
// branch-name prefix convention used in Layer87 repositories:
//
//   - bugfix/ fix/ patch/ dependabot/ → patch bump
//   - feature/ feat/ minor/           → minor bump
//   - major/                          → major bump
//
// Returns the bumped version string (e.g. "1.3.0").
// The base version is the latest tag reachable from HEAD (or "0.0.0" if none).
//
// Example:
//
//	dagger call next-version --source=. --branch=feature/my-feature
func (m *L87Relctl) NextVersion(ctx context.Context, source *dagger.Directory, branch string) (string, error) {
	base, err := m.Version(ctx, source)
	if err != nil || base == "dev" {
		base = "0.0.0"
	}
	bump := bumpFromBranch(branch)
	out, err := dag.Container().
		From(gitImage).
		WithExec([]string{"sh", "-c", bumpScript(base, bump)}).
		Stdout(ctx)
	if err != nil {
		return base, nil //nolint:nilerr
	}
	return strings.TrimSpace(out), nil
}

// bumpFromBranch maps a branch-name prefix to a semver bump level.
func bumpFromBranch(branch string) string {
	prefix := branch
	if idx := strings.Index(branch, "/"); idx > 0 {
		prefix = branch[:idx]
	}
	switch strings.ToLower(prefix) {
	case "major":
		return "major"
	case "feature", "feat", "minor":
		return "minor"
	default: // bugfix, fix, patch, dependabot, anything else → patch
		return "patch"
	}
}

// bumpScript returns a sh one-liner that bumps a semver string.
// Relies only on POSIX shell arithmetic — no extra tools needed.
func bumpScript(version, bump string) string {
	// Strip leading "v" prefix if present
	v := strings.TrimPrefix(version, "v")
	parts := strings.SplitN(v, ".", 3)
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	switch bump {
	case "major":
		return `echo "$(( ` + parts[0] + ` + 1 )).0.0"`
	case "minor":
		return `echo "` + parts[0] + `.$(( ` + parts[1] + ` + 1 )).0"`
	default:
		return `echo "` + parts[0] + `.` + parts[1] + `.$(( ` + parts[2] + ` + 1 ))"`
	}
}
