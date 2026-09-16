package practice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/conchi/study-learning/learning"
	"github.com/conchi/study-learning/mastery"
	learningmodel "github.com/conchi/study-learning/model"
	"github.com/conchi/study-learning/poemcontent"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/conchi/poem-server/internal/studyplan"
)

var (
	ErrPlanNotFound     = errors.New("plan not found")
	ErrItemNotFound     = errors.New("plan item not found")
	ErrItemCompleted    = errors.New("plan item completed")
	ErrPlanIncomplete   = errors.New("plan incomplete")
	ErrClientIDConflict = errors.New("client id belongs to another question")
)

type AnswerInput struct {
	ClientID    string   `json:"clientId" binding:"required"`
	OptionIndex int      `json:"optionIndex"`
	SelectedID  string    `json:"selectedId"`
	AnswerID    string    `json:"answerId"`
	Sequence    []string `json:"sequence"`
	CostMs      int      `json:"costMs"`
}

type AnswerResult struct {
	Correct     bool              `json:"correct"`
	AnswerIndex int               `json:"answerIndex"`
	CanRetry    bool              `json:"canRetry"`
	Tries       int               `json:"tries"`
	Status      string            `json:"status"`
	Mastery     learning.StateDTO `json:"mastery"`
	Explanation string            `json:"explanation,omitempty"`
}

type Service struct {
	db       *gorm.DB
	learning *learning.Service
}

type AttemptReceipt struct {
	ChildID    int64  `gorm:"primaryKey"`
	ClientID   string `gorm:"primaryKey;column:client_id"`
	KpID       int64  `gorm:"column:kp_id"`
	QuestionID int64  `gorm:"column:question_id"`
	PlanID     int64  `gorm:"column:plan_id"`
	ItemID     int64  `gorm:"column:item_id"`
	ResultJSON string `gorm:"column:result_json"`
	CreatedAt  time.Time
}

func (AttemptReceipt) TableName() string { return "poem_attempt_receipts" }

func NewService(db *gorm.DB, cfg mastery.Config) *Service {
	_ = db.AutoMigrate(&AttemptReceipt{})
	return &Service{db: db, learning: learning.NewService(cfg)}
}

func (s *Service) Answer(ctx context.Context, childID, planID, itemID int64, input AnswerInput) (AnswerResult, error) {
	if strings.TrimSpace(input.ClientID) == "" {
		return AnswerResult{}, errors.New("client id is required")
	}
	if strings.TrimSpace(input.SelectedID) == "" && strings.TrimSpace(input.AnswerID) != "" {
		input.SelectedID = strings.TrimSpace(input.AnswerID)
	}
	var output AnswerResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var planRow studyplan.StudyPlan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND child_id = ? AND subject_code = ?", planID, childID, "poem").First(&planRow).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPlanNotFound
			}
			return err
		}
		var child learningmodel.Child
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&child, childID).Error; err != nil {
			return err
		}
		var item learningmodel.PlanItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND plan_id = ?", itemID, planID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrItemNotFound
			}
			return err
		}
		isCorrect, selected, answerDisplay, err := judgePoem(item, input)
		if err != nil {
			return err
		}

		var receipt AttemptReceipt
		if err := tx.Where("child_id = ? AND client_id = ?", childID, input.ClientID).First(&receipt).Error; err == nil {
			if !receiptMatches(receipt, planID, item) {
				return ErrClientIDConflict
			}
			return json.Unmarshal([]byte(receipt.ResultJSON), &output)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var existing learningmodel.Attempt
		if err := tx.Where("child_id = ? AND client_id = ?", childID, input.ClientID).First(&existing).Error; err == nil {
			if existing.KpID != item.KpID || existing.QuestionID == nil || *existing.QuestionID != item.QuestionID {
				return ErrClientIDConflict
			}
			visibleAnswer := answerDisplay
			if !existing.IsCorrect && item.Status != "completed" && item.Tries < 2 {
				visibleAnswer = -1
			}
			output = AnswerResult{Correct: existing.IsCorrect, AnswerIndex: visibleAnswer,
				CanRetry: item.Status != "completed" && !existing.IsCorrect && item.Tries < 2,
				Tries:    item.Tries, Status: item.Status, Mastery: loadMastery(tx, childID, item.KpID)}
			if existing.IsCorrect || item.Tries >= 2 {
				output.Explanation = item.Explanation
			}
			return saveReceipt(tx, childID, input.ClientID, planID, item, output)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if item.Status == "completed" {
			return ErrItemCompleted
		}

		state, applied, err := s.learning.ApplyOne(tx, childID, learning.AttemptInput{
			ClientID: input.ClientID, KpID: item.KpID, QuestionID: &item.QuestionID, PlanItemID: &item.ID,
			Selected: selected, SkillCode: questionCode(tx, item.QuestionID),
			IsCorrect: isCorrect, CostMs: input.CostMs, Source: mastery.SourceQuiz, At: time.Now(),
		})
		if err != nil {
			return err
		}
		if !applied {
			if err := tx.Where("child_id = ? AND client_id = ?", childID, input.ClientID).First(&receipt).Error; err == nil {
				if !receiptMatches(receipt, planID, item) {
					return ErrClientIDConflict
				}
				return json.Unmarshal([]byte(receipt.ResultJSON), &output)
			}
			return errors.New("attempt exists without a replay receipt")
		}
		tries := item.Tries + 1
		status := "pending"
		completedNow := isCorrect || tries >= 2
		updates := map[string]any{
			"tries": tries, "cost_ms": item.CostMs + input.CostMs,
			"picks": appendPick(item.Picks, selected), "status": status,
		}
		if completedNow {
			status = "completed"
			now := time.Now()
			updates["status"] = status
			updates["answered_at"] = &now
		}
		if err := tx.Model(&learningmodel.PlanItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			return err
		}
		if completedNow {
			planUpdates := map[string]any{"done_count": gorm.Expr("done_count + 1")}
			if isCorrect {
				planUpdates["correct_count"] = gorm.Expr("correct_count + 1")
			}
			if err := tx.Model(&studyplan.StudyPlan{}).Where("id = ?", planID).Updates(planUpdates).Error; err != nil {
				return err
			}
		}
		visibleAnswer := answerDisplay
		if !completedNow {
			visibleAnswer = -1
		}
		output = AnswerResult{Correct: isCorrect, AnswerIndex: visibleAnswer,
			CanRetry: !isCorrect && tries < 2, Tries: tries, Status: status, Mastery: state}
		if completedNow {
			output.Explanation = item.Explanation
		}
		return saveReceipt(tx, childID, input.ClientID, planID, item, output)
	})
	return output, err
}

func receiptMatches(receipt AttemptReceipt, planID int64, item learningmodel.PlanItem) bool {
	return receipt.KpID == item.KpID && receipt.QuestionID == item.QuestionID &&
		receipt.PlanID == planID && receipt.ItemID == item.ID
}

func saveReceipt(tx *gorm.DB, childID int64, clientID string, planID int64, item learningmodel.PlanItem, output AnswerResult) error {
	encoded, err := json.Marshal(output)
	if err != nil {
		return err
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&AttemptReceipt{
		ChildID: childID, ClientID: clientID, KpID: item.KpID, QuestionID: item.QuestionID,
		PlanID: planID, ItemID: item.ID, ResultJSON: string(encoded), CreatedAt: time.Now(),
	}).Error
}

func (s *Service) Finish(ctx context.Context, childID, planID int64) (studyplan.StudyPlan, error) {
	var row studyplan.StudyPlan
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND child_id = ? AND subject_code = ?", planID, childID, "poem").First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPlanNotFound
			}
			return err
		}
		if row.Status == "completed" {
			return nil
		}
		if row.DoneCount < row.TargetCount {
			return ErrPlanIncomplete
		}
		now := time.Now()
		stars := 1
		if row.TargetCount > 0 && row.CorrectCount*100/row.TargetCount >= 80 {
			stars = 3
		} else if row.TargetCount > 0 && row.CorrectCount*100/row.TargetCount >= 60 {
			stars = 2
		}
		if err := tx.Model(&row).Updates(map[string]any{"status": "completed", "completed_at": &now, "stars": stars}).Error; err != nil {
			return err
		}
		row.Status, row.CompletedAt, row.Stars = "completed", &now, stars
		return nil
	})
	return row, err
}

func appendPick(existing string, option string) string {
	if existing == "" {
		return option
	}
	return existing + "," + option
}

func questionCode(tx *gorm.DB, questionID int64) string {
	var code string
	_ = tx.Table("questions").Select("code").Where("id = ?", questionID).Scan(&code).Error
	return poemcontent.SkillForKind(code)
}

func judgePoem(item learningmodel.PlanItem, input AnswerInput) (bool, string, int, error) {
	if strings.TrimSpace(item.QuestionSnapshot) == "" {
		return false, "", -1, fmt.Errorf("missing poem snapshot")
	}
	converted, err := poemcontent.PlanExampleFromSnapshot(item.QuestionSnapshot, "")
	if err != nil {
		return false, "", -1, err
	}
	in := poemcontent.AnswerInput{Kind: converted.Example.Kind, SelectedID: strings.TrimSpace(input.SelectedID), Sequence: input.Sequence}
	if poemcontent.KindForSkill(converted.Example.Kind) != poemcontent.KindRecite && in.SelectedID == "" && input.OptionIndex >= 0 && input.OptionIndex < len(converted.Example.Options) {
		in.SelectedID = converted.Example.Options[input.OptionIndex].ID
	}
	return poemcontent.IsCorrect(converted.Example, in), poemcontent.EncodeInput(in), choiceDisplayIndex(converted.Example, converted.Example.AnswerID), nil
}

func choiceDisplayIndex(example poemcontent.PoemExample, answerID string) int {
	for i, o := range example.Options {
		if o.ID == answerID {
			return i
		}
	}
	return -1
}

func loadMastery(tx *gorm.DB, childID, kpID int64) learning.StateDTO {
	var row learningmodel.MasteryState
	if tx.Where("child_id = ? AND kp_id = ?", childID, kpID).First(&row).Error != nil {
		return learning.StateDTO{KpID: kpID, Status: string(mastery.StatusNotStarted)}
	}
	accuracy := 0.0
	if row.Attempts > 0 {
		accuracy = float64(row.Correct) / float64(row.Attempts)
	}
	return learning.StateDTO{KpID: kpID, Status: row.Status, Streak: row.Streak,
		Attempts: row.Attempts, Accuracy: accuracy, IntervalDays: row.IntervalDays, DueAt: row.DueAt}
}
