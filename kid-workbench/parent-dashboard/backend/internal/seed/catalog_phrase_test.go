package seed

import (
	"encoding/json"
	"testing"
)

func TestPhraseCatalogHasSceneAndReplyPairs(t *testing.T) {
	var scenes, replies int
	for _, mod := range phraseModules() {
		for _, kp := range mod.Kps {
			var p struct {
				Kind    string `json:"kind"`
				Zh      string `json:"zh"`
				Scene   string `json:"scene"`
				ReplyTo string `json:"replyTo"`
			}
			if err := json.Unmarshal([]byte(kp.Payload), &p); err != nil {
				t.Fatal(err)
			}
			if p.Kind != "phrase" || p.Zh == "" || p.Scene == "" {
				t.Fatalf("%s payload incomplete: %+v", kp.Code, p)
			}
			scenes++
			if p.ReplyTo != "" {
				replies++
			}
		}
	}
	if scenes != 32 {
		t.Fatalf("scenes = %d", scenes)
	}
	if replies != 4 {
		t.Fatalf("reply pairs = %d", replies)
	}
}
