package math

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/conchi/study-learning/mathcontent"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrDetailConflict = errors.New("素材版本已变化，请刷新后重试")

type detailDraft struct {
	ID                string `gorm:"primaryKey;size:80"`
	Revision          int
	PublishedRevision int
	OrderNo           int
	Content           string `gorm:"type:text"`
	UpdatedAt         time.Time
}

func (detailDraft) TableName() string { return "math_detail_drafts" }

type detailRevision struct {
	ID          string `gorm:"primaryKey;size:80"`
	Revision    int    `gorm:"primaryKey;autoIncrement:false"`
	Content     string `gorm:"type:text"`
	PublishedAt time.Time
}

func (detailRevision) TableName() string { return "math_detail_revisions" }

func (s *Service) InitializeDetails() error {
	if err := s.db.AutoMigrate(&detailDraft{}, &detailRevision{}); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		for i, d := range mathcontent.Defaults().Items {
			if e := mathcontent.Validate(d); e != nil {
				return e
			}
			raw, e := json.Marshal(d)
			if e != nil {
				return e
			}
			row := detailDraft{ID: d.ID, Revision: 1, PublishedRevision: 1, OrderNo: i, Content: string(raw), UpdatedAt: time.Now()}
			res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected > 0 {
				if e := tx.Create(&detailRevision{ID: d.ID, Revision: 1, Content: string(raw), PublishedAt: time.Now()}).Error; e != nil {
					return e
				}
			}
		}
		return nil
	})
}

func (s *Service) Details(ctx context.Context, published bool) (mathcontent.Catalog, error) {
	out := mathcontent.Catalog{SchemaVersion: 1, Items: []mathcontent.MathDetail{}}
	var rows []detailDraft
	if e := s.db.WithContext(ctx).Order("order_no,id").Find(&rows).Error; e != nil {
		return out, e
	}
	for _, r := range rows {
		content := r.Content
		if published {
			if r.PublishedRevision == 0 {
				continue
			}
			var version detailRevision
			if e := s.db.WithContext(ctx).Where("id=? AND revision=?", r.ID, r.PublishedRevision).First(&version).Error; e != nil {
				return out, e
			}
			content = version.Content
		}
		var d mathcontent.MathDetail
		if e := json.Unmarshal([]byte(content), &d); e != nil {
			return out, e
		}
		d.PublishedRevision = r.PublishedRevision
		out.Items = append(out.Items, d)
	}
	return out, nil
}

func (s *Service) SaveDetail(ctx context.Context, d mathcontent.MathDetail) (mathcontent.MathDetail, error) {
	if e := mathcontent.Validate(d); e != nil {
		return d, e
	}
	expected := d.Revision
	d.Revision++
	d.PublishedRevision = 0
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row detailDraft
		if e := tx.Where("id=?", d.ID).First(&row).Error; e != nil {
			return e
		}
		if row.Revision != expected {
			return ErrDetailConflict
		}
		var old mathcontent.MathDetail
		if e := json.Unmarshal([]byte(row.Content), &old); e != nil {
			return e
		}
		if old.GroupID != d.GroupID || old.Example.Kind != d.Example.Kind {
			return errors.New("不能改变现有详情的分组或题型身份")
		}
		if detailSpeechText(old) != detailSpeechText(d) && d.Example.AudioURL == old.Example.AudioURL {
			d.Example.AudioURL = ""
		}
		raw, e := json.Marshal(d)
		if e != nil {
			return e
		}
		res := tx.Model(&detailDraft{}).Where("id=? AND revision=?", d.ID, expected).Updates(map[string]any{"revision": d.Revision, "content": string(raw), "updated_at": time.Now()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrDetailConflict
		}
		d.PublishedRevision = row.PublishedRevision
		return nil
	})
	return d, err
}

func (s *Service) PublishDetail(ctx context.Context, id string, revision int) (mathcontent.MathDetail, error) {
	var d mathcontent.MathDetail
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row detailDraft
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", id).First(&row).Error; e != nil {
			return e
		}
		if row.Revision != revision {
			return ErrDetailConflict
		}
		if e := json.Unmarshal([]byte(row.Content), &d); e != nil {
			return e
		}
		if e := mathcontent.Validate(d); e != nil {
			return e
		}
		if e := s.verifyDetailAudio(ctx, d); e != nil {
			return e
		}
		if e := s.verifyDetailImages(ctx, d); e != nil {
			return e
		}
		if row.PublishedRevision == revision {
			d.PublishedRevision = revision
			return nil
		}
		if e := tx.Create(&detailRevision{ID: id, Revision: revision, Content: row.Content, PublishedAt: time.Now()}).Error; e != nil {
			return e
		}
		res := tx.Model(&detailDraft{}).Where("id=? AND revision=?", id, revision).Update("published_revision", revision)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrDetailConflict
		}
		d.PublishedRevision = revision
		return nil
	})
	return d, err
}
