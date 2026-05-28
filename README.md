<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/buildlike-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="docs/buildlike-light.svg">
  <img alt="buildlike ASCII animation" src="docs/buildlike-light.svg">
</picture>

## Build

Requires [Go 1.26+](https://go.dev/dl/). Clone the repo, then from the project root:

**macOS / Linux**

```sh
go build -o buildlike ./cmd/buildlike
./buildlike
```

**Windows (PowerShell)**

```powershell
go build -o buildlike.exe ./cmd/buildlike
.\buildlike.exe
```

**Any platform (no binary)**

```sh
go run ./cmd/buildlike
```

### Flags

- `-seed <int>` — RNG seed (0 = time-based)
- `-no-color` — disable colors for monochrome terminals
