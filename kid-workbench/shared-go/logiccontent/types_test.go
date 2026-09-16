package logiccontent

import (
	"strings"
	"testing"
)

func TestCuratedMaterialsValidate(t *testing.T) {
	for _, spec := range CuratedMaterials() {
		if err := Validate(spec.Example); err != nil {
			t.Fatalf("%s: %v", spec.Code, err)
		}
		if spec.Example.Kind == "pattern" && strings.Contains(spec.Title, "成对") {
			if spec.Example.AnswerID != "bn2" {
				t.Fatalf("成对水果应补全第二根香蕉，而不是复用序列中的香蕉")
			}
		}
	}
}

func TestClassifyUniqueOdd(t *testing.T) {
	ex := mustSpec("logic-classify-animal")
	if got := oddOneOut(ex); len(got) != 1 || got[0] != "car" {
		t.Fatalf("classify odd=%v", got)
	}
}

func TestDiffRejectsAllDifferent(t *testing.T) {
	ex := mustSpec("logic-diff-dir")
	if err := Validate(ex); err != nil {
		t.Fatal(err)
	}
	bad := ex
	bad.Objects[0].Attrs["direction"] = "up"
	bad.Objects[1].Attrs["direction"] = "down"
	bad.Objects[2].Attrs["direction"] = "left"
	bad.Objects[3].Attrs["direction"] = "right"
	if err := Validate(bad); err == nil {
		t.Fatal("四个方向全不同必须失败")
	}
}

func TestCompareNeedsScale(t *testing.T) {
	ex := mustSpec("logic-compare-visual")
	if ex.Objects[0].Scale < ex.Objects[1].Scale {
		t.Fatal("公交车必须更大")
	}
	bad := ex
	for i := range bad.Objects {
		bad.Objects[i].Scale = 0
	}
	if err := Validate(bad); err == nil {
		t.Fatal("画面尺寸比较缺 scale 必须失败")
	}
}

func TestOrderDuplicateInstances(t *testing.T) {
	ex := mustSpec("logic-order-dup")
	ids := map[string]bool{}
	for _, o := range ex.Objects {
		if ids[o.ID] {
			t.Fatal("id 必须唯一")
		}
		ids[o.ID] = true
	}
	if ex.Objects[0].Glyph != ex.Objects[2].Glyph {
		t.Fatal("重复图形实例应共用 glyph")
	}
	ok, err := Judge(ex, AnswerInput{Sequence: []string{"d1", "d2", "d3"}})
	if err != nil || !ok {
		t.Fatal(err)
	}
	wrong, _ := Judge(ex, AnswerInput{Sequence: []string{"d3", "d2", "d1"}})
	if wrong {
		t.Fatal("错误顺序不应判定正确")
	}
}

func TestShuffleDoesNotChangeAnswer(t *testing.T) {
	ex := mustSpec("logic-classify-animal")
	shuffled := ShuffleIDs(ex.Options, 42)
	if len(shuffled) != len(ex.Options) {
		t.Fatal("shuffle length")
	}
	ex.Options = shuffled
	if err := Validate(ex); err != nil {
		t.Fatal(err)
	}
	ok, err := Judge(ex, AnswerInput{SelectedID: "car"})
	if err != nil || !ok {
		t.Fatal("打乱选项后答案 id 仍应成立")
	}
}

func TestUnrotatedArrowPointsRight(t *testing.T) {
	svg := string(RenderSVG(Object{ID: "t0", Caption: "向右", Glyph: "arrow", Fill: "#2563eb"}))
	if strings.Contains(svg, `points="12,32`) {
		t.Fatalf("默认箭头仍指向左，和「向右」标签冲突: %s", svg)
	}
	if !strings.Contains(svg, `points="52,32`) {
		t.Fatalf("默认箭头的尖应在右侧: %s", svg)
	}
	turned := string(RenderSVG(Object{ID: "t90", Caption: "向下", Glyph: "arrow", Fill: "#2563eb", Rotate: 90}))
	if !strings.Contains(turned, `rotate(90 32 32)`) {
		t.Fatalf("向下应绕中心顺时针 90 度: %s", turned)
	}
}

func TestAABBNotCopyAnswerIndex(t *testing.T) {
	ex := mustSpec("logic-pattern-aabb")
	shown := strings.Join(ex.Sequence, ",")
	if shown != "a1,a2,bn1" {
		t.Fatalf("shown=%s", shown)
	}
	if ex.AnswerID == "bn1" {
		t.Fatal("不应把已展示的香蕉当作下一格")
	}
}

func mustSpec(code string) LogicExample {
	for _, spec := range CuratedMaterials() {
		if spec.Code == code {
			return spec.Example
		}
	}
	panic(code)
}
