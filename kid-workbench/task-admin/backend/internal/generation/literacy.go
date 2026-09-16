package generation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/conchi/study-learning/literacycontract"
	"math/rand"
	"sort"
	"strings"
)

type MediaRef struct {
	RevisionID string `json:"revisionId"`
	Kind       string `json:"kind"`
	SHA256     string `json:"sha256"`
}
type Capability struct {
	Ready   bool     `json:"ready"`
	Reasons []string `json:"reasons"`
}
type Material struct {
	WritingTemplate *MediaRef             `json:"writingTemplate,omitempty"`
	QuestionTypes   []string              `json:"-"`
	KpID            int64                 `json:"kpId"`
	Text            string                `json:"text"`
	ModuleCode      string                `json:"moduleCode"`
	ModuleName      string                `json:"moduleName"`
	RevisionID      string                `json:"revisionId"`
	SourceRevision  string                `json:"sourceRevision"`
	Glyph           *MediaRef             `json:"glyph,omitempty"`
	Sense           *MediaRef             `json:"sense,omitempty"`
	Speech          *MediaRef             `json:"speech,omitempty"`
	Capabilities    map[string]Capability `json:"capabilities,omitempty"`
}
type Scope struct {
	ModuleCodes []string `json:"moduleCodes"`
	KpIDs       []int64  `json:"kpIds"`
}
type Spec struct {
	SubjectCode     string         `json:"subjectCode"`
	Kind            string         `json:"kind"`
	Scope           Scope          `json:"scope"`
	TargetCount     int            `json:"targetCount"`
	TypeCounts      map[string]int `json:"typeCounts"`
	DistractorScope string         `json:"distractorScope"`
}
type Option struct {
	ID    string    `json:"id"`
	KpID  int64     `json:"kpId"`
	Text  string    `json:"text"`
	Image *MediaRef `json:"image,omitempty"`
	Audio *MediaRef `json:"audio,omitempty"`
}
type Stem struct {
	Text  string    `json:"text,omitempty"`
	Image *MediaRef `json:"image,omitempty"`
	Audio *MediaRef `json:"audio,omitempty"`
}
type Snapshot struct {
	Interaction             string          `json:"interaction,omitempty"`
	ResponseSchemaVersion   int             `json:"responseSchemaVersion,omitempty"`
	EvaluationPolicyVersion string          `json:"evaluationPolicyVersion,omitempty"`
	Presentation            map[string]bool `json:"presentation,omitempty"`
	WritingTemplate         *MediaRef       `json:"writingTemplate,omitempty"`
	SourceQuestionID        int64           `json:"sourceQuestionId,omitempty"`
	SchemaVersion           int             `json:"schemaVersion"`
	SubjectCode             string          `json:"subjectCode"`
	KpID                    int64           `json:"kpId"`
	TargetText              string          `json:"targetText"`
	QuestionType            string          `json:"questionType"`
	SkillCode               string          `json:"skillCode"`
	TemplateVersion         string          `json:"templateVersion"`
	Prompt                  string          `json:"prompt"`
	Stem                    Stem            `json:"stem"`
	Options                 []Option        `json:"options"`
	AnswerOptionID          string          `json:"answerOptionId"`
	Explanation             string          `json:"explanation"`
	MaterialRevisionIDs     []string        `json:"materialRevisionIds"`
}

func Normalize(s Spec) (Spec, error) {
	if s.SubjectCode == "" {
		s.SubjectCode = "literacy"
	}
	if s.Kind == "" {
		s.Kind = "practice"
	}
	if s.TargetCount == 0 {
		s.TargetCount = 10
	}
	if s.DistractorScope == "" {
		s.DistractorScope = "module"
	}
	if s.SubjectCode != "literacy" || (s.Kind != "practice" && s.Kind != "review") {
		return s, fmt.Errorf("仅支持识字普通或复习任务")
	}
	if len(s.Scope.ModuleCodes) != 1 || strings.TrimSpace(s.Scope.ModuleCodes[0]) == "" {
		return s, fmt.Errorf("请选择一个识字组")
	}
	if s.TargetCount < 1 || s.TargetCount > 20 {
		return s, fmt.Errorf("题量必须为 1–20")
	}
	if s.DistractorScope != "module" && s.DistractorScope != "subject" {
		return s, fmt.Errorf("无效的干扰项范围")
	}
	if len(s.TypeCounts) == 0 {
		s.TypeCounts = map[string]int{"glyph_sense": (s.TargetCount + 1) / 2}
		if s.TargetCount > 1 {
			s.TypeCounts["sense_char"] = s.TargetCount / 2
		}
	}
	n := 0
	for k, v := range s.TypeCounts {
		if !literacycontract.Supports(k) || v < 1 {
			return s, fmt.Errorf("题型或数量无效")
		}
		n += v
	}
	if n != s.TargetCount {
		return s, fmt.Errorf("题型数量合计必须等于 %d", s.TargetCount)
	}
	ids := map[int64]bool{}
	for _, id := range s.Scope.KpIDs {
		if id <= 0 || ids[id] {
			return s, fmt.Errorf("知识点编号无效或重复")
		}
		ids[id] = true
	}
	return s, nil
}
func Generate(in Spec, materials []Material, seed int64) ([]Snapshot, error) {
	s, err := Normalize(in)
	if err != nil {
		return nil, err
	}
	all := append([]Material(nil), materials...)
	sort.Slice(all, func(i, j int) bool { return all[i].KpID < all[j].KpID })
	selected := map[int64]bool{}
	for _, id := range s.Scope.KpIDs {
		selected[id] = true
	}
	targets := []Material{}
	pool := []Material{}
	seen := map[int64]bool{}
	for _, m := range all {
		if seen[m.KpID] {
			return nil, fmt.Errorf("素材重复")
		}
		seen[m.KpID] = true
		if m.RevisionID == "" || m.Text == "" {
			continue
		}
		if s.DistractorScope == "subject" || m.ModuleCode == s.Scope.ModuleCodes[0] {
			pool = append(pool, m)
		}
		if m.ModuleCode == s.Scope.ModuleCodes[0] && (len(selected) == 0 || selected[m.KpID]) {
			targets = append(targets, m)
		}
	}
	if len(selected) > 0 && len(targets) != len(selected) {
		return nil, fmt.Errorf("所选字不属于当前组或素材不可用")
	}
	r := rand.New(rand.NewSource(seed))
	result := []Snapshot{}
	for _, def := range literacycontract.Registry().Types {
		typ := def.Code
		count := s.TypeCounts[typ]
		if count == 0 {
			continue
		}
		eligible := []Material{}
		for _, m := range targets {
			if ReadyFor(m, typ) {
				eligible = append(eligible, m)
			}
		}
		if count > len(eligible) {
			return nil, fmt.Errorf("%s 可出题仅 %d 道，需要 %d 道", def.Label, len(eligible), count)
		}
		r.Shuffle(len(eligible), func(i, j int) { eligible[i], eligible[j] = eligible[j], eligible[i] })
		for i := 0; i < count; i++ {
			q, e := Build(eligible[i], typ, pool, r)
			if e != nil {
				return nil, e
			}
			result = append(result, q)
		}
	}
	r.Shuffle(len(result), func(i, j int) { result[i], result[j] = result[j], result[i] })
	return result, nil
}
func ReadyFor(m Material, typ string) bool {
	if m.Speech == nil || m.RevisionID == "" || m.Text == "" {
		return false
	}
	if typ == "write_char" {
		return m.WritingTemplate != nil
	}
	return m.Sense != nil && m.Glyph != nil
}
func Build(target Material, typ string, pool []Material, r *rand.Rand) (Snapshot, error) {
	if typ == "write_char" {
		if !ReadyFor(target, typ) {
			return Snapshot{}, fmt.Errorf("听写素材缺少音频或书写模板")
		}
		q := Snapshot{SchemaVersion: 2, SubjectCode: "literacy", KpID: target.KpID, TargetText: target.Text, QuestionType: typ, SkillCode: typ, TemplateVersion: "literacy-writing-v1", Interaction: "handwriting", ResponseSchemaVersion: 2, EvaluationPolicyVersion: "ink-match-v1", Presentation: map[string]bool{"showTarget": false}, Prompt: "听读音，在田字格里写出汉字", Stem: Stem{Audio: target.Speech}, WritingTemplate: target.WritingTemplate, MaterialRevisionIDs: []string{target.RevisionID}}
		return q, ValidateSnapshot(q)
	}

	q := Snapshot{SchemaVersion: 2, Interaction: "choice", ResponseSchemaVersion: 2, SubjectCode: "literacy", KpID: target.KpID, TargetText: target.Text, QuestionType: typ, SkillCode: typ, TemplateVersion: "literacy-choice-v1", AnswerOptionID: fmt.Sprintf("kp:%d", target.KpID), Explanation: fmt.Sprintf("这道题对应的汉字是『%s』", target.Text)}
	if typ == "glyph_sense" {
		q.Prompt = "看字，选出对应的图片"
		q.Stem = Stem{Text: target.Text, Image: target.Glyph}
	} else if typ == "sense_char" {
		q.Prompt = "看图，选出对应的汉字"
		q.Stem = Stem{Image: target.Sense, Audio: target.Speech}
	} else {
		return q, fmt.Errorf("未知题型")
	}
	candidates := append([]Material(nil), pool...)
	r.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
	picked := []Material{target}
	texts := map[string]bool{target.Text: true}
	images := map[string]bool{}
	if target.Sense != nil {
		images[target.Sense.SHA256] = true
	}
	for _, m := range candidates {
		if len(picked) == 4 {
			break
		}
		if m.KpID == target.KpID || texts[m.Text] || m.Speech == nil || m.Sense == nil || m.Glyph == nil || m.RevisionID == "" || (typ == "glyph_sense" && images[m.Sense.SHA256]) {
			continue
		}
		picked = append(picked, m)
		texts[m.Text] = true
		images[m.Sense.SHA256] = true
	}
	if len(picked) != 4 {
		return q, fmt.Errorf("『%s』缺少 3 个不重复的可用干扰项", target.Text)
	}
	for _, m := range picked {
		o := Option{ID: fmt.Sprintf("kp:%d", m.KpID), KpID: m.KpID, Text: m.Text, Audio: m.Speech}
		if typ == "glyph_sense" {
			o.Image = m.Sense
		} else {
			o.Image = m.Glyph
		}
		q.Options = append(q.Options, o)
		q.MaterialRevisionIDs = append(q.MaterialRevisionIDs, m.RevisionID)
	}
	sort.Strings(q.MaterialRevisionIDs)
	r.Shuffle(len(q.Options), func(i, j int) { q.Options[i], q.Options[j] = q.Options[j], q.Options[i] })
	return q, ValidateSnapshot(q)
}
func ValidateSnapshot(q Snapshot) error {
	if q.QuestionType == "write_char" {
		if q.SchemaVersion != 2 || q.SubjectCode != "literacy" || q.KpID <= 0 || q.TargetText == "" || q.SkillCode != "write_char" || q.Interaction != "handwriting" || q.ResponseSchemaVersion != 2 || q.EvaluationPolicyVersion != "ink-match-v1" || q.WritingTemplate == nil || q.WritingTemplate.Kind != "writing_template" || q.WritingTemplate.SHA256 == "" || q.WritingTemplate.RevisionID == "" || q.Stem.Audio == nil || q.Stem.Audio.Kind != "speech" || q.Stem.Text != "" || q.Stem.Image != nil || len(q.Options) != 0 || q.AnswerOptionID != "" {
			return fmt.Errorf("听写快照无效")
		}
		validHash := func(v string) bool {
			if len(v) != 64 {
				return false
			}
			_, e := hex.DecodeString(v)
			return e == nil
		}
		if !validHash(q.WritingTemplate.RevisionID) || !validHash(q.WritingTemplate.SHA256) || !validHash(q.Stem.Audio.RevisionID) || !validHash(q.Stem.Audio.SHA256) {
			return fmt.Errorf("听写素材引用缺少有效摘要")
		}
		if q.WritingTemplate.RevisionID != q.Stem.Audio.RevisionID || len(q.MaterialRevisionIDs) != 1 || q.MaterialRevisionIDs[0] != q.WritingTemplate.RevisionID {
			return fmt.Errorf("听写素材修订不一致")
		}
		return nil
	}

	if (q.SchemaVersion != 1 && q.SchemaVersion != 2) || q.SubjectCode != "literacy" || q.KpID <= 0 || q.TargetText == "" || len(q.Options) != 4 || q.SkillCode != q.QuestionType || (q.QuestionType != "glyph_sense" && q.QuestionType != "sense_char") {
		return fmt.Errorf("题目快照无效")
	}
	if q.SchemaVersion == 2 && (q.Interaction != "choice" || q.ResponseSchemaVersion != 2 || q.Stem.Image == nil) {
		return fmt.Errorf("选择题缺少素材字图或响应协议")
	}
	if q.QuestionType == "sense_char" && q.Stem.Image == nil {
		return fmt.Errorf("题干缺少义图")
	}
	seen := map[string]bool{}
	texts := map[string]bool{}
	images := map[string]bool{}
	answers := 0
	for _, o := range q.Options {
		if o.ID == "" || seen[o.ID] || o.Text == "" || texts[o.Text] || o.Audio == nil {
			return fmt.Errorf("题目选项重复或缺少音频")
		}
		seen[o.ID] = true
		texts[o.Text] = true
		if q.SchemaVersion == 2 && o.Image == nil {
			return fmt.Errorf("选择项缺少素材图片")
		}
		if q.QuestionType == "glyph_sense" {
			if o.Image == nil || o.Image.SHA256 == "" || images[o.Image.SHA256] {
				return fmt.Errorf("选项义图缺失或重复")
			}
			images[o.Image.SHA256] = true
		}
		if o.ID == q.AnswerOptionID {
			answers++
			if o.KpID != q.KpID {
				return fmt.Errorf("答案知识点不匹配")
			}
		}
	}
	if answers != 1 {
		return fmt.Errorf("正确答案必须唯一")
	}
	return nil
}
func Fingerprint(q Snapshot) string {
	q.SourceQuestionID = 0
	q.Options = append([]Option(nil), q.Options...)
	sort.Slice(q.Options, func(i, j int) bool { return q.Options[i].ID < q.Options[j].ID })
	data, _ := json.Marshal(q)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
