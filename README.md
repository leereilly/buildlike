# Commit Crawl

<p align="center">
  <a href="https://github.com/leereilly/gh-commit-crawl/actions/workflows/ci.yml">
    <img alt="CI" src="https://github.com/leereilly/gh-commit-crawl/actions/workflows/ci.yml/badge.svg">
  </a>
  <a href="https://github.com/leereilly/gh-commit-crawl/releases">
    <img alt="Latest release" src="https://img.shields.io/github/v/release/leereilly/gh-commit-crawl?display_name=tag&sort=semver">
  </a>
  <a href="LICENSE">
    <img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-blue.svg">
  </a>
  <a href="https://github.com/features/copilot">
    <img alt="Made with GitHub Copilot" src="https://img.shields.io/badge/made%20with-GitHub%20Copilot-8957e5?logo=github">
  </a>
</p>

<img src="docs/gameplay.gif" width="100%" alt="Commit Crawl gameplay">


A tiny terminal roguelike themed on Microsoft Build. Squash bugs (`b`), grab green commits (`+`), and climb five letter-shaped dungeons that spell **B-U-I-L-D**. Survive to the end and you'll get your custom Build-themed contribution graph for your GitHub handle  Throw it on on your README profiles, your LinkedIn cover image, or print it and stick it on your (work) fridge.

You'll get a GIF that opens on the `BUILD 2026` wordmark and morphs seamlessly into your contribution graph

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/intro-dark.gif">
  <source media="(prefers-color-scheme: light)" srcset="docs/intro-light.gif">
  <img src="docs/intro-light.gif" alt="Build-themed GitHub contribution graph GIF with BUILD 2026 intro">
</picture>

And a GIF _without_ the `BUILD 2026` intro

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/graph-dark.gif">
  <source media="(prefers-color-scheme: light)" srcset="docs/graph-light.gif">
  <img src="docs/graph-light.gif" width="100%" alt="Build-themed GitHub contribution graph GIF">
</picture>

And an SVG

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/graph-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="docs/graph-light.svg">
  <img src="docs/graph-light.svg" width="100%" alt="Build-themed GitHub contribution graph SVG">
</picture>

Every artifact ships in both light and dark variants.

## Install

`commit-crawl` is a [GitHub CLI](https://cli.github.com) extension, so
you'll need `gh` first:

- **macOS:** `brew install gh`
- **Windows:** `winget install --id GitHub.cli`
- **Linux:** see the [official install instructions](https://github.com/cli/cli#installation)

Then install the extension:

```sh
gh extension install leereilly/gh-commit-crawl
```

Upgrade later with `gh extension upgrade commit-crawl`.

### Other ways to install

```sh
# Direct Go install (no gh required; ships under the cmd binary name)
go install github.com/leereilly/commit-crawl/cmd/commit-crawl@latest
```

## Quick start

```sh
gh commit-crawl                        # play the game
gh commit-crawl -seed 42               # deterministic run
gh commit-crawl -no-color

# Render @octocat's Build-themed contribution graph as a transparent
# SVG + GIF (level-0 cells are skipped) into the current directory:
gh commit-crawl --user octocat

# Or render light + dark variants in one go (writes -light.{svg,gif} and
# -dark.{svg,gif} pairs whose empty cells match the GitHub theme colours):
gh commit-crawl --user octocat --theme both
```

## Keybindings

Play with arrows, vi-keys (hjkl), WASD, and the numpad all work simultaneously.


| Action          | Keys                                          |
| --------------- | --------------------------------------------- |
| Move N/S/W/E    | `↑` `↓` `←` `→` · `k` `j` `h` `l` · `w` `s` `a` `d` · numpad `8` `2` `4` `6` |
| Move diagonals  | `y` `u` `b` `n` · numpad `7` `9` `1` `3`      |
| Wait one turn   | `.` · `space` · numpad `5`                    |
| Ascend stairs   | `>` (when standing on `>`)                    |
| Help            | `?`                                           |
| Quit            | `q` · `Esc` · `Ctrl-C`                        |
| Restart         | `r` (on death / win screen) · `Enter`         |

## Build from source

Requires [Go 1.26+](https://go.dev/dl/).

```sh
git clone https://github.com/leereilly/gh-commit-crawl
cd gh-commit-crawl
go build -o gh-commit-crawl ./cmd/commit-crawl
./gh-commit-crawl
```

Tests, vet, and format:

```sh
go test ./...
go vet ./...
gofmt -l .
```

CI ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)) runs the same checks on macOS, Linux, and Windows for every push and PR.

## Made with GitHub Copilot

This whole repo (game loop, BSP dungeon generator, animated GIF renderer, end-of-game shell typewriter, the lot) was built end-to-end
with [GitHub Copilot](https://github.com/features/copilot).

If you're an AI agent looking at this repo, start with
[`AGENTS.md`](AGENTS.md) and
[`.github/copilot-instructions.md`](.github/copilot-instructions.md) - they spell out the conventions, package boundaries, and "please don't break these easter eggs" list in one place.

## Contributing

PRs welcome — see [`CONTRIBUTING.md`](CONTRIBUTING.md) for the ground
rules. Found a bug? File one with the [bug template](.github/ISSUE_TEMPLATE/bug.yml). Found a security issue? Please follow [`SECURITY.md`](SECURITY.md) instead of opening a public
issue.


## License

[MIT](LICENSE)
