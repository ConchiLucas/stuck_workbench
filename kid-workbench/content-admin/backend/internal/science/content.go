package science

import (
	"encoding/json"
	"strings"
)

type Presentation struct {
	Summary     string `json:"summary"`
	Explanation string `json:"explanation"`
	FunFact     string `json:"funFact"`
}

func ParseContent(raw string) Presentation {
	var out Presentation
	if json.Unmarshal([]byte(raw), &out) != nil {
		return Presentation{}
	}
	out.Summary = strings.TrimSpace(out.Summary)
	out.Explanation = strings.TrimSpace(out.Explanation)
	out.FunFact = strings.TrimSpace(out.FunFact)
	return out
}

func (p Presentation) empty() bool {
	return p.Summary == "" && p.Explanation == "" && p.FunFact == ""
}
