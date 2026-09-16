package quiz

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/conchi/study-learning/logiccontent"
	"gorm.io/gorm"
)

var (
	ErrInvalidType = errors.New("无效的逻辑题型")
	ErrNoMaterial  = errors.New("没有足够的逻辑题目生成练习")
)

var modules = map[string]string{
	"pattern":      "pattern",
	"classify":     "classify",
	"order":        "order",
	"shape_reason": "shape_reason",
	"diff":         "diff",
	"compare":      "compare",
}

type Visual struct {
	Kind  string   `json:"kind,omitempty"`
	Items []string `json:"items,omitempty"`
}

type Option struct {
	ID    int64  `json:"id"`
	Label string `json:"label,omitempty"`
	Emoji string `json:"emoji,omitempty"`
}

func (o Option) Display() string {
	if strings.TrimSpace(o.Emoji) != "" {
		return o.Emoji
	}
	return o.Label
}

type Question struct {
	InstanceID  string                      `json:"instanceId"`
	Type        string                       `json:"type"`
	Stem        string                       `json:"stem"`
	TargetID    int64                       `json:"targetId"`
	Visual      Visual                       `json:"visual"`
	Options     []Option                    `json:"options"`
	AnswerIndex int                         `json:"answerIndex"`
	Example     *logiccontent.LogicExample   `json:"example,omitempty"`
}

type storedRow struct {
	QuestionID int64
	KpID       int64
	Code       string
	Stem       string
	Options    string
	Answer     string
	Visual     string
	Payload    string
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Generate(ctx context.Context, quizType string, excluded []int64) (Question, error) {
	quizType = strings.TrimSpace(quizType)
	module, ok := modules[quizType]
	if !ok {
		return Question{}, ErrInvalidType
	}
	rows, err := s.loadStored(ctx, module)
	if err != nil {
		return Question{}, err
	}
	candidates := make([]storedRow, 0, len(rows))
	for _, row := range rows {
		if contains(excluded, row.KpID) {
			continue
		}
		candidates = append(candidates, row)
	}
	if len(candidates) == 0 {
		return Question{}, ErrNoMaterial
	}
	row := candidates[randIndex(len(candidates))]
	if example, err := logiccontent.ExampleFromPayload(row.Payload, row.Visual, "", row.KpID); err == nil {
		example = logiccontent.ShuffleExample(example, row.QuestionID)
		options := make([]Option, 0, len(example.Options))
		answerIndex := 0
		ids := example.Options
		if example.Kind == "order" {
			ids = example.DisplayOrder
		}
		for i, id := range ids {
			o, _ := logiccontent.ObjectByID(example.Objects, id)
			options = append(options, Option{ID: int64(i), Label: o.Caption})
			if id == example.AnswerID {
				answerIndex = i
			}
		}
		visual := Visual{Kind: "seq", Items: captionsOf(example, example.Sequence)}
		return Question{
			InstanceID: newID(), Type: quizType, Stem: example.Prompt, TargetID: row.KpID,
			Visual: visual, Options: options, AnswerIndex: answerIndex, Example: &example,
		}, nil
	}
	var raw []struct {
		Label string `json:"label"`
		Emoji string `json:"emoji"`
	}
	if err := json.Unmarshal([]byte(row.Options), &raw); err != nil || len(raw) < 2 {
		return Question{}, ErrNoMaterial
	}
	var answer struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal([]byte(row.Answer), &answer); err != nil || answer.Index < 0 || answer.Index >= len(raw) {
		return Question{}, ErrNoMaterial
	}
	var visual Visual
	_ = json.Unmarshal([]byte(row.Visual), &visual)
	options := make([]Option, len(raw))
	for i, item := range raw {
		options[i] = Option{ID: int64(i), Label: item.Label, Emoji: item.Emoji}
		if options[i].Display() == "" {
			return Question{}, ErrNoMaterial
		}
	}
	return Question{
		InstanceID: newID(), Type: quizType, Stem: row.Stem, TargetID: row.KpID,
		Visual: visual, Options: options, AnswerIndex: answer.Index,
	}, nil
}

func (s *Service) loadStored(ctx context.Context, module string) ([]storedRow, error) {
	var rows []storedRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT q.id AS question_id, q.kp_id, q.code, q.stem, q.options, q.answer, q.visual, COALESCE(kp.payload,'') AS payload
		FROM questions q
		JOIN knowledge_points kp ON kp.id = q.kp_id
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects sub ON sub.id = m.subject_id
		WHERE sub.code = 'logic' AND ((m.code = ? AND q.code = 'pick1') OR q.code = ?)
		ORDER BY CASE WHEN q.code = ? THEN 0 ELSE 1 END, q.id`, module, module, module).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	structured := make([]storedRow, 0, len(rows))
	legacy := make([]storedRow, 0, len(rows))
	for _, row := range rows {
		if row.Code == module {
			structured = append(structured, row)
		} else {
			legacy = append(legacy, row)
		}
	}
	if len(structured) > 0 {
		return structured, nil
	}
	return legacy, nil
}

func captionsOf(e logiccontent.LogicExample, ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if o, ok := logiccontent.ObjectByID(e.Objects, id); ok {
			out = append(out, o.Caption)
		}
	}
	return out
}

func contains(ids []int64, id int64) bool {
	for _, item := range ids {
		if item == id {
			return true
		}
	}
	return false
}

func randIndex(n int) int {
	if n <= 1 {
		return 0
	}
	v, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(v.Int64())
}

func newID() string {
	buf := make([]byte, 12)
	_, _ = cryptorand.Read(buf)
	return hex.EncodeToString(buf)
}
