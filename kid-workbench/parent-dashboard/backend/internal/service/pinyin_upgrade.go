package service

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/conchi/study-learning/mastery"
	learningmodel "github.com/conchi/study-learning/model"
	"github.com/conchi/study-learning/pinyincatalog"
	"github.com/conchi/study-workbench/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type PinyinUpgradeReport struct {
	DryRun           bool  `json:"dry_run"`
	Children         int   `json:"children"`
	Points           int   `json:"points"`
	ChangedToPartial int   `json:"changed_to_partial"`
	PriorRewards     int   `json:"prior_rewards"`
	SyllableMappings int64 `json:"syllable_mappings"`
	AlreadyApplied   int   `json:"already_applied"`
	CatalogAvailable bool  `json:"catalog_available"`
}

var rollbackPreview = errors.New("rollback pinyin upgrade preview")

func UpgradePinyin(d *gorm.DB, apply bool) (PinyinUpgradeReport, error) {
	out := PinyinUpgradeReport{DryRun: !apply, CatalogAvailable: true}
	err := d.Transaction(func(tx *gorm.DB) error {
		if e := pinyincatalog.Sync(context.Background(), tx); e != nil {
			if errors.Is(e, pinyincatalog.ErrUnavailable) {
				out.CatalogAvailable = false
			} else {
				return e
			}
		}
		if e := tx.Model(&learningmodel.PinyinSyllableLink{}).Count(&out.SyllableMappings).Error; e != nil {
			return e
		}
		type target struct {
			ChildID, KpID int64
			ModuleCode    string
		}
		var targets []target
		if e := tx.Raw(`SELECT ms.child_id,ms.kp_id,m.code AS module_code FROM mastery_states ms JOIN knowledge_points kp ON kp.id=ms.kp_id JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id WHERE s.code='pinyin' ORDER BY ms.child_id,ms.kp_id`).Scan(&targets).Error; e != nil {
			return e
		}
		children := map[int64]bool{}
		for _, v := range targets {
			// Same lock order as all child answer writes.
			var child model.Child
			if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&child, v.ChildID).Error; e != nil {
				return e
			}
			var n int64
			if e := tx.Model(&learningmodel.PinyinUpgradeAudit{}).Where("child_id=? AND kp_id=? AND rule_version=2", v.ChildID, v.KpID).Count(&n).Error; e != nil {
				return e
			}
			if n > 0 {
				out.AlreadyApplied++
				continue
			}
			var modern int64
			if e := tx.Model(&learningmodel.PinyinMasteryMilestone{}).Where("child_id=? AND kp_id=? AND rule_version=2", v.ChildID, v.KpID).Count(&modern).Error; e != nil {
				return e
			}
			if modern == 0 {
				if e := tx.Table("pinyin_answer_receipts pr").Joins("JOIN attempts a ON a.id=pr.attempt_id").Where("a.child_id=? AND a.kp_id=?", v.ChildID, v.KpID).Count(&modern).Error; e != nil {
					return e
				}
			}
			if modern > 0 {
				out.AlreadyApplied++
				continue
			}
			var before model.MasteryState
			if e := tx.Where("child_id=? AND kp_id=?", v.ChildID, v.KpID).First(&before).Error; e != nil {
				return e
			}
			var skills []model.MasterySkill
			if e := tx.Where("child_id=? AND kp_id=?", v.ChildID, v.KpID).Find(&skills).Error; e != nil {
				return e
			}
			byCode := map[string]model.MasterySkill{}
			for _, sk := range skills {
				byCode[sk.SkillCode] = sk
			}
			_, status := skillsFromRows(mastery.SkillsFor("pinyin", v.ModuleCode), byCode, time.Now())
			if mastery.IsSkillDone(mastery.Status(before.Status)) && !mastery.IsSkillDone(status) {
				out.ChangedToPartial++
			}
			var rewards int64
			if e := tx.Model(&model.FlowerLedger{}).Where("child_id=? AND ref_type=? AND ref_id=? AND reason=?", v.ChildID, "knowledge_point", v.KpID, "mastered").Count(&rewards).Error; e != nil {
				return e
			}
			if rewards > 0 {
				out.PriorRewards++
			}
			out.Points++
			children[v.ChildID] = true
			snapshot, e := json.Marshal(before)
			if e != nil {
				return e
			}
			audit := learningmodel.PinyinUpgradeAudit{ChildID: v.ChildID, KpID: v.KpID, RuleVersion: 2, BeforeSnapshot: string(snapshot), HadReward: rewards > 0, AppliedAt: time.Now()}
			if e := tx.Create(&audit).Error; e != nil {
				return e
			}
			var counts struct{ Attempts, Correct int }
			if e := tx.Model(&model.Attempt{}).Select("COUNT(*) AS attempts,COALESCE(SUM(CASE WHEN is_correct THEN 1 ELSE 0 END),0) AS correct").Where("child_id=? AND kp_id=? AND source <> 'parent_mark'", v.ChildID, v.KpID).Scan(&counts).Error; e != nil {
				return e
			}
			if e := tx.Model(&model.MasteryState{}).Where("child_id=? AND kp_id=?", v.ChildID, v.KpID).Updates(map[string]any{"status": string(status), "mastered_at": nil, "attempts": counts.Attempts, "correct": counts.Correct, "updated_at": time.Now()}).Error; e != nil {
				return e
			}
		}
		out.Children = len(children)
		if !apply {
			return rollbackPreview
		}
		return nil
	})
	if errors.Is(err, rollbackPreview) {
		err = nil
	}
	return out, err
}
