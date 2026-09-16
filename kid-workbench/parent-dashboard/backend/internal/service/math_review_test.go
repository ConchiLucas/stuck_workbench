package service

import "testing"

func TestBuildMathReviewUsesFrozenSnapshotWithoutSpeechForCalc(t *testing.T) {
	raw := `{"questionId":42,"code":"calc","stem":"2 + 5 = ?","options":[{"label":"5"},{"label":"6"},{"label":"7"},{"label":"8"}],"answerIndex":2,"visual":{"kind":"equation","a":2,"b":5,"operator":"+"},"audioObjectKey":"math/questions/42.mp3"}`
	r := buildMathReview(raw, "0,2", 1, 7, 9)
	if r.Example == nil || r.Example.Kind != "choice" || r.Example.Prompt != "2 + 5" || r.Selected != "7" || r.Example.AudioURL != "" || r.AudioMutable {
		t.Fatalf("%+v", r)
	}
}

func TestBuildMathReviewKeepsFindAudioProxyAndMutableHint(t *testing.T) {
	raw := `{"questionId":50,"code":"find","options":[{"shape":"circle"},{"shape":"square"},{"shape":"triangle"},{"shape":"star"}],"answerIndex":0,"audioObjectKey":"math/questions/50.mp3"}`
	r := buildMathReview(raw, "1", 1, 7, 9)
	if r.Example == nil || r.Example.Kind != "audio-shape" || r.Example.AudioURL != "/api/math/children/1/plans/7/items/9/speech.mp3" || !r.AudioMutable || r.Selected != "正方形" {
		t.Fatalf("%+v", r)
	}
}

func TestBuildMathReviewMissingSnapshot(t *testing.T) {
	r := buildMathReview("", "", 1, 0, 0)
	if r.Example != nil || r.UnavailableReason == "" {
		t.Fatalf("%+v", r)
	}
}
