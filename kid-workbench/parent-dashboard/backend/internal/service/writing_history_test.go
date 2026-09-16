package service

import "testing"

func TestFrozenWritingHistory(t *testing.T) {
	h, e := frozenHistory(`{"schemaVersion":2,"subjectCode":"literacy","questionType":"write_char","targetText":"山","interaction":"handwriting","prompt":"听音写字","stem":{"audio":{"revisionId":"rev","kind":"speech"}}}`)
	if e != nil {
		t.Fatal(e)
	}
	if h.Options != "[]" || h.Answer != "{}" {
		t.Fatalf("writing rendered as choice: %+v", h)
	}
}
