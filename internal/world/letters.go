package world

// Letter masks for the five BUILD floors. Each letter is authored at a small
// "stamp" size (8 columns × 10 rows) using '#' for inside-mask and '.' for
// outside. Strokes are intentionally 3 stamp cells thick (≈3× the previous
// design) so the letters read clearly as B, U, I, L, D at level scale. The
// stamp is scaled up to mask grid size by repeating each logical cell sx × sy
// times. With sx=9, sy=2 we get a 72×20 letter that fits inside the standard
// 80×22 level with a comfortable margin.
//
// At runtime, LetterFor(depth) returns a LetterMask sized exactly to a level
// (level coordinates), with the letter centered. BSP generation and rendering
// both consult the mask: cells outside the mask are forced to TileWall, so the
// dungeon's silhouette is the letter itself.

const (
	maskLevelW  = 80
	maskLevelH  = 22
	maskScaleX  = 9
	maskScaleY  = 2
	maskLetterW = 72                              // 8 * sx
	maskLetterH = 20                              // 10 * sy
	maskOffsetX = (maskLevelW - maskLetterW) / 2  // = 4
	maskOffsetY = (maskLevelH - maskLetterH) / 2  // = 1
)

// LetterMask is a level-sized boolean grid. cells[y][x] == true means the cell
// is inside the letter (BSP may carve it; outside cells stay TileWall).
type LetterMask struct {
	W, H    int
	cells   [][]bool
	Letter  byte // 'B', 'U', 'I', 'L', 'D'
	ColorIx int  // index into palette.Cycle: B=0,U=1,I=2,L=3,D=4
}

func (m *LetterMask) Contains(p Point) bool {
	if m == nil {
		return true
	}
	if p.X < 0 || p.Y < 0 || p.X >= m.W || p.Y >= m.H {
		return false
	}
	return m.cells[p.Y][p.X]
}

// Edge reports whether p is inside the mask and has at least one orthogonal
// neighbor outside the mask. Used by the renderer to tint the letter outline.
func (m *LetterMask) Edge(p Point) bool {
	if m == nil || !m.Contains(p) {
		return false
	}
	for _, d := range [4]Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		n := Point{p.X + d.X, p.Y + d.Y}
		if !m.Contains(n) {
			return true
		}
	}
	return false
}

// Stamp returns the small (8×10) logical bitmap for the given letter ('B',
// 'U', 'I', 'L', 'D'). Each row is a string of length 8 where '#' marks an
// inside cell and '.' marks an outside cell. Returns nil for unknown letters.
// Exported so UI code (e.g. the teleport animation) can render the same
// silhouette used to build the level mask.
func Stamp(letter byte) []string {
	switch letter {
	case 'B':
		return []string{
			"########",
			"########",
			"###..###",
			"###..###",
			"########",
			"########",
			"###..###",
			"###..###",
			"########",
			"########",
		}
	case 'U':
		return []string{
			"###..###",
			"###..###",
			"###..###",
			"###..###",
			"###..###",
			"###..###",
			"###..###",
			"########",
			"########",
			"########",
		}
	case 'I':
		return []string{
			"########",
			"########",
			"..####..",
			"..####..",
			"..####..",
			"..####..",
			"..####..",
			"..####..",
			"########",
			"########",
		}
	case 'L':
		return []string{
			"###.....",
			"###.....",
			"###.....",
			"###.....",
			"###.....",
			"###.....",
			"###.....",
			"########",
			"########",
			"########",
		}
	case 'D':
		return []string{
			"######..",
			"########",
			"###...##",
			"###....#",
			"###....#",
			"###....#",
			"###....#",
			"###...##",
			"########",
			"######..",
		}
	}
	return nil
}

// LetterFor returns the mask for floor depth (1=B, 2=U, 3=I, 4=L, 5=D).
func LetterFor(depth int) *LetterMask {
	letters := []byte{'B', 'U', 'I', 'L', 'D'}
	if depth < 1 || depth > len(letters) {
		return nil
	}
	letter := letters[depth-1]
	return makeMask(letter, depth-1)
}

func makeMask(letter byte, colorIx int) *LetterMask {
	st := Stamp(letter)
	if st == nil {
		return nil
	}
	m := &LetterMask{
		W:       maskLevelW,
		H:       maskLevelH,
		Letter:  letter,
		ColorIx: colorIx,
	}
	m.cells = make([][]bool, m.H)
	for y := 0; y < m.H; y++ {
		m.cells[y] = make([]bool, m.W)
	}
	// Paint scaled stamp into the level grid at (maskOffsetX, maskOffsetY).
	for sy, line := range st {
		for sx := 0; sx < len(line); sx++ {
			if line[sx] != '#' {
				continue
			}
			x0 := maskOffsetX + sx*maskScaleX
			y0 := maskOffsetY + sy*maskScaleY
			for dy := 0; dy < maskScaleY; dy++ {
				for dx := 0; dx < maskScaleX; dx++ {
					m.cells[y0+dy][x0+dx] = true
				}
			}
		}
	}
	return m
}
