package contribgraph

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"math"
	"math/rand"
	"sort"
	"time"
)

// Knobs that shape the BUILD-2026 → contribution-graph intro animation.
// They sum to gifFrames (= 60) so the produced GIF stays the same length
// as the plain RenderGIF and the wall-clock loop is identical.
const (
	introHoldFrames  = 12 // wordmark held, no drift
	introMorphFrames = 12 // wordmark → graph crossfade
	introGraphFrames = 24 // contribution graph drift
	introOutroFrames = 12 // graph → wordmark crossfade (loop seam)
	introTotalFrames = introHoldFrames + introMorphFrames + introGraphFrames + introOutroFrames
)

// RenderIntroGIF produces a looping GIF that opens on the BUILD 2026
// wordmark, crossfades into the user's animated contribution graph
// (same drift + sparkle as RenderGIF), then crossfades back to the
// wordmark so the loop seam is invisible.
//
// `data` is the user's normal year-graph payload — pass anything you'd
// hand to RenderGIF. The wordmark is overlaid on top of the same 53×7
// lattice, so cells the wordmark touches but the user's data doesn't
// (and vice versa) still crossfade cleanly between empty grid and live
// colour. When `scale <= 0` the package-level gifScale is used.
//
// Theme behaviour matches RenderGIFWithTheme: ThemeNone leaves the empty
// grid transparent; ThemeLight / ThemeDark fill it with the matching
// GitHub colour.
func RenderIntroGIF(data *Data, username string, palette []string, theme Theme, scale int) ([]byte, error) {
	if data == nil {
		data = &Data{}
	}
	if len(palette) == 0 {
		palette = Palette
	}
	if scale <= 0 {
		scale = gifScale
	}

	wordmark := WordmarkBuild2026Data()

	const (
		cell   = 10
		gap    = 3
		stride = cell + gap
		radius = 2
	)
	weeks := len(wordmark.Weeks)
	if len(data.Weeks) > weeks {
		weeks = len(data.Weeks)
	}
	if weeks <= 0 {
		return emptyGIF()
	}
	width := weeks*stride - gap
	height := 7*stride - gap

	// Walk the union of (week, day) cells across both the wordmark and
	// the user's contribution data. For each cell we remember its
	// wordmark colour (level-4 palette pick, or theme.EmptyCell when
	// the wordmark doesn't touch this cell) and its graph base colour
	// (palette pick by date+level, or theme.EmptyCell when the user
	// didn't commit that day). Per-frame drift modulates the graph
	// colour later; the wordmark side stays flat.
	type cellState struct {
		x, y        int
		wmRGB       color.RGBA
		graphRGB    color.RGBA
		hasWordmark bool
		hasGraph    bool
	}
	all := make([]cellState, 0, weeks*7)
	var liveRGBs []color.RGBA
	for wi := 0; wi < weeks; wi++ {
		var wmWeek, gWeek *Week
		if wi < len(wordmark.Weeks) {
			wmWeek = &wordmark.Weeks[wi]
		}
		if wi < len(data.Weeks) {
			gWeek = &data.Weeks[wi]
		}
		for d := 0; d < 7; d++ {
			s := cellState{
				x:        wi * stride,
				y:        d * stride,
				wmRGB:    theme.EmptyCell,
				graphRGB: theme.EmptyCell,
			}
			if wmWeek != nil && d < len(wmWeek.ContributionDays) {
				wd := wmWeek.ContributionDays[d]
				if wd.Level > 0 {
					date := datePlus(wmWeek.FirstDay, d)
					r, g, b := hexToRGB(cellColor(palette, date, wordmarkLevel))
					s.wmRGB = color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xFF}
					s.hasWordmark = true
					liveRGBs = append(liveRGBs, s.wmRGB)
				}
			}
			if gWeek != nil && d < len(gWeek.ContributionDays) {
				gd := gWeek.ContributionDays[d]
				if gd.Level > 0 {
					date := datePlus(gWeek.FirstDay, d)
					r, g, b := hexToRGB(cellColor(palette, date, gd.Level))
					s.graphRGB = color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xFF}
					s.hasGraph = true
					liveRGBs = append(liveRGBs, s.graphRGB)
				}
			}
			all = append(all, s)
		}
	}

	// Cells that never light up in either phase are static through the
	// whole loop — peel them off so the per-frame inner loop only
	// touches genuinely changing cells.
	type pos struct{ x, y int }
	var dead []pos
	live := make([]cellState, 0, len(all))
	for _, s := range all {
		if !s.hasWordmark && !s.hasGraph {
			dead = append(dead, pos{x: s.x, y: s.y})
			continue
		}
		live = append(live, s)
	}

	sortedPalette := uniqueHSVSorted(liveRGBs)
	paletteIdx := make(map[color.RGBA]int, len(sortedPalette))
	for i, c := range sortedPalette {
		paletteIdx[c] = i
	}
	n := len(sortedPalette)

	// Per-cell drift / sparkle oscillator — identical maths to the
	// plain RenderGIF. Only graph cells get an oscillator; wordmark-
	// only cells stay flat in their phase. Seeded so the same handle
	// produces the same drift across runs and machines.
	rng := rand.New(rand.NewSource(gifSeed))
	driftKs := []int{1, 1, 1, 2, 2, 3}
	sparkleKs := []int{2, 3, 3, 4, 5, 5, 7}
	type osc struct {
		phase float64
		dk    int
		damp  float64
		doff  float64
		sk    int
		soff  float64
	}
	oscs := make([]osc, len(live))
	for i, s := range live {
		if !s.hasGraph {
			continue
		}
		o := osc{
			phase: float64(paletteIdx[s.graphRGB]),
			dk:    driftKs[rng.Intn(len(driftKs))],
			damp:  4.0 + rng.Float64()*4.0,
		}
		if rng.Intn(2) == 1 {
			o.doff = math.Pi
		}
		o.sk = sparkleKs[rng.Intn(len(sparkleKs))]
		if rng.Intn(2) == 1 {
			o.soff = math.Pi
		}
		oscs[i] = o
	}

	// graphColorAt returns the contribution-graph colour of live cell i
	// at drift step t (0..introGraphFrames-1). Cells with no graph
	// contribution return theme.EmptyCell so the crossfade lands on
	// the empty grid instead of jumping there.
	omega := 2 * math.Pi / float64(introGraphFrames)
	graphColorAt := func(i, t int) color.RGBA {
		s := live[i]
		if !s.hasGraph || n == 0 {
			return theme.EmptyCell
		}
		o := oscs[i]
		pos := o.phase + o.damp*math.Sin(float64(o.dk)*omega*float64(t)+o.doff)
		fl := math.Floor(pos)
		iBase := int(fl)
		frac := pos - fl
		a := sortedPalette[modPositive(iBase, n)]
		b := sortedPalette[modPositive(iBase+1, n)]
		base := lerpRGB(a, b, frac)
		sp := math.Max(0, math.Sin(float64(o.sk)*omega*float64(t)+o.soff))
		sp = math.Pow(sp, 6)
		base.R = clampU8(math.Round(lerpFloat(float64(base.R), 255, 0.55*sp)))
		base.G = clampU8(math.Round(lerpFloat(float64(base.G), 255, 0.55*sp)))
		base.B = clampU8(math.Round(lerpFloat(float64(base.B), 255, 0.55*sp)))
		return base
	}

	// Compose the per-frame colour table. Pre-computing the whole
	// timeline up front lets the median-cut quantiser see every
	// distinct colour that will appear on screen, so static regions
	// stay stable and don't shimmer due to per-frame re-quantisation.
	cellRGBs := make([][]color.RGBA, introTotalFrames)
	freq := map[color.RGBA]int{}
	phaseB := introHoldFrames
	phaseC := phaseB + introMorphFrames
	phaseD := phaseC + introGraphFrames
	for t := 0; t < introTotalFrames; t++ {
		row := make([]color.RGBA, len(live))
		for i, s := range live {
			var c color.RGBA
			switch {
			case t < phaseB:
				c = s.wmRGB
			case t < phaseC:
				u := float64(t-phaseB) / float64(introMorphFrames)
				c = lerpRGB(s.wmRGB, graphColorAt(i, 0), u)
			case t < phaseD:
				c = graphColorAt(i, t-phaseC)
			default:
				u := float64(t-phaseD) / float64(introOutroFrames)
				c = lerpRGB(graphColorAt(i, introGraphFrames-1), s.wmRGB, u)
			}
			row[i] = c
			freq[c]++
		}
		cellRGBs[t] = row
	}
	if len(dead) > 0 && !theme.Transparent() {
		freq[theme.EmptyCell] += len(dead)
	}

	// Median-cut quantise the full timeline down to gifMaxColors so
	// the encoded palette is shared across every frame; without that
	// step empty cells would shimmer between near-identical greys
	// across the loop.
	distinct := make([]color.RGBA, 0, len(freq))
	for c := range freq {
		distinct = append(distinct, c)
	}
	sort.Slice(distinct, func(i, j int) bool {
		a, b := distinct[i], distinct[j]
		if a.R != b.R {
			return a.R < b.R
		}
		if a.G != b.G {
			return a.G < b.G
		}
		return a.B < b.B
	})
	weights := make([]int, len(distinct))
	for i, c := range distinct {
		weights[i] = freq[c]
	}
	quantised := medianCut(distinct, weights, gifMaxColors)
	fullPalette := make(color.Palette, 0, len(quantised)+1)
	fullPalette = append(fullPalette, transparentRGBA)
	for _, c := range quantised {
		fullPalette = append(fullPalette, c)
	}
	rgbToIdx := make(map[color.RGBA]uint8, len(distinct))
	for _, c := range distinct {
		rgbToIdx[c] = uint8(nearestPaletteIndex(fullPalette, c, 1))
	}
	emptyIdx := uint8(0)
	if !theme.Transparent() {
		emptyIdx = uint8(nearestPaletteIndex(fullPalette, theme.EmptyCell, 1))
	}

	// Frame encoding: identical delta-frame scheme to RenderGIFWithTheme.
	sw, sh := width*scale, height*scale
	sc := scale
	bounds := image.Rect(0, 0, sw, sh)
	frames := make([]*image.Paletted, introTotalFrames)
	delays := make([]int, introTotalFrames)
	disposals := make([]byte, introTotalFrames)

	base := image.NewPaletted(bounds, fullPalette)
	if !theme.Transparent() {
		for _, dc := range dead {
			drawRoundedRect(base, dc.x*sc, dc.y*sc, cell*sc, cell*sc, radius*sc, emptyIdx)
		}
	}
	for i, lc := range live {
		drawRoundedRect(base, lc.x*sc, lc.y*sc, cell*sc, cell*sc, radius*sc, rgbToIdx[cellRGBs[0][i]])
	}
	frames[0] = base
	delays[0] = gifFrameMs / 10
	disposals[0] = gif.DisposalNone

	for t := 1; t < introTotalFrames; t++ {
		var changed []int
		for i := range live {
			if rgbToIdx[cellRGBs[t-1][i]] != rgbToIdx[cellRGBs[t][i]] {
				changed = append(changed, i)
			}
		}
		if len(changed) == 0 {
			stub := image.NewPaletted(image.Rect(0, 0, 1, 1), fullPalette)
			frames[t] = stub
			delays[t] = gifFrameMs / 10
			disposals[t] = gif.DisposalNone
			continue
		}
		minX, minY := math.MaxInt32, math.MaxInt32
		maxX, maxY := 0, 0
		for _, i := range changed {
			lc := live[i]
			x0 := lc.x * sc
			y0 := lc.y * sc
			x1 := x0 + cell*sc
			y1 := y0 + cell*sc
			if x0 < minX {
				minX = x0
			}
			if y0 < minY {
				minY = y0
			}
			if x1 > maxX {
				maxX = x1
			}
			if y1 > maxY {
				maxY = y1
			}
		}
		sub := image.NewPaletted(image.Rect(minX, minY, maxX, maxY), fullPalette)
		for _, i := range changed {
			lc := live[i]
			drawRoundedRect(sub, lc.x*sc, lc.y*sc, cell*sc, cell*sc, radius*sc, rgbToIdx[cellRGBs[t][i]])
		}
		frames[t] = sub
		delays[t] = gifFrameMs / 10
		disposals[t] = gif.DisposalNone
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, &gif.GIF{
		Image:           frames,
		Delay:           delays,
		Disposal:        disposals,
		LoopCount:       0,
		BackgroundIndex: 0,
		Config: image.Config{
			ColorModel: fullPalette,
			Width:      sw,
			Height:     sh,
		},
	}); err != nil {
		return nil, fmt.Errorf("contribgraph: encode intro gif: %w", err)
	}
	return buf.Bytes(), nil
}

// datePlus returns the date string `firstDay + dayOffset days`. Used so
// the intro renderer can seed cellColor with the right per-cell date
// even when firstDay is empty (in which case it returns "").
func datePlus(firstDay string, dayOffset int) string {
	if firstDay == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02", firstDay)
	if err != nil {
		return ""
	}
	return t.AddDate(0, 0, dayOffset).Format("2006-01-02")
}
