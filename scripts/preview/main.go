// Command preview renders every contribution-graph artifact embedded in
// the README. It produces:
//
//	contribution-graph-light.svg
//	contribution-graph-light.gif
//	contribution-graph-dark.svg
//	contribution-graph-dark.gif
//	build-2026-intro-light.gif
//	build-2026-intro-dark.gif
//
// All six files are written to --out (default "."). The handle whose
// contribution data drives the year-graph variants is fetched from the
// public .contribs endpoint exactly once and reused across themes, so the
// per-cell palette pick stays identical between the light and dark
// variants for a given handle.
//
// The BUILD 2026 wordmark variants come from contribgraph.WordmarkBuild2026Data
// — they don't touch the network and are byte-stable across machines.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/leereilly/commit-crawl/internal/contribgraph"
)

func main() {
	handle := flag.String("handle", "leereilly", "GitHub handle whose contribution data fills the year-graph variants")
	outDir := flag.String("out", ".", "directory to write the rendered README artifacts into")
	wordmarkOnly := flag.Bool("wordmark-only", false, "skip the year-graph variants and only render the BUILD 2026 wordmark GIFs")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fail(err)
	}

	var yearData *contribgraph.Data
	if !*wordmarkOnly {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		data, err := contribgraph.Fetch(ctx, nil, *handle)
		if err != nil {
			fail(fmt.Errorf("fetch %s: %w", *handle, err))
		}
		yearData = data
		base := filepath.Join(*outDir, "contribution-graph.svg")
		if err := contribgraph.GenerateReadmeAssetsFromData(data, *handle, base); err != nil {
			fail(err)
		}
		for _, theme := range []contribgraph.Theme{contribgraph.ThemeLight, contribgraph.ThemeDark} {
			fmt.Printf("wrote %s\n", filepath.Join(*outDir, "contribution-graph-"+theme.Name+".svg"))
			fmt.Printf("wrote %s\n", filepath.Join(*outDir, "contribution-graph-"+theme.Name+".gif"))
			fmt.Printf("wrote %s\n", filepath.Join(*outDir, contribgraph.IntroFileName+"-"+theme.Name+".gif"))
		}
		return
	}

	for _, theme := range []contribgraph.Theme{contribgraph.ThemeLight, contribgraph.ThemeDark} {
		gifPath := filepath.Join(*outDir, contribgraph.IntroFileName+"-"+theme.Name+".gif")
		// The intro GIF opens on the BUILD 2026 wordmark and crossfades
		// into the user's animated contribution graph. When --wordmark-only
		// is set or the year-graph fetch was skipped, fall back to the
		// wordmark itself so the loop still plays cleanly — just without
		// the contribution-graph mid-phase.
		source := yearData
		if source == nil {
			source = contribgraph.WordmarkBuild2026Data()
		}
		gifBytes, err := contribgraph.RenderIntroGIF(source, *handle, nil, theme, contribgraph.IntroGIFScale)
		if err != nil {
			fail(err)
		}
		if err := os.WriteFile(gifPath, gifBytes, 0o644); err != nil {
			fail(err)
		}
		fmt.Printf("wrote %s\n", gifPath)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "preview: %v\n", err)
	os.Exit(1)
}
