# Contributing to `gh-commit-crawl`

Thanks for poking at the source! This is a small, opinionated project but
PRs are very welcome — especially bug fixes, accessibility improvements,
new dungeon-letter renderings, and Build-themed flavor text.

## TL;DR

```sh
git clone https://github.com/leereilly/gh-commit-crawl
cd gh-commit-crawl
go build -o gh-commit-crawl ./cmd/commit-crawl
./gh-commit-crawl
```

Before you push:

```sh
go test ./...
go vet ./...
gofmt -l .            # must print nothing
```

CI (`.github/workflows/ci.yml`) runs the same checks on macOS, Linux, and
Windows. Please make sure your branch is green before requesting review.

## Ground rules

- **Pure Go.** No JavaScript, no Python, no shelling out in the hot path.
  The one intentional shellout is `gh` for the `--pr` flag.
- **No new runtime dependencies** without a clear reason. We're at exactly
  one (`tcell`) and we'd like to stay close to that.
- **Easter eggs are load-bearing.** Don't accidentally remove the Konami
  code, the triple-tap warp, Clippy, the jester, the Copilot mascot, or the
  rick roll. If you intentionally change one, call it out in the PR.
- **Determinism.** Anything seeded by `--seed` must be reproducible across
  platforms — the demo GIFs depend on it.
- **Style.** Run `gofmt` and follow the existing comment style: short godoc
  on exported symbols, paragraph comments above non-obvious logic.

For deeper conventions (package boundaries, AI-agent guidance, release
process) see [`AGENTS.md`](AGENTS.md).

## Filing issues

- **Bug:** what did you run, what did you expect, what happened? Terminal
  emulator and OS version please.
- **Feature:** describe the use case first, the implementation second.
- **Security:** please follow [`SECURITY.md`](SECURITY.md) instead of opening
  a public issue.

## Code of Conduct

This project follows the
[Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).
See [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md).
