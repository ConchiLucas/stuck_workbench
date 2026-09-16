package mathcontent

import "testing"

func TestPlanExampleFromCalcAndStorySnapshots(t *testing.T) {
	calc, err := PlanExampleFromSnapshot(`{"questionId":42,"code":"calc","stem":"2 + 5 = ?","options":[{"label":"5"},{"label":"6"},{"label":"7"},{"label":"8"}],"answerIndex":2,"visual":{"kind":"equation","a":2,"b":5,"operator":"+"},"audioObjectKey":"math/questions/42.mp3"}`, 1)
	if err != nil {
		t.Fatal(err)
	}
	if calc.Example.Kind != "choice" || calc.Example.Prompt != "2 + 5" || calc.Example.Answer != "7" || calc.Selected != "6" || calc.Example.AudioURL != "" {
		t.Fatalf("calc %+v", calc.Example)
	}
	story, err := PlanExampleFromSnapshot(`{"code":"story","options":[{"label":"6"},{"label":"7"},{"label":"8"},{"label":"9"}],"answerIndex":1,"visual":{"kind":"add","leftCount":2,"rightCount":5,"object":"apple"}}`, 1)
	if err != nil {
		t.Fatal(err)
	}
	if story.Example.Kind != "objects" || story.Example.Object != "苹果" || story.Example.Answer != "7" {
		t.Fatalf("story %+v", story.Example)
	}
}

func TestPlanExampleFromShapeSnapshots(t *testing.T) {
	find, err := PlanExampleFromSnapshot(`{"code":"find","options":[{"shape":"circle"},{"shape":"square"},{"shape":"triangle"},{"shape":"star"}],"answerIndex":0,"audioObjectKey":"math/questions/50.mp3"}`, 2)
	if err != nil {
		t.Fatal(err)
	}
	if find.Example.Kind != "audio-shape" || find.Example.Answer != "圆形" || find.Selected != "三角形" || !find.AudioMutable {
		t.Fatalf("find %+v selected=%s", find.Example, find.Selected)
	}
	name, err := PlanExampleFromSnapshot(`{"code":"name","options":[{"label":"圆形"},{"label":"正方形"},{"label":"三角形"},{"label":"五角星"}],"answerIndex":2,"visual":{"kind":"shape","shape":"triangle"}}`, -1)
	if err != nil {
		t.Fatal(err)
	}
	if name.Example.Kind != "shape-name" || name.Example.ShapeKeys[0] != "triangle" || name.Selected != "" {
		t.Fatalf("name %+v", name.Example)
	}
}

func TestLastPickIndex(t *testing.T) {
	if LastPickIndex("0,2") != 2 || LastPickIndex("") != -1 || LastPickIndex("x") != -1 {
		t.Fatal("pick parse")
	}
}
