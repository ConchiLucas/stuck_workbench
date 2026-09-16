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
	ErrInvalidType = errors.New("无效的英语题型")
	ErrNoMaterial  = errors.New("没有足够的英语单词生成题目")
)

var abstractLook = map[string]struct{}{
	"hello": {}, "hi": {}, "please": {}, "sorry": {}, "thanks": {}, "thank": {},
	"goodbye": {}, "bye": {}, "yes": {}, "no": {}, "morning": {}, "noon": {},
	"evening": {}, "night": {}, "yesterday": {}, "today": {}, "tomorrow": {},
}

type Visual struct {
	Kind     string `json:"kind"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
}

type Option struct {
	ID       int64  `json:"id"`
	Label    string `json:"label,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
}

type Question struct {
	InstanceID  string   `json:"instanceId"`
	Type        string   `json:"type"`
	Stem        string   `json:"stem"`
	TargetID    int64    `json:"targetId"`
	SpeechText  string   `json:"speechText,omitempty"`
	SpeechURL   string   `json:"speechUrl,omitempty"`
	Visual      Visual   `json:"visual"`
	Options     []Option `json:"options"`
	AnswerIndex int      `json:"answerIndex"`
}

type wordRow struct {
	KpID      int64
	Word      string
	Payload   string
	Module    string
	HasSense  bool
	HasSpeech bool
}

type meaning struct {
	MeaningZh string `json:"meaningZh"`
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Generate(ctx context.Context, quizType string, excluded []int64) (Question, error) {
	quizType = strings.TrimSpace(quizType)
	if quizType != "listen" && quizType != "look" {
		return Question{}, ErrInvalidType
	}
	rows, err := s.loadWords(ctx)
	if err != nil {
		return Question{}, err
	}
	candidates := make([]wordRow, 0, len(rows))
	for _, row := range rows {
		if meaningZh(row) == "" {
			continue
		}
		if quizType == "look" {
			if _, skip := abstractLook[strings.ToLower(row.Word)]; skip {
				continue
			}
		}
		if quizType == "listen" && !row.HasSpeech {
			continue
		}
		if contains(excluded, row.KpID) {
			continue
		}
		candidates = append(candidates, row)
	}
	if len(candidates) == 0 {
		return Question{}, ErrNoMaterial
	}
	target := candidates[randIndex(len(candidates))]
	distractors := pickDistractors(target, rows, excluded)
	if len(distractors) < 3 {
		return Question{}, ErrNoMaterial
	}
	options := make([]Option, 0, 4)
	for _, row := range append([]wordRow{target}, distractors...) {
		option := Option{ID: row.KpID, Label: meaningZh(row)}
		if row.HasSense {
			option.ImageURL = fmt.Sprintf("/api/v1/english/words/%d/sense.png", row.KpID)
		}
		options = append(options, option)
	}
	shuffle(options)
	question := Question{
		InstanceID: newID(), Type: quizType, TargetID: target.KpID,
		Options: options, AnswerIndex: indexOf(options, target.KpID),
		SpeechText: target.Word,
	}
	if target.HasSpeech {
		question.SpeechURL = fmt.Sprintf("/api/v1/english/words/%d/speech.mp3", target.KpID)
	}
	if quizType == "listen" {
		question.Stem = "听一听，选出你听到的单词"
		question.Visual = Visual{Kind: "sound"}
	} else {
		question.Stem = "哪一张图是这个单词？"
		question.Visual = Visual{Kind: "word", Text: target.Word}
	}
	return question, nil
}

func (s *Service) loadWords(ctx context.Context) ([]wordRow, error) {
	var rows []wordRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT kp.id AS kp_id, kp.title AS word, kp.payload, m.code AS module,
			CASE WHEN COALESCE(ea.sense_image_url,'')<>'' THEN 1 ELSE 0 END AS has_sense,
			CASE WHEN COALESCE(ea.speech_audio_url,'')<>'' THEN 1 ELSE 0 END AS has_speech
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		LEFT JOIN english_assets ea ON ea.kp_id = kp.id
		WHERE s.code = 'english'
		ORDER BY m.order_no, kp.order_no, kp.id`).Scan(&rows).Error
	return rows, err
}

func pickDistractors(target wordRow, all []wordRow, excluded []int64) []wordRow {
	same, other := []wordRow{}, []wordRow{}
	for _, row := range all {
		if row.KpID == target.KpID || contains(excluded, row.KpID) || strings.EqualFold(row.Word, target.Word) || meaningZh(row) == "" {
			continue
		}
		if row.Module == target.Module {
			same = append(same, row)
		} else {
			other = append(other, row)
		}
	}
	shuffle(same)
	shuffle(other)
	out := append(same, other...)
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}

func meaningZh(row wordRow) string {
	var meta meaning
	_ = json.Unmarshal([]byte(row.Payload), &meta)
	return strings.TrimSpace(meta.MeaningZh)
}

func contains(ids []int64, id int64) bool {
	for _, item := range ids {
		if item == id {
			return true
		}
	}
	return false
}

func indexOf(options []Option, id int64) int {
	for i, option := range options {
		if option.ID == id {
			return i
		}
	}
	return 0
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
