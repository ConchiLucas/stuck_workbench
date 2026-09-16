package service

import "testing"

func TestBuildPhraseReviewRewritesFrozenMedia(t *testing.T) {
	raw := `{"schema":1,"kind":"listen_zh","skillCode":"listen_zh","selected":"2","example":{"kind":"listen_zh","speech":"Good morning.","speechUrl":"/api/v1/phrase/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"1","label":"早上好。"},{"id":"2","label":"下午好。"}],"answerId":"1"}}`
	r := buildPhraseReview(raw, "2")
	if r.Example == nil || r.Selected != "2" || r.Example.SpeechURL != "/api/phrase/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3" {
		t.Fatalf("%+v", r)
	}
}

func TestBuildPhraseReviewKeepsLiveSpeechURLUnproxied(t *testing.T) {
	raw := `{"schema":1,"kind":"listen_zh","skillCode":"listen_zh","example":{"kind":"listen_zh","speech":"Good morning.","speechUrl":"/api/v1/phrase/items/20/speech.mp3","options":[{"id":"20","label":"早上好。"},{"id":"21","label":"下午好。"}],"answerId":"20"}}`
	r := buildPhraseReview(raw, "21")
	if r.Example == nil || r.Selected != "21" || r.Example.SpeechURL != "/api/v1/phrase/items/20/speech.mp3" {
		t.Fatalf("%+v", r)
	}
}

func TestBuildPhraseReviewIgnoresAccumulatedPicks(t *testing.T) {
	raw := `{"schema":1,"kind":"listen_zh","skillCode":"listen_zh","example":{"kind":"listen_zh","speech":"Good morning.","speechUrl":"/api/v1/phrase/items/20/speech.mp3","options":[{"id":"20","label":"早上好。"},{"id":"21","label":"下午好。"}],"answerId":"20"}}`
	r := buildPhraseReview(raw, `[{"clientId":"a","optionIndex":1}]`)
	if r.Example == nil || r.Selected != "" {
		t.Fatalf("accumulated picks must not become selected: %+v", r)
	}
}

func TestBuildPhraseReviewMissingSnapshot(t *testing.T) {
	r := buildPhraseReview("", "")
	if r.UnavailableReason == "" {
		t.Fatal("expected unavailable reason")
	}
}

func TestBuildPhraseReviewRejectsCodeTypeOnlySnapshot(t *testing.T) {
	r := buildPhraseReview(`{"code":"listen_zh","type":"choice"}`, "2")
	if r.Example != nil || r.UnavailableReason == "" {
		t.Fatalf("legacy code/type snapshot must stay unrestorable: %+v", r)
	}
}
