package mathcontent

import "testing"

func TestDefaultsCoverTwelveValidDetails(t *testing.T) {
	c := Defaults()
	if len(c.Items) != 12 {
		t.Fatalf("got %d details", len(c.Items))
	}
	ids := map[string]bool{}
	for _, d := range c.Items {
		if ids[d.ID] {
			t.Fatal("duplicate", d.ID)
		}
		ids[d.ID] = true
		if err := Validate(d); err != nil {
			t.Fatalf("%s: %v", d.ID, err)
		}
	}
}
func TestRejectsAmbiguousAnswersAndInvalidQuantities(t *testing.T) {
	d := Defaults().Items[0]
	d.Example.Answer = "999"
	if Validate(d) == nil {
		t.Fatal("answer outside options accepted")
	}
	d = Defaults().Items[5]
	d.Example.Counts = []int{2, 8}
	if Validate(d) == nil {
		t.Fatal("subtracting more than available accepted")
	}
	d = Defaults().Items[0]
	d.Example.Options = []string{"8", "8"}
	if Validate(d) == nil {
		t.Fatal("duplicate options accepted")
	}
}

func TestRejectsIncorrectEquationAndJudgeAnswers(t *testing.T) {
	d := Defaults().Items[0]
	d.Example.Answer = "7"
	if Validate(d) == nil {
		t.Fatal("incorrect arithmetic answer accepted")
	}
	d = Defaults().Items[3]
	d.Example.Answer = "对"
	if Validate(d) == nil {
		t.Fatal("incorrect judgement accepted")
	}
}

func TestRejectsShapeAndCourseContradictions(t *testing.T) {
	d := Defaults().Items[9]
	d.Example.Answer = "圆形"
	if Validate(d) == nil {
		t.Fatal("triangle rendered with circle answer")
	}
	d = Defaults().Items[8]
	d.Example.ShapeKeys = []string{"circle"}
	if Validate(d) == nil {
		t.Fatal("truncated shape options")
	}
	d = Defaults().Items[0]
	d.Example.Prompt = "9 − 1 = ?"
	if Validate(d) == nil {
		t.Fatal("subtraction inside addition course")
	}
	d = Defaults().Items[3]
	d.Example.Statements = []string{"20 + 20 = 12"}
	if Validate(d) == nil {
		t.Fatal("judge outside course range")
	}
}

func TestPublishedAudioMustUseImmutableMaterialReference(t *testing.T) {
	d := Defaults().Items[0]
	d.Example.AudioURL = "https://example.com/mutable.mp3"
	if Validate(d) == nil {
		t.Fatal("mutable remote media accepted")
	}
	d.Example.AudioURL = "/api/v1/math/task-media/" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" + ".mp3"
	if Validate(d) != nil {
		t.Fatal("frozen task media rejected")
	}
}

func TestPublishedImageMustUseImmutableMaterialReference(t *testing.T) {
	d := Defaults().Items[1]
	d.Example.ObjectImageURL = "https://example.com/star.png"
	if Validate(d) == nil {
		t.Fatal("mutable remote image accepted")
	}
	d.Example.ObjectImageURL = "/api/v1/math/detail-image/" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" + ".png"
	if err := Validate(d); err != nil {
		t.Fatal(err)
	}
	d.Example.ObjectImageURL = "/api/v1/math/task-media/" + "bbbbbbbbaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" + ".webp"
	if err := Validate(d); err != nil {
		t.Fatal(err)
	}
}

func TestChoicePromptMayOmitEquals(t *testing.T) {
	d := Defaults().Items[0]
	d.Example.Prompt = "3 + 5"
	d.Example.Answer = "8"
	if err := Validate(d); err != nil {
		t.Fatal(err)
	}
	d.Example.Prompt = "3 + 5 = ?"
	if err := Validate(d); err != nil {
		t.Fatal(err)
	}
}
