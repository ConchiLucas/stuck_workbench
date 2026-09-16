package plan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	learningmodel "github.com/conchi/study-learning/model"
	"gorm.io/gorm"
)

var (
	ErrChildNotFound = errors.New("child not found")
	ErrPlanNotFound  = errors.New("plan not found")
	ErrNoQuestions   = errors.New("no pinyin questions")
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Create(ctx context.Context, childID int64, count int) (Detail, error) {
	if count <= 0 {
		count = 8
	}
	if count > 20 {
		count = 20
	}
	var childCount int64
	if err := s.db.WithContext(ctx).Table("children").Where("id = ?", childID).Count(&childCount).Error; err != nil {
		return Detail{}, err
	}
	if childCount == 0 {
		return Detail{}, ErrChildNotFound
	}

	type candidate struct {
		ID, KpID                  int64
		Code, Type, Stem, Options string
		Answer, Visual, Speech    string
	}
	var candidates []candidate
	if err := s.db.WithContext(ctx).Table("questions q").
		Select("q.id, q.kp_id, q.code, q.type, q.stem, q.options, q.answer, q.visual, q.speech").
		Joins("JOIN knowledge_points kp ON kp.id = q.kp_id").
		Joins("JOIN modules m ON m.id = kp.module_id").
		Joins("JOIN subjects s ON s.id = m.subject_id").
		Where("s.code = ? AND q.code IN ?", "pinyin", []string{"inword", "listen"}).
		Order("m.order_no, kp.order_no, q.id").Limit(count).Scan(&candidates).Error; err != nil {
		return Detail{}, err
	}
	if len(candidates) == 0 {
		return Detail{}, ErrNoQuestions
	}

	row := StudyPlan{ChildID: childID, PlanDate: time.Now().Format("2006-01-02"), SubjectCode: "pinyin", Status: "pending", TargetCount: len(candidates), CreatedAt: time.Now()}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("study_plans").Where("child_id = ? AND plan_date = ?", childID, row.PlanDate).
			Select("COALESCE(MAX(seq_no), 0)").Scan(&row.SeqNo).Error; err != nil {
			return err
		}
		row.SeqNo++
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		for index, candidate := range candidates {
			order := shuffledOrder(optionCount(candidate.Options), row.ID+candidate.ID)
			snapshot, err := json.Marshal(map[string]any{"code": candidate.Code, "type": candidate.Type})
			if err != nil {
				return err
			}
			item := learningmodel.PlanItem{
				PlanID: row.ID, Seq: index + 1, KpID: candidate.KpID, QuestionID: candidate.ID,
				Bucket: "new", Status: "pending", OptionOrder: formatOrder(order),
				QuestionStem: candidate.Stem, QuestionOptions: candidate.Options, QuestionAnswer: candidate.Answer,
				QuestionVisual: candidate.Visual, QuestionSpeech: candidate.Speech,
				QuestionSnapshot: string(snapshot), ContentSnapshotVersion: 1,
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Detail{}, err
	}
	return s.Get(ctx, childID, row.ID)
}

func (s *Service) Get(ctx context.Context, childID, planID int64) (Detail, error) {
	var row StudyPlan
	err := s.db.WithContext(ctx).Where("id = ? AND child_id = ? AND subject_code = ?", planID, childID, "pinyin").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Detail{}, ErrPlanNotFound
	}
	if err != nil {
		return Detail{}, err
	}

	type itemRow struct {
		ID, KpID, QuestionID                              int64
		Seq, Tries                                        int
		Bucket, Status, Picks, OptionOrder                string
		Letter, Code, Type, Stem, Options, Visual, Speech string
		QuestionSnapshot                                  string
	}
	var rows []itemRow
	if err := s.db.WithContext(ctx).Table("plan_items pi").
		Select(`pi.id, pi.kp_id, pi.question_id, pi.seq, pi.tries, pi.bucket, pi.status, pi.picks, pi.option_order,
			kp.title AS letter, q.code, q.type, pi.question_snapshot,
			COALESCE(NULLIF(pi.question_stem, ''), q.stem) AS stem,
			COALESCE(NULLIF(pi.question_options, ''), q.options) AS options,
			COALESCE(NULLIF(pi.question_visual, ''), q.visual) AS visual,
			COALESCE(NULLIF(pi.question_speech, ''), q.speech) AS speech`).
		Joins("JOIN questions q ON q.id = pi.question_id").
		Joins("JOIN knowledge_points kp ON kp.id = pi.kp_id").
		Where("pi.plan_id = ?", row.ID).Order("pi.seq").Scan(&rows).Error; err != nil {
		return Detail{}, err
	}
	items := make([]Item, 0, len(rows))
	for _, value := range rows {
		var snapshot struct{ Code, Type string }
		if json.Unmarshal([]byte(value.QuestionSnapshot), &snapshot) == nil {
			if snapshot.Code != "" {
				value.Code = snapshot.Code
			}
			if snapshot.Type != "" {
				value.Type = snapshot.Type
			}
		}
		order := value.OptionOrder
		if order == "" {
			order = formatOrder(identityOrder(optionCount(value.Options)))
			if err := s.db.WithContext(ctx).Table("plan_items").Where("id = ?", value.ID).Update("option_order", order).Error; err != nil {
				return Detail{}, err
			}
		}
		reordered, err := reorderOptions(value.Options, order)
		if err != nil {
			return Detail{}, fmt.Errorf("question %d options: %w", value.QuestionID, err)
		}
		items = append(items, Item{ID: value.ID, Seq: value.Seq, KpID: value.KpID, Letter: value.Letter,
			Bucket: value.Bucket, Status: value.Status, Tries: value.Tries, Picks: value.Picks, OptionOrder: order,
			Question: Question{ID: value.QuestionID, Code: value.Code, Type: value.Type, Stem: value.Stem,
				Options: reordered, Visual: rawJSON(value.Visual, "{}"), Speech: rawJSON(value.Speech, "{}")}})
	}
	return Detail{Plan: row, Items: items}, nil
}

func (s *Service) Start(ctx context.Context, childID, planID int64) (Detail, error) {
	result := s.db.WithContext(ctx).Model(&StudyPlan{}).
		Where("id = ? AND child_id = ? AND subject_code = ?", planID, childID, "pinyin").
		Where("status = ?", "pending").Updates(map[string]any{"status": "doing", "started_at": time.Now()})
	if result.Error != nil {
		return Detail{}, result.Error
	}
	return s.Get(ctx, childID, planID)
}

func optionCount(raw string) int {
	var values []json.RawMessage
	if json.Unmarshal([]byte(raw), &values) != nil {
		return 0
	}
	return len(values)
}

func shuffledOrder(count int, seed int64) []int {
	order := identityOrder(count)
	rand.New(rand.NewSource(seed)).Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	return order
}

func identityOrder(count int) []int {
	order := make([]int, count)
	for index := range order {
		order[index] = index
	}
	return order
}

func formatOrder(order []int) string {
	values := make([]string, len(order))
	for index, value := range order {
		values[index] = strconv.Itoa(value)
	}
	return strings.Join(values, ",")
}

func ParseOrder(value string) ([]int, error) {
	if value == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	order := make([]int, len(parts))
	for index, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		order[index] = value
	}
	return order, nil
}

func reorderOptions(raw, orderValue string) (json.RawMessage, error) {
	var original []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &original); err != nil {
		return nil, err
	}
	order, err := ParseOrder(orderValue)
	if err != nil {
		return nil, err
	}
	reordered := make([]json.RawMessage, len(order))
	for index, originalIndex := range order {
		if originalIndex < 0 || originalIndex >= len(original) {
			return nil, errors.New("option order out of range")
		}
		reordered[index] = original[originalIndex]
	}
	return json.Marshal(reordered)
}

func rawJSON(value, fallback string) json.RawMessage {
	if !json.Valid([]byte(value)) {
		value = fallback
	}
	return json.RawMessage(value)
}
