package logic

import (
	"encoding/json"
	"strings"

	"github.com/conchi/study-learning/logiccontent"
	"gorm.io/gorm"
)

type Payload struct {
	Kind   string   `json:"kind"`
	Seq    []string `json:"seq"`
	A      string   `json:"a"`
	Wrong  []string `json:"wrong"`
	Prompt string   `json:"prompt"`
	Speech string   `json:"speech"`
}

func ParsePayload(raw string) Payload {
	var p Payload
	_ = json.Unmarshal([]byte(raw), &p)
	p.Kind = strings.TrimSpace(p.Kind)
	p.A = strings.TrimSpace(p.A)
	p.Prompt = strings.TrimSpace(p.Prompt)
	p.Speech = strings.TrimSpace(p.Speech)

	seq := make([]string, 0, len(p.Seq))
	for _, item := range p.Seq {
		item = strings.TrimSpace(item)
		if item != "" {
			seq = append(seq, item)
		}
	}
	p.Seq = seq

	wrong := make([]string, 0, len(p.Wrong))
	for _, item := range p.Wrong {
		item = strings.TrimSpace(item)
		if item != "" {
			wrong = append(wrong, item)
		}
	}
	p.Wrong = wrong

	if p.Prompt == "" {
		p.Prompt = defaultPrompt(p.Kind)
	}
	if p.Speech == "" {
		p.Speech = p.Prompt
	}
	return p
}

func defaultPrompt(kind string) string {
	switch kind {
	case "pattern":
		return "下一个是哪个？"
	case "classify":
		return "哪个和其他不一样？"
	case "order":
		return "正确的顺序是？"
	case "diff":
		return "找出不一样的那个"
	case "compare":
		return "哪个更大？"
	default:
		return "选一选"
	}
}

type ItemDTO struct {
	KpID        int64                      `json:"kpId"`
	Title       string                      `json:"title"`
	Kind        string                      `json:"kind"`
	Seq         []string                   `json:"seq"`
	Answer      string                     `json:"answer"`
	Wrong       []string                   `json:"wrong"`
	Prompt      string                     `json:"prompt"`
	Speech      string                     `json:"speech"`
	Rule        string                     `json:"rule,omitempty"`
	Example     *logiccontent.LogicExample `json:"example,omitempty"`
	GlyphURLs   map[string]string           `json:"glyphUrls,omitempty"`
	Difficulty  int                        `json:"difficulty"`
	ModuleCode  string                      `json:"moduleCode"`
	ModuleName  string                     `json:"moduleName"`
	ModuleOrder int                        `json:"moduleOrder"`
	KpOrder     int                        `json:"kpOrder"`
}

type GroupDTO struct {
	ModuleCode  string    `json:"moduleCode"`
	ModuleName  string    `json:"moduleName"`
	ModuleOrder int       `json:"moduleOrder"`
	Items       []ItemDTO `json:"items"`
}

type ListResult struct {
	View   string     `json:"view"`
	Total  int        `json:"total"`
	Groups []GroupDTO `json:"groups,omitempty"`
	Items  []ItemDTO  `json:"items,omitempty"`
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type kpRow struct {
	ID          int64
	Title       string
	Payload     string
	Difficulty  int
	KpOrder     int
	ModuleCode  string
	ModuleName  string
	ModuleOrder int
}

func (s *Service) List(view string) (ListResult, error) {
	if view == "" {
		view = "groups"
	}
	var rows []kpRow
	err := s.db.Raw(`
		SELECT kp.id AS id, kp.title AS title, kp.payload AS payload, kp.difficulty AS difficulty,
		       kp.order_no AS kp_order, m.code AS module_code, m.name AS module_name, m.order_no AS module_order
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = ?
		ORDER BY m.order_no, kp.order_no, kp.id
	`, "logic").Scan(&rows).Error
	if err != nil {
		return ListResult{}, err
	}

	dtos := make([]ItemDTO, 0, len(rows))
	for _, row := range rows {
		content := ParsePayload(row.Payload)
		diff := row.Difficulty
		if diff < 1 {
			diff = 1
		}
		dto := ItemDTO{
			KpID:        row.ID,
			Title:       row.Title,
			Kind:        content.Kind,
			Seq:         content.Seq,
			Answer:      content.A,
			Wrong:       content.Wrong,
			Prompt:      content.Prompt,
			Speech:      content.Speech,
			Difficulty:  diff,
			ModuleCode:  row.ModuleCode,
			ModuleName:  row.ModuleName,
			ModuleOrder: row.ModuleOrder,
			KpOrder:     row.KpOrder,
		}
		if example, err := logiccontent.ExampleFromPayload(row.Payload, "", "", row.ID); err == nil {
			dto.Kind = example.Kind
			dto.Prompt = example.Prompt
			dto.Rule = example.Rule.Explain
			dto.Example = &example
			dto.GlyphURLs = example.ImageURLs
			dto.Seq = captionsFrom(example, example.Sequence)
			if example.Kind == "order" {
				dto.Seq = captionsFrom(example, example.CorrectSequence)
				dto.Answer = strings.Join(captionsFrom(example, example.CorrectSequence), " → ")
			} else if o, ok := logiccontent.ObjectByID(example.Objects, example.AnswerID); ok {
				dto.Answer = o.Caption
				wrong := []string{}
				for _, id := range example.Options {
					if id != example.AnswerID {
						if w, ok := logiccontent.ObjectByID(example.Objects, id); ok {
							wrong = append(wrong, w.Caption)
						}
					}
				}
				dto.Wrong = wrong
			}
		}
		dtos = append(dtos, dto)
	}

	out := ListResult{View: view, Total: len(dtos)}
	if view == "table" {
		out.Items = dtos
		return out, nil
	}
	byMod := map[string]*GroupDTO{}
	order := []string{}
	for _, dto := range dtos {
		g, ok := byMod[dto.ModuleCode]
		if !ok {
			g = &GroupDTO{
				ModuleCode:  dto.ModuleCode,
				ModuleName:  dto.ModuleName,
				ModuleOrder: dto.ModuleOrder,
				Items:       []ItemDTO{},
			}
			byMod[dto.ModuleCode] = g
			order = append(order, dto.ModuleCode)
		}
		g.Items = append(g.Items, dto)
	}
	out.Groups = make([]GroupDTO, 0, len(order))
	for _, code := range order {
		out.Groups = append(out.Groups, *byMod[code])
	}
	return out, nil
}

func captionsFrom(e logiccontent.LogicExample, ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if o, ok := logiccontent.ObjectByID(e.Objects, id); ok {
			out = append(out, o.Caption)
		}
	}
	return out
}

func (s *Service) Glyph(kpID int64, objectID string) ([]byte, string, error) {
	return logiccontent.LiveGlyph(s.db, kpID, objectID)
}
