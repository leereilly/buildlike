# Commit Crawl

A tiny terminal roguelike themed on Microsoft Build. Squash bugs (`b`), grab green commits (`+`), and climb five letter-shaped dungeons that spell **B-U-I-L-D**.

<img src="gameplay.gif" width="100%" alt="Commit Crawl gameplay">

Survive to the end and `commit-crawl` will render a Build-themed contribution graph for your GitHub handle, but you have to earn it. Throw it on your README profile, your LinkedIn header, or print it and stick it on your (work) fridge.

You'll get a GIF

<img src="contribution-graph.gif" width="100%" alt="Build-themed GitHub contribution graph GIF">

And an SVG

<img src="contribution-graph.svg" width="100%" alt="Build-themed GitHub contribution graph SVG">

## Install

`commit-crawl` is a [GitHub CLI](https://cli.github.com) extension, so you'll need `gh` first:

- **macOS:** `brew install gh`
- **Windows:** `winget install --id GitHub.cli`
- **Linux:** see the [official install instructions](https://github.com/cli/cli#installation)

Then install the extension:

```sh
gh extension install leereilly/gh-commit-crawl
```

Upgrade later with `gh extension upgrade commit-crawl`.

## Build from source

Requires [Go 1.26+](https://go.dev/dl/).

```sh
go build -o gh-commit-crawl ./cmd/commit-crawl
./gh-commit-crawl
```


## License

[MIT](LICENSE)

