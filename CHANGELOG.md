# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `--copilot-plays` autopilot mode: Copilot drives the entire run end-to-end
  (title → username → all five BUILD floors) using a deterministic BFS
  pathfinder. Pairs with `--seed` for reproducible demo recordings.
- `--as <handle>` flag: pre-fill the GitHub handle on the username screen so
  autopilot runs can be fully unattended.
- `--output PATH` and `--format svg,gif` flags for `--user` mode: pick the
  output filename and which artifacts to render.
- `--pr` flag: prints the exact `gh` CLI commands you'd run to open a pull
  request adding the Build-themed contribution graph to your profile README.
- `--version` flag and a `commit-crawl completion {bash,zsh,fish,powershell}`
  helper that emits the rune-by-rune sequence to remap autopilot keys.
- Custom `--help` output with examples and a keybinding table.
- `AGENTS.md` and `.github/copilot-instructions.md` documenting how AI agents
  should contribute to this repo.
- `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, issue and PR
  templates.
- `.github/workflows/ci.yml`: `go test -race`, `go vet`, and `gofmt -l` on
  Ubuntu / macOS / Windows for every push and PR.

### Changed
- README rewritten: light/dark hero art via `<picture>`, "Made with GitHub
  Copilot" badge and section, feature list, keybinding table, easter-egg
  reference, multi-channel install instructions, and a star-history image.

### Removed
- `scripts/make_svg.py` — the SVG/GIF renderer is now pure Go and the
  helper script was no longer used.

## [v0.1.0] - prior releases

See the [Releases page](https://github.com/leereilly/gh-commit-crawl/releases)
for tagged history before this changelog was introduced.
