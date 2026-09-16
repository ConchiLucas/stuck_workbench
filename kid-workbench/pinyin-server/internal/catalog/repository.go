package catalog

import (
	"context"

	"gorm.io/gorm"
)

const pinyinSubject = "pinyin"

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ListModules(ctx context.Context) ([]Module, error) {
	modules := make([]Module, 0)
	err := r.db.WithContext(ctx).Raw(`
		SELECT m.code, m.name, m.order_no, COUNT(kp.id) AS item_count
		FROM modules m
		JOIN subjects s ON s.id = m.subject_id
		JOIN knowledge_points kp ON kp.module_id = m.id
		WHERE s.code = ? AND m.code <> 'syllables'
		GROUP BY m.id, m.code, m.name, m.order_no
		ORDER BY m.order_no, m.id`, pinyinSubject).Scan(&modules).Error
	return modules, err
}

func (r *Repository) ListItems(ctx context.Context, moduleCode string) ([]Item, error) {
	items := make([]Item, 0)
	err := r.itemQuery(ctx).Where("m.code = ?", moduleCode).
		Order("m.order_no, kp.order_no, kp.id").Scan(&items).Error
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
	return item, nil
}

func (r *Repository) itemQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table("knowledge_points kp").
		Select(`kp.id AS kp_id,
			COALESCE(NULLIF(pa.letter, ''), kp.title) AS letter,
			m.code AS module_code, m.name AS module_name,
			COALESCE(pa.solo_text, '') AS solo_text,
			COALESCE(pa.word_text, '') AS word_text,
			CASE WHEN COALESCE(pa.solo_speech_url, '') <> '' THEN TRUE ELSE FALSE END AS has_solo_speech,
			CASE WHEN COALESCE(pa.word_speech_url, '') <> '' THEN TRUE ELSE FALSE END AS has_word_speech,
			CASE WHEN COALESCE(pa.glyph_image_url, '') <> '' THEN TRUE ELSE FALSE END AS has_glyph,
			kp.order_no AS order_no`).
		Joins("JOIN modules m ON m.id = kp.module_id").
		Joins("JOIN subjects s ON s.id = m.subject_id").
		Joins("LEFT JOIN pinyin_assets pa ON pa.kp_id = kp.id").
		Where("s.code = ? AND m.code <> ?", pinyinSubject, "syllables")
}
