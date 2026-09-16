package learning

import (
	"time"

	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-learning/model"
	"gorm.io/gorm"
)

// finalizeMath only runs for classified, unassisted math answers. A representative
// skill's timestamp is not the moment all required skills became complete.
// The existing reward receipt is the durable proof that a KP was counted before.
// ApplyOne holds the child's row lock, and the receipt is written in the same
// transaction as this state, daily counters, and the flower balance.
func finalizeMath(tx *gorm.DB, childID, kpID int64, before mastery.State, state *mastery.State, at time.Time) (bool, error) {
	state.MasteredAt = nil
	newly := false
	if mastery.IsSkillDone(state.Status) {
		var receipts []model.FlowerLedger
		if err := tx.Where("child_id=? AND reason=? AND ref_type=? AND ref_id=?", childID, "mastered", "knowledge_point", kpID).Order("created_at ASC,id ASC").Limit(1).Find(&receipts).Error; err != nil {
			return false, err
		}
		switch {
		case len(receipts) > 0:
			state.MasteredAt = &receipts[0].CreatedAt
		case mastery.IsSkillDone(before.Status) && before.MasteredAt != nil:
			state.MasteredAt = before.MasteredAt
		default:
			completed := at
			state.MasteredAt = &completed
			newly = true
		}
	}
	var total struct{ Attempts, Correct int }
	if err := tx.Model(&model.Attempt{}).Select("COUNT(*) AS attempts,COALESCE(SUM(CASE WHEN is_correct THEN 1 ELSE 0 END),0) AS correct").Where("child_id=? AND kp_id=? AND source<>?", childID, kpID, mastery.SourceParentMark).Scan(&total).Error; err != nil {
		return false, err
	}
	state.Attempts, state.Correct = total.Attempts, total.Correct
	next := stateFromEngine(childID, kpID, *state)
	return newly, upsertMastery(tx, &next)
}
