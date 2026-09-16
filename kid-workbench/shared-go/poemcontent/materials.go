package poemcontent

import (
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type storedQuestion struct {
	KpID       int64
	Code       string
	Type       string
	Stem       string
	Options    string
	Answer     string
	Visual     string
	Speech     string
	Difficulty int
}

func (storedQuestion) TableName() string { return "questions" }

func EnsureMaterials(db *gorm.DB) error {
	if err := MigrateMedia(db); err != nil {
		return err
	}
	works, err := LoadWorks(db)
	if err != nil {
		return err
	}
	for _, w := range works {
		for _, line := range w.Lines {
			if err := EnsureLineSpeech(db, w.KpID, line.Ord, line.Text); err != nil {
				return err
			}
		}
	}
	return EnsureQuestions(db, works)
}

func LoadWorks(db *gorm.DB) ([]Work, error) {
	type row struct {
		ID, KpID                    int64
		Code, Title, Payload, Module string
	}
	var rows []row
	err := db.Raw(`
		SELECT kp.id AS id, kp.id AS kp_id, kp.code, kp.title, COALESCE(kp.payload,'') AS payload, m.code AS module
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'poem'
		ORDER BY m.order_no, kp.order_no, kp.id`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	works := make([]Work, 0, len(rows))
	for _, r := range rows {
		w := ParseWork(r.ID, r.Code, r.Title, r.Payload)
		if w.WorkID == "" {
			w.WorkID = strings.TrimSpace(r.Code)
		}
		if len(w.Lines) == 0 {
			continue
		}
		works = append(works, w)
	}
	return works, nil
}

func EnsureQuestions(db *gorm.DB, works []Work) error {
	if len(works) == 0 {
		var err error
		works, err = LoadWorks(db)
		if err != nil {
			return err
		}
	}
	kinds := []string{KindTitle, KindFill, KindCouplet, KindRecite}
	for i, w := range works {
		for _, kind := range kinds {
			ex, err := Generate(GenerateOpts{Kind: kind, Work: w, All: works, Seed: w.KpID*10 + int64(len(kind)), KpID: w.KpID})
			if err != nil {
				continue
			}
			speech := "{}"
			if ex.SpeechURL != "" {
				raw, _ := json.Marshal(map[string]string{"url": ex.SpeechURL, "text": ex.SpeechText})
				speech = string(raw)
			}
			q := storedQuestion{
				KpID: w.KpID, Code: kind, Type: questionType(kind), Stem: ex.Prompt,
				Options: StoredOptions(ex), Answer: StoredAnswer(ex), Visual: StoredVisual(ex),
				Speech: speech, Difficulty: 1,
			}
			if err := db.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "kp_id"}, {Name: "code"}},
				DoUpdates: clause.AssignmentColumns([]string{"type", "stem", "options", "answer", "visual", "speech"}),
			}).Create(&q).Error; err != nil {
				return fmt.Errorf("poem question %s/%s: %w", w.WorkID, kind, err)
			}
		}
		_ = i
	}
	return nil
}

func questionType(kind string) string {
	if kind == KindRecite {
		return "order"
	}
	return "choice"
}
