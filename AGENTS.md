# AGENTS.md

Instructions for AI coding agents (and a short orientation for humans) working in this repository.
`CLAUDE.md` points here.

relctl is a provider-agnostic release management CLI for CI/CD pipelines (Go). The CLI reference
and user documentation live in the separate `docs` repository; the GitHub Action and reusable
workflows live in `relctl-action`.

## Public repository — hard rules

This repository is public. Everything committed, and everything written to issues,
pull requests, comments, commit messages and release notes, is world-readable.

- Never include internal information from private projects of the owning organisation:
  no private repository names or links, no internal hostnames, URLs or addresses, no
  customer or tenant names, no ticket or issue references into private systems, no
  secrets, tokens or credentials, no internal process names or standards.
- Issues may be *motivated* by internal needs, but must be written generically:
  describe the problem and the proposal, not where it came from. Use neutral
  examples (`example-org/example-repo`).
- If in doubt, leave it out. If something internal slipped in, remove it from the
  text and tell a maintainer — edit history on GitHub may still show it.
- Write all text in English (code, comments, docs, commits, issues, PRs).
- Commits carry no AI attribution: no `Co-Authored-By` trailers for tools, no session IDs or
  session links, no tool names in commit messages, PR bodies or comments. Commit as the
  maintainer who requested the change.

## Development

- Go version is pinned in `go.mod`. Build: `go build ./...`. Tests: `go test ./internal/...`.
  Full quality gate (as in CI): `make audit` (vet, staticcheck, govulncheck, golangci-lint, race tests).
- Format with `gofmt`; keep `go mod tidy` clean.
- Layout: `cmd/relctl` (entrypoint), `internal/app/relctl` (commands, services, SCM portal),
  `internal/pkg` (reusable packages: `semver`, `versioning`, `conventionalcommits`, …).
- New logic must come with unit tests, written so they run without network access or a real
  GitHub repository (inject fakes, keep decision logic in pure functions).
- Do not mock-pass: never skip, delete or weaken a test to get green.

## Releases and versioning

- Pull requests are squash-merged to `main`; every merge to `main` creates a release.
- The SemVer bump comes from the PR branch prefix: `fix/`, `bugfix/`, `patch/`, `dependabot/` → patch;
  `feat/`, `feature/`, `minor/` → minor; `major/` or any prefix ending in `!` (e.g. `feat!/`) → major.
- PR titles and commits follow Conventional Commits (`feat:`, `fix:`, `feat!:` …).
- Behaviour changes need a matching update in the `docs` repository and, if workflow inputs or
  permissions change, in `relctl-action`.

## Workflow

- Develop on a descriptively named branch (`feat/…`, `fix/…`, `docs/…`, `chore/…`), never push to
  `main` directly. Do not use tool-generated branch names.
- Open pull requests only when asked.
