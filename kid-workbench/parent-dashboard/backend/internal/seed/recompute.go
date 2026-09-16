package seed

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
)

func Recompute(gdb *gorm.DB, cfg mastery.Config, childID int64) error {
	r := repo.New(gdb)
	return gdb.Transaction(func(tx *gorm.DB) error {
		// The demo replayer cannot reconstruct v2 mastery events from raw facts.
		// Refuse before deleting daily totals once this child has entered v2.
		for _, table := range []string{"pinyin_upgrade_audits", "pinyin_answer_receipts", "pinyin_mastery_milestones"} {
			var count int64
			if err := tx.Table(table).Where("child_id = ?", childID).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return fmt.Errorf("legacy demo recompute cannot rebuild pinyin v2 history; use the audited pinyin-upgrade command")
			}
		}
		// Pinyin's per-skill v2 state is maintained by the dedicated audited upgrade.
		// This legacy demo replayer must not infer three-skill mastery from untyped facts.
		var pinyinIDs []int64
		if err := tx.Raw(`SELECT kp.id FROM knowledge_points kp JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id WHERE s.code='pinyin'`).Scan(&pinyinIDs).Error; err != nil {
			return err
		}
		pinyin := map[int64]bool{}
		for _, id := range pinyinIDs {
			pinyin[id] = true
		}
		deletion := tx.Where("child_id = ?", childID)
		if len(pinyinIDs) > 0 {
			deletion = deletion.Where("kp_id NOT IN ?", pinyinIDs)
		}
		if err := deletion.Delete(&model.MasteryState{}).Error; err != nil {
			return err
		}
		if err := tx.Where("child_id = ?", childID).Delete(&model.DailyStat{}).Error; err != nil {
			return err
		}

		var rows []model.Attempt
		if err := tx.Where("child_id = ?", childID).
			Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
			return err
		}

		difficulty := map[int64]int{}
		states := map[int64]mastery.State{}

		for _, a := range rows {
			if pinyin[a.KpID] {
				if a.Source != mastery.SourceParentMark {
					if err := r.BumpDailyStat(tx, childID, a.CreatedAt, model.DailyStat{PracticeSec: maxInt(a.CostMs/1000, 1), Attempts: 1, Correct: boolToInt(a.IsCorrect)}); err != nil {
						return err
					}
				}
				continue
			}
			d, cached := difficulty[a.KpID]
			if !cached {
				kp, err := r.GetKnowledgePoint(tx, a.KpID)
				if err != nil {
					return err
				}
				d = kp.Difficulty
				difficulty[a.KpID] = d
			}

			cur, seen := states[a.KpID]
			if !seen {
				cur = mastery.NewState()
			}
			next := mastery.Apply(cur, mastery.Attempt{
				Correct: a.IsCorrect, At: a.CreatedAt, Source: a.Source,
			}, d, cfg)
			states[a.KpID] = next

			newly := 0
			if cur.MasteredAt == nil && next.MasteredAt != nil {
				newly = 1
			}
			reviewDone := 0
			if cur.Status == mastery.StatusMastered && a.IsCorrect {
				reviewDone = 1
			}
			if a.Source != mastery.SourceParentMark {
				if err := r.BumpDailyStat(tx, childID, a.CreatedAt, model.DailyStat{
					PracticeSec: maxInt(a.CostMs/1000, 1), Attempts: 1,
					Correct: boolToInt(a.IsCorrect), NewlyMastered: newly, ReviewDone: reviewDone,
				}); err != nil {
					return err
				}
			}
		}

		for kpID, st := range states {
			row := model.MasteryState{
				ChildID: childID, KpID: kpID, Status: string(st.Status),
				Attempts: st.Attempts, Correct: st.Correct, Streak: st.Streak,
				BestStreak: st.BestStreak, Ease: st.Ease, IntervalDays: st.IntervalDays,
				MasteredAt: st.MasteredAt,
			}
			if !st.DueAt.IsZero() {
				t := st.DueAt
				row.DueAt = &t
			}
			if !st.FirstSeenAt.IsZero() {
				t := st.FirstSeenAt
				row.FirstSeenAt = &t
			}
			if err := r.UpsertMastery(tx, &row); err != nil {
				return err
			}
		}
		return nil
	})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
