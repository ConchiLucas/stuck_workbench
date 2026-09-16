// Package pinyincatalog maps content-owned syllables to stable learning points.
package pinyincatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/conchi/study-learning/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

var ErrUnavailable = errors.New("pinyin syllable catalog unavailable")
var ErrIdentityConflict = errors.New("pinyin syllable identity changed")

type asset struct {
	ID                     int64
	InitialText, FinalText string
	Tone                   int
	SyllableText           string
	Difficulty             int
	Enabled                bool
}
type module struct {
	ID         int64
	SubjectID  int64
	Code, Name string
	OrderNo    int
}

func Sync(ctx context.Context, db *gorm.DB) error {
	if !db.Migrator().HasTable("pinyin_syllable_assets") {
		return ErrUnavailable
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize syncs without locking a child's learning writes.
		var subject struct{ ID int64 }
		if e := tx.Table("subjects").Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", "pinyin").First(&subject).Error; e != nil {
			return fmt.Errorf("%w: %v", ErrUnavailable, e)
		}
		var assets []asset
		if e := tx.Table("pinyin_syllable_assets").Order("id").Find(&assets).Error; e != nil {
			return e
		}
		var existing []model.PinyinSyllableLink
		if e := tx.Find(&existing).Error; e != nil {
			return e
		}
		byID := map[int64]model.PinyinSyllableLink{}
		for _, l := range existing {
			byID[l.AssetID] = l
		}
		for _, a := range assets {
			if l, ok := byID[a.ID]; ok && (l.InitialText != a.InitialText || l.FinalText != a.FinalText || l.Tone != a.Tone) {
				return fmt.Errorf("%w: asset %d", ErrIdentityConflict, a.ID)
			}
		}
		m := module{SubjectID: subject.ID, Code: "syllables", Name: "音节拼读", OrderNo: 100}
		if e := tx.Table("modules").Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "subject_id"}, {Name: "code"}}, DoNothing: true}).Create(&m).Error; e != nil {
			return e
		}
		if e := tx.Table("modules").Where("subject_id = ? AND code = ?", subject.ID, m.Code).First(&m).Error; e != nil {
			return e
		}
		if e := tx.Model(&model.PinyinSyllableLink{}).Where("1=1").Update("enabled", false).Error; e != nil {
			return e
		}
		for _, a := range assets {
			// Inactive material with no learning identity does not create new course content.
			if _, ok := byID[a.ID]; !ok && !a.Enabled {
				continue
			}
			payload, _ := json.Marshal(map[string]any{"kind": "syllable", "initial": a.InitialText, "final": a.FinalText, "tone": a.Tone, "syllable": a.SyllableText})
			kp := model.KnowledgePoint{ModuleID: m.ID, Code: fmt.Sprintf("py-syllable-%d", a.ID), Title: a.SyllableText, Payload: string(payload), Difficulty: max(1, a.Difficulty), OrderNo: int(a.ID)}
			if e := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "module_id"}, {Name: "code"}}, DoUpdates: clause.AssignmentColumns([]string{"title", "payload", "difficulty", "order_no"})}).Create(&kp).Error; e != nil {
				return e
			}
			if e := tx.Where("module_id = ? AND code = ?", m.ID, kp.Code).First(&kp).Error; e != nil {
				return e
			}
			l := model.PinyinSyllableLink{AssetID: a.ID, KpID: kp.ID, InitialText: a.InitialText, FinalText: a.FinalText, Tone: a.Tone, SyllableText: a.SyllableText, Enabled: a.Enabled, SyncedAt: time.Now()}
			if e := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "asset_id"}}, DoUpdates: clause.AssignmentColumns([]string{"syllable_text", "enabled", "synced_at"})}).Create(&l).Error; e != nil {
				return e
			}
		}
		return nil
	})
}
