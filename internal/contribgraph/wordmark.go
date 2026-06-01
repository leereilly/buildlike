package contribgraph

// BUILD 2026 wordmark generator.
//
// The contribution-graph renderer normally consumes a year's worth of
// .contribs JSON for a real GitHub handle, but the README also wants a
// stylised "BUILD 2026" hero animation rendered through the same code
// path. WordmarkBuild2026 returns a synthetic *Data shaped exactly like
// the real payload — 53 weeks × 7 days — where the per-cell Level values
// have been hand-arranged so the contributing cells spell out the
// "BUILD 2026" wordmark across rows 1..5 of the grid.

import (
	"hash/fnv"
	"strings"
)

// wordmarkLayout is the source-of-truth bitmap that the renderer rasterises
// into the contribution grid. Each character is described by a slice of
// equal-width strings (rows); 'X' marks a contributing cell and '.' marks
// an empty cell. The five-row height matches BUILD 2026's calendar
// placement (rows 1..5 of the 7-row grid, leaving one empty row of breathing
// space top and bottom).
var wordmarkBuild2026 = []wordmarkGlyph{
	{ch: 'B', rows: []string{
		"XXXX.",
		"X...X",
		"XXXX.",
		"X...X",
		"XXXX.",
	}},
	{ch: 'U', rows: []string{
		"X...X",
		"X...X",
		"X...X",
		"X...X",
		"XXXXX",
	}},
	{ch: 'I', rows: []string{
		"XXX",
		".X.",
		".X.",
		".X.",
		"XXX",
	}},
	{ch: 'L', rows: []string{
		"X....",
		"X....",
		"X....",
		"X....",
		"XXXXX",
	}},
	{ch: 'D', rows: []string{
		"XXXX.",
		"X...X",
		"X...X",
		"X...X",
		"XXXX.",
	}},
	{ch: ' ', rows: []string{
		".",
		".",
		".",
		".",
		".",
	}},
	{ch: '2', rows: []string{
		"XXXX.",
		"....X",
		"XXXX.",
		"X....",
		"XXXXX",
	}},
	{ch: '0', rows: []string{
		".XXX.",
		"X...X",
		"X...X",
		"X...X",
		".XXX.",
	}},
	{ch: '2', rows: []string{
		"XXXX.",
		"....X",
		"XXXX.",
		"X....",
		"XXXXX",
	}},
	{ch: '6', rows: []string{
		".XXXX",
		"X....",
		"XXXX.",
		"X...X",
		".XXX.",
	}},
}

type wordmarkGlyph struct {
	ch   rune
	rows []string
}

// wordmarkLevel is the contribution Level assigned to every wordmark
// cell. Level 4 is the brightest tier, which makes the letters pop
// against the empty grid in any theme without depending on the seeded
// per-date palette pick going one way or the other.
const wordmarkLevel = 4

// WordmarkBuild2026Data returns a *Data whose Weeks/Days are arranged so
// that the contributing cells (Level > 0) spell out "BUILD 2026" across
// a standard 53-week × 7-day contribution grid.
//
// The result is suitable for passing directly to Render, RenderWithTheme,
// RenderGIF or RenderGIFWithTheme — the renderers don't distinguish
// synthetic payloads from real .contribs JSON.
//
// The returned payload is deterministic: every call returns an equivalent
// graph, and its date fields are populated for 2026 so the per-cell
// palette pick stays stable across runs.
func WordmarkBuild2026Data() *Data {
	const (
		weeks   = 53
		rows    = 7
		topPad  = 1 // empty row above the wordmark
		gapCols = 1 // empty column between glyphs
	)

	// Lay each glyph's row strings into a 7×weeks grid of '.'/'X' so
	// the wordmark is centred horizontally and sits between rows 1..5.
	totalCols := 0
	for i, g := range wordmarkBuild2026 {
		totalCols += len(g.rows[0])
		if i < len(wordmarkBuild2026)-1 {
			totalCols += gapCols
		}
	}
	// Widen the gap between "BUILD" and "2026" so the wordmark reads as
	// two words rather than one ten-character blob. The ' ' glyph is
	// 1 col wide, surrounded by the standard 1-col gaps on each side,
	// giving a 3-column visual gap between the 'D' and the '2' — wide
	// enough to read as a word break while keeping the whole wordmark
	// inside the 53-week contribution grid.
	extraSpaceCols := 0
	totalCols += extraSpaceCols
	if totalCols > weeks {
		totalCols = weeks
	}
	leftPad := (weeks - totalCols) / 2

	grid := make([][]byte, rows)
	for r := range grid {
		grid[r] = make([]byte, weeks)
		for c := range grid[r] {
			grid[r][c] = '.'
		}
	}
	col := leftPad
	for _, g := range wordmarkBuild2026 {
		w := len(g.rows[0])
		for r, line := range g.rows {
			for x := 0; x < w; x++ {
				if line[x] == 'X' && col+x < weeks {
					grid[topPad+r][col+x] = 'X'
				}
			}
		}
		col += w + gapCols
		if g.ch == ' ' {
			col += extraSpaceCols
		}
	}

	// Convert the grid into Weeks/Days. Each week needs a FirstDay so
	// the rendered cells get stable per-date palette picks; we anchor
	// to the first Sunday of 2026 (2026-01-04) and step seven days per
	// column.
	firstSunday := "2026-01-04"
	out := &Data{
		Schema:             "v2",
		From:               firstSunday,
		To:                 "2026-12-31",
		TotalContributions: 0,
		Months:             []Month{{Month: "2026-01", TotalWeeks: weeks}},
	}
	out.Weeks = make([]Week, weeks)
	for wi := 0; wi < weeks; wi++ {
		days := make([]Day, rows)
		for d := 0; d < rows; d++ {
			level := 0
			count := 0
			if grid[d][wi] == 'X' {
				level = wordmarkLevel
				// Salt the per-cell count so total_contributions looks
				// plausible. Not strictly required — the renderer only
				// reads Level — but it keeps the synthetic payload
				// faithful to the real schema.
				count = 1 + int(hashCol(wi, d))%9
				out.TotalContributions += count
			}
			days[d] = Day{Weekday: d, Count: count, Level: level}
		}
		out.Weeks[wi] = Week{
			Index:            wi,
			FirstDay:         dateForWeek(wi),
			ContributionDays: days,
		}
	}
	return out
}

// dateForWeek returns the FirstDay string for the wi-th week of the
// synthetic 2026 wordmark calendar. The dates are fixed (no time.Now())
// so the rendered GIF/SVG bytes stay reproducible.
func dateForWeek(wi int) string {
	// Manually advance day-of-month from 2026-01-04 in 7-day steps so
	// the function has no external dependency on the time package. The
	// renderer only uses these strings as a deterministic palette seed
	// (it never validates calendar arithmetic), so a simple closed-form
	// month/day table is fine.
	days := wi * 7
	month := 1
	day := 4 + days
	monthLengths := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	for month <= 12 && day > monthLengths[month-1] {
		day -= monthLengths[month-1]
		month++
	}
	if month > 12 {
		month = 12
		day = monthLengths[11]
	}
	var b strings.Builder
	b.WriteString("2026-")
	if month < 10 {
		b.WriteByte('0')
	}
	b.WriteString(itoa(month))
	b.WriteByte('-')
	if day < 10 {
		b.WriteByte('0')
	}
	b.WriteString(itoa(day))
	return b.String()
}

// itoa is a tiny base-10 conversion helper used only by dateForWeek so
// we don't have to pull strconv into the wordmark path.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// hashCol gives each (week, day) cell a stable per-cell count derived
// from a tiny FNV hash. Only used to populate the Count field of
// contributing days in the synthetic payload.
func hashCol(week, day int) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte{byte(week), byte(day)})
	return h.Sum32()
}
