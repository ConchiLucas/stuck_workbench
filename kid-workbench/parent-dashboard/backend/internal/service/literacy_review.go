package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type LiteracyReview struct {
	Question          *LiteracyReviewQuestion `json:"question,omitempty"`
	SelectedOptionID  string                  `json:"selected_option_id,omitempty"`
	AnswerOptionID    string                  `json:"answer_option_id,omitempty"`
	UnavailableReason string                  `json:"unavailable_reason,omitempty"`
}
type LiteracyReviewQuestion struct {
	ID           string                 `json:"id"`
	QuestionType string                 `json:"questionType"`
	Interaction  string                 `json:"interaction"`
	Stem         LiteracyReviewStem     `json:"stem"`
	Options      []LiteracyReviewOption `json:"options"`
}
type LiteracyReviewStem struct {
	Text  string `json:"text,omitempty"`
	Image string `json:"image,omitempty"`
}
type LiteracyReviewOption struct {
	ID    string `json:"id"`
	Text  string `json:"text,omitempty"`
	Image string `json:"image,omitempty"`
	Audio string `json:"audio,omitempty"`
}
type frozenMedia struct{ RevisionID, Kind, SHA256 string }

var revisionPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

func reviewMedia(ref *frozenMedia, audio bool) (string, error) {
	if ref == nil {
		return "", nil
	}
	if !revisionPattern.MatchString(ref.RevisionID) || !revisionPattern.MatchString(ref.SHA256) || (audio && ref.Kind != "speech") || (!audio && ref.Kind != "glyph" && ref.Kind != "sense") {
		return "", fmt.Errorf("历史媒体引用缺失或无效")
	}
	return "/api/literacy/material-revisions/" + ref.RevisionID + "/media/" + ref.Kind, nil
}
func unavailableReview(reason string) *LiteracyReview {
	return &LiteracyReview{UnavailableReason: reason}
}
func buildLiteracyReview(version int64, raw, order, selected string) *LiteracyReview {
	if version <= 0 {
		return unavailableReview("未保存可信版本快照，无法还原历史题面")
	}
	var s struct {
		SchemaVersion                                                      int
		SubjectCode, QuestionType, Interaction, AnswerOptionID, TargetText string
		Stem                                                               struct {
			Text         string
			Image, Audio *frozenMedia
		}
		Options []struct {
			ID, Text     string
			Image, Audio *frozenMedia
		}
	}
	if json.Unmarshal([]byte(raw), &s) != nil || (s.SchemaVersion != 1 && s.SchemaVersion != 2) || s.SubjectCode != "literacy" || s.QuestionType != "glyph_sense" || (s.Interaction != "" && s.Interaction != "choice") || len(s.Options) != 4 {
		return unavailableReview("历史题目快照缺失或无效")
	}
	q := &LiteracyReviewQuestion{ID: strconv.FormatInt(version, 10), QuestionType: "glyph_sense", Interaction: "choice", Stem: LiteracyReviewStem{Text: s.Stem.Text}, Options: []LiteracyReviewOption{}}
	if q.Stem.Text == "" {
		q.Stem.Text = s.TargetText
	}
	if s.Stem.Image == nil || s.Stem.Image.Kind != "glyph" {
		return unavailableReview("历史字图缺失或类型无效")
	}
	var err error
	if q.Stem.Image, err = reviewMedia(s.Stem.Image, false); err != nil {
		return unavailableReview(err.Error())
	}
	if s.Stem.Audio != nil {
		return unavailableReview("历史题干包含当前题型不支持的冻结音频")
	}
	if strings.TrimSpace(q.Stem.Text) == "" && q.Stem.Image == "" {
		return unavailableReview("历史题干缺失")
	}
	seen := map[string]bool{}
	for _, o := range s.Options {
		if o.ID == "" || seen[o.ID] {
			return unavailableReview("历史选项标识无效")
		}
		seen[o.ID] = true
		if o.Image == nil || o.Image.Kind != "sense" {
			return unavailableReview("历史选项义图缺失或类型无效")
		}
		image, e := reviewMedia(o.Image, false)
		if e != nil {
			return unavailableReview(e.Error())
		}
		audio, e := reviewMedia(o.Audio, true)
		if e != nil {
			return unavailableReview(e.Error())
		}
		if strings.TrimSpace(o.Text) == "" && image == "" && audio == "" {
			return unavailableReview("历史选项媒体或文字缺失")
		}
		q.Options = append(q.Options, LiteracyReviewOption{ID: o.ID, Text: o.Text, Image: image, Audio: audio})
	}
	if !seen[s.AnswerOptionID] || (selected != "" && !seen[selected]) {
		return unavailableReview("历史答案或作答选项标识无效")
	}
	if order != "" {
		parts := strings.Split(order, ",")
		if len(parts) != len(q.Options) {
			return unavailableReview("历史选项顺序无效")
		}
		ordered := make([]LiteracyReviewOption, len(parts))
		used := map[int]bool{}
		for i, p := range parts {
			j, e := strconv.Atoi(strings.TrimSpace(p))
			if e != nil || j < 0 || j >= len(parts) || used[j] {
				return unavailableReview("历史选项顺序无效")
			}
			used[j] = true
			ordered[i] = q.Options[j]
		}
		q.Options = ordered
	}
	return &LiteracyReview{Question: q, SelectedOptionID: selected, AnswerOptionID: s.AnswerOptionID}
}
func snapshotGlyph(raw string) bool {
	var s struct{ QuestionType string }
	return json.Unmarshal([]byte(raw), &s) == nil && s.QuestionType == "glyph_sense"
}

func snapshotTypeMissing(raw string) bool {
	var s struct{ QuestionType string }
	return json.Unmarshal([]byte(raw), &s) != nil || s.QuestionType == ""
}
