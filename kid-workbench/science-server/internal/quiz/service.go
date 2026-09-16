package quiz

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrInvalidType = errors.New("无效的科普题型")
	ErrNoMaterial  = errors.New("没有足够的科普题目生成练习")
)

type Visual struct {
	Kind  string `json:"kind"`
	Key   string `json:"key,omitempty"`
	Text  string `json:"text,omitempty"`
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

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Generate(ctx context.Context, quizType string, excluded []int64) (Question, error) {
	quizType = strings.TrimSpace(quizType)
	if quizType != "choice" {
		return Question{}, ErrInvalidType
	}
	rows, err := s.loadStored(ctx)
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
	var opts []struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}
	if err := json.Unmarshal([]byte(row.Options), &opts); err != nil || len(opts) < 2 {
		return Question{}, ErrNoMaterial
	}
	var answer struct {
		Index int    `json:"index"`
		ID    string `json:"id"`
	}
	if err := json.Unmarshal([]byte(row.Answer), &answer); err != nil {
		return Question{}, ErrNoMaterial
	}
	answerIndex := answer.Index
	if strings.TrimSpace(answer.ID) != "" {
		answerIndex = -1
		for i, item := range opts {
			id := strings.TrimSpace(item.ID)
			if id == "" {
				id = item.Label
			}
			if id == answer.ID {
				answerIndex = i
				break
			}
		}
	}
	if answerIndex < 0 || answerIndex >= len(opts) {
		return Question{}, ErrNoMaterial
	}
	var visual Visual
	_ = json.Unmarshal([]byte(row.Visual), &visual)
	if visual.Text == "" && visual.Key != "" {
		visual.Text = visual.Key
	}
	options := make([]Option, len(opts))
	for i, item := range opts {
		options[i] = Option{ID: int64(i), Label: item.Label}
	}
	return Question{
		InstanceID: newID(), Type: "choice", Stem: row.Stem, TargetID: row.KpID,
		Visual: visual, Options: options, AnswerIndex: answerIndex,
	}, nil
}

func (s *Service) loadStored(ctx context.Context) ([]storedRow, error) {
	var rows []storedRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT q.id AS question_id, q.kp_id, q.code, q.stem, q.options, q.answer, q.visual
		FROM questions q
		JOIN knowledge_points kp ON kp.id = q.kp_id
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'science' AND q.code IN ('recognize','choice')
		ORDER BY q.id`).Scan(&rows).Error
	return rows, err
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
