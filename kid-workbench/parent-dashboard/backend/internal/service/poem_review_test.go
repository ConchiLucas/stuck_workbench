package service

import "testing"

func TestBuildPoemReviewRewritesFrozenMedia(t *testing.T) {
	raw := `{"schema":1,"kind":"title","skillCode":"title","example":{"kind":"title","prompt":"这首诗叫什么？","line":"床前明月光","workId":"pm001","options":[{"id":"pm002","label":"春晓"},{"id":"pm001","label":"静夜思"}],"answerId":"pm001","speechUrl":"/api/v1/poem/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.wav"}}`
	r := buildPoemReview(raw, "pm002")
	if r.Example == nil || r.Example.SpeechURL != "/api/poem/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.wav" {
		t.Fatalf("expected rewritten frozen speech, got %#v", r.Example)
	}
	if len(r.Facts) == 0 {
		t.Fatal("expected structured title error facts")
	}
}

func TestBuildPoemReviewMissingSnapshot(t *testing.T) {
	r := buildPoemReview("", "")
	if r.Example != nil || r.UnavailableReason == "" {
		t.Fatalf("%+v", r)
	}
}

func TestBuildPoemReviewRejectsCodeTypeOnlySnapshot(t *testing.T) {
	r := buildPoemReview(`{"code":"title","type":"choice"}`, "pm001")
	if r.Example != nil || r.UnavailableReason == "" {
		t.Fatalf("%+v", r)
	}
}
