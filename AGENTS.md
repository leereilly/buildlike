# AGENTS.md

This file is a one-stop briefing for AI coding agents (GitHub Copilot, Claude,
Cursor, Aider, etc.) working on `gh-commit-crawl`. Humans should read
[`CONTRIBUTING.md`](CONTRIBUTING.md); the conventions below apply equally to
both, but agents tend to need the explicit form.

## What this repo is

A `gh` CLI extension written in Go. The headline command is
[`gh commit-crawl`](README.md) — a tiny terminal roguelike themed on Microsoft
Build. Surviving the run renders a Build-themed GitHub contribution graph (SVG
+ animated GIF) for the player's handle.

Everything ships from one Go module (`github.com/leereilly/commit-crawl`).
There is no JavaScript, no Python, no C — and we intend to keep it that way.

## Layout

```
cmd/commit-crawl/       main package (CLI entrypoint, flag parsing, run loop)
internal/contribgraph/  GitHub .contribs JSON → SVG + animated GIF
internal/entity/        player, bug, jester, powerup model
internal/game/          game state machine, input, autopilot
internal/rng/           seeded RNG wrapper
internal/ui/            tcell renderers + screen-by-screen logic
internal/world/         BSP dungeon generation, letter masks, LOS, tiles
docs/                   SVG hero art used in the README
scripts/                helper scripts (recordings, demos)
.github/                workflows, issue/PR templates, Copilot instructions
```

## Build, test, lint

```sh
go build -o gh-commit-crawl ./cmd/commit-crawl   # produces the extension binary
go test ./...                                    # full suite (fast: < 5s)
go vet ./...
gofmt -l .                                       # must print nothing
```

Go 1.26+ (see `go.mod`). No external test deps — everything uses the standard
library plus `tcell`.

## Conventions

- **Pure Go, no shellouts in the hot path.** Rendering, generation, and game
  logic must never invoke external binaries. The one exception is the `--pr`
  flag, which is allowed to `exec.LookPath("gh")` because that's literally
  its job.
- **Package boundaries matter.** `internal/world` knows nothing about tcell.
  `internal/contribgraph` knows nothing about the game. `internal/game` is
  the only package allowed to import both `entity` and `world` heavily; UI
  packages import `game` read-only.
- **Tests live next to code.** `_test.go` files in the same package. Use
  table-driven tests where it improves clarity. Prefer deterministic seeds
  (`rng.New(0xC0DE)`) over `time.Now()` in tests.
- **Comments explain *why*, not *what*.** The style is: short godoc on
  exported symbols, paragraph-length comments above non-obvious blocks of
  logic (see `internal/game/game.go` for the house style).
- **No new dependencies without a reason.** If you're tempted to add one,
  open an issue first.
- **No secrets.** Ever. The `.contribs` endpoint is anonymous; if you need to
  authenticate something, take a PR-time review.

## Easter eggs (please don't break)

The game is built around discovery. Several easter eggs are load-bearing for
the demo:

- **Konami code** on the title screen → invincibility for the run.
- **Triple-tap `1`–`5`** on the title screen → warp straight to that BUILD
  floor (level skip).
- **Clippy** appears on floor 2 (`U`) and fades after a few keystrokes.
- **Jester** (`j`) spawns on a random BUILD floor each run.
- **Copilot mascot** blinks in the side panel on every keypress.
- **Rick roll** plays after you ascend the final stairs, before the win
  screen.

If your change touches `internal/game` or `internal/ui`, please add a quick
manual-test note in your PR confirming the relevant easter egg still works.

## Autopilot mode

`--copilot-plays` makes Copilot drive the run end-to-end (used for demos and
recordings). The pathfinder lives in `internal/game/autopilot.go` and is
intentionally simple — BFS, no learning, no models. If you improve it, keep it
deterministic given the same `--seed` so demo GIFs stay reproducible.

## Release

Tag `vX.Y.Z` on `main`. `.github/workflows/release.yml` runs
[`cli/gh-extension-precompile`](https://github.com/cli/gh-extension-precompile)
to build cross-platform binaries and publish a release `gh extension install`
can consume. Pre-release tags (`-rc.N`) are auto-flipped to non-prerelease so
the latest-release lookup still finds them.

## Copilot-specific notes

See [`.github/copilot-instructions.md`](.github/copilot-instructions.md) for
the prompt-level guidance GitHub Copilot picks up automatically. The two
files overlap on purpose — `AGENTS.md` is the human-readable canonical doc;
the `.github/copilot-instructions.md` file is the machine-readable subset.
