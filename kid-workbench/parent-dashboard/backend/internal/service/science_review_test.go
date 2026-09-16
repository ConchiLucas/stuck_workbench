package service

import "testing"

func TestBuildScienceReviewRewritesFrozenMedia(t *testing.T) {
	raw := `{"schema":1,"kind":"match","skillCode":"match","example":{"kind":"match","prompt":"连","matchSources":[{"id":"frog","label":"青蛙"},{"id":"bear","label":"北极熊"}],"matchTargets":[{"id":"forest","label":"热带雨林","imageUrl":"/api/v1/science/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png"},{"id":"ice","label":"冰原"}],"matchAnswers":{"frog":"forest","bear":"ice"}}}`
	r := buildScienceReview(raw, `{"kind":"match","pairs":{"frog":"ice","bear":"forest"}}`)
	if r.Example == nil || r.Example.MatchTargets[0].ImageURL != "/api/science/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png" {
		t.Fatalf("expected rewritten frozen image, got %#v", r.Example)
	}
	if len(r.Facts) == 0 {
		t.Fatal("expected structured match error facts")
	}
}

func TestBuildScienceReviewMissingSnapshot(t *testing.T) {
	r := buildScienceReview("", "")
	if r.Example != nil || r.UnavailableReason == "" {
		t.Fatalf("%+v", r)
	}
}

func TestBuildScienceReviewRejectsCodeTypeOnlySnapshot(t *testing.T) {
	r := buildScienceReview(`{"code":"choice","type":"choice"}`, "duck")
	if r.Example != nil || r.UnavailableReason == "" {
		t.Fatalf("%+v", r)
	}
}
