[![Go Report Card](https://goreportcard.com/badge/github.com/layer87-labs/relctl)](https://goreportcard.com/report/github.com/layer87-labs/relctl)
[![GitHub release](https://img.shields.io/github/release/layer87-labs/relctl.svg)](https://github.com/layer87-labs/relctl/releases/latest)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/layer87-labs/relctl.svg)](https://github.com/layer87-labs/relctl)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/layer87-labs/relctl/blob/main/LICENSE)

[![Publish Release](https://github.com/layer87-labs/relctl/actions/workflows/Release.yaml/badge.svg)](https://github.com/layer87-labs/relctl/actions/workflows/Release.yaml)
[![gh-pages](https://github.com/layer87-labs/relctl/actions/workflows/pages/pages-build-deployment/badge.svg)](https://github.com/layer87-labs/relctl/actions/workflows/pages/pages-build-deployment)

# relctl

**Description**: relctl is a provider-agnostic release management CLI for CI/CD pipelines.
It supports two versioning schemes – [SemVer](https://semver.org/) and [CalVer](https://calver.org/) –
and integrates with GitHub, GitHub Enterprise, GitLab, and Jenkins.

- **Technology stack**: Go, Cobra CLI
- **Status**: Stable
- **Supported environments**:
  - GitHub & GitHub Enterprise (GitHub Actions)
  - GitLab (GitLab CI)
  - Jenkins Pipelines
- **Versioning schemes**: SemVer (branch-prefix driven) · CalVer (date-based, git-tag driven)

## Getting Started

Download the [latest release](https://github.com/layer87-labs/relctl/releases/latest/download/relctl)
and add it to your `PATH`, or use the
[layer87-labs/relctl-action](https://github.com/layer87-labs/relctl-action) in GitHub Actions.

## Versioning Schemes

### SemVer (default)

relctl derives the SemVer bump level from the source branch name:

| Branch prefix | Bump |
|---|---|
| `bugfix/`, `fix/`, `patch/`, `dependabot/` | Patch |
| `feature/`, `feat/`, `minor/` | Minor |
| `major/` | Major |

### CalVer

Format: **`YYYY.MM.DD.N`** – e.g. `2026.06.01.3`

- `YYYY.MM.DD` – current date in UTC
- `N` – monotonically increasing counter per day, 1-based

N is calculated **exclusively from local Git tags** – no SCM API call, fully provider-agnostic.
relctl lists all tags matching `YYYY.MM.DD.*` for today and sets N to `max(N) + 1` (or `1` if no tag exists yet).

> **Prerequisite**: the repository must be checked out with full tag history (`fetch-depth: 0`).
> This is already a general relctl requirement.

## Configuration

### `.relctl.yaml` (repo-level config file)

Place a `.relctl.yaml` in your repository root to set project-wide defaults:

```yaml
version_scheme: calver   # semver (default) | calver
default_branch: main
```

The file is optional. When absent, relctl behaves exactly as before (SemVer, `main`).

**Priority**: `--version-scheme` flag > `.relctl.yaml` > built-in default (SemVer)

### CLI Flags

| Flag | Scope | Description |
|---|---|---|
| `--config <path>` | `relctl`, `release` | Override config file path (default: `.relctl.yaml`) |
| `--version-scheme <scheme>` | `release` | `semver` or `calver`; overrides config file |

## Usage

### SemVer release (existing workflow, unchanged)

```bash
# After merging a PR – relctl reads the PR branch and GitHub API
relctl release create
relctl release publish --release-id "$RELCTL_RELEASE_ID" --asset "file=dist/binary"
```

### CalVer release via config file

```yaml
# .relctl.yaml
version_scheme: calver
```

```bash
# No branch prefix or PR context required
relctl release create
# → e.g. 2026.06.01.1

relctl release publish --release-id "$RELCTL_RELEASE_ID" --asset "file=dist/binary"
```

### CalVer release via flag (no config file needed)

```bash
relctl release create --version-scheme calver
# → e.g. 2026.06.01.1

# Second release on the same day (git tag 2026.06.01.1 already exists)
relctl release create --version-scheme calver
# → 2026.06.01.2
```

### Dry-run (preview version without creating a release)

```bash
relctl release create --version-scheme calver --dry-run
# Would create new release with version: 2026.06.01.1
```

### GitHub Actions example

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0   # required: full tag history for N calculation

- name: Create CalVer release
  run: relctl release create --version-scheme calver
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Examples

More examples are available in the [examples section](https://layer87-labs.github.io/relctl/docs/examples)
of the documentation.

## Frequently Asked Questions

See the [Q&A section](https://layer87-labs.github.io/relctl/docs/questions_and_answers).

## Getting Help

Please file an issue in this repository's [Issue Tracker](https://github.com/layer87-labs/relctl/issues).

## Community

- [Contributing](https://github.com/layer87-labs/.github/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://github.com/layer87-labs/.github/blob/main/CODE_OF_CONDUCT.md)
- [Security Policy](https://github.com/layer87-labs/.github/blob/main/SECURITY.md)

## License

relctl is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).

## Credits

- [SemVer](https://semver.org/)
- [CalVer](https://calver.org/)
- [Cobra CLI](https://github.com/spf13/cobra)
- [go-git](https://github.com/go-git/go-git)
