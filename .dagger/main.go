// Package main is the entry point for the l87-relctl Dagger module.
//
// l87-relctl exposes the full relctl CLI as a Dagger module so that CI
// pipelines can:
//   - Query git metadata (CommitHash, Version, NextVersion)
//   - Build the relctl binary from source (Binary)
//   - Get a pre-authenticated runtime container (Container)
//   - Create and publish GitHub releases (ReleaseCreate, ReleasePublish)
//   - Retrieve pull-request information (PrInfo)
package main

import (
	"context"
	"fmt"
	"strings"

	"dagger/l-87-relctl/internal/dagger"
)

const (
	// gitImage is used for lightweight git operations (no auth needed).
	gitImage = "alpine/git:2"
	// goImage is used to compile the relctl binary.
	goImage = "golang:1.26-alpine"
	// buildPkg is the Go package path where ldflags inject version info.
	buildPkg = "github.com/layer87-labs/relctl/internal/app/build"
	// defaultServer is the GitHub instance used when none is specified.
	defaultServer = "https://github.com"
)

// L87Relctl is the root Dagger module type.
type L87Relctl struct{}

// ─── Git helpers (no auth required) ─────────────────────────────────────────

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
// Returns "unknown" when the repository has no commits or no .git directory.
//
// Example:
//
//	dagger call commit-hash --source=.
func (m *L87Relctl) CommitHash(ctx context.Context, source *dagger.Directory) (string, error) {
	out, err := m.gitCtr(source).
		WithExec([]string{"git", "rev-parse", "--short", "HEAD"}).
		Stdout(ctx)
	if err != nil {
		return "unknown", nil //nolint:nilerr // best-effort
	}
	return strings.TrimSpace(out), nil
}

// Version returns the latest SemVer git tag reachable from HEAD.
// Returns "dev" when no tag is found or the repository has no tags yet.
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

// NextVersion computes the next SemVer bump based on the branch-name prefix:
//
//   - bugfix/ fix/ patch/ dependabot/ (and everything else) → patch bump
//   - feature/ feat/ minor/                                  → minor bump
//   - major/                                                 → major bump
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

// ─── Binary build ────────────────────────────────────────────────────────────

// Binary builds the relctl binary from source and returns the compiled file.
// Version and commit hash are embedded at compile time via ldflags.
// When not provided, both values are auto-detected from the git history.
//
// Example:
//
//	dagger call binary --source=. export --path=./out/relctl
func (m *L87Relctl) Binary(
	ctx context.Context,
	source *dagger.Directory,
	// +optional
	version string,
	// +optional
	commitHash string,
) *dagger.File {
	if version == "" {
		v, _ := m.Version(ctx, source)
		version = v
	}
	if commitHash == "" {
		h, _ := m.CommitHash(ctx, source)
		commitHash = h
	}
	ldflags := fmt.Sprintf(
		"-s -w -X %s.Version=%s -X %s.CommitHash=%s",
		buildPkg, version, buildPkg, commitHash,
	)
	return dag.Container().
		From(goImage).
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("relctl-go-mod")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("relctl-go-build")).
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithEnvVariable("CGO_ENABLED", "0").
		WithExec([]string{
			"go", "build",
			"-ldflags", ldflags,
			"-o", "/out/relctl",
			"./cmd/relctl",
		}).
		File("/out/relctl")
}

// ─── Runtime container ───────────────────────────────────────────────────────

// Container returns a minimal Alpine container with the relctl binary and
// GitHub credentials pre-configured. Use this as an escape hatch to run any
// relctl subcommand not covered by the typed helper functions.
//
// The container is configured in GitHub Actions detection mode so relctl
// automatically picks up the credentials:
//   - CI=true
//   - GITHUB_SERVER_URL=<server>
//   - GITHUB_REPOSITORY=<repository>
//   - GITHUB_TOKEN=<token> (injected as a secret — never exposed in logs)
//
// The source directory is mounted at /repo and set as the working directory.
//
// Example:
//
//	dagger call container --source=. --token=env:GITHUB_TOKEN \
//	  --repository=my-org/my-repo \
//	  with-exec --args="relctl,release,create,--dry-run" stdout
func (m *L87Relctl) Container(
	ctx context.Context,
	source *dagger.Directory,
	token *dagger.Secret,
	repository string,
	// +optional
	server string,
) *dagger.Container {
	if server == "" {
		server = defaultServer
	}
	return dag.Container().
		From("alpine:3").
		WithFile("/usr/local/bin/relctl", m.Binary(ctx, source, "", "")).
		WithMountedDirectory("/repo", source).
		WithWorkdir("/repo").
		WithEnvVariable("CI", "true").
		WithEnvVariable("GITHUB_SERVER_URL", server).
		WithEnvVariable("GITHUB_REPOSITORY", repository).
		WithSecretVariable("GITHUB_TOKEN", token)
}

// ─── Release commands ────────────────────────────────────────────────────────

// ReleaseCreate creates a new GitHub release by running `relctl release create`.
// Returns the command stdout which contains the newly created release ID.
//
// Example:
//
//	dagger call release-create \
//	  --source=. --token=env:GITHUB_TOKEN --repository=my-org/my-repo
func (m *L87Relctl) ReleaseCreate(
	ctx context.Context,
	source *dagger.Directory,
	token *dagger.Secret,
	repository string,
	// +optional
	server string,
	// +optional
	body string,
	// +optional
	patchLevel string,
	// +optional
	version string,
	// +optional
	mergeSha string,
	// +optional
	releaseBranch string,
	// +optional
	releasePrefix string,
	// +optional
	prNumber int,
	// +optional
	dryRun bool,
	// +optional
	hotfix bool,
) (string, error) {
	args := []string{"relctl", "release", "create"}
	if body != "" {
		args = append(args, "--body", body)
	}
	if patchLevel != "" {
		args = append(args, "--patch-level", patchLevel)
	}
	if version != "" {
		args = append(args, "--version", version)
	}
	if mergeSha != "" {
		args = append(args, "--merge-sha", mergeSha)
	}
	if releaseBranch != "" {
		args = append(args, "--release-branch", releaseBranch)
	}
	if releasePrefix != "" {
		args = append(args, "--release-prefix", releasePrefix)
	}
	if prNumber != 0 {
		args = append(args, "--prnumber", fmt.Sprintf("%d", prNumber))
	}
	if dryRun {
		args = append(args, "--dry-run")
	}
	if hotfix {
		args = append(args, "--hotfix")
	}
	out, err := m.Container(ctx, source, token, repository, server).
		WithExec(args).
		Stdout(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ReleasePublish publishes a previously created GitHub release and optionally
// uploads assets. Run ReleaseCreate first to obtain the release ID.
//
// Assets are specified as typed strings:
//   - "file=<path>"   — upload a single file
//   - "zip=<dir>"     — zip a directory and upload it
//   - "tgz=<dir>"     — tar.gz a directory and upload it
//
// Example:
//
//	dagger call release-publish \
//	  --source=. --token=env:GITHUB_TOKEN --repository=my-org/my-repo \
//	  --release-id=12345 \
//	  --assets=file=./out/relctl_linux_amd64 \
//	  --assets=file=./out/relctl_linux_arm64
func (m *L87Relctl) ReleasePublish(
	ctx context.Context,
	source *dagger.Directory,
	token *dagger.Secret,
	repository string,
	releaseID int,
	// +optional
	server string,
	// +optional
	assets []string,
	// +optional
	body string,
	// +optional
	mergeSha string,
	// +optional
	prNumber int,
) (string, error) {
	args := []string{
		"relctl", "release", "publish",
		"--release-id", fmt.Sprintf("%d", releaseID),
	}
	for _, a := range assets {
		args = append(args, "--asset", a)
	}
	if body != "" {
		args = append(args, "--body", body)
	}
	if mergeSha != "" {
		args = append(args, "--merge-sha", mergeSha)
	}
	if prNumber != 0 {
		args = append(args, "--prnumber", fmt.Sprintf("%d", prNumber))
	}
	out, err := m.Container(ctx, source, token, repository, server).
		WithExec(args).
		Stdout(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ─── Pull-request commands ───────────────────────────────────────────────────

// PrInfo retrieves pull-request information by running `relctl pr info`.
// Specify either a PR number or the merge commit SHA to identify the PR.
//
// Example:
//
//	dagger call pr-info \
//	  --source=. --token=env:GITHUB_TOKEN --repository=my-org/my-repo \
//	  --number=42
func (m *L87Relctl) PrInfo(
	ctx context.Context,
	source *dagger.Directory,
	token *dagger.Secret,
	repository string,
	// +optional
	server string,
	// +optional
	number int,
	// +optional
	mergeSha string,
) (string, error) {
	args := []string{"relctl", "pr", "info"}
	if number != 0 {
		args = append(args, "--number", fmt.Sprintf("%d", number))
	}
	if mergeSha != "" {
		args = append(args, "--merge-commit-sha", mergeSha)
	}
	out, err := m.Container(ctx, source, token, repository, server).
		WithExec(args).
		Stdout(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ─── Internal helpers ────────────────────────────────────────────────────────

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
