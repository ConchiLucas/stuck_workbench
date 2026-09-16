package service

import "testing"

func TestBuildEnglishReviewRewritesFrozenMedia(t *testing.T) {
	raw := `{"schema":1,"kind":"audio-choice","skillCode":"listen","selected":"2","example":{"kind":"audio-choice","speech":"apple","speechUrl":"/api/v1/english/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"1","label":"苹果","picture":"/api/v1/english/task-media/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb.jpg"},{"id":"2","label":"小狗","picture":"/api/v1/english/task-media/cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc.jpg"}],"answerId":"1"}}`
	r := buildEnglishReview(raw, "2")
	if r.Example == nil || r.Selected != "2" || r.Example.SpeechURL != "/api/english/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3" {
		t.Fatalf("%+v", r)
	}
}

func TestBuildEnglishReviewRewritesLiveWordMedia(t *testing.T) {
	raw := `{"schema":1,"kind":"audio-choice","skillCode":"listen","example":{"kind":"audio-choice","speech":"apple","speechUrl":"/api/v1/english/words/20/speech.mp3","options":[{"id":"20","label":"苹果","picture":"/api/v1/english/words/20/sense.png"},{"id":"21","label":"小狗","picture":"/api/v1/english/words/21/sense.png"}],"answerId":"20"}}`
	r := buildEnglishReview(raw, "21")
	if r.Example == nil || r.Selected != "21" || r.Example.SpeechURL != "/api/english/words/20/speech.mp3" {
		t.Fatalf("%+v", r)
	}
}

func TestBuildEnglishReviewIgnoresAccumulatedPicks(t *testing.T) {
	raw := `{"schema":1,"kind":"audio-choice","skillCode":"listen","example":{"kind":"audio-choice","speech":"apple","speechUrl":"/api/v1/english/words/20/speech.mp3","options":[{"id":"20","label":"苹果"},{"id":"21","label":"小狗"}],"answerId":"20"}}`
	r := buildEnglishReview(raw, `[{"clientId":"a","optionIndex":1}]`)
	if r.Example == nil || r.Selected != "" {
		t.Fatalf("accumulated picks must not become selected: %+v", r)
	}
}

func TestBuildEnglishReviewMissingSnapshot(t *testing.T) {
	r := buildEnglishReview("", "")
	if r.UnavailableReason == "" {
		t.Fatal("expected unavailable reason")
	}
}

func TestBuildEnglishReviewRejectsCodeTypeOnlySnapshot(t *testing.T) {
	r := buildEnglishReview(`{"code":"listen","type":"choice"}`, "2")
	if r.Example != nil || r.UnavailableReason == "" {
		t.Fatalf("legacy code/type snapshot must stay unrestorable: %+v", r)
	}
}

func TestBuildEnglishReviewKeepsOrderAndInput(t *testing.T) {
	order := `{"schema":1,"kind":"card-builder","skillCode":"build","selected":"is This an apple","example":{"kind":"card-builder","prompt":"把单词排成一句话","speechUrl":"/api/v1/english/sentences/this-is-an-apple/speech.mp3","bank":["This","is","an","apple"],"answer":"This is an apple"}}`
	r := buildEnglishReview(order, "This is an apple")
	if r.Example == nil || r.Selected != "This is an apple" || r.ResponseKind != "order" {
		t.Fatalf("%+v", r)
	}
	input := `{"schema":1,"kind":"input-gap","skillCode":"type","selected":"aple","example":{"kind":"input-gap","prompt":"写出这个单词","answer":"apple"}}`
	r = buildEnglishReview(input, "")
	if r.Example == nil || r.Selected != "aple" || r.ResponseKind != "input" {
		t.Fatalf("%+v", r)
	}
}
