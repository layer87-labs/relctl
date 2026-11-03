<p align="center">
  <img alt="relctl: release control for CI/CD" src="assets/banner.png">
</p>

<p align="center">
  <a href="https://github.com/layer87-labs/relctl/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/layer87-labs/relctl?display_name=tag&sort=semver"></a>
  <a href="https://github.com/layer87-labs/relctl/actions/workflows/release.yaml?query=branch%3Adev"><img alt="Build (dev)" src="https://img.shields.io/github/actions/workflow/status/layer87-labs/relctl/release.yaml?branch=dev&label=build"></a>
  <a href="go.mod"><img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/layer87-labs/relctl"></a>
  <a href="https://goreportcard.com/report/github.com/layer87-labs/relctl"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/layer87-labs/relctl"></a>
  <a href="LICENSE"><img alt="License: Apache-2.0" src="https://img.shields.io/badge/license-Apache--2.0-blue.svg"></a>
</p>

# relctl

**relctl** is a release management CLI for CI/CD pipelines. It works out the next version of your project,
creates the GitHub release and keeps your pipeline free of release boilerplate. It is the successor of
[awesome-ci](https://github.com/fullstack-devops/awesome-ci); see the
[migration guide](https://labs.layer87.de/docs/relctl/migrating-from-awesome-ci).

- **Source control**: GitHub and GitHub Enterprise (GitLab as source control is not supported;
  [open an issue](https://github.com/layer87-labs/relctl/issues) if you need it).
- **CI system**: GitHub Actions and Jenkins are detected automatically. Any other pipeline (including GitLab CI)
  connects once with `relctl connect github`.
- **Four ways to get the version**: branch prefix, Conventional Commits, date, or a reviewed `CHANGELOG.md`.

## Quick start

**1. Install.** In GitHub Actions use [`relctl-action`](https://github.com/layer87-labs/relctl-action), which
installs a verified binary (see the example below). Elsewhere, download the release asset and check it against
`sha256sum.txt` (Linux amd64 shown; asset names are `relctl_<version>_<os>-<arch>`):

```bash
VERSION=2.11.0   # see https://github.com/layer87-labs/relctl/releases/latest
BASE=https://github.com/layer87-labs/relctl/releases/download/$VERSION
curl -fsSLO "$BASE/relctl_${VERSION}_linux-amd64"
curl -fsSLO "$BASE/sha256sum.txt"
grep " relctl_${VERSION}_linux-amd64\$" sha256sum.txt | sha256sum -c -
install -m 0755 "relctl_${VERSION}_linux-amd64" /usr/local/bin/relctl
```

**2. Run the three commands that matter.** All of them read the token from `GITHUB_TOKEN`:

```bash
relctl pr info                                                  # on a pull request: the version it would release
relctl release create --merge-sha "$GITHUB_SHA"                 # on main: compute the version, create a draft release
relctl release publish --release-id "$RELCTL_RELEASE_ID" \
  --asset "file=dist/app"                                       # upload assets and publish
```

`release create` exports `RELCTL_VERSION` and `RELCTL_RELEASE_ID` for the following steps. Add `--dry-run` to
preview. In GitHub Actions:

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

Without a config file relctl uses SemVer from the branch prefix of the merged pull request:

| Branch prefix | Bump | Example |
| --- | --- | --- |
| `fix/`, `bugfix/`, `patch/`, `dependabot/` | patch | 1.2.3 → 1.2.4 |
| `feat/`, `feature/`, `minor/` | minor | 1.2.3 → 1.3.0 |
| `major/`, or any prefix ending in `!` (`feat!/`) | major | 1.2.3 → 2.0.0 |

To override it, comment on the pull request before merging (the comment must start with the command):
`relctl_patch_level: major` forces a bump level, `relctl_version_override: 2.1.0` pins the version.

## Reviewed releases (flow 3)

Work lands on `dev`; the version is written down in `CHANGELOG.md` and visible in review:

1. Every push to `dev` runs `relctl release create --prerelease` and publishes a candidate `X.Y.Z-rc.N`.
2. To release, open a pull request `dev` → `main` that dates the topmost heading (`## 1.4.0 — 2026-10-02`).
3. `relctl release gate --sha "$HEAD_SHA"` is the required check: the heading is dated and has notes, the version
   is not released yet, and a candidate exists for exactly this commit.
4. Merge with a **merge commit**, never a squash. The push to `main` runs `relctl release create` and publishes `X.Y.Z`.
5. `relctl release forward` brings `main` back into `dev`.

Details: [release flows](https://labs.layer87.de/docs/relctl/release-flows). Flows 1 and 2 release on every merge
to `main`:

| | Flow 1 | Flow 2 | Flow 3 |
| --- | --- | --- | --- |
| Release | every merge to `main` | every merge to `main` | merging a release pull request |
| Version from | branch prefix | commit messages or date | `CHANGELOG.md` heading |
| `version_scheme` | `semver` (default) | `conventional-commits`, `calver` | `changelog` |
| Good for | libraries, CLIs | services, apps | products with deliberate releases |

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

## Documentation

- [Documentation](https://labs.layer87.de/docs/relctl/intro): CLI reference, version schemes, release flows
- [Migrating from awesome-ci](https://labs.layer87.de/docs/relctl/migrating-from-awesome-ci), including a runbook an AI agent can follow
- [relctl-action](https://github.com/layer87-labs/relctl-action): setup action and reusable workflows

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
