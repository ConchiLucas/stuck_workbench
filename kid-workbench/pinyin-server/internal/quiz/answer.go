package quiz

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/conchi/study-learning/learning"
	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-learning/model"
	"github.com/conchi/study-learning/pinyincontract"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Service) Answer(ctx context.Context, childID int64, id string, in pinyincontract.AnswerRequest) (pinyincontract.AnswerResult, error) {
	if strings.TrimSpace(in.ClientID) == "" || len(in.ClientID) > 64 || in.OptionID == "" || len(in.OptionID) > 128 || in.CostMs < 0 || in.CostMs > 3600000 {
		return pinyincontract.AnswerResult{}, ErrInvalidRequest
	}
	var out pinyincontract.AnswerResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Match the shared learning engine's lock order across every service.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&model.Child{}, childID).Error; err != nil {
			return notFound(err)
		}
		var instance model.PinyinQuizInstance
		if err := tx.Where("id = ? AND child_id = ?", id, childID).First(&instance).Error; err != nil {
			return notFound(err)
		}
		var receipt model.PinyinAnswerReceipt
		err := tx.Where("child_id = ? AND client_id = ?", childID, in.ClientID).First(&receipt).Error
		if err == nil {
			if receipt.InstanceID != id || receipt.SelectedOptionID != in.OptionID {
				return ErrConflict
			}
			return json.Unmarshal([]byte(receipt.ResponseSnapshot), &out)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var count int64
		if err := tx.Model(&model.PinyinAnswerReceipt{}).Where("instance_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrConflict
		}
		now := time.Now()
		if !now.Before(instance.ExpiresAt) {
			return ErrExpired
		}
		var q pinyincontract.GeneratedQuestion
		if err := json.Unmarshal([]byte(instance.PublicSnapshot), &q); err != nil {
			return err
		}
		valid := false
		for _, option := range q.Options {
			valid = valid || option.ID == in.OptionID
		}
		if !valid {
			return ErrInvalidRequest
		}
		correct := in.OptionID == instance.AnswerOptionID
		state, applied, err := learning.NewService(s.cfg).ApplyOne(tx, childID, learning.AttemptInput{ClientID: in.ClientID, KpID: instance.KpID, SkillCode: instance.SkillCode, IsCorrect: correct, CostMs: in.CostMs, Source: mastery.SourceQuiz, At: now})
		if err != nil {
			return err
		}
		if !applied {
			return ErrConflict
		}
		var attempt model.Attempt
		if err := tx.Where("child_id = ? AND client_id = ?", childID, in.ClientID).First(&attempt).Error; err != nil {
			return err
		}
		var skill model.MasterySkill
		if err := tx.Where("child_id = ? AND kp_id = ? AND skill_code = ?", childID, instance.KpID, instance.SkillCode).First(&skill).Error; err != nil {
			return err
		}
		engine := mastery.State{Status: mastery.Status(skill.Status)}
		if skill.DueAt != nil {
			engine.DueAt = *skill.DueAt
		}
		out = pinyincontract.AnswerResult{InstanceID: id, AttemptID: attempt.ID, SelectedOptionID: in.OptionID, Correct: correct, AnswerOptionID: instance.AnswerOptionID, Skill: pinyincontract.SkillResult{Code: instance.SkillCode, Status: string(mastery.Display(engine, now))}, Knowledge: pinyincontract.KnowledgeResult{KpID: instance.KpID, Status: state.Status, NewlyMastered: state.NewlyMastered}}
		data, err := json.Marshal(out)
		if err != nil {
			return err
		}
		return tx.Create(&model.PinyinAnswerReceipt{ChildID: childID, ClientID: in.ClientID, InstanceID: id, AttemptID: attempt.ID, SkillCode: instance.SkillCode, SelectedOptionID: in.OptionID, ResponseSnapshot: string(data), CreatedAt: now}).Error
	})
	if err != nil {
		return pinyincontract.AnswerResult{}, err
	}
	return out, nil
}
