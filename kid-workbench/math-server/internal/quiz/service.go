package quiz

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrInvalidType = errors.New("无效的算数题型")
	ErrNoMaterial  = errors.New("没有足够的算数题目生成练习")
)

type Visual struct {
	Kind  string `json:"kind"`
	Text  string `json:"text,omitempty"`
	A     int    `json:"a,omitempty"`
	B     int    `json:"b,omitempty"`
	Emoji string `json:"emoji,omitempty"`
}

type Option struct {
	ID    int64  `json:"id"`
	Label string `json:"label,omitempty"`
}

type Question struct {
	InstanceID  string   `json:"instanceId"`
	Type        string   `json:"type"`
	Stem        string   `json:"stem"`
	TargetID    int64    `json:"targetId"`
	Visual      Visual   `json:"visual"`
	Options     []Option `json:"options"`
	AnswerIndex int      `json:"answerIndex"`
}

type storedRow struct {
	QuestionID int64
	KpID       int64
	Code       string
	Stem       string
	Options    string
	Answer     string
	Visual     string
}

type kpRow struct {
	KpID    int64
	Payload string
	Module  string
}

type payload struct {
	Kind  string `json:"kind"`
	A     int    `json:"a"`
	B     int    `json:"b"`
	Emoji string `json:"emoji"`
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Generate(ctx context.Context, quizType string, excluded []int64) (Question, error) {
	quizType = strings.TrimSpace(quizType)
	switch quizType {
	case "equation":
		return s.fromStored(ctx, quizType, "calc", excluded)
	case "story":
		return s.fromStored(ctx, quizType, "story", excluded)
	case "shape":
		return s.fromStored(ctx, quizType, "name", excluded)
	case "missing":
		return s.generateMissing(ctx, excluded)
	case "judge":
		return s.generateJudge(ctx, excluded)
	default:
		return Question{}, ErrInvalidType
	}
}

func (s *Service) fromStored(ctx context.Context, quizType, code string, excluded []int64) (Question, error) {
	rows, err := s.loadStored(ctx, code)
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
	var labels []struct {
		Label string `json:"label"`
	}
	if err := json.Unmarshal([]byte(row.Options), &labels); err != nil || len(labels) < 2 {
		return Question{}, ErrNoMaterial
	}
	var answer struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal([]byte(row.Answer), &answer); err != nil || answer.Index < 0 || answer.Index >= len(labels) {
		return Question{}, ErrNoMaterial
	}
	var visual Visual
	_ = json.Unmarshal([]byte(row.Visual), &visual)
	options := make([]Option, len(labels))
	for i, item := range labels {
		options[i] = Option{ID: int64(i), Label: item.Label}
	}
	return Question{
		InstanceID: newID(), Type: quizType, Stem: row.Stem, TargetID: row.KpID,
		Visual: visual, Options: options, AnswerIndex: answer.Index,
	}, nil
}

func (s *Service) loadStored(ctx context.Context, code string) ([]storedRow, error) {
	var rows []storedRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT q.id AS question_id, q.kp_id, q.code, q.stem, q.options, q.answer, q.visual
		FROM questions q
		JOIN knowledge_points kp ON kp.id = q.kp_id
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'math' AND q.code = ?
		ORDER BY q.id`, code).Scan(&rows).Error
	return rows, err
}

func (s *Service) loadArithmetic(ctx context.Context, excluded []int64) ([]kpRow, error) {
	var rows []kpRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT kp.id AS kp_id, kp.payload, m.code AS module
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'math' AND m.code IN ('add10','sub10')
		ORDER BY kp.id`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]kpRow, 0, len(rows))
	for _, row := range rows {
		if contains(excluded, row.KpID) {
			continue
		}
		var meta payload
		if json.Unmarshal([]byte(row.Payload), &meta) != nil {
			continue
		}
		if meta.Kind != "add" && meta.Kind != "sub" {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func (s *Service) generateMissing(ctx context.Context, excluded []int64) (Question, error) {
	rows, err := s.loadArithmetic(ctx, excluded)
	if err != nil {
		return Question{}, err
	}
	if len(rows) == 0 {
		return Question{}, ErrNoMaterial
	}
	row := rows[randIndex(len(rows))]
	var meta payload
	_ = json.Unmarshal([]byte(row.Payload), &meta)
	result, sign := compute(meta)
	hole := randIndex(3)
	answer, stem := missingParts(meta, result, sign, hole)
	options, index := consecutiveOptions(answer)
	return Question{
		InstanceID: newID(), Type: "missing", Stem: stem, TargetID: row.KpID,
		Visual: Visual{Kind: meta.Kind, Text: stem, A: meta.A, B: meta.B, Emoji: meta.Emoji},
		Options: options, AnswerIndex: index,
	}, nil
}

func (s *Service) generateJudge(ctx context.Context, excluded []int64) (Question, error) {
	rows, err := s.loadArithmetic(ctx, excluded)
	if err != nil {
		return Question{}, err
	}
	if len(rows) == 0 {
		return Question{}, ErrNoMaterial
	}
	row := rows[randIndex(len(rows))]
	var meta payload
	_ = json.Unmarshal([]byte(row.Payload), &meta)
	result, sign := compute(meta)
	shown := result
	correct := true
	if randIndex(2) == 1 {
		shown = result + 1
		correct = false
	}
	stem := fmt.Sprintf("%d %s %d = %d，对吗？", meta.A, sign, meta.B, shown)
	answerIndex := 0
	if !correct {
		answerIndex = 1
	}
	return Question{
		InstanceID: newID(), Type: "judge", Stem: stem, TargetID: row.KpID,
		Visual: Visual{Kind: meta.Kind, Text: stem, A: meta.A, B: meta.B, Emoji: meta.Emoji},
		Options: []Option{{ID: 0, Label: "对"}, {ID: 1, Label: "错"}},
		AnswerIndex: answerIndex,
	}, nil
}

func compute(meta payload) (int, string) {
	if meta.Kind == "sub" {
		return meta.A - meta.B, "−"
	}
	return meta.A + meta.B, "+"
}

func missingParts(meta payload, result int, sign string, hole int) (int, string) {
	switch hole {
	case 0:
		return meta.A, fmt.Sprintf("□ %s %d = %d", sign, meta.B, result)
	case 1:
		return meta.B, fmt.Sprintf("%d %s □ = %d", meta.A, sign, result)
	default:
		return result, fmt.Sprintf("%d %s %d = □", meta.A, sign, meta.B)
	}
}

func consecutiveOptions(n int) ([]Option, int) {
	start := n - 1
	if start < 0 {
		start = 0
	}
	labels := []int{start, start + 1, start + 2, start + 3}
	order := []int{0, 1, 2, 3}
	shuffle(order)
	options := make([]Option, 4)
	answerIndex := 0
	for i, from := range order {
		value := labels[from]
		options[i] = Option{ID: int64(i), Label: fmt.Sprintf("%d", value)}
		if value == n {
			answerIndex = i
		}
	}
	return options, answerIndex
}

func contains(ids []int64, id int64) bool {
	for _, item := range ids {
		if item == id {
			return true
		}
	}
	return false
}

func shuffle[T any](items []T) {
	for i := len(items) - 1; i > 0; i-- {
		j := randIndex(i + 1)
		items[i], items[j] = items[j], items[i]
	}
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
