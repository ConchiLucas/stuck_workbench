package plan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/conchi/study-learning/handwriting"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var ErrTemplateUnavailable = errors.New("writing template unavailable")

type CachedTemplate struct {
	RevisionID   string `gorm:"primaryKey"`
	SHA256       string
	TemplateJSON string
}

func (CachedTemplate) TableName() string { return "literacy_writing_template_cache" }
func (s *Service) cacheRevisionTemplates(ctx context.Context, revisionID int64) error {
	var rows []struct{ SnapshotJSON string }
	if err := s.db.WithContext(ctx).Table("question_versions").Select("snapshot_json").Where("revision_id = ?", revisionID).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		snap, err := ParseSnapshot(row.SnapshotJSON)
		if err != nil {
			return err
		}
		if snap.WritingTemplate == nil {
			continue
		}
		ref := snap.WritingTemplate
		var cached CachedTemplate
		if err := s.db.WithContext(ctx).Where("revision_id = ? AND sha256 = ?", ref.RevisionID, ref.SHA256).First(&cached).Error; err == nil {
			if err := ValidateCachedTemplate(cached, snap); err != nil {
				return err
			}
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if len(ref.RevisionID) != 64 || len(ref.SHA256) != 64 {
			return ErrTemplateUnavailable
		}
		if _, err := hex.DecodeString(ref.RevisionID); err != nil {
			return ErrTemplateUnavailable
		}
		base := strings.TrimRight(os.Getenv("APP_CONTENT_ADMIN_URL"), "/")
		if base == "" {
			base = "http://localhost:19091"
		}
		req, err := http.NewRequestWithContext(ctx, "GET", base+"/api/v1/material-revisions/"+ref.RevisionID+"/media/writing_template", nil)
		if err != nil {
			return ErrTemplateUnavailable
		}
		client := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		resp, err := client.Do(req)
		if err != nil {
			return ErrTemplateUnavailable
		}
		b, readErr := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
		resp.Body.Close()
		if readErr != nil || resp.StatusCode != 200 || len(b) > 2*1024*1024 {
			return ErrTemplateUnavailable
		}
		hash := sha256.Sum256(b)
		if fmt.Sprintf("%x", hash) != ref.SHA256 {
			return ErrTemplateUnavailable
		}
		var template handwriting.Template
		if json.Unmarshal(b, &template) != nil || handwriting.ValidateTemplate(template) != nil || template.Character != snap.TargetText {
			return ErrTemplateUnavailable
		}
		cached = CachedTemplate{RevisionID: ref.RevisionID, SHA256: ref.SHA256, TemplateJSON: string(b)}
		if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&cached).Error; err != nil {
			return err
		}
	}
	return nil
}

// ValidateCachedTemplate verifies immutable bytes again on cache hits, so corrupt
// or mismatched entries never create an unanswerable learning plan.
func ValidateCachedTemplate(cached CachedTemplate, snap Snapshot) error {
	if snap.WritingTemplate == nil || cached.RevisionID != snap.WritingTemplate.RevisionID || cached.SHA256 != snap.WritingTemplate.SHA256 {
		return ErrTemplateUnavailable
	}
	hash := sha256.Sum256([]byte(cached.TemplateJSON))
	if fmt.Sprintf("%x", hash) != cached.SHA256 {
		return ErrTemplateUnavailable
	}
	var template handwriting.Template
	if json.Unmarshal([]byte(cached.TemplateJSON), &template) != nil || handwriting.ValidateTemplate(template) != nil || template.Character != snap.TargetText {
		return ErrTemplateUnavailable
	}
	return nil
}
