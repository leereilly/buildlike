# Buildlike

A tiny terminal roguelike themed on Microsoft Build. Squash bugs (`b`), grab green commits (`+`), and climb five letter-shaped dungeons that spell **B-U-I-L-D**.

Survive to the end and `buildlike` will render a Build-themed contribution graph for your GitHub handle, but you have to earn it. Throw it on your README profile, your LinkedIn header, or print it and stink it on your (work) fridge.

![contribution graph](contribution-graph.svg)

## Build and play

Requires [Go 1.26+](https://go.dev/dl/).

```sh
go run ./cmd/buildlike
```

Or build a binary:

```sh
go build -o buildlike ./cmd/buildlike
./buildlike
```


## License

[MIT](LICENSE)
