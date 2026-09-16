package catalog

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

const scienceSubject = "science"

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ListModules(ctx context.Context) ([]Module, error) {
	var modules []Module
	err := r.db.WithContext(ctx).Raw(`
		SELECT m.code, m.name, m.order_no, COUNT(kp.id) AS item_count
		FROM modules m
		JOIN subjects s ON s.id = m.subject_id
		JOIN knowledge_points kp ON kp.module_id = m.id
		JOIN science_assets sa ON sa.kp_id = kp.id AND sa.review_status = 'published'
		WHERE s.code = ?
		GROUP BY m.id, m.code, m.name, m.order_no
		ORDER BY m.order_no, m.id`, scienceSubject).Scan(&modules).Error
	return modules, err
}

func (r *Repository) ListItems(ctx context.Context, moduleCode string) ([]Item, error) {
	var items []Item
	err := r.itemQuery(ctx).Where("m.code = ?", moduleCode).
		Order("m.order_no, kp.order_no, kp.id").Scan(&items).Error
	localizeAssets(items)
	return items, err
}

func (r *Repository) GetItem(ctx context.Context, kpID int64) (Item, error) {
	var item Item
	result := r.itemQuery(ctx).Where("kp.id = ?", kpID).Limit(1).Scan(&item)
	if result.Error != nil {
		return Item{}, result.Error
	}
	if result.RowsAffected == 0 {
		return Item{}, gorm.ErrRecordNotFound
	}
	localizeAssets([]Item{item})
	if item.HasSenseImage {
		item.SenseImageURL = fmt.Sprintf("/api/v1/science/items/%d/sense.png?v=%d", item.KpID, item.ContentVersion)
	}
	if item.HasGlyphImage {
		item.GlyphImageURL = fmt.Sprintf("/api/v1/science/items/%d/glyph.png?v=%d", item.KpID, item.ContentVersion)
	}
	if item.HasSpeechAudio {
		item.SpeechURL = fmt.Sprintf("/api/v1/science/items/%d/speech.mp3?v=%d", item.KpID, item.ContentVersion)
	}
	return item, nil
}

func (r *Repository) itemQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table("knowledge_points kp").
		Select(`kp.id AS kp_id, kp.title,
			m.code AS module_code, m.name AS module_name,
			kp.difficulty, sa.summary, sa.explanation, sa.fun_fact, sa.content_version,
			sa.glyph_object_key, sa.sense_object_key, sa.speech_object_key,
			CASE WHEN COALESCE(sa.sense_image_url, '') <> '' THEN TRUE ELSE FALSE END AS has_sense_image,
			CASE WHEN COALESCE(sa.glyph_image_url, '') <> '' THEN TRUE ELSE FALSE END AS has_glyph_image,
			CASE WHEN COALESCE(sa.speech_audio_url, '') <> '' THEN TRUE ELSE FALSE END AS has_speech_audio,
			EXISTS(SELECT 1 FROM questions q WHERE q.kp_id = kp.id AND q.code = 'recognize') AS has_practice,
			kp.order_no AS order_no`).
		Joins("JOIN modules m ON m.id = kp.module_id").
		Joins("JOIN subjects s ON s.id = m.subject_id").
		Joins("JOIN science_assets sa ON sa.kp_id = kp.id AND sa.review_status = 'published'").
		Where("s.code = ?", scienceSubject)
}

func localizeAssets(items []Item) {
	for index := range items {
		item := &items[index]
		if item.HasSenseImage {
			item.SenseImageURL = fmt.Sprintf("/api/v1/science/items/%d/sense.png?v=%d", item.KpID, item.ContentVersion)
		}
		if item.HasGlyphImage {
			item.GlyphImageURL = fmt.Sprintf("/api/v1/science/items/%d/glyph.png?v=%d", item.KpID, item.ContentVersion)
		}
		if item.HasSpeechAudio {
			item.SpeechURL = fmt.Sprintf("/api/v1/science/items/%d/speech.mp3?v=%d", item.KpID, item.ContentVersion)
		}
	}
}
