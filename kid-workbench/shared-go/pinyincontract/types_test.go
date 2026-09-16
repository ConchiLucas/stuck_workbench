package pinyincontract

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPublicQuestionContractDoesNotRevealAnswer(t *testing.T) {
	q := GeneratedQuestion{InstanceID: "instance", Type: "shape", TargetID: 7, KpID: 100, Options: []Option{{ID: "7"}}}
	encoded, err := json.Marshal(InstanceSnapshot{GeneratedQuestion: q})
	if err != nil {
		t.Fatal(err)
	}
	s := string(encoded)
	for _, forbidden := range []string{"answerIndex", "answerOptionId", "acceptedResult"} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("public snapshot exposes %s: %s", forbidden, s)
		}
	}
	if !strings.Contains(s, `"id":"7"`) || !strings.Contains(s, `"kpId":100`) {
		t.Fatal(s)
	}
}
