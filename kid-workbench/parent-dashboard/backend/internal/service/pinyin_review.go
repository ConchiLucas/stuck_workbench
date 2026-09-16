package service

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
)

type PinyinReview struct {
	Question          *PinyinReviewQuestion `json:"question,omitempty"`
	SelectedOptionID  string                `json:"selected_option_id,omitempty"`
	UnavailableReason string                `json:"unavailable_reason,omitempty"`
}

type PinyinReviewQuestion struct {
	ID        string               `json:"id"`
	Type      string               `json:"type"`
	Stem      string               `json:"stem,omitempty"`
	SpeechURL string               `json:"speechUrl,omitempty"`
	Visual    PinyinReviewVisual   `json:"visual"`
	Options   []PinyinReviewOption `json:"options"`
}

type PinyinReviewVisual struct {
	Kind     string `json:"kind"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
	Initial  string `json:"initial,omitempty"`
	Final    string `json:"final,omitempty"`
	Syllable string `json:"syllable,omitempty"`
}

type PinyinReviewOption struct {
	ID        string `json:"id"`
	Label     string `json:"label,omitempty"`
	SpeechURL string `json:"speechUrl,omitempty"`
}

var pinyinMediaPath = regexp.MustCompile(`^/api/v1/pinyin/items/([1-9][0-9]*)/(speech/(?:solo|word|word-[1-4])\.mp3|glyph\.png)$`)

var pinyinSyllableVersion = regexp.MustCompile(`^v=[0-9a-f]{16}$`)
var pinyinSyllableMediaPath = regexp.MustCompile(`^/api/v1/pinyin/syllables/([1-9][0-9]*)/speech\.mp3$`)

func unavailablePinyinReview(reason string) *PinyinReview {
	return &PinyinReview{UnavailableReason: reason}
}

func rewritePinyinMedia(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil || u.Fragment != "" {
		return ""
	}
	path := u.Path
	if path == "" {
		path = raw
	}
	if pinyinSyllableMediaPath.MatchString(path) {
		if u.RawQuery != "" && !pinyinSyllableVersion.MatchString(u.RawQuery) {
			return ""
		}
		result := strings.Replace(path, "/api/v1/pinyin/", "/api/pinyin/", 1)
		if u.RawQuery != "" {
			result += "?" + u.RawQuery
		}
		return result
	}
	if u.RawQuery != "" {
		return ""
	}
	m := pinyinMediaPath.FindStringSubmatch(path)
	if m == nil {
		return ""
	}
	return "/api/pinyin/items/" + m[1] + "/" + strings.TrimPrefix(path, "/api/v1/pinyin/items/"+m[1]+"/")
}

func buildPinyinReview(raw, selected string) *PinyinReview {
	if strings.TrimSpace(raw) == "" {
		return unavailablePinyinReview("未保存可信题目快照，无法还原历史题面")
	}
	var s struct {
		InstanceID string
		Type       string
		Stem       string
		SpeechURL  string
		Visual     PinyinReviewVisual
		Options    []PinyinReviewOption
	}
	if json.Unmarshal([]byte(raw), &s) != nil {
		return unavailablePinyinReview("历史题目快照缺失或无效")
	}
	switch s.Type {
	case "listen", "inword", "shape", "blend":
	default:
		return unavailablePinyinReview("历史题目快照缺失或无效")
	}
	if len(s.Options) != 4 {
		return unavailablePinyinReview("历史选项标识无效")
	}
	if s.Type == "inword" && strings.TrimSpace(s.Visual.Text) == "" {
		return unavailablePinyinReview("历史题干缺失")
	}
	if s.Type == "shape" && strings.TrimSpace(s.Visual.Text) == "" {
		return unavailablePinyinReview("历史题干缺失")
	}
	if s.Type == "blend" && (strings.TrimSpace(s.Visual.Initial) == "" || strings.TrimSpace(s.Visual.Final) == "") {
		return unavailablePinyinReview("历史题干缺失")
	}
	seen := map[string]bool{}
	options := make([]PinyinReviewOption, 0, 4)
	for _, option := range s.Options {
		if option.ID == "" || seen[option.ID] {
			return unavailablePinyinReview("历史选项标识无效")
		}
		seen[option.ID] = true
		if s.Type != "shape" && s.Type != "blend" && strings.TrimSpace(option.Label) == "" {
			return unavailablePinyinReview("历史选项媒体或文字缺失")
		}
		options = append(options, PinyinReviewOption{
			ID:        option.ID,
			Label:     option.Label,
			SpeechURL: rewritePinyinMedia(option.SpeechURL),
		})
	}
	if selected != "" && !seen[selected] {
		return unavailablePinyinReview("历史答案或作答选项标识无效")
	}
	id := strings.TrimSpace(s.InstanceID)
	if id == "" {
		id = "history"
	}
	return &PinyinReview{
		SelectedOptionID: selected,
		Question: &PinyinReviewQuestion{
			ID:        id,
			Type:      s.Type,
			Stem:      s.Stem,
			SpeechURL: rewritePinyinMedia(s.SpeechURL),
			Visual: PinyinReviewVisual{
				Kind:     s.Visual.Kind,
				Text:     s.Visual.Text,
				ImageURL: rewritePinyinMedia(s.Visual.ImageURL),
				Initial:  s.Visual.Initial,
				Final:    s.Visual.Final,
				Syllable: s.Visual.Syllable,
			},
			Options: options,
		},
	}
}
