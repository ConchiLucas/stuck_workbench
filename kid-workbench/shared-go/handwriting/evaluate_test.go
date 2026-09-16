package handwriting

import (
	"encoding/json"
	"os"
	"testing"
)

func TestEvaluate(t *testing.T) {
	template := Template{SchemaVersion: 1, Character: "十", Strokes: []string{"M 100 500 L 900 500", "M 500 100 L 500 900"}, Medians: [][][]float64{{{100, 500}, {900, 500}}, {{500, 100}, {500, 900}}}}
	cases := []struct {
		name    string
		strokes [][]Point
		pass    bool
	}{
		{"correct", [][]Point{{{.1, .5, 0}, {.9, .5, 1}}, {{.5, .1, 2}, {.5, .9, 3}}}, true},
		{"missing", [][]Point{{{.1, .5, 0}, {.9, .5, 1}}}, false},
		{"blank", nil, false},
		{"translated scaled", [][]Point{{{.3, .5, 0}, {.7, .5, 1}}, {{.5, .3, 2}, {.5, .7, 3}}}, true},
		{"diagonal", [][]Point{{{.1, .1, 0}, {.9, .9, 1}}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := Evaluate(tc.strokes, template, PolicyV1)
			if err != nil {
				t.Fatal(err)
			}
			if (r.Outcome == "passed") != tc.pass {
				t.Fatalf("result %+v", r)
			}
		})
	}
	if _, err := Evaluate([][]Point{{{2, 0, 0}}}, template, PolicyV1); err == nil {
		t.Fatal("unbounded coordinate accepted")
	}
	if _, err := Evaluate(nil, Template{}, PolicyV1); err == nil {
		t.Fatal("missing template accepted")
	}
	if _, err := Evaluate(nil, template, "future"); err == nil {
		t.Fatal("unknown policy accepted")
	}
	if err := ValidateStrokes([][]Point{{{0, 0, 2}, {1, 1, 1}}}); err == nil {
		t.Fatal("decreasing time accepted")
	}
}

func TestRealTemplates(t *testing.T) {
	for _, char := range []string{"山", "水", "一", "的"} {
		t.Run(char, func(t *testing.T) {
			b, err := os.ReadFile("testdata/" + char + ".json")
			if err != nil {
				t.Fatal(err)
			}
			var template Template
			if err = json.Unmarshal(b, &template); err != nil {
				t.Fatal(err)
			}
			strokes := make([][]Point, len(template.Medians))
			clock := 0.0
			for i, s := range template.Medians {
				for _, p := range s {
					strokes[i] = append(strokes[i], Point{X: p[0] / 1024, Y: 1 - p[1]/1024, T: clock})
					clock++
				}
			}
			r, err := Evaluate(strokes, template, PolicyV1)
			if err != nil || r.Outcome != "passed" {
				t.Fatalf("real template match %+v %v", r, err)
			}
			if len(strokes) > 1 {
				r, err = Evaluate(strokes[:1], template, PolicyV1)
				if err != nil || r.Outcome != "not_passed" {
					t.Fatalf("missing strokes %+v %v", r, err)
				}
			}
		})
	}
}

func TestDenseScribbleDoesNotPass(t *testing.T) {
	b, e := os.ReadFile("testdata/的.json")
	if e != nil {
		t.Fatal(e)
	}
	var template Template
	json.Unmarshal(b, &template)
	ink := [][]Point{}
	for y := 0; y < 32; y++ {
		ink = append(ink, []Point{{X: 0, Y: float64(y) / 31, T: float64(y * 2)}, {X: 1, Y: float64(y) / 31, T: float64(y*2 + 1)}})
	}
	r, e := Evaluate(ink, template, PolicyV1)
	if e != nil {
		t.Fatal(e)
	}
	if r.Outcome != "not_passed" {
		t.Fatalf("dense scribble accepted %+v", r)
	}
}

func TestSparseHorizontalScribbleDoesNotPass(t *testing.T) {
	for _, char := range []string{"山", "水", "的"} {
		b, e := os.ReadFile("testdata/" + char + ".json")
		if e != nil {
			t.Fatal(e)
		}
		var template Template
		json.Unmarshal(b, &template)
		ink := [][]Point{}
		for y := 0; y < 4; y++ {
			ink = append(ink, []Point{{X: 0, Y: float64(y) / 3, T: float64(y * 2)}, {X: 1, Y: float64(y) / 3, T: float64(y*2 + 1)}})
		}
		r, e := Evaluate(ink, template, PolicyV1)
		if e != nil {
			t.Fatal(e)
		}
		if r.Outcome != "not_passed" {
			t.Fatalf("%s sparse scribble accepted %+v", char, r)
		}
	}
}
