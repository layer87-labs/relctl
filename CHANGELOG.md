# Changelog

Releases are made from the topmost `## X.Y.Z` heading (see [AGENTS.md](AGENTS.md)). Earlier releases are
listed on the [releases page](https://github.com/layer87-labs/relctl/releases).

## 2.12.1 — 2026-11-13

- `relctl release forward` recreates the development branch at the head of the production branch when it does not
  exist. This happens when the repository deletes head branches on merge and the release pull request was
  `dev` into `main`; before, the command failed with `404 Not Found` while comparing the branches. The branch is only recreated
  when it is the `development_branch` of `.relctl.yaml`, never for a name given with `--to`. Other errors still fail
  the command. The result is reported as `RELCTL_FORWARD_RESULT=recreated` (also with `--dry-run`, which creates
  nothing and prints `Would recreate`).

## 2.12.0 — 2026-11-13

- SemVer scheme: breaking-change detection. relctl reads the pull request title, its description and its own commits:
  `fix!:` / `fix(scope)!:` in the title or a commit subject, or a `BREAKING CHANGE: ` / `BREAKING-CHANGE: ` footer at
  the start of a line (not in a code block, an HTML comment or a `<details>` block; the description of `renovate/`
  and `dependabot/` pull requests is not read). Before, a pull request from `fix/...` titled `fix(forms)!: ...` was
  released as a patch without notice. The new setting `breaking_markers` in `.relctl.yaml` decides what happens when
  the branch prefix gives a patch or minor bump: `warn` (default) keeps the bump and warns in the log, the GitHub
  Actions run summary and on top of the release notes; `major` raises the bump to major; `off` ignores the markers.
  **The default becomes `major` in 3.0.0.** Set `breaking_markers: major` to get it now.
- `relctl pr info` prints where the breaking-change marker was found.
- Built with Go 1.26.11.

## 2.11.0 — 2026-11-03

- `relctl pr info` knows the version scheme (`--version-scheme`, `--config`, or `.relctl.yaml`). For `calver`,
  `conventional-commits` and `changelog` it prints the version the pull request would release instead of a SemVer
  version derived from the branch prefix. It also exports `RELCTL_VERSION_SCHEME`.
- The help comment on pull requests matches the scheme: only SemVer offers `relctl_patch_level` and
  `relctl_version_override`; the other schemes say what decides their version.
- The version variables have one meaning each. `relctl release create` exports `RELCTL_VERSION` (what was created,
  `X.Y.Z` or `X.Y.Z-rc.N`), `RELCTL_NEXT_VERSION` (the release, so `X.Y.Z` also for a candidate),
  `RELCTL_LATEST_VERSION` (the latest published release, never a candidate) and `RELCTL_PRERELEASE`. Before,
  `RELCTL_NEXT_VERSION` of a candidate was the candidate.
- `relctl pr info` no longer fails on a pull request whose merge commit GitHub has not computed yet.
- `tag_prefix` in `.relctl.yaml` puts a prefix in front of every tag and release (`v` gives `v1.2.3`); the version
  stays plain. Before, a repository that tags `v1.2.3` never had a latest release, because the tags were not found.
- Components: parts of one repository that are released independently (`components` in `.relctl.yaml`, `--component`
  on `release create`, `publish`, `gate` and `pr info`). A component has its own tag prefix (default `<name>/`), its
  own latest release and its own CalVer counter, so `whisper/202611.3.1` and `router-v202611.3.1` do not interfere.
  Components are released with the `calver` or `changelog` scheme.
- `relctl release changed --component NAME` tells whether a file that belongs to the component (glob `paths`) changed
  since its latest release, so a pipeline builds only what changed. It exports `RELCTL_CHANGED`,
  `RELCTL_CHANGED_FILES`, `RELCTL_COMPONENT` and `RELCTL_LATEST_VERSION`, and needs no access to GitHub.
- `relctl release create` also exports `RELCTL_TAG` (the version with the tag prefix) and `RELCTL_COMPONENT`.
- `relctl docs --out DIR` writes the command line reference as Markdown pages for Docusaurus (hidden command; the
  output is deterministic). The documentation site regenerates its reference from it.

## 2.10.2 — 2026-11-02

- `relctl release create` no longer hands out a release id for a release that is already published. A pipeline
  that ran again for a released version built its binaries and uploaded them into the published release; now it
  stops after `release create`.
- The download links in the asset table of a release text point to the final addresses; before, they pointed to a
  temporary address that stopped working once the release was published.

## 2.10.1 — 2026-11-02

- Release notes: releases created by the `semver`, `conventional-commits` and `calver` schemes now carry the notes
  GitHub generates (merged pull requests since the last published release, grouped by `.github/release.yml`).
  Before, only the asset list was written.
- `--body` accepts a text again; before, any value that was not an existing file failed.
- The uploaded assets are listed as a table with size and SHA-256.

## 2.10.0 — 2026-11-02

- `relctl connect github` stores no token any more: `.relctl-connection.yaml` holds server and repository only,
  the token is read from `GITHUB_TOKEN` on every call. A legacy `.awesomerc.yaml` is still read and replaced on
  the next connect.
- The export file for local shells is now `relctl.env` (was `awesomeci.env`).
- `relctl release forward` and the branch settings `production_branch` and `development_branch` in `.relctl.yaml`.
- `relctl release create --prerelease` (release candidates `X.Y.Z-rc.N`) and `relctl release gate`.
- Version scheme `changelog`: the version is the topmost heading of `CHANGELOG.md`.
- A major bump for `feat!/…` style branch names, and pull request help comments are posted again.
