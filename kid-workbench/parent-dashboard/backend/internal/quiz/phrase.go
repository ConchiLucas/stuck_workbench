package quiz

import "encoding/json"

type phrasePayload struct {
	Kind    string   `json:"kind"`
	Zh      string   `json:"zh"`
	Wrong   []string `json:"wrong"`
	Scene   string   `json:"scene"`
	ReplyTo string   `json:"replyTo"`
}

func phraseSpecs(kp Kp) []Spec {
	var p phrasePayload
	if err := json.Unmarshal([]byte(kp.Payload), &p); err != nil || p.Kind != "phrase" || p.Zh == "" {
		return nil
	}
	if len(p.Wrong) < optionCount-1 {
		return nil
	}
	pool := make([]string, 0, len(kp.Siblings))
	for _, s := range kp.Siblings {
		if s != kp.Title {
			pool = append(pool, s)
		}
	}
	if len(pool) < optionCount-1 {
		return nil
	}

	specs := make([]Spec, 0, 4)
	{
		opts, idx := labelOptions(p.Zh, [][]string{p.Wrong}, rngFor(kp.ID, 1))
		attachOptionKps(opts, kp.SiblingZhIDs, "")
		if kp.ID > 0 {
			opts[idx].KpID = kp.ID
		}
		specs = append(specs, Spec{
			Code: "listen_zh", Stem: "听一听，选出中文意思",
			Options: opts, AnswerIndex: idx,
			Speech: Speech{Text: kp.Title, Lang: LangEN},
		})
	}
	{
		opts, idx := labelOptions(kp.Title, [][]string{pool}, rngFor(kp.ID, 2))
		attachOptionKps(opts, kp.SiblingIDs, "")
		specs = append(specs, Spec{
			Code: "listen_en", Stem: "听一听，点出这句英语",
			Options: opts, AnswerIndex: idx,
			Speech: Speech{Text: kp.Title, Lang: LangEN},
		})
	}
	if p.Scene != "" {
		opts, idx := labelOptions(kp.Title, [][]string{pool}, rngFor(kp.ID, 3))
		attachOptionKps(opts, kp.SiblingIDs, "")
		specs = append(specs, Spec{
			Code: "scene", Stem: "这种时候该说哪一句？",
			Options: opts, AnswerIndex: idx,
			Visual: Visual{Kind: "scene", Text: p.Scene},
			Speech: Speech{Text: kp.Title, Lang: LangEN},
		})
	}
	if p.ReplyTo != "" {
		opts, idx := labelOptions(kp.Title, [][]string{pool}, rngFor(kp.ID, 4))
		attachOptionKps(opts, kp.SiblingIDs, "")
		specs = append(specs, Spec{
			Code: "reply", Stem: "对方说了这句话，你怎么答？",
			Options: opts, AnswerIndex: idx,
			Visual: Visual{Kind: "prompt", Text: p.ReplyTo},
			Speech: Speech{Text: p.ReplyTo, Lang: LangEN},
		})
	}
	return specs
}
