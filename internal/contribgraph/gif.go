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

// Knobs controlling the GIF output. These match the historical
// scripts/make_gif.py defaults so the rendered animation looks the same
// without a Python interpreter on the user's machine.
const (
	gifFrames    = 60     // total frames per loop
	gifFrameMs   = 80     // wall-clock duration of each frame
	gifScale     = 2      // upscaling factor over the SVG cell geometry
	gifSeed      = 0xB011 // RNG seed for per-cell oscillator parameters
	gifMaxColors = 255    // 256 minus the reserved transparent slot
)

// emptyCellRGB is GitHub's light-theme "no contributions" cell colour. The
// SVG renderer skips these cells entirely (transparent background), but
// the GIF draws them as static rounded squares so the calendar layout is
// readable even on a sparse year.
var emptyCellRGB = color.RGBA{0xEB, 0xED, 0xF0, 0xFF}

// transparentRGBA is the colour reserved at palette index 0. Go's image/gif
// encoder treats any palette entry with A=0 as the transparent index, and
// we put it at index 0 so the default zero-value of an image.Paletted's
// pixel buffer renders as transparent.
var transparentRGBA = color.RGBA{0, 0, 0, 0}

// RenderGIF produces an animated GIF version of the same graph Render
// produces. Each contributing cell drifts through the BUILD palette at its
// own integer-cycles-per-loop frequency so frame[N-1] transitions cleanly
// into frame[0]; a separate sparkle oscillator adds an occasional bright
// twinkle. Empty cells render as static GitHub-grey rounded squares.
//
// The returned bytes are a complete, self-contained looping GIF safe to
// write straight to disk.
func RenderGIF(data *Data, username string, palette []string) ([]byte, error) {
	if data == nil {
		return emptyGIF()
	}
	if len(palette) == 0 {
		palette = Palette
	}

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
	if width <= 0 || height <= 0 {
		return emptyGIF()
	}

	type liveCell struct {
		x, y int
		rgb  color.RGBA
	}
	type deadCell struct {
		x, y int
	}
	var live []liveCell
	var dead []deadCell
	for wi, w := range data.Weeks {
		firstDay, _ := time.Parse("2006-01-02", w.FirstDay)
		for _, d := range w.ContributionDays {
			x := wi * stride
			y := d.Weekday * stride
			if d.Level <= 0 {
				dead = append(dead, deadCell{x: x, y: y})
				continue
			}
			dateStr := ""
			if !firstDay.IsZero() {
				dateStr = firstDay.AddDate(0, 0, d.Weekday).Format("2006-01-02")
			}
			r, g, b := hexToRGB(cellColor(palette, dateStr, d.Level))
			live = append(live, liveCell{
				x:   x,
				y:   y,
				rgb: color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xFF},
			})
		}
	}

	liveRGBs := make([]color.RGBA, len(live))
	for i, c := range live {
		liveRGBs[i] = c.rgb
	}
	sortedPalette := uniqueHSVSorted(liveRGBs)
	paletteIdx := make(map[color.RGBA]int, len(sortedPalette))
	for i, c := range sortedPalette {
		paletteIdx[c] = i
	}
	n := len(sortedPalette)

	// Per-cell oscillator parameters. Seeded so the animation is
	// identical across runs and platforms.
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
	for i, c := range live {
		o := osc{
			phase: float64(paletteIdx[c.rgb]),
			dk:    driftKs[rng.Intn(len(driftKs))],
			damp:  4.0 + rng.Float64()*4.0, // uniform [4, 8)
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

	// Pre-compute the per-frame colour of every live cell, and a weighted
	// histogram so the palette quantiser can prefer hot colours.
	omega := 2 * math.Pi / float64(gifFrames)
	cellRGBs := make([][]color.RGBA, gifFrames)
	freq := map[color.RGBA]int{}
	for t := 0; t < gifFrames; t++ {
		row := make([]color.RGBA, len(live))
		for i, o := range oscs {
			if n == 0 {
				// No palette to drift through (no live cells at all);
				// degenerate but kept for symmetry with the dead branch.
				continue
			}
			pos := o.phase + o.damp*math.Sin(float64(o.dk)*omega*float64(t)+o.doff)
			fl := math.Floor(pos)
			iBase := int(fl)
			frac := pos - fl
			a := sortedPalette[modPositive(iBase, n)]
			b := sortedPalette[modPositive(iBase+1, n)]
			base := lerpRGB(a, b, frac)

			// max(0, sin) then ^6 narrows the sparkle to brief peaks
			// rather than a continuous pulse.
			sp := math.Max(0, math.Sin(float64(o.sk)*omega*float64(t)+o.soff))
			sp = math.Pow(sp, 6)
			base.R = clampU8(math.Round(lerpFloat(float64(base.R), 255, 0.55*sp)))
			base.G = clampU8(math.Round(lerpFloat(float64(base.G), 255, 0.55*sp)))
			base.B = clampU8(math.Round(lerpFloat(float64(base.B), 255, 0.55*sp)))

			row[i] = base
			freq[base]++
		}
		cellRGBs[t] = row
	}
	if len(dead) > 0 {
		freq[emptyCellRGB] += len(dead)
	}

	// Quantise the union of every distinct (cell, frame) RGB down to
	// gifMaxColors entries via weighted median cut. Doing the quantisation
	// over the whole timeline rather than per-frame is what keeps static
	// regions (the empty grey grid, lulls in the sparkle) from shimmering
	// frame-to-frame.
	distinct := make([]color.RGBA, 0, len(freq))
	for c := range freq {
		distinct = append(distinct, c)
	}
	// Map iteration order is randomised in Go; sort here so the median-cut
	// input (and therefore the resulting palette and the encoded GIF bytes)
	// is the same across runs and machines.
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
	if len(dead) > 0 {
		emptyIdx = rgbToIdx[emptyCellRGB]
	}

	// Build and encode the frames. Each frame uses the same shared palette
	// so all frame *image.Paletted instances quantise the same RGB to the
	// same index, eliminating per-frame palette drift.
	//
	// We use a delta-frame scheme: frame 0 is a full image; subsequent
	// frames are clipped to the bounding box of cells whose quantised
	// palette index changed from the previous frame, with unchanged
	// pixels left at palette[0] (the transparent index) so the decoder's
	// disposal=DisposalNone leaves the previous frame's pixels in place.
	// This shrinks the encoded file by a large factor on graphs where
	// most cells don't visibly change frame-to-frame.
	sw, sh := width*gifScale, height*gifScale
	sc := gifScale
	bounds := image.Rect(0, 0, sw, sh)

	frames := make([]*image.Paletted, gifFrames)
	delays := make([]int, gifFrames)
	disposals := make([]byte, gifFrames)

	// Frame 0: full image, baseline for every subsequent delta.
	base := image.NewPaletted(bounds, fullPalette)
	for _, dc := range dead {
		drawRoundedRect(base, dc.x*sc, dc.y*sc, cell*sc, cell*sc, radius*sc, emptyIdx)
	}
	if gifFrames > 0 {
		for i, lc := range live {
			drawRoundedRect(base, lc.x*sc, lc.y*sc, cell*sc, cell*sc, radius*sc, rgbToIdx[cellRGBs[0][i]])
		}
	}
	frames[0] = base
	delays[0] = gifFrameMs / 10
	disposals[0] = gif.DisposalNone

	for t := 1; t < gifFrames; t++ {
		var changed []int
		for i := range live {
			if rgbToIdx[cellRGBs[t-1][i]] != rgbToIdx[cellRGBs[t][i]] {
				changed = append(changed, i)
			}
		}
		if len(changed) == 0 {
			// No visible change vs previous frame; emit a 1×1 transparent
			// stub so the loop timing still advances correctly without
			// re-encoding any pixels.
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
	err := gif.EncodeAll(&buf, &gif.GIF{
		Image:           frames,
		Delay:           delays,
		Disposal:        disposals,
		LoopCount:       0, // 0 == loop forever
		BackgroundIndex: 0,
		Config: image.Config{
			ColorModel: fullPalette,
			Width:      sw,
			Height:     sh,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("contribgraph: encode gif: %w", err)
	}
	return buf.Bytes(), nil
}

// emptyGIF returns a 1×1 fully-transparent looping GIF. We hand this back
// for degenerate inputs (nil data, zero weeks) instead of erroring so
// callers always get a writeable file.
func emptyGIF() ([]byte, error) {
	pal := color.Palette{transparentRGBA}
	img := image.NewPaletted(image.Rect(0, 0, 1, 1), pal)
	var buf bytes.Buffer
	err := gif.EncodeAll(&buf, &gif.GIF{
		Image:     []*image.Paletted{img},
		Delay:     []int{gifFrameMs / 10},
		LoopCount: 0,
		Config: image.Config{
			ColorModel: pal,
			Width:      1,
			Height:     1,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("contribgraph: encode empty gif: %w", err)
	}
	return buf.Bytes(), nil
}

// uniqueHSVSorted returns the unique cell colours sorted by HSV. We use
// HSV order so per-cell drift sweeps neighbouring hues (red → orange →
// yellow → green …) rather than jumping around the spectrum.
func uniqueHSVSorted(cs []color.RGBA) []color.RGBA {
	seen := make(map[color.RGBA]bool, len(cs))
	out := make([]color.RGBA, 0, len(cs))
	for _, c := range cs {
		if seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		hi, si, vi := rgbToHSV(out[i])
		hj, sj, vj := rgbToHSV(out[j])
		if hi != hj {
			return hi < hj
		}
		if si != sj {
			return si < sj
		}
		return vi < vj
	})
	return out
}

// drawRoundedRect paints a rounded rectangle of size w×h, top-left at
// (x0,y0), into img at palette index idx. Pixels are inside the shape iff
// their distance from the nearest "corner-anchor" point (the clamp of the
// pixel into the inner straight band) is ≤ radius. The result is binary
// (no antialiasing) which is the only thing GIF's palette index encoding
// can natively express.
func drawRoundedRect(img *image.Paletted, x0, y0, w, h, radius int, idx uint8) {
	if radius < 0 {
		radius = 0
	}
	if 2*radius > w {
		radius = w / 2
	}
	if 2*radius > h {
		radius = h / 2
	}
	r2 := radius * radius
	for dy := 0; dy < h; dy++ {
		var ccy int
		switch {
		case dy < radius:
			ccy = radius
		case dy > h-1-radius:
			ccy = h - 1 - radius
		default:
			ccy = dy
		}
		ey := dy - ccy
		for dx := 0; dx < w; dx++ {
			var ccx int
			switch {
			case dx < radius:
				ccx = radius
			case dx > w-1-radius:
				ccx = w - 1 - radius
			default:
				ccx = dx
			}
			ex := dx - ccx
			if ex*ex+ey*ey > r2 {
				continue
			}
			img.SetColorIndex(x0+dx, y0+dy, idx)
		}
	}
}

// rgbToHSV converts an 8-bit RGB triple to HSV in [0, 1].
func rgbToHSV(c color.RGBA) (h, s, v float64) {
	r := float64(c.R) / 255
	g := float64(c.G) / 255
	b := float64(c.B) / 255
	mx := math.Max(r, math.Max(g, b))
	mn := math.Min(r, math.Min(g, b))
	v = mx
	d := mx - mn
	if mx == 0 {
		s = 0
	} else {
		s = d / mx
	}
	if d == 0 {
		return 0, s, v
	}
	switch mx {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	case b:
		h = (r-g)/d + 4
	}
	h /= 6
	return
}

func lerpFloat(a, b, t float64) float64 { return a + (b-a)*t }

func lerpRGB(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: clampU8(math.Round(lerpFloat(float64(a.R), float64(b.R), t))),
		G: clampU8(math.Round(lerpFloat(float64(a.G), float64(b.G), t))),
		B: clampU8(math.Round(lerpFloat(float64(a.B), float64(b.B), t))),
		A: 0xFF,
	}
}

func clampU8(f float64) uint8 {
	switch {
	case f <= 0:
		return 0
	case f >= 255:
		return 255
	default:
		return uint8(f)
	}
}

// modPositive returns ((i mod n) + n) mod n, i.e. a non-negative residue
// for negative i. Go's % operator preserves the sign of the dividend, so
// we need this when iBase can be negative after the sine drift.
func modPositive(i, n int) int {
	if n <= 0 {
		return 0
	}
	r := i % n
	if r < 0 {
		r += n
	}
	return r
}

// medianCut reduces colors to at most maxN representative entries via
// weighted median-cut quantisation. weights[i] is the number of pixels (or
// (cell, frame) pairs in our case) that would map to colors[i] in the
// original image.
func medianCut(colors []color.RGBA, weights []int, maxN int) []color.RGBA {
	if maxN <= 0 {
		return nil
	}
	if len(colors) == 0 {
		return nil
	}
	if len(colors) <= maxN {
		out := make([]color.RGBA, len(colors))
		copy(out, colors)
		return out
	}

	type box struct {
		cs []color.RGBA
		ws []int
	}
	// Defensive copy: median cut sorts in place, so we don't want to
	// scramble the caller's input slices.
	initCs := append([]color.RGBA(nil), colors...)
	initWs := append([]int(nil), weights...)
	boxes := []box{{cs: initCs, ws: initWs}}

	for len(boxes) < maxN {
		pick := -1
		bestSpan := 0
		for i, b := range boxes {
			if len(b.cs) < 2 {
				continue
			}
			if s := channelSpan(b.cs); s > bestSpan {
				bestSpan = s
				pick = i
			}
		}
		if pick == -1 {
			break
		}
		b := boxes[pick]
		axis := longestAxis(b.cs)
		sortByAxis(b.cs, b.ws, axis)

		total := 0
		for _, w := range b.ws {
			total += w
		}
		half := total / 2
		cum := 0
		split := len(b.cs) / 2
		for i, w := range b.ws {
			cum += w
			if cum > half {
				split = i + 1
				break
			}
		}
		if split < 1 {
			split = 1
		}
		if split >= len(b.cs) {
			split = len(b.cs) - 1
		}
		b1 := box{cs: b.cs[:split], ws: b.ws[:split]}
		b2 := box{cs: b.cs[split:], ws: b.ws[split:]}

		// Replace boxes[pick] with b1 and append b2.
		boxes[pick] = b1
		boxes = append(boxes, b2)
	}

	out := make([]color.RGBA, 0, len(boxes))
	for _, b := range boxes {
		if len(b.cs) == 0 {
			continue
		}
		var rSum, gSum, bSum, wSum int
		for j, c := range b.cs {
			w := b.ws[j]
			if w <= 0 {
				w = 1
			}
			rSum += int(c.R) * w
			gSum += int(c.G) * w
			bSum += int(c.B) * w
			wSum += w
		}
		if wSum == 0 {
			wSum = 1
		}
		out = append(out, color.RGBA{
			R: uint8(rSum / wSum),
			G: uint8(gSum / wSum),
			B: uint8(bSum / wSum),
			A: 0xFF,
		})
	}
	return out
}

func channelSpan(cs []color.RGBA) int {
	if len(cs) == 0 {
		return 0
	}
	rMin, gMin, bMin := 255, 255, 255
	rMax, gMax, bMax := 0, 0, 0
	for _, c := range cs {
		r, g, b := int(c.R), int(c.G), int(c.B)
		if r < rMin {
			rMin = r
		}
		if r > rMax {
			rMax = r
		}
		if g < gMin {
			gMin = g
		}
		if g > gMax {
			gMax = g
		}
		if b < bMin {
			bMin = b
		}
		if b > bMax {
			bMax = b
		}
	}
	return max3(rMax-rMin, gMax-gMin, bMax-bMin)
}

func longestAxis(cs []color.RGBA) int {
	rMin, gMin, bMin := 255, 255, 255
	rMax, gMax, bMax := 0, 0, 0
	for _, c := range cs {
		r, g, b := int(c.R), int(c.G), int(c.B)
		if r < rMin {
			rMin = r
		}
		if r > rMax {
			rMax = r
		}
		if g < gMin {
			gMin = g
		}
		if g > gMax {
			gMax = g
		}
		if b < bMin {
			bMin = b
		}
		if b > bMax {
			bMax = b
		}
	}
	rs, gs, bs := rMax-rMin, gMax-gMin, bMax-bMin
	switch {
	case rs >= gs && rs >= bs:
		return 0
	case gs >= bs:
		return 1
	default:
		return 2
	}
}

func sortByAxis(cs []color.RGBA, ws []int, axis int) {
	type pair struct {
		c color.RGBA
		w int
	}
	pairs := make([]pair, len(cs))
	for i := range cs {
		pairs[i] = pair{c: cs[i], w: ws[i]}
	}
	sort.Slice(pairs, func(i, j int) bool {
		var a, b uint8
		switch axis {
		case 0:
			a, b = pairs[i].c.R, pairs[j].c.R
		case 1:
			a, b = pairs[i].c.G, pairs[j].c.G
		default:
			a, b = pairs[i].c.B, pairs[j].c.B
		}
		return a < b
	})
	for i, p := range pairs {
		cs[i] = p.c
		ws[i] = p.w
	}
}

func max3(a, b, c int) int {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}

// nearestPaletteIndex returns the index in p of the colour closest to c
// in squared-RGB space, ignoring entries before startAt (so we can
// reserve palette[0] for transparency).
func nearestPaletteIndex(p color.Palette, c color.RGBA, startAt int) int {
	if startAt < 0 {
		startAt = 0
	}
	if startAt >= len(p) {
		return 0
	}
	bestI := startAt
	bestD := math.MaxInt32
	for i := startAt; i < len(p); i++ {
		pc, ok := p[i].(color.RGBA)
		if !ok {
			r, g, b, a := p[i].RGBA()
			pc = color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
		}
		dr := int(c.R) - int(pc.R)
		dg := int(c.G) - int(pc.G)
		db := int(c.B) - int(pc.B)
		d := dr*dr + dg*dg + db*db
		if d < bestD {
			bestD = d
			bestI = i
		}
	}
	return bestI
}
