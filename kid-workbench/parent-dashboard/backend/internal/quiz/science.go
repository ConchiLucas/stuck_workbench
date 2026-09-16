package quiz

import "encoding/json"

type factPayload struct {
	Kind  string   `json:"kind"`
	Q     string   `json:"q"`
	A     string   `json:"a"`
	Wrong []string `json:"wrong"`
	Emoji string   `json:"emoji"`
}

func scienceSpecs(kp Kp) []Spec {
	var p factPayload
	if err := json.Unmarshal([]byte(kp.Payload), &p); err != nil || p.Kind != "fact" || p.Q == "" || p.A == "" {
		return nil
	}
	if len(p.Wrong) < optionCount-1 {
		return nil
	}

	opts, idx := labelOptions(p.A, [][]string{p.Wrong}, rngFor(kp.ID, 1))
	sp := Spec{
		Code: "recognize", Stem: p.Q,
		Options: opts, AnswerIndex: idx,
		Speech: Speech{Text: p.Q, Lang: LangZH},
	}
	if p.Emoji != "" {
		sp.Visual = Visual{Kind: "emoji", Emoji: p.Emoji}
	}
	return []Spec{sp}
}
