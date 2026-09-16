// Package handwriting implements versioned template coverage matching, not OCR.
package handwriting

import (
	"fmt"
	"math"
)

const PolicyV1 = "ink-match-v1"
const gridSize = 64

// These thresholds are immutable for this initial server policy. A future
// adjustment must introduce a new policy version, never re-score receipts.
const minStrokeCover = .52
const minMeanCover = .62
const minPrecision = .22
const maxInkFraction = .65
const minIndividualStrokeCover = .70

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	T float64 `json:"t"`
}
type Template struct {
	SchemaVersion    int           `json:"schemaVersion"`
	Character        string        `json:"character"`
	Strokes          []string      `json:"strokes"`
	Medians          [][][]float64 `json:"medians"`
	CoordinateSystem struct {
		Width    float64 `json:"width"`
		Height   float64 `json:"height"`
		YAxis    string  `json:"yAxis"`
		Baseline float64 `json:"baseline"`
	} `json:"coordinateSystem"`
	Source struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		License string `json:"license"`
	} `json:"source"`
	License string `json:"license,omitempty"`
}
type EvaluationResult struct {
	Outcome          string             `json:"outcome"`
	Assistance       string             `json:"assistance"`
	EvaluatorVersion string             `json:"evaluatorVersion"`
	Metrics          map[string]float64 `json:"metrics"`
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func ValidateStrokes(strokes [][]Point) error {
	if len(strokes) > 64 {
		return fmt.Errorf("at most 64 strokes")
	}
	total := 0
	last := -1.0
	for _, s := range strokes {
		if len(s) == 0 || len(s) > 512 {
			return fmt.Errorf("each stroke requires 1 to 512 points")
		}
		total += len(s)
		for _, p := range s {
			if !finite(p.X) || !finite(p.Y) || !finite(p.T) || p.X < 0 || p.X > 1 || p.Y < 0 || p.Y > 1 || p.T < 0 || p.T < last {
				return fmt.Errorf("invalid stroke coordinate or timestamp")
			}
			last = p.T
		}
	}
	if total > 8192 {
		return fmt.Errorf("at most 8192 points")
	}
	return nil
}
func ValidateTemplate(t Template) error {
	if t.SchemaVersion != 1 || len([]rune(t.Character)) != 1 || len(t.Medians) == 0 || len(t.Medians) > 64 || len(t.Strokes) != len(t.Medians) {
		return fmt.Errorf("invalid writing template")
	}
	if t.CoordinateSystem.Width != 0 && t.CoordinateSystem.Width != 1024 || t.CoordinateSystem.Height != 0 && t.CoordinateSystem.Height != 1024 || t.CoordinateSystem.YAxis != "" && t.CoordinateSystem.YAxis != "up" {
		return fmt.Errorf("unsupported template coordinates")
	}
	n := 0
	for i, s := range t.Medians {
		if len(s) < 2 || len(s) > 512 || t.Strokes[i] == "" {
			return fmt.Errorf("invalid template stroke")
		}
		n += len(s)
		for _, p := range s {
			if len(p) != 2 || !finite(p[0]) || !finite(p[1]) || p[0] < -128 || p[0] > 1152 || p[1] < -128 || p[1] > 1152 {
				return fmt.Errorf("invalid template median")
			}
		}
	}
	if n > 8192 {
		return fmt.Errorf("template has too many points")
	}
	return nil
}

// Evaluate uses the legacy 64-cell grid and coverage thresholds (0.52/0.62/0.22).
// Pointer coordinates are converted to the Hanzi template's upward Y axis before
// bounding-box normalization, preserving translation and scale invariance.
func Evaluate(strokes [][]Point, t Template, policy string) (EvaluationResult, error) {
	r := EvaluationResult{Outcome: "not_passed", Assistance: "none", EvaluatorVersion: PolicyV1, Metrics: map[string]float64{"minStrokeCover": 0, "meanStrokeCover": 0, "precision": 0}}
	if policy != PolicyV1 {
		return r, fmt.Errorf("unsupported evaluation policy")
	}
	if err := ValidateTemplate(t); err != nil {
		return r, err
	}
	if err := ValidateStrokes(strokes); err != nil {
		return r, err
	}
	if len(strokes) == 0 {
		return r, nil
	}
	user := make([][]Point, len(strokes))
	for i, s := range strokes {
		for _, p := range s {
			user[i] = append(user[i], Point{X: p.X * 1024, Y: (1 - p.Y) * 1024})
		}
	}
	model := make([][]Point, len(t.Medians))
	for i, s := range t.Medians {
		for _, p := range s {
			model[i] = append(model[i], Point{X: p[0], Y: p[1]})
		}
	}
	user = normalize(user)
	model = normalize(model)
	ink := rasterize(user, 3)
	standard := rasterize(model, 4)
	minCover := 1.0
	mean := 0.0
	individualCover := 1.0
	individualInk := make([][]bool, len(user))
	for i, stroke := range user {
		individualInk[i] = rasterize([][]Point{stroke}, 3)
	}
	for _, s := range model {
		best := 0.0
		for _, grid := range individualInk {
			best = math.Max(best, strokeCoverage(s, grid))
		}
		individualCover = math.Min(individualCover, best)
		hit := 0
		for i := 0; i < 20; i++ {
			at := float64(i) / 19 * float64(len(s)-1)
			idx := int(math.Min(float64(len(s)-2), math.Floor(at)))
			f := at - float64(idx)
			p := Point{X: s[idx].X + (s[idx+1].X-s[idx].X)*f, Y: s[idx].Y + (s[idx+1].Y-s[idx].Y)*f}
			if nearby(ink, p, 5) {
				hit++
			}
		}
		cover := float64(hit) / 20
		minCover = math.Min(minCover, cover)
		mean += cover
	}
	mean /= float64(len(model))
	sum, overlap := 0, 0
	for i, v := range ink {
		if v {
			sum++
			if standard[i] {
				overlap++
			}
		}
	}
	precision := 0.0
	if sum > 0 {
		precision = float64(overlap) / float64(sum)
	}
	r.Metrics = map[string]float64{"minStrokeCover": minCover, "meanStrokeCover": mean, "precision": precision, "minIndividualStrokeCover": individualCover, "inkFraction": float64(sum) / (gridSize * gridSize)}
	if individualCover >= minIndividualStrokeCover && minCover >= minStrokeCover && mean >= minMeanCover && precision >= minPrecision && float64(sum)/(gridSize*gridSize) <= maxInkFraction {
		r.Outcome = "passed"
	}
	return r, nil
}
func normalize(s [][]Point) [][]Point {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, stroke := range s {
		for _, p := range stroke {
			minX = math.Min(minX, p.X)
			minY = math.Min(minY, p.Y)
			maxX = math.Max(maxX, p.X)
			maxY = math.Max(maxY, p.Y)
		}
	}
	w, h := math.Max(1, maxX-minX), math.Max(1, maxY-minY)
	size := math.Max(w, h)
	pad := size * .1
	left, top := minX-(size-w)/2-pad, minY-(size-h)/2-pad
	for _, stroke := range s {
		for i := range stroke {
			stroke[i].X = (stroke[i].X - left) / (size + 2*pad)
			stroke[i].Y = (stroke[i].Y - top) / (size + 2*pad)
		}
	}
	return s
}
func stamp(grid []bool, p Point, r int) {
	x, y := int(math.Round(p.X*63)), int(math.Round(p.Y*63))
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			px, py := x+dx, y+dy
			if dx*dx+dy*dy <= r*r && px >= 0 && py >= 0 && px < 64 && py < 64 {
				grid[py*64+px] = true
			}
		}
	}
}
func rasterize(s [][]Point, r int) []bool {
	g := make([]bool, 4096)
	for _, stroke := range s {
		if len(stroke) == 1 {
			stamp(g, stroke[0], r)
		}
		for i := 1; i < len(stroke); i++ {
			a, b := stroke[i-1], stroke[i]
			n := int(math.Max(1, math.Ceil(math.Hypot(b.X-a.X, b.Y-a.Y)*128)))
			for j := 0; j <= n; j++ {
				f := float64(j) / float64(n)
				stamp(g, Point{X: a.X + (b.X-a.X)*f, Y: a.Y + (b.Y-a.Y)*f}, r)
			}
		}
	}
	return g
}
func nearby(g []bool, p Point, r int) bool {
	x, y := int(math.Round(p.X*63)), int(math.Round(p.Y*63))
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			px, py := x+dx, y+dy
			if px >= 0 && py >= 0 && px < 64 && py < 64 && g[py*64+px] {
				return true
			}
		}
	}
	return false
}

func strokeCoverage(stroke []Point, ink []bool) float64 {
	hit := 0
	for i := 0; i < 20; i++ {
		at := float64(i) / 19 * float64(len(stroke)-1)
		idx := int(math.Min(float64(len(stroke)-2), math.Floor(at)))
		f := at - float64(idx)
		p := Point{X: stroke[idx].X + (stroke[idx+1].X-stroke[idx].X)*f, Y: stroke[idx].Y + (stroke[idx+1].Y-stroke[idx].Y)*f}
		if nearby(ink, p, 5) {
			hit++
		}
	}
	return float64(hit) / 20
}
