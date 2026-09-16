package poemcontent

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

type GenerateOpts struct {
	Kind  string
	Work  Work
	All   []Work
	Seed  int64
	KpID  int64
}

func Generate(opts GenerateOpts) (PoemExample, error) {
	kind := KindForSkill(opts.Kind)
	if kind == "" {
		return PoemExample{}, fmt.Errorf("未知古诗题型")
	}
	w := NormalizeWork(opts.Work)
	if len(w.Lines) == 0 {
		return PoemExample{}, fmt.Errorf("作品没有诗句")
	}
	all := opts.All
	if len(all) == 0 {
		all = []Work{w}
	}
	var example PoemExample
	var err error
	switch kind {
	case KindTitle:
		example, err = generateTitle(w, all, opts.Seed)
	case KindFill:
		example, err = generateFill(w, opts.Seed)
	case KindCouplet:
		example, err = generateCouplet(w, all, opts.Seed)
	case KindRecite:
		example, err = generateRecite(w, opts.Seed)
	}
	if err != nil {
		return PoemExample{}, err
	}
	example.WorkID = w.WorkID
	example.WorkTitle = w.Title
	example.Author = w.Author
	example.Dynasty = w.Dynasty
	example.Edition = w.Edition
	if example.Edition == "" {
		example.Edition = Edition
	}
	attachLiveSpeech(&example, opts.KpID)
	example = ShuffleExample(example, opts.Seed)
	if err := Validate(example); err != nil {
		return PoemExample{}, err
	}
	return example, nil
}

func generateTitle(w Work, all []Work, seed int64) (PoemExample, error) {
	if !titleUnique(w, all) {
		return PoemExample{}, fmt.Errorf("作品 %s 的诗文无法单独区分同名或同文作品", w.Title)
	}
	line := distinguishingLine(w, all)
	if line.Text == "" {
		return PoemExample{}, fmt.Errorf("作品 %s 没有足以区分身份的诗句", w.Title)
	}
	opts := []Choice{{ID: w.WorkID, Label: w.Title}}
	seen := map[string]bool{w.WorkID: true, w.Title: true}
	for _, other := range all {
		other = NormalizeWork(other)
		if other.WorkID == w.WorkID || seen[other.WorkID] || seen[other.Title] {
			continue
		}
		seen[other.WorkID] = true
		seen[other.Title] = true
		opts = append(opts, Choice{ID: other.WorkID, Label: other.Title})
		if len(opts) == 4 {
			break
		}
	}
	if len(opts) < 4 {
		return PoemExample{}, fmt.Errorf("选诗名缺少足够的候选作品")
	}
	return PoemExample{
		Kind: KindTitle, Prompt: "这首诗叫什么？", Line: line.Text, LineID: line.ID,
		SourceLine: line.Text, Options: opts[:4], AnswerID: w.WorkID,
		Explanation: fmt.Sprintf("这是%s〔%s〕的《%s》。", w.Dynasty, w.Author, w.Title),
		Tip:          "先读诗句，再选作品名称。同名作品看具体篇目。",
	}, nil
}

func generateFill(w Work, seed int64) (PoemExample, error) {
	line, idx, ch, ok := pickGap(w, seed)
	if !ok {
		return PoemExample{}, fmt.Errorf("补字找不到合适的缺字位置")
	}
	answerID := fmt.Sprintf("char:%s#%d", string(ch), idx)
	opts := []Choice{{ID: answerID, Label: string(ch)}}
	seen := map[rune]bool{ch: true}
	for _, r := range distractorChars(w, ch, seed) {
		if seen[r] {
			continue
		}
		seen[r] = true
		opts = append(opts, Choice{ID: fmt.Sprintf("char:%s", string(r)), Label: string(r)})
		if len(opts) == 4 {
			break
		}
	}
	if len(opts) < 4 {
		return PoemExample{}, fmt.Errorf("补字缺少候选项")
	}
	display := DisplayFillLine(line.Text, []int{idx})
	return PoemExample{
		Kind: KindFill, Prompt: "缺的字是哪个？", Line: display, LineID: line.ID,
		SourceLine: line.Text, GapIndexes: []int{idx}, Options: opts, AnswerID: answerID,
		AudioMissingReason: "补字题不朗读缺字，以免直接说出答案。",
		Explanation:         fmt.Sprintf("原句是「%s」。", line.Text),
		Tip:                 "看□两边的字，想一想原诗。",
	}, nil
}

func generateCouplet(w Work, all []Work, seed int64) (PoemExample, error) {
	if len(w.Lines) < 2 {
		return PoemExample{}, fmt.Errorf("诗句不足，无法出下一句")
	}
	upperIdx := int(uint64(seed) % uint64(len(w.Lines)-1))
	upper := w.Lines[upperIdx]
	next := w.Lines[upperIdx+1]
	opts := []Choice{{ID: next.ID, Label: next.Text}}
	seen := map[string]bool{next.ID: true, next.Text: true, upper.Text: true}
	for _, other := range all {
		other = NormalizeWork(other)
		for _, line := range other.Lines {
			if seen[line.ID] || seen[line.Text] {
				continue
			}
			seen[line.ID] = true
			seen[line.Text] = true
			opts = append(opts, Choice{ID: line.ID, Label: line.Text})
			if len(opts) == 4 {
				break
			}
		}
		if len(opts) == 4 {
			break
		}
	}
	if len(opts) < 4 {
		return PoemExample{}, fmt.Errorf("选下一句缺少候选项")
	}
	return PoemExample{
		Kind: KindCouplet, Prompt: "下一句是哪一句？", Line: upper.Text, LineID: upper.ID,
		SourceLine: upper.Text, UpperLineID: upper.ID, NextLineID: next.ID,
		Options: opts, AnswerID: next.ID,
		Explanation: fmt.Sprintf("《%s》中，「%s」的下一句是「%s」。", w.Title, upper.Text, next.Text),
		Tip:         "下一句必须来自同一首诗的正确顺序。",
	}, nil
}

func generateRecite(w Work, seed int64) (PoemExample, error) {
	if len(w.Lines) < 2 {
		return PoemExample{}, fmt.Errorf("诗句不足，无法排顺序")
	}
	items := make([]Node, len(w.Lines))
	correct := make([]string, len(w.Lines))
	for i, line := range w.Lines {
		items[i] = Node{ID: line.ID, Label: line.Text}
		correct[i] = line.ID
	}
	display := ShuffleIDs(append([]string{}, correct...), seed)
	if sameOrder(display, correct) && len(display) > 1 {
		display[0], display[len(display)-1] = display[len(display)-1], display[0]
	}
	n := len(w.Lines)
	prompt := fmt.Sprintf("按顺序点出这%d句", n)
	return PoemExample{
		Kind: KindRecite, Prompt: prompt, Line: w.Lines[0].Text,
		SequenceItems: items, SequenceDisplayOrder: display, CorrectSequence: correct,
		AudioMissingReason: "排顺序不按正确句序朗读，以免泄露答案。",
		Explanation:         fmt.Sprintf("《%s》共 %d 句，按作品保存的行序排列。", w.Title, n),
		Tip:                 "点诗句填进空位。排错了也能看到你排出的顺序。",
	}, nil
}

func titleUnique(w Work, all []Work) bool {
	full := strings.Join(Texts(w), "")
	if distinguishingLine(w, all).Text == "" {
		return false
	}
	for _, other := range all {
		other = NormalizeWork(other)
		if other.WorkID == w.WorkID {
			continue
		}
		ot := strings.Join(Texts(other), "")
		if full == ot {
			return false
		}
		if len(full) < len(ot) && strings.HasPrefix(ot, full) {
			return false
		}
	}
	return true
}

func distinguishingLine(w Work, all []Work) Line {
	for _, line := range w.Lines {
		if lineTextUnique(line.Text, w.WorkID, all) {
			return line
		}
	}
	return Line{}
}

func lineTextUnique(text, workID string, all []Work) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	for _, other := range all {
		other = NormalizeWork(other)
		if other.WorkID == workID {
			continue
		}
		for _, line := range other.Lines {
			if line.Text == text {
				return false
			}
		}
	}
	return true
}

func pickGap(w Work, seed int64) (Line, int, rune, bool) {
	type cand struct {
		line Line
		idx  int
		ch   rune
	}
	var cands []cand
	for _, line := range w.Lines {
		runes := []rune(line.Text)
		for i, r := range runes {
			if IsPunct(r) || !unicode.Is(unicode.Han, r) {
				continue
			}
			cands = append(cands, cand{line, i, r})
		}
	}
	if len(cands) == 0 {
		return Line{}, 0, 0, false
	}
	c := cands[int(uint64(seed)%uint64(len(cands)))]
	return c.line, c.idx, c.ch, true
}

func distractorChars(w Work, answer rune, seed int64) []rune {
	pool := []rune{'天', '地', '人', '山', '水', '月', '花', '风', '日', '云', '春', '秋', '江', '雪', '鸟', '草'}
	out := make([]rune, 0, 8)
	for _, r := range pool {
		if r != answer {
			out = append(out, r)
		}
	}
	for _, line := range w.Lines {
		for _, r := range line.Text {
			if r != answer && unicode.Is(unicode.Han, r) && !IsPunct(r) {
				out = append(out, r)
			}
		}
	}
	return ShuffleRunes(out, seed)
}

func attachLiveSpeech(e *PoemExample, kpID int64) {
	if kpID <= 0 || !NeedsSpeech(e.Kind) {
		return
	}
	ord := LineOrd(e.LineID)
	e.SpeechURL = fmt.Sprintf("/api/v1/poem/items/%d/speech/%d.wav", kpID, ord)
	e.SpeechText = e.SourceLine
	if e.SpeechText == "" {
		e.SpeechText = e.Line
	}
}

func ShuffleExample(e PoemExample, seed int64) PoemExample {
	switch KindForSkill(e.Kind) {
	case KindTitle, KindFill, KindCouplet:
		ids := make([]string, len(e.Options))
		for i, o := range e.Options {
			ids[i] = o.ID
		}
		order := ShuffleIDs(ids, seed)
		if len(order) > 1 && order[0] == e.AnswerID {
			order[0], order[1] = order[1], order[0]
		}
		e.Options = reorderChoices(e.Options, order)
		e.OptionOrder = order
	case KindRecite:
		if len(e.SequenceDisplayOrder) == 0 {
			ids := make([]string, len(e.SequenceItems))
			for i, n := range e.SequenceItems {
				ids[i] = n.ID
			}
			e.SequenceDisplayOrder = ShuffleIDs(ids, seed)
		}
		e.SequenceItems = reorderNodes(e.SequenceItems, e.SequenceDisplayOrder)
	}
	return e
}

func ShuffleIDs(ids []string, seed int64) []string {
	out := append([]string{}, ids...)
	x := uint64(seed)
	if x == 0 {
		x = 1
	}
	for i := len(out) - 1; i > 0; i-- {
		x = x*6364136223846793005 + 1
		j := int(x % uint64(i+1))
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func ShuffleRunes(ids []rune, seed int64) []rune {
	out := append([]rune{}, ids...)
	x := uint64(seed)
	if x == 0 {
		x = 1
	}
	for i := len(out) - 1; i > 0; i-- {
		x = x*6364136223846793005 + 1
		j := int(x % uint64(i+1))
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func reorderChoices(opts []Choice, order []string) []Choice {
	byID := map[string]Choice{}
	for _, o := range opts {
		byID[o.ID] = o
	}
	out := make([]Choice, 0, len(order))
	for _, id := range order {
		if o, ok := byID[id]; ok {
			out = append(out, o)
		}
	}
	return out
}

func reorderNodes(nodes []Node, order []string) []Node {
	byID := map[string]Node{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	out := make([]Node, 0, len(order))
	for _, id := range order {
		if n, ok := byID[id]; ok {
			out = append(out, n)
		}
	}
	return out
}

func sameOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func StoredOptions(e PoemExample) string {
	switch KindForSkill(e.Kind) {
	case KindRecite:
		raw, _ := json.Marshal(map[string]any{"items": e.SequenceItems, "displayOrder": e.SequenceDisplayOrder})
		return string(raw)
	default:
		raw, _ := json.Marshal(e.Options)
		return string(raw)
	}
}

func StoredAnswer(e PoemExample) string {
	switch KindForSkill(e.Kind) {
	case KindRecite:
		raw, _ := json.Marshal(map[string]any{"order": e.CorrectSequence})
		return string(raw)
	default:
		raw, _ := json.Marshal(map[string]any{"id": e.AnswerID})
		return string(raw)
	}
}

func StoredVisual(e PoemExample) string {
	raw, _ := json.Marshal(map[string]any{
		"kind": "poem-line", "text": e.Line, "lineId": e.LineID, "sourceLine": e.SourceLine,
		"gapIndexes": e.GapIndexes, "workId": e.WorkID, "upperLineId": e.UpperLineID, "nextLineId": e.NextLineID,
	})
	return string(raw)
}
