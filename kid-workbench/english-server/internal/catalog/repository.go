package catalog

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"
)

const englishSubject = "english"

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ListModules(ctx context.Context) ([]Module, error) {
	modules := make([]Module, 0)
	err := r.db.WithContext(ctx).Raw(`SELECT m.code, m.name, m.order_no, COUNT(kp.id) AS word_count
		FROM modules m JOIN subjects s ON s.id=m.subject_id JOIN knowledge_points kp ON kp.module_id=m.id
		WHERE s.code=? GROUP BY m.id,m.code,m.name,m.order_no ORDER BY m.order_no,m.id`, englishSubject).Scan(&modules).Error
	return modules, err
}

func (r *Repository) ListWords(ctx context.Context, moduleCode string) ([]Word, error) {
	var rows []wordRow
	if err := r.wordQuery(ctx).Where("m.code = ?", moduleCode).Order("m.order_no,kp.order_no,kp.id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return decodeWords(rows), nil
}

func (r *Repository) GetWord(ctx context.Context, kpID int64) (Word, error) {
	var row wordRow
	result := r.wordQuery(ctx).Where("kp.id = ?", kpID).Limit(1).Scan(&row)
	if result.Error != nil {
		return Word{}, result.Error
	}
	if result.RowsAffected == 0 {
		return Word{}, gorm.ErrRecordNotFound
	}
	return decodeWord(row), nil
}

type wordRow struct {
	KpID, OrderNo                         int64
	Word, Payload, ModuleCode, ModuleName string
	HasGlyph, HasSense, HasSpeech         bool
}

func (r *Repository) wordQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table("knowledge_points kp").Select(`kp.id AS kp_id,kp.title AS word,kp.payload,
		m.code AS module_code,m.name AS module_name,kp.order_no,
		CASE WHEN COALESCE(ea.glyph_image_url,'')<>'' THEN TRUE ELSE FALSE END AS has_glyph,
		CASE WHEN COALESCE(ea.sense_image_url,'')<>'' THEN TRUE ELSE FALSE END AS has_sense,
		CASE WHEN COALESCE(ea.speech_audio_url,'')<>'' THEN TRUE ELSE FALSE END AS has_speech`).
		Joins("JOIN modules m ON m.id=kp.module_id").Joins("JOIN subjects s ON s.id=m.subject_id").
		Joins("LEFT JOIN english_assets ea ON ea.kp_id=kp.id").Where("s.code = ?", englishSubject)
}

type metadata struct {
	MeaningZh, Phonetic, PartOfSpeech, Example, ExampleMeaningZh string
}

func decodeWords(rows []wordRow) []Word {
	words := make([]Word, 0, len(rows))
	for _, row := range rows {
		words = append(words, decodeWord(row))
	}
	return words
}

func decodeWord(row wordRow) Word {
	var meta metadata
	_ = json.Unmarshal([]byte(row.Payload), &meta)
	return Word{KpID: row.KpID, Word: row.Word, MeaningZh: meta.MeaningZh, Phonetic: meta.Phonetic,
		PartOfSpeech: meta.PartOfSpeech, Example: meta.Example, ExampleMeaningZh: meta.ExampleMeaningZh,
		ModuleCode: row.ModuleCode, ModuleName: row.ModuleName, HasGlyph: row.HasGlyph,
		HasSense: row.HasSense, HasSpeech: row.HasSpeech, OrderNo: int(row.OrderNo)}
}
