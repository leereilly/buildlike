// Package contribgraph fetches a user's GitHub contribution data from the
// public ".contribs" JSON endpoint and renders it as a standalone SVG.
//
// The rendered graph keeps GitHub's level-0..level-4 intensity semantics but
// swaps the green color ramp for a deterministically-seeded pick from a
// 20-color Microsoft-Build-inspired palette: the same date always gets the
// same base hue across renders, and intensity is conveyed by lightness
// rather than saturation alone.
//
// The package is intentionally self-contained — no external SVG / image /
// font / CSS dependencies — so the resulting file embeds cleanly in
// READMEs, GitHub Pages, blog posts, etc.
package contribgraph

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// EndpointFmt is the GitHub-served JSON endpoint that returns the schema-v2
// contribution payload for a given handle. The %s placeholder is replaced
// with the bare username (no leading '@'). It's a var rather than a const
// so tests can repoint it at a local httptest server without touching the
// network.
var EndpointFmt = "https://github.com/%s.contribs"

// DefaultOutputPath is the file name Generate writes to when the caller
// passes the empty string for outPath. It's deliberately relative — the
// caller decides which working directory the file lands in.
const DefaultOutputPath = "contribution-graph.svg"

// Day is a single cell inside a Week.
type Day struct {
	Weekday int `json:"weekday"`
	Count   int `json:"count"`
	Level   int `json:"level"`
}

// Week is one column of the rendered grid. FirstDay anchors the dates of
// every Day in ContributionDays (weekday 0 == FirstDay).
type Week struct {
	Index            int    `json:"index"`
	FirstDay         string `json:"first_day"`
	ContributionDays []Day  `json:"contribution_days"`
}

// Month is one entry of the month-label index along the top of the graph.
// TotalWeeks tells us how many calendar weeks this month spans in the
// rendered grid so we can position the next label.
type Month struct {
	Month      string `json:"month"`
	TotalWeeks int    `json:"total_weeks"`
}

// Data is the subset of the .contribs JSON we render. Fields we don't use
// (colors_full, private_contributions_included, etc.) are intentionally
// ignored so future schema additions don't break parsing.
type Data struct {
	Schema             string  `json:"schema"`
	GeneratedAt        string  `json:"generated_at"`
	From               string  `json:"from"`
	To                 string  `json:"to"`
	TotalContributions int     `json:"total_contributions"`
	Weeks              []Week  `json:"weeks"`
	Months             []Month `json:"months"`
}

// Palette is the 20-color Microsoft-Build-inspired wheel used in place of
// GitHub's green ramp. Each day deterministically picks one entry as its
// base hue (seeded by date string), and intensity is then expressed by
// tinting/shading that base — see cellColor.
var Palette = []string{
	"#2e98c6", "#307ac7", "#2f66c6", "#4c5299", "#8e407d",
	"#c42664", "#d1244a", "#eb2612", "#eb3915", "#ed5c1b",
	"#f07f21", "#f0b528", "#f2ba00", "#b1b629", "#9eb527",
	"#70b62d", "#6eb72d", "#52a970", "#46a28f", "#3f9ea3",
}

// userAgent identifies us to GitHub. The endpoint is unauthenticated but
// having a clear UA helps if behaviour ever needs to be debugged from
// access logs.
const userAgent = "commit-crawl-contribgraph/1 (+https://github.com/leereilly/commit-crawl)"

// Fetch downloads and decodes the .contribs JSON for username. A nil
// client falls back to a sensible default with a short timeout so callers
// can't accidentally hang the program.
func Fetch(ctx context.Context, client *http.Client, username string) (*Data, error) {
	if strings.TrimSpace(username) == "" {
		return nil, fmt.Errorf("contribgraph: empty username")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	url := fmt.Sprintf(EndpointFmt, username)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("contribgraph: new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("contribgraph: GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("contribgraph: GET %s: %s", url, resp.Status)
	}
	var d Data
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, fmt.Errorf("contribgraph: decode: %w", err)
	}
	return &d, nil
}

// Render produces a standalone SVG document for username from data. When
// palette is nil/empty the package-level Palette is used. The returned
// bytes are a complete, self-contained `<?xml ... ?><svg>...</svg>` blob
// safe to write straight to disk or embed in an HTML page.
//
// The output intentionally contains no visible text (no month or weekday
// labels, no summary line, no per-cell tooltips), uses GitHub's exact
// contribution-grid geometry (10×10 cells with a 3 px gap, 7 rows per
// week), and renders against a transparent background using only the
// 20 colors in Palette — i.e. only the colors found in the Microsoft
// Build wordmark the palette was sampled from.
func Render(data *Data, username string, palette []string) []byte {
	return RenderWithTheme(data, username, palette, ThemeNone)
}

// RenderWithTheme is Render plus an explicit Theme. ThemeNone reproduces
// Render's transparent-background output byte-for-byte; ThemeLight /
// ThemeDark add a full grid of rounded empty-cell rects in the theme's
// EmptyCell colour so the graph reads cleanly on light or dark README
// backgrounds without relying on CSS.
func RenderWithTheme(data *Data, username string, palette []string, theme Theme) []byte {
	if data == nil {
		return nil
	}
	if len(palette) == 0 {
		palette = Palette
	}

	// Geometry matches the rendered github.com contribution calendar:
	// 10×10 cells with a 3 px gap between columns and between rows.
	const (
		cell   = 10
		gap    = 3
		stride = cell + gap
		radius = 2
	)

	weeks := len(data.Weeks)
	width := 0
	if weeks > 0 {
		width = weeks*stride - gap
	}
	height := 7*stride - gap

	var b strings.Builder

	label := fmt.Sprintf("@%s's Build-themed GitHub contribution graph", username)

	fmt.Fprintln(&b, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprintf(&b,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" role="img" aria-label="%s">`+"\n",
		width, height, width, height, xmlEscape(label))

	// Day cells. Level-0 cells are either skipped (transparent theme)
	// or rendered in the theme's empty-cell colour so the grid
	// silhouette is visible on flat backgrounds.
	fmt.Fprintln(&b, `  <g shape-rendering="geometricPrecision">`)
	emptyFill := theme.EmptyCellHex()
	for wi, w := range data.Weeks {
		firstDay, _ := time.Parse("2006-01-02", w.FirstDay)
		for _, d := range w.ContributionDays {
			x := wi * stride
			y := d.Weekday * stride
			if d.Level <= 0 {
				if theme.Transparent() {
					continue
				}
				fmt.Fprintf(&b,
					"    <rect x=\"%d\" y=\"%d\" width=\"%d\" height=\"%d\" rx=\"%d\" ry=\"%d\" fill=\"%s\"/>\n",
					x, y, cell, cell, radius, radius, emptyFill)
				continue
			}
			dateStr := ""
			if !firstDay.IsZero() {
				dateStr = firstDay.AddDate(0, 0, d.Weekday).Format("2006-01-02")
			}
			fmt.Fprintf(&b,
				"    <rect x=\"%d\" y=\"%d\" width=\"%d\" height=\"%d\" rx=\"%d\" ry=\"%d\" fill=\"%s\"/>\n",
				x, y, cell, cell, radius, radius, cellColor(palette, dateStr, d.Level))
		}
	}
	fmt.Fprintln(&b, `  </g>`)

	fmt.Fprintln(&b, `</svg>`)
	return []byte(b.String())
}

// cellColor returns the SVG fill string for a single day cell. Level-0
// cells return the empty string — the renderer treats that as "skip" so
// the transparent background shows through. Levels 1..4 deterministically
// pick one of the in-palette BUILD colors seeded by both the date and the
// level, so the rendered graph only ever contains colors that already
// appear in the source artwork (no tinting or shading to in-between
// values).
func cellColor(palette []string, date string, level int) string {
	if level <= 0 {
		return ""
	}
	if len(palette) == 0 {
		palette = Palette
	}
	return palette[seedIndex(date+":"+strconv.Itoa(level), len(palette))]
}

// seedIndex hashes seed with FNV-1a and reduces it modulo n. We use FNV
// rather than crypto hashing because we just need a fast, stable,
// well-distributed integer; the same date string always produces the same
// index across runs and platforms.
func seedIndex(seed string, n int) int {
	if n <= 0 {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	return int(h.Sum32() % uint32(n))
}

// hexToRGB parses a `#rrggbb` hex string into its three channel values.
// Retained as a small public-ish helper for future tooling (e.g. picking
// the best foreground color over a generated cell); it's intentionally
// not exported because the surface area of this package is otherwise
// frozen around Render/Fetch/Generate.
func hexToRGB(s string) (int, int, int) {
	if len(s) < 7 || s[0] != '#' {
		return 0, 0, 0
	}
	r, _ := strconv.ParseUint(s[1:3], 16, 8)
	g, _ := strconv.ParseUint(s[3:5], 16, 8)
	b, _ := strconv.ParseUint(s[5:7], 16, 8)
	return int(r), int(g), int(b)
}

// xmlEscape is the safe form of stuffing arbitrary user-supplied text
// (usernames!) into <text>, <title>, and <desc> bodies.
func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// Generate is the one-shot helper: fetch the JSON for username, render the
// SVG, and write it to outPath (or DefaultOutputPath when empty). It
// returns the rendered bytes alongside any error so callers can also
// surface or embed the SVG without re-reading the file.
//
// Generate writes the transparent (no-theme) variant — use GenerateThemed
// to also emit light/dark companions.
func Generate(ctx context.Context, client *http.Client, username, outPath string) ([]byte, error) {
	data, err := Fetch(ctx, client, username)
	if err != nil {
		return nil, err
	}
	return GenerateFromData(data, username, outPath)
}

// GenerateThemed is Generate plus a Theme: it fetches once and renders an
// SVG + GIF pair into outPath / its .gif sibling using the supplied theme.
// Useful for producing per-theme README assets in a single network round
// trip.
func GenerateThemed(ctx context.Context, client *http.Client, username, outPath string, theme Theme) ([]byte, error) {
	data, err := Fetch(ctx, client, username)
	if err != nil {
		return nil, err
	}
	return GenerateFromDataThemed(data, username, outPath, theme)
}

// GenerateFromData renders an already-fetched Data payload for username
// and writes BOTH the Build-themed SVG and a matching animated GIF. The
// SVG is written to outPath (or DefaultOutputPath when empty) and the
// GIF is written to the same path with the extension swapped from .svg
// to .gif (any other extension is preserved and .gif is appended).
// Useful in tests where we want to avoid the network entirely.
//
// Returns the SVG bytes alongside any error. If the SVG write succeeds
// but the GIF render or write fails, the SVG is still left on disk and
// the GIF-related error is returned.
//
// Empty cells render as transparent — call GenerateFromDataThemed to
// pick a light / dark grid background instead.
func GenerateFromData(data *Data, username, outPath string) ([]byte, error) {
	return GenerateFromDataThemed(data, username, outPath, ThemeNone)
}

// GenerateFromDataThemed is GenerateFromData with an explicit Theme. It
// writes a themed SVG to outPath and a themed GIF to the matching .gif
// path; both have empty cells in theme.EmptyCell (or transparent when
// theme.Transparent()).
func GenerateFromDataThemed(data *Data, username, outPath string, theme Theme) ([]byte, error) {
	if outPath == "" {
		outPath = DefaultOutputPath
	}
	svg := RenderWithTheme(data, username, nil, theme)
	if err := os.WriteFile(outPath, svg, 0o644); err != nil {
		return nil, fmt.Errorf("contribgraph: write %s: %w", outPath, err)
	}
	gifBytes, err := RenderGIFWithTheme(data, username, nil, theme, 0)
	if err != nil {
		return svg, err
	}
	gifPath := gifPathFor(outPath)
	if err := os.WriteFile(gifPath, gifBytes, 0o644); err != nil {
		return svg, fmt.Errorf("contribgraph: write %s: %w", gifPath, err)
	}
	return svg, nil
}

// gifPathFor derives the GIF output path that pairs with an SVG output
// path. A `.svg` (case-insensitive) suffix is swapped for `.gif`;
// anything else gets `.gif` appended.
func gifPathFor(svgPath string) string {
	if len(svgPath) >= 4 && strings.EqualFold(svgPath[len(svgPath)-4:], ".svg") {
		return svgPath[:len(svgPath)-4] + ".gif"
	}
	return svgPath + ".gif"
}

// IntroGIFScale is the GIF upscaling factor used for the BUILD 2026 intro
// loop in the README. Exposed so other callers (the dungeon-end goroutine,
// the preview script) can stay byte-identical to the committed artifacts.
const IntroGIFScale = 3

// IntroFileName is the basename (no extension) of the BUILD 2026 intro
// GIFs that ship with the README. The themed variants land next to the
// contribution-graph artifacts as `build-2026-intro-<theme>.gif`.
const IntroFileName = "build-2026-intro"

// GenerateReadmeAssets fetches `username`'s contribution data once and
// writes the full set of README-ready artifacts derived from `outPath`:
//
//   - <base>-light.svg, <base>-light.gif
//   - <base>-dark.svg,  <base>-dark.gif
//   - <dir>/build-2026-intro-light.gif
//   - <dir>/build-2026-intro-dark.gif
//
// where <base> is `outPath` with any `.svg`/`.gif` suffix stripped and
// <dir> is the directory containing `outPath`. An empty `outPath` falls
// back to DefaultOutputPath, matching Generate.
//
// This is the production entrypoint used by the dungeon-end goroutine
// and the preview script: callers get a complete, theme-aware artifact
// set from a single network round-trip.
func GenerateReadmeAssets(ctx context.Context, client *http.Client, username, outPath string) error {
	data, err := Fetch(ctx, client, username)
	if err != nil {
		return err
	}
	return GenerateReadmeAssetsFromData(data, username, outPath)
}

// GenerateReadmeAssetsFromData is GenerateReadmeAssets without the fetch
// step — useful for tests and for callers that already have a Data value
// (e.g. the preview script's `--wordmark-only` mode).
func GenerateReadmeAssetsFromData(data *Data, username, outPath string) error {
	if outPath == "" {
		outPath = DefaultOutputPath
	}
	base := stripGraphExt(outPath)
	dir := filepath.Dir(outPath)
	for _, theme := range []Theme{ThemeLight, ThemeDark} {
		svgPath := base + "-" + theme.Name + ".svg"
		gifPath := base + "-" + theme.Name + ".gif"
		introPath := filepath.Join(dir, IntroFileName+"-"+theme.Name+".gif")

		svg := RenderWithTheme(data, username, nil, theme)
		if err := os.WriteFile(svgPath, svg, 0o644); err != nil {
			return fmt.Errorf("contribgraph: write %s: %w", svgPath, err)
		}
		gifBytes, err := RenderGIFWithTheme(data, username, nil, theme, 0)
		if err != nil {
			return fmt.Errorf("contribgraph: render gif (%s): %w", theme.Name, err)
		}
		if err := os.WriteFile(gifPath, gifBytes, 0o644); err != nil {
			return fmt.Errorf("contribgraph: write %s: %w", gifPath, err)
		}
		introBytes, err := RenderIntroGIF(data, username, nil, theme, IntroGIFScale)
		if err != nil {
			return fmt.Errorf("contribgraph: render intro (%s): %w", theme.Name, err)
		}
		if err := os.WriteFile(introPath, introBytes, 0o644); err != nil {
			return fmt.Errorf("contribgraph: write %s: %w", introPath, err)
		}
	}
	return nil
}

// stripGraphExt removes a trailing `.svg` or `.gif` (case-insensitive)
// from p so the per-theme suffix can be appended cleanly. Anything else
// is returned untouched so non-standard paths like "/tmp/graph" still
// produce sensible "/tmp/graph-light.svg" etc.
func stripGraphExt(p string) string {
	if len(p) >= 4 {
		tail := strings.ToLower(p[len(p)-4:])
		if tail == ".svg" || tail == ".gif" {
			return p[:len(p)-4]
		}
	}
	return p
}
