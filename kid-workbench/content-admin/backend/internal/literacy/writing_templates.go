package literacy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm/clause"
	"io"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const WritingTemplateLimit int64 = 2 << 20

var ErrInvalidWritingTemplate = errors.New("无效书写模板")

type WritingTemplate struct {
	SchemaVersion    int    `json:"schemaVersion"`
	Character        string `json:"character"`
	CoordinateSystem struct {
		Width    int    `json:"width"`
		Height   int    `json:"height"`
		YAxis    string `json:"yAxis"`
		Baseline int    `json:"baseline"`
	} `json:"coordinateSystem"`
	Strokes []string      `json:"strokes"`
	Medians [][][]float64 `json:"medians"`
	Source  struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"source"`
	License string `json:"license"`
}
type WritingTemplateDetail struct {
	KpID             int64           `json:"kpId"`
	Version          string          `json:"version"`
	ValidationStatus string          `json:"validationStatus"`
	Template         WritingTemplate `json:"template"`
}
type writingTemplateRow struct {
	Version   string `gorm:"primaryKey"`
	KpID      int64  `gorm:"primaryKey"`
	Content   string
	CreatedAt time.Time
}

func (writingTemplateRow) TableName() string { return "literacy_writing_templates" }

var svgPath = regexp.MustCompile(`^[MLCQZmlcqz0-9., +\-\r\n\t]+$`)

func decodeWritingTemplate(data []byte, character string) (WritingTemplate, []byte, error) {
	var t WritingTemplate
	invalid := func(reason string) (WritingTemplate, []byte, error) {
		return t, nil, fmt.Errorf("%w: %s", ErrInvalidWritingTemplate, reason)
	}
	if len(data) == 0 || int64(len(data)) > WritingTemplateLimit {
		return invalid("模板大小超限")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return invalid(err.Error())
	}
	if dec.Decode(new(any)) != io.EOF {
		return invalid("多余JSON")
	}
	if t.SchemaVersion != 1 || utf8.RuneCountInString(t.Character) != 1 || t.Character != character {
		return invalid("字符或版本不匹配")
	}
	if t.CoordinateSystem.Width != 1024 || t.CoordinateSystem.Height != 1024 || t.CoordinateSystem.YAxis != "up" || t.CoordinateSystem.Baseline != 900 {
		return invalid("坐标系须为HanziWriter 1024/up/900")
	}
	u, e := url.Parse(t.Source.URL)
	if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || strings.TrimSpace(t.Source.Name) == "" || strings.TrimSpace(t.License) == "" {
		return invalid("缺来源或许可")
	}
	if len(t.Strokes) == 0 || len(t.Strokes) > 64 || len(t.Strokes) != len(t.Medians) {
		return invalid("笔画与中线数量不匹配")
	}
	total := 0
	for i, path := range t.Strokes {
		if len(path) > 32768 || !validWritingPath(path) || !strings.HasPrefix(strings.TrimSpace(path), "M ") || !strings.HasSuffix(strings.ToUpper(strings.TrimSpace(path)), "Z") {
			return invalid("无效笔画路径")
		}
		pts := t.Medians[i]
		if len(pts) < 2 || len(pts) > 512 {
			return invalid("中线点数超限")
		}
		total += len(pts)
		for _, p := range pts {
			if len(p) != 2 {
				return invalid("无效中线坐标")
			}
			for _, v := range p {
				if math.IsNaN(v) || math.IsInf(v, 0) || v < -124 || v > 1024 {
					return invalid("中线坐标超限")
				}
			}
		}
	}
	if total > 8192 {
		return invalid("中线总点数超限")
	}
	b, err := json.Marshal(t)
	return t, b, err
}
func (s *Service) ImportWritingTemplate(ctx context.Context, id int64, data []byte) (WritingTemplateDetail, error) {
	out := WritingTemplateDetail{}
	err := s.withMaterialLocks(ctx, []int64{id}, func(locked *Service) error {
		var a Asset
		if err := locked.db.First(&a, "kp_id = ?", id).Error; err != nil {
			return err
		}
		t, b, err := decodeWritingTemplate(data, a.CharText)
		if err != nil {
			return err
		}
		version := digest(b)
		row := writingTemplateRow{Version: version, KpID: id, Content: string(b), CreatedAt: time.Now().UTC()}
		if err := locked.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
			return err
		}
		if err := locked.db.Model(&a).Update("writing_template_version", version).Error; err != nil {
			return err
		}
		out = WritingTemplateDetail{id, version, "valid", t}
		return nil
	})
	return out, err
}
func (s *Service) WritingTemplate(ctx context.Context, id int64, version string) (WritingTemplateDetail, error) {
	var a Asset
	if err := s.db.WithContext(ctx).First(&a, "kp_id = ?", id).Error; err != nil {
		return WritingTemplateDetail{}, err
	}
	if version == "" {
		version = a.WritingTemplateVersion
	}
	var row writingTemplateRow
	if err := s.db.WithContext(ctx).First(&row, "version = ? AND kp_id = ?", version, id).Error; err != nil {
		return WritingTemplateDetail{}, err
	}
	t, _, err := decodeWritingTemplate([]byte(row.Content), a.CharText)
	if err != nil {
		return WritingTemplateDetail{}, err
	}
	if digest([]byte(row.Content)) != version {
		return WritingTemplateDetail{}, ErrInvalidWritingTemplate
	}
	return WritingTemplateDetail{id, version, "valid", t}, nil
}
func writingReasons(a Asset) []string {
	r := []string{}
	if a.CharText == "" {
		r = append(r, "missing_text")
	}
	if a.SpeechAudioURL == "" {
		r = append(r, "missing_speech_audio")
	}
	if a.WritingTemplateVersion == "" {
		r = append(r, "missing_writing_template")
	}
	if a.MaterialPending {
		r = append(r, "media_update_incomplete")
	}
	return r
}

// Accept the absolute M/L/Q/C/Z subset emitted by Hanzi Writer, with complete
// command groups and bounded finite coordinates. Paths are data, never markup.
func validWritingPath(path string) bool {
	if !svgPath.MatchString(path) {
		return false
	}
	tokens := regexp.MustCompile(`[MLCQZ]|[-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)`).FindAllString(path, -1)
	if strings.Join(tokens, "") != strings.NewReplacer(" ", "", ",", "", "\n", "", "\r", "", "\t", "").Replace(path) {
		return false
	}
	if len(tokens) < 6 || tokens[0] != "M" || tokens[len(tokens)-1] != "Z" {
		return false
	}
	for i := 0; i < len(tokens); {
		cmd := tokens[i]
		i++
		n := 0
		switch cmd {
		case "M", "L":
			n = 2
		case "Q":
			n = 4
		case "C":
			n = 6
		case "Z":
			return i == len(tokens)
		default:
			return false
		}
		start := i
		for i < len(tokens) && !strings.Contains("MLCQZ", tokens[i]) {
			v, err := strconv.ParseFloat(tokens[i], 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < -124 || v > 1024 {
				return false
			}
			i++
		}
		if i == start || (i-start)%n != 0 {
			return false
		}
	}
	return false
}
