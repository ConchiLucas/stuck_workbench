package quiz

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"strings"

	"github.com/conchi/study-learning/chengyucontent"
)

const optionCount = 4

var Codes = []string{"meaning", "pick", "pinyin", "example"}

type Payload struct {
	Kind    string   `json:"kind"`
	Pinyin  string   `json:"pinyin"`
	Meaning string   `json:"meaning"`
	Example string   `json:"example"`
	Wrong   []string `json:"wrong"`
}

type Sibling struct {
	KpID  int64
	Title string
}

type Snapshot struct {
	Code, Type, Stem, Options, Answer, Visual, Speech string
}

func ParsePayload(raw string) (Payload, bool) {
	var p Payload
	if json.Unmarshal([]byte(raw), &p) != nil || p.Kind != "chengyu" {
		return Payload{}, false
	}
	p.Pinyin = strings.TrimSpace(p.Pinyin)
	p.Meaning = strings.TrimSpace(p.Meaning)
	p.Example = strings.TrimSpace(p.Example)
	return p, p.Meaning != ""
}

func IsCode(code string) bool {
	for _, item := range Codes {
		if item == code {
			return true
		}
	}
	return false
}

func Build(code, title string, p Payload, siblings []Sibling, seed, targetKpID int64) (Snapshot, bool) {
	if !IsCode(code) || strings.TrimSpace(title) == "" || p.Meaning == "" {
		return Snapshot{}, false
	}
	pool := make([]Sibling, 0, len(siblings))
	for _, item := range siblings {
		if item.Title != title && item.KpID > 0 {
			pool = append(pool, item)
		}
	}
	switch code {
	case "meaning":
		if len(p.Wrong) < optionCount-1 {
			return Snapshot{}, false
		}
		opts, idx := meaningOptions(p.Meaning, p.Wrong, seed)
		return snapshot("meaning", "这个成语是什么意思？", opts, idx, visual(map[string]any{"kind": "char", "text": title}), speech(title)), true
	case "pick":
		if len(pool) < optionCount-1 {
			return Snapshot{}, false
		}
		opts, idx := idiomOptions(Sibling{KpID: targetKpID, Title: title}, pool, seed+1)
		return snapshot("pick", "看意思，点出这个成语", opts, idx, visual(map[string]any{"kind": "meaning", "text": p.Meaning}), speech(title)), true
	case "pinyin":
		if p.Pinyin == "" || len(pool) < optionCount-1 {
			return Snapshot{}, false
		}
		opts, idx := idiomOptions(Sibling{KpID: targetKpID, Title: title}, pool, seed+2)
		return snapshot("pinyin", "", opts, idx, visual(map[string]any{"kind": "pinyin", "text": p.Pinyin}), speech(title)), true
	default:
		if p.Example == "" || len(pool) < optionCount-1 {
			return Snapshot{}, false
		}
		blank, err := chengyucontent.FirstBlank(p.Example, title)
		if err != nil {
			return Snapshot{}, false
		}
		opts, idx := idiomOptions(Sibling{KpID: targetKpID, Title: title}, pool, seed+3)
		return snapshot("example", "", opts, idx, visual(map[string]any{
			"kind": "example", "text": blank.Blanked, "full": blank.Full, "blanked": blank.Blanked,
			"target": blank.Target, "start": blank.Start, "length": blank.Length,
		}), speech(title)), true
	}
}

func snapshot(code, stem string, opts []map[string]any, idx int, visual, speech string) Snapshot {
	options, _ := json.Marshal(opts)
	answer, _ := json.Marshal(map[string]int{"index": idx})
	return Snapshot{Code: code, Type: "choice", Stem: stem, Options: string(options), Answer: string(answer), Visual: visual, Speech: speech}
}

func visual(v map[string]any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func speech(text string) string {
	raw, _ := json.Marshal(map[string]string{"text": text, "lang": "zh-CN"})
	return string(raw)
}

func meaningOptions(correct string, wrong []string, seed int64) ([]map[string]any, int) {
	rng := rand.New(rand.NewSource(seed))
	picked := make([]string, 0, optionCount-1)
	taken := map[string]bool{correct: true}
	shuffled := append([]string{}, wrong...)
	rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	for _, item := range shuffled {
		if len(picked) == optionCount-1 {
			break
		}
		if taken[item] || strings.TrimSpace(item) == "" {
			continue
		}
		taken[item] = true
		picked = append(picked, item)
	}
	opts := []map[string]any{{"id": "label:" + correct, "label": correct, "kpId": 0}}
	for _, item := range picked {
		opts = append(opts, map[string]any{"id": "label:" + item, "label": item, "kpId": 0})
	}
	rng.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
	idx := 0
	for i, opt := range opts {
		if opt["label"] == correct {
			idx = i
			break
		}
	}
	return opts, idx
}

func idiomOptions(correct Sibling, pool []Sibling, seed int64) ([]map[string]any, int) {
	rng := rand.New(rand.NewSource(seed))
	picked := make([]Sibling, 0, optionCount-1)
	taken := map[int64]bool{correct.KpID: true}
	takenTitle := map[string]bool{correct.Title: true}
	shuffled := append([]Sibling{}, pool...)
	rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	for _, item := range shuffled {
		if len(picked) == optionCount-1 {
			break
		}
		if item.KpID == 0 || taken[item.KpID] || takenTitle[item.Title] || strings.TrimSpace(item.Title) == "" {
			continue
		}
		taken[item.KpID] = true
		takenTitle[item.Title] = true
		picked = append(picked, item)
	}
	opts := []map[string]any{{"id": strconv.FormatInt(correct.KpID, 10), "label": correct.Title, "kpId": correct.KpID}}
	for _, item := range picked {
		opts = append(opts, map[string]any{"id": strconv.FormatInt(item.KpID, 10), "label": item.Title, "kpId": item.KpID})
	}
	rng.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
	idx := 0
	for i, opt := range opts {
		if opt["kpId"] == correct.KpID {
			idx = i
			break
		}
	}
	return opts, idx
}
