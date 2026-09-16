package quiz

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/conchi/study-learning/mastery"
	"gorm.io/gorm"
)

var (
	ErrInvalidType = errors.New("无效的拼音题型")
	ErrNoMaterial  = errors.New("没有足够的拼音素材生成题目")
)

type Visual struct {
	Kind     string `json:"kind"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
	Initial  string `json:"initial,omitempty"`
	Final    string `json:"final,omitempty"`
	Syllable string `json:"syllable,omitempty"`
}

type Option struct {
	ID         int64  `json:"id"`
	Label      string `json:"label,omitempty"`
	SpeechText string `json:"speechText,omitempty"`
	SpeechURL  string `json:"speechUrl,omitempty"`
}

type Question struct {
	InstanceID  string   `json:"instanceId"`
	Type        string   `json:"type"`
	Stem        string   `json:"stem"`
	TargetID    int64    `json:"targetId"`
	SpeechText  string   `json:"speechText,omitempty"`
	SpeechURL   string   `json:"speechUrl,omitempty"`
	Visual      Visual   `json:"visual"`
	Options     []Option `json:"options"`
	AnswerIndex int      `json:"answerIndex"`
}

type letterAsset struct {
	KpID          int64  `gorm:"column:kp_id;primaryKey"`
	Letter        string `gorm:"column:letter"`
	ModuleCode    string `gorm:"column:module_code"`
	SoloText      string `gorm:"column:solo_text"`
	WordText      string `gorm:"column:word_text"`
	WordExamples  string `gorm:"column:word_examples"`
	GlyphImageURL string `gorm:"column:glyph_image_url"`
}

func (letterAsset) TableName() string { return "pinyin_assets" }

type syllableAsset struct {
	ID           int64  `gorm:"column:id"`
	InitialText  string `gorm:"column:initial_text"`
	FinalText    string `gorm:"column:final_text"`
	Tone         int    `gorm:"column:tone"`
	SyllableText string `gorm:"column:syllable_text"`
	SpeechText   string `gorm:"column:speech_text"`
	SpeechURL    string `gorm:"column:speech_url"`
	Enabled      bool   `gorm:"column:enabled"`
}

func (syllableAsset) TableName() string { return "pinyin_syllable_assets" }

var quizConfusion = [][]string{
	{"b", "d", "p", "q"}, {"m", "n", "f", "h"}, {"g", "k", "h", "j"},
	{"j", "q", "x", "y"}, {"zh", "ch", "sh", "r"}, {"z", "c", "s", "zh"},
	{"t", "d", "l", "n"}, {"y", "w", "m", "n"}, {"a", "o", "e", "i"},
	{"i", "u", "ü", "e"}, {"ai", "ei", "ui", "ao"}, {"ao", "ou", "iu", "ai"},
	{"ie", "üe", "er", "iu"}, {"an", "en", "in", "un"},
	{"un", "ün", "in", "en"}, {"ang", "eng", "an", "en"},
}

type Service struct {
	db  *gorm.DB
	cfg mastery.Config
}

func NewService(db *gorm.DB, configs ...mastery.Config) *Service {
	cfg := mastery.DefaultConfig()
	if len(configs) > 0 {
		cfg = configs[0]
	}
	return &Service{db: db, cfg: cfg}
}

func (s *Service) Generate(ctx context.Context, quizType string, excludedTargetIDs []int64) (Question, error) {
	quizType = strings.TrimSpace(quizType)
	switch quizType {
	case "listen", "inword", "shape":
		return s.generateAssetQuiz(ctx, quizType, excludedTargetIDs)
	case "blend":
		return s.generateBlendQuiz(ctx, excludedTargetIDs)
	default:
		return Question{}, ErrInvalidType
	}
}

func (s *Service) generateAssetQuiz(ctx context.Context, quizType string, excluded []int64) (Question, error) {
	var assets []letterAsset
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
		return Question{}, err
	}
	if len(assets) < 4 {
		return Question{}, ErrNoMaterial
	}
	target, err := chooseAsset(assets, excluded)
	if err != nil {
		return Question{}, err
	}
	distractors := assetDistractors(target, assets)
	if len(distractors) < 3 {
		return Question{}, ErrNoMaterial
	}

	options := make([]Option, 0, 4)
	for _, asset := range append([]letterAsset{target}, distractors[:3]...) {
		option := Option{ID: asset.KpID, Label: asset.Letter}
		if quizType == "shape" {
			option.SpeechText = asset.SoloText
			option.SpeechURL = soloPublicURL(asset.KpID)
		}
		options = append(options, option)
	}
	shuffle(options)
	question := Question{
		InstanceID: newInstanceID(), Type: quizType, TargetID: target.KpID,
		Options: options, AnswerIndex: optionIndex(options, target.KpID),
	}
	switch quizType {
	case "listen":
		question.Stem = "听一听，选出你听到的拼音"
		question.SpeechText = target.SoloText
		question.SpeechURL = soloPublicURL(target.KpID)
		question.Visual = Visual{Kind: "sound"}
	case "inword":
		word, index := pickWordExample(target)
		question.Stem = "听一听，这个字里藏着哪个拼音？"
		question.SpeechText = word
		question.SpeechURL = wordExamplePublicURL(target.KpID, index)
		question.Visual = Visual{Kind: "char", Text: word}
	case "shape":
		question.Stem = "看一看，选出四线格里拼音的读音"
		question.Visual = Visual{Kind: "glyph", Text: target.Letter, ImageURL: glyphPublicURL(target.KpID)}
	}
	return question, nil
}

func (s *Service) generateBlendQuiz(ctx context.Context, excluded []int64) (Question, error) {
	var syllables []syllableAsset
	if err := s.db.WithContext(ctx).Where("enabled = ? AND speech_url <> ''", true).Order("id").Find(&syllables).Error; err != nil {
		return Question{}, err
	}
	if len(syllables) < 4 {
		return Question{}, ErrNoMaterial
	}
	target, err := chooseSyllable(syllables, excluded)
	if err != nil {
		return Question{}, err
	}
	pool := make([]syllableAsset, 0, len(syllables)-1)
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
		return Question{}, ErrNoMaterial
	}
	options := make([]Option, 0, 4)
	for _, item := range append([]syllableAsset{target}, pool[:3]...) {
		options = append(options, Option{ID: item.ID, Label: item.SyllableText, SpeechText: item.SpeechText, SpeechURL: item.SpeechURL})
	}
	shuffle(options)
	return Question{
		InstanceID: newInstanceID(), Type: "blend", Stem: "把声母和韵母拼在一起",
		TargetID: target.ID, Options: options, AnswerIndex: optionIndex(options, target.ID),
		Visual: Visual{Kind: "blend", Initial: target.InitialText, Final: target.FinalText, Syllable: target.SyllableText},
	}, nil
}

func pickWordExample(asset letterAsset) (string, int) {
	words := parseWordExamples(asset.WordExamples, asset.WordText)
	if len(words) == 0 {
		return asset.WordText, 0
	}
	index, err := randomIndex(len(words))
	if err != nil {
		return words[0], 0
	}
	return words[index], index
}

func parseWordExamples(raw, fallback string) []string {
	raw = strings.TrimSpace(raw)
	if raw != "" {
		var words []string
		if err := json.Unmarshal([]byte(raw), &words); err == nil && len(words) > 0 {
			return words
		}
	}
	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return []string{fallback}
	}
	return nil
}

func soloPublicURL(kpID int64) string {
	return fmt.Sprintf("/api/v1/pinyin/items/%d/speech/solo.mp3", kpID)
}

func wordPublicURL(kpID int64) string {
	return fmt.Sprintf("/api/v1/pinyin/items/%d/speech/word.mp3", kpID)
}

func wordExamplePublicURL(kpID int64, index int) string {
	if index <= 0 {
		return wordPublicURL(kpID)
	}
	return fmt.Sprintf("/api/v1/pinyin/items/%d/speech/word-%d.mp3", kpID, index)
}

func glyphPublicURL(kpID int64) string {
	return fmt.Sprintf("/api/v1/pinyin/items/%d/glyph.png", kpID)
}

func chooseAsset(items []letterAsset, excluded []int64) (letterAsset, error) {
	available := make([]letterAsset, 0, len(items))
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
		return letterAsset{}, err
	}
	return available[index], nil
}

func chooseSyllable(items []syllableAsset, excluded []int64) (syllableAsset, error) {
	available := make([]syllableAsset, 0, len(items))
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
		return syllableAsset{}, err
	}
	return available[index], nil
}

func assetDistractors(target letterAsset, items []letterAsset) []letterAsset {
	confusing := map[string]bool{}
	for _, group := range quizConfusion {
		if containsString(group, target.Letter) {
			for _, letter := range group {
				confusing[letter] = true
			}
		}
	}
	var first, rest []letterAsset
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
		return 0, ErrNoMaterial
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

func optionIndex(options []Option, id int64) int {
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

func containsSyllable(items []syllableAsset, id int64) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}
