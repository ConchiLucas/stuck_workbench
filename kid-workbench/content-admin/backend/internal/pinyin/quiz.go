package pinyin

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

var (
	ErrInvalidQuizType = errors.New("无效的拼音题型")
	ErrNoQuizMaterial  = errors.New("没有足够的拼音素材生成题目")
)

type QuizVisual struct {
	Kind     string `json:"kind"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
	Initial  string `json:"initial,omitempty"`
	Final    string `json:"final,omitempty"`
	Syllable string `json:"syllable,omitempty"`
}

type QuizOption struct {
	ID         int64  `json:"id"`
	Label      string `json:"label,omitempty"`
	SpeechText string `json:"speechText,omitempty"`
	SpeechURL  string `json:"speechUrl,omitempty"`
}

type QuizQuestion struct {
	InstanceID  string       `json:"instanceId"`
	Type        string       `json:"type"`
	Stem        string       `json:"stem"`
	TargetID    int64        `json:"targetId"`
	SpeechText  string       `json:"speechText,omitempty"`
	SpeechURL   string       `json:"speechUrl,omitempty"`
	Visual      QuizVisual   `json:"visual"`
	Options     []QuizOption `json:"options"`
	AnswerIndex int          `json:"answerIndex"`
}

type SyllableAsset struct {
	ID           int64  `gorm:"column:id" json:"id"`
	InitialText  string `gorm:"column:initial_text" json:"initialText"`
	FinalText    string `gorm:"column:final_text" json:"finalText"`
	Tone         int    `gorm:"column:tone" json:"tone"`
	SyllableText string `gorm:"column:syllable_text" json:"syllableText"`
	SpeechText   string `gorm:"column:speech_text" json:"speechText"`
	SpeechURL    string `gorm:"column:speech_url" json:"speechUrl"`
	Enabled      bool   `gorm:"column:enabled" json:"enabled"`
}

func (SyllableAsset) TableName() string { return "pinyin_syllable_assets" }

var quizConfusion = [][]string{
	{"b", "d", "p", "q"}, {"m", "n", "f", "h"}, {"g", "k", "h", "j"},
	{"j", "q", "x", "y"}, {"zh", "ch", "sh", "r"}, {"z", "c", "s", "zh"},
	{"t", "d", "l", "n"}, {"y", "w", "m", "n"}, {"a", "o", "e", "i"},
	{"i", "u", "ü", "e"}, {"ai", "ei", "ui", "ao"}, {"ao", "ou", "iu", "ai"},
	{"ie", "üe", "er", "iu"}, {"an", "en", "in", "un"},
	{"un", "ün", "in", "en"}, {"ang", "eng", "an", "en"},
}

func (s *Service) GenerateQuiz(ctx context.Context, quizType string, excludedTargetIDs []int64) (QuizQuestion, error) {
	quizType = strings.TrimSpace(quizType)
	switch quizType {
	case "listen", "inword", "shape":
		return s.generateAssetQuiz(ctx, quizType, excludedTargetIDs)
	case "blend":
		return s.generateBlendQuiz(ctx, excludedTargetIDs)
	default:
		return QuizQuestion{}, ErrInvalidQuizType
	}
}

func (s *Service) generateAssetQuiz(ctx context.Context, quizType string, excluded []int64) (QuizQuestion, error) {
	var assets []Asset
	query := s.db.WithContext(ctx).Order("module_order, kp_order, kp_id")
	switch quizType {
	case "listen":
		query = query.Where("solo_text <> ''")
	case "inword":
		query = query.Where("word_text <> ''")
	case "shape":
		query = query.Where("solo_text <> '' AND glyph_image_url <> ''")
	}
	if err := query.Find(&assets).Error; err != nil {
		return QuizQuestion{}, err
	}
	if len(assets) < 4 {
		return QuizQuestion{}, ErrNoQuizMaterial
	}
	target, err := chooseAsset(assets, excluded)
	if err != nil {
		return QuizQuestion{}, err
	}
	distractors := assetDistractors(target, assets)
	if len(distractors) < 3 {
		return QuizQuestion{}, ErrNoQuizMaterial
	}

	options := make([]QuizOption, 0, 4)
	for _, asset := range append([]Asset{target}, distractors[:3]...) {
		option := QuizOption{ID: asset.KpID, Label: asset.Letter}
		if quizType == "shape" {
			option.SpeechText = asset.SoloText
			option.SpeechURL = soloPublicURL(asset.KpID)
		}
		options = append(options, option)
	}
	shuffle(options)
	answerIndex := optionIndex(options, target.KpID)
	question := QuizQuestion{
		InstanceID: newInstanceID(), Type: quizType, TargetID: target.KpID,
		Options: options, AnswerIndex: answerIndex,
	}
	switch quizType {
	case "listen":
		question.Stem = "听一听，选出你听到的拼音"
		question.SpeechText = target.SoloText
		question.SpeechURL = soloPublicURL(target.KpID)
		question.Visual = QuizVisual{Kind: "sound"}
	case "inword":
		word, index := pickWordExample(target)
		question.Stem = "听一听，这个字里藏着哪个拼音？"
		question.SpeechText = word
		question.SpeechURL = wordExamplePublicURL(target.KpID, index)
		question.Visual = QuizVisual{Kind: "char", Text: word}
	case "shape":
		question.Stem = "看一看，选出四线格里拼音的读音"
		question.Visual = QuizVisual{Kind: "glyph", Text: target.Letter, ImageURL: glyphPublicURL(target.KpID)}
	}
	return question, nil
}

func (s *Service) generateBlendQuiz(ctx context.Context, excluded []int64) (QuizQuestion, error) {
	var syllables []SyllableAsset
	if err := s.db.WithContext(ctx).Where("enabled = ? AND speech_url <> ''", true).Order("id").Find(&syllables).Error; err != nil {
		return QuizQuestion{}, err
	}
	if len(syllables) < 4 {
		return QuizQuestion{}, ErrNoQuizMaterial
	}
	target, err := chooseSyllable(syllables, excluded)
	if err != nil {
		return QuizQuestion{}, err
	}
	pool := make([]SyllableAsset, 0, len(syllables)-1)
	for _, item := range syllables {
		if item.ID != target.ID && item.FinalText == target.FinalText && item.Tone == target.Tone {
			pool = append(pool, item)
		}
	}
	shuffle(pool)
	for _, item := range syllables {
		if item.ID == target.ID || containsSyllable(pool, item.ID) {
			continue
		}
		pool = append(pool, item)
	}
	if len(pool) < 3 {
		return QuizQuestion{}, ErrNoQuizMaterial
	}
	options := make([]QuizOption, 0, 4)
	for _, item := range append([]SyllableAsset{target}, pool[:3]...) {
		options = append(options, QuizOption{ID: item.ID, Label: item.SyllableText, SpeechText: item.SpeechText, SpeechURL: item.SpeechURL})
	}
	shuffle(options)
	return QuizQuestion{
		InstanceID: newInstanceID(), Type: "blend", Stem: "把声母和韵母拼在一起",
		TargetID: target.ID, Options: options, AnswerIndex: optionIndex(options, target.ID),
		Visual: QuizVisual{Kind: "blend", Initial: target.InitialText, Final: target.FinalText, Syllable: target.SyllableText},
	}, nil
}

func pickWordExample(asset Asset) (string, int) {
	words := ParseWordExamples(asset.WordExamples, asset.WordText)
	if len(words) == 0 {
		return asset.WordText, 0
	}
	index, err := randomIndex(len(words))
	if err != nil {
		return words[0], 0
	}
	return words[index], index
}

func chooseAsset(items []Asset, excluded []int64) (Asset, error) {
	available := make([]Asset, 0, len(items))
	for _, item := range items {
		if !containsID(excluded, item.KpID) {
			available = append(available, item)
		}
	}
	if len(available) == 0 {
		available = items
	}
	index, err := randomIndex(len(available))
	if err != nil {
		return Asset{}, err
	}
	return available[index], nil
}

func chooseSyllable(items []SyllableAsset, excluded []int64) (SyllableAsset, error) {
	available := make([]SyllableAsset, 0, len(items))
	for _, item := range items {
		if !containsID(excluded, item.ID) {
			available = append(available, item)
		}
	}
	if len(available) == 0 {
		available = items
	}
	index, err := randomIndex(len(available))
	if err != nil {
		return SyllableAsset{}, err
	}
	return available[index], nil
}

func assetDistractors(target Asset, items []Asset) []Asset {
	confusing := map[string]bool{}
	for _, group := range quizConfusion {
		if containsString(group, target.Letter) {
			for _, letter := range group {
				confusing[letter] = true
			}
		}
	}
	var first, rest []Asset
	for _, item := range items {
		if item.KpID == target.KpID || item.ModuleCode != target.ModuleCode {
			continue
		}
		if confusing[item.Letter] {
			first = append(first, item)
		} else {
			rest = append(rest, item)
		}
	}
	shuffle(first)
	shuffle(rest)
	return append(first, rest...)
}

func randomIndex(size int) (int, error) {
	if size <= 0 {
		return 0, ErrNoQuizMaterial
	}
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(size)))
	if err != nil {
		return 0, fmt.Errorf("生成随机题目失败: %w", err)
	}
	return int(value.Int64()), nil
}

func shuffle[T any](items []T) {
	for i := len(items) - 1; i > 0; i-- {
		j, err := randomIndex(i + 1)
		if err == nil {
			items[i], items[j] = items[j], items[i]
		}
	}
}

func newInstanceID() string {
	buf := make([]byte, 12)
	if _, err := cryptorand.Read(buf); err != nil {
		return fmt.Sprintf("quiz-%d", len(buf))
	}
	return hex.EncodeToString(buf)
}

func optionIndex(options []QuizOption, id int64) int {
	for index, option := range options {
		if option.ID == id {
			return index
		}
	}
	return -1
}

func containsID(items []int64, id int64) bool {
	for _, item := range items {
		if item == id {
			return true
		}
	}
	return false
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func containsSyllable(items []SyllableAsset, id int64) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}
