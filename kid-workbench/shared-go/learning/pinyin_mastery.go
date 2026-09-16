package learning

import (
	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-learning/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

// finalizePinyin records the first completion under the three-skill contract.
// The caller holds the child's row lock and owns the transaction.
func finalizePinyin(tx *gorm.DB, childID, kpID int64, state *mastery.State, at time.Time) (bool, error) {
	state.MasteredAt = nil
	newly := false
	if mastery.IsSkillDone(state.Status) {
		milestone := model.PinyinMasteryMilestone{ChildID: childID, KpID: kpID, RuleVersion: 2, FirstCompletedAt: at}
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&milestone)
		if result.Error != nil {
			return false, result.Error
		}
		newly = result.RowsAffected == 1
		if e := tx.Where("child_id=? AND kp_id=? AND rule_version=2", childID, kpID).First(&milestone).Error; e != nil {
			return false, e
		}
		state.MasteredAt = &milestone.FirstCompletedAt
	}
	// Rollup's representative skill is not a total number of answers.
	var total struct{ Attempts, Correct int }
	if e := tx.Model(&model.Attempt{}).Select("COUNT(*) AS attempts, COALESCE(SUM(CASE WHEN is_correct THEN 1 ELSE 0 END),0) AS correct").Where("child_id=? AND kp_id=? AND source <> ?", childID, kpID, mastery.SourceParentMark).Scan(&total).Error; e != nil {
		return false, e
	}
	state.Attempts = total.Attempts
	state.Correct = total.Correct
	next := stateFromEngine(childID, kpID, *state)
	return newly, upsertMastery(tx, &next)
}
func pinyinRewardExists(tx *gorm.DB, childID, kpID int64) (bool, error) {
	var n int64
	e := tx.Model(&model.FlowerLedger{}).Where("child_id=? AND reason=? AND ref_type=? AND ref_id=?", childID, "mastered", "knowledge_point", kpID).Count(&n).Error
	return n > 0, e
}
