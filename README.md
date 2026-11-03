<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/lockup-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="assets/lockup-light.svg">
    <img alt="relctl" src="assets/lockup-light.svg" width="360">
  </picture>
</p>

[![Go Report Card](https://goreportcard.com/badge/github.com/layer87-labs/relctl)](https://goreportcard.com/report/github.com/layer87-labs/relctl)
[![GitHub release](https://img.shields.io/github/release/layer87-labs/relctl.svg)](https://github.com/layer87-labs/relctl/releases/latest)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/layer87-labs/relctl.svg)](https://github.com/layer87-labs/relctl)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/layer87-labs/relctl/blob/main/LICENSE)

# relctl

**relctl** is a release management CLI for CI/CD pipelines. It works out the next version of your
project, creates the GitHub release and keeps your pipeline free of release boilerplate. It is the
successor of [awesome-ci](https://github.com/fullstack-devops/awesome-ci) (see
[migrating](https://labs.layer87.de/docs/relctl/migrating-from-awesome-ci)).

- **Source control**: GitHub and GitHub Enterprise.
- **CI system**: GitHub Actions and Jenkins are detected automatically. Any other pipeline (including GitLab CI)
  connects once with `relctl connect github`.
- **Four ways to get the version**: from the branch prefix, from Conventional Commits, from the date, or
  from a reviewed `CHANGELOG.md`.
- **Technology**: Go, Cobra. Status: stable.

> GitLab as source control system is not supported. If you need it, please
> [open an issue](https://github.com/layer87-labs/relctl/issues) and describe your use case.

## Quick start (5 minutes, GitHub Actions)

**1. Name your branches by what they change.** The prefix decides the version bump of the pull request:

| Branch prefix | Bump | Example |
| --- | --- | --- |
| `fix/`, `bugfix/`, `patch/`, `dependabot/` | patch | 1.2.3 → 1.2.4 |
| `feat/`, `feature/`, `minor/` | minor | 1.2.3 → 1.3.0 |
| `major/`, or any prefix ending in `!` (`feat!/`) | major | 1.2.3 → 2.0.0 |

**2. Add a release workflow.** Every merge to `main` creates a draft release, builds, and publishes it:

```yaml
# .github/workflows/release.yaml
name: Release
on:
  push:
    branches: [main]

permissions:
  contents: write        # create and publish releases
  pull-requests: write   # relctl posts a help comment on pull requests

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with: { fetch-depth: 0 }

      - uses: layer87-labs/relctl-action@v1
        with: { github-token: "${{ secrets.GITHUB_TOKEN }}" }

      - id: release
        run: relctl release create --merge-sha ${{ github.sha }}
        env: { GITHUB_TOKEN: "${{ secrets.GITHUB_TOKEN }}" }

      # ... build your artifacts, use ${{ steps.release.outputs.RELCTL_VERSION }} ...

      - run: relctl release publish --release-id ${{ steps.release.outputs.RELCTL_RELEASE_ID }} --asset "file=dist/app"
        env: { GITHUB_TOKEN: "${{ secrets.GITHUB_TOKEN }}" }
```

**3. Preview the version on pull requests** (optional): `relctl pr info` prints the version a pull request
would produce, and relctl posts a help comment on the pull request.

That is all. Without a config file relctl uses SemVer from the branch prefix.

### Overriding the version on a pull request

Comment on the pull request before merging. The comment must start with the command:

| Comment | Effect |
| --- | --- |
| `relctl_patch_level: major` | force a bump level |
| `relctl_version_override: 2.1.0` | pin the version |

## Choose your flow

| | Flow 1 | Flow 2 | Flow 3 |
| --- | --- | --- | --- |
| Release | every merge to `main` | every merge to `main` | merging a release pull request |
| Version from | branch prefix | commit messages or date | `CHANGELOG.md` heading |
| `version_scheme` | `semver` (default) | `conventional-commits`, `calver` | `changelog` |
| Good for | libraries, CLIs | services, apps | products with deliberate releases |

### Flow 3 in short: reviewed releases from a development branch

Work lands on `dev`; the version is written down in `CHANGELOG.md` and visible in review:

```markdown
## 1.4.0                   <- upcoming (on dev)
## 1.4.0 — 2026-10-02      <- released: version, dash, ISO date
```

```bash
relctl release create --prerelease     # every push to dev: 1.4.0-rc.1, rc.2, ... (prerelease)
relctl release gate --sha "$HEAD_SHA"  # required check on the release pull request (dev -> main)
relctl release create                  # on main: creates 1.4.0 from the dated heading, idempotent
relctl release forward                 # afterwards: bring main back into dev
```

The gate passes only if the heading is dated and has notes, the version is not released yet, and a release
candidate exists for exactly the commit being released. See the
[release flows](https://labs.layer87.de/docs/relctl/release-flows).

## Configuration: `.relctl.yaml`

Optional. Place it in the repository root. Every key is optional:

```yaml
# how the version is calculated: semver (default) | conventional-commits | calver | changelog
version_scheme: semver

# changelog scheme: the file with the "## X.Y.Z — YYYY-MM-DD" headings (default CHANGELOG.md)
changelog_file: CHANGELOG.md

# branches for `relctl release forward`
production_branch: main      # receives the releases (default: main)
development_branch: dev      # default: the default branch of the repository

# tags and releases get this prefix: "v" gives v1.2.3 (the version itself stays 1.2.3)
tag_prefix: v

# independently released parts of the repository (own tags, own latest release, own counter)
components:
  whisper:
    paths: ["deploy/whisper/**", "deploy/Containerfile.whisper"]
    version_scheme: calver   # components use calver or changelog
```

`default_branch` is the former name of `production_branch` and is still read.

**Priority**: command-line flag > `.relctl.yaml` > built-in default.

## Commands

| Command | What it does |
| --- | --- |
| `relctl release create` | compute the version and create a draft release (`--prerelease` for a release candidate, `--dry-run` to preview) |
| `relctl release publish` | publish a draft release and upload assets (`--asset "file=…"`, `zip=`, `tgz=`) |
| `relctl release gate` | check that the changelog may be released (see flow 3) |
| `relctl release forward` | bring the production branch back into the development branch (fast-forward or pull request) |
| `relctl release changed --component NAME` | tell whether the files of a component changed since its latest release, so a pipeline builds only what changed (`--component` also works with `release create`, `publish`, `gate` and `pr info`) |
| `relctl pr info` | version a pull request would produce |
| `relctl parse json\|yaml\|version` | read values from JSON, YAML or version strings |
| `relctl transform group-by` | group a JSON list by a key |
| `relctl connect github` | store server and repository in `.relctl-connection.yaml` for pipelines that cannot be detected (GitLab CI with a GitHub repository, local shell). The file holds no secret; the token is read from `GITHUB_TOKEN` on every call and never stored |

Variables are exported to the pipeline (GitHub Actions: `$GITHUB_ENV` and step outputs):
`RELCTL_VERSION` (what was created; `X.Y.Z-rc.N` for a candidate), `RELCTL_TAG` (with the tag prefix),
`RELCTL_NEXT_VERSION` (the release), `RELCTL_LATEST_VERSION` (never a candidate), `RELCTL_RELEASE_ID` (empty for a
release that is already published), `RELCTL_PRERELEASE`, `RELCTL_COMPONENT`, `RELCTL_CHANGED`, `RELCTL_PR`, …
`RELCTL_SILENT=true` switches the help comment on pull requests off.

## Version schemes

- **semver** (default): the bump comes from the branch prefix of the merged pull request (table above).
- **conventional-commits**: the bump comes from the commit messages since the last published release.
  `feat!:` or a `BREAKING CHANGE:` footer is major, `feat` is minor, `fix` and other types are patch.
  Squash-merged history only; the repository needs full history (`fetch-depth: 0`).
- **calver**: `YYYYMM.D.N` (for example `202611.2.1`): year and month, day of the month, and the number of the
  release that day. Calculated from local git tags, needs `fetch-depth: 0`.
- **changelog**: the topmost `## X.Y.Z — YYYY-MM-DD` heading of the changelog. Strict format: no `v`, no
  suffix, a real date. An undated heading is not released, and the release is idempotent.

## Installation

- **GitHub Actions** (recommended): [`layer87-labs/relctl-action`](https://github.com/layer87-labs/relctl-action)
  installs a verified binary.
- **Manually**: download `relctl_<version>_<os>-<arch>` and `sha256sum.txt` from the
  [latest release](https://github.com/layer87-labs/relctl/releases/latest), verify the checksum and add the
  binary to your `PATH`.

## Documentation

Full documentation, including a migration guide from awesome-ci and a runbook an AI agent can follow to migrate
a repository: **https://labs.layer87.de/docs/relctl/intro**

## Getting help

Please file an issue in this repository's [issue tracker](https://github.com/layer87-labs/relctl/issues).

## Community

- [Contributing](https://github.com/layer87-labs/.github/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://github.com/layer87-labs/.github/blob/main/CODE_OF_CONDUCT.md)
- [Security Policy](https://github.com/layer87-labs/.github/blob/main/SECURITY.md)

## License

relctl is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).

## Credits

- [SemVer](https://semver.org/)
- [CalVer](https://calver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Cobra CLI](https://github.com/spf13/cobra)
- [go-git](https://github.com/go-git/go-git)
