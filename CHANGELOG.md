# Changelog

Releases are made from the topmost `## X.Y.Z` heading (see [AGENTS.md](AGENTS.md)). Earlier releases are
listed on the [releases page](https://github.com/layer87-labs/relctl/releases).

## 2.10.0 — 2026-11-02

- `relctl connect github` stores no token any more: `.relctl-connection.yaml` holds server and repository only,
  the token is read from `GITHUB_TOKEN` on every call. A legacy `.awesomerc.yaml` is still read and replaced on
  the next connect.
- The export file for local shells is now `relctl.env` (was `awesomeci.env`).
- `relctl release forward` and the branch settings `production_branch` and `development_branch` in `.relctl.yaml`.
- `relctl release create --prerelease` (release candidates `X.Y.Z-rc.N`) and `relctl release gate`.
- Version scheme `changelog`: the version is the topmost heading of `CHANGELOG.md`.
- A major bump for `feat!/…` style branch names, and pull request help comments are posted again.
