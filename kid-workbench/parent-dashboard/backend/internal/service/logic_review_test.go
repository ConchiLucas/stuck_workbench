package service

import "testing"

func TestBuildLogicReviewRewritesFrozenMedia(t *testing.T) {
	raw := `{"schema":1,"kind":"classify","prompt":"哪个不属于这一类？","rule":{"type":"odd-one-out","dimension":"kingdom","inGroup":"animal","explain":"动物"},"objects":[{"id":"cat","caption":"猫","glyph":"cat","attrs":{"category":"animal"}},{"id":"car","caption":"汽车","glyph":"car","attrs":{"category":"vehicle"}}],"options":["cat","car"],"answerId":"car","imageUrls":{"car":"/api/v1/logic/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.svg"}}`
	r := buildLogicReview(raw, `{"selectedId":"cat"}`)
	if r.Example == nil || r.Example.ImageURLs["car"] != "/api/logic/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.svg" {
		t.Fatalf("expected rewritten frozen image, got %#v", r.Example)
	}
	if len(r.Facts) == 0 {
		t.Fatal("expected structured choice facts")
	}
}

func TestBuildLogicReviewMissingSnapshot(t *testing.T) {
	r := buildLogicReview("", "")
	if r.Example != nil || r.UnavailableReason == "" {
		t.Fatalf("%+v", r)
	}
}

func TestBuildLogicReviewRejectsCodeTypeOnlySnapshot(t *testing.T) {
	r := buildLogicReview(`{"code":"classify","type":"choice"}`, "car")
	if r.Example != nil || r.UnavailableReason == "" {
		t.Fatalf("%+v", r)
	}
}
