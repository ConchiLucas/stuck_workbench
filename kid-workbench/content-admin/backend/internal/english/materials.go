package english

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/conchi/study-content-admin/internal/storage"
)

type Sentence struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	Code           string    `gorm:"size:64" json:"code"`
	Text           string    `json:"text"`
	TokensJSON     string    `gorm:"column:tokens_json;type:text" json:"-"`
	TargetKpID     int64     `json:"targetKpId"`
	SpeechAudioURL string    `json:"speechAudioUrl"`
	ContentHash    string    `json:"contentHash"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (Sentence) TableName() string { return "english_sentences" }

type Passage struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	Code            string    `gorm:"size:64" json:"code"`
	Passage         string    `gorm:"type:text" json:"passage"`
	Prompt          string    `json:"prompt"`
	AnswerKpID      int64     `json:"answerKpId"`
	OptionKpIDsJSON string    `gorm:"column:option_kp_ids_json;type:text" json:"-"`
	ContentHash     string    `json:"contentHash"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (Passage) TableName() string { return "english_passages" }

type SentenceDTO struct {
	ID             int64    `json:"id"`
	Code           string     `json:"code"`
	Text           string     `json:"text"`
	Tokens         []string `json:"tokens"`
	TargetKpID     int64     `json:"targetKpId"`
	TargetWord     string     `json:"targetWord"`
	SpeechAudioURL string    `json:"speechAudioUrl"`
	ContentHash    string     `json:"contentHash"`
}

type PassageDTO struct {
	ID          int64   `json:"id"`
	Code        string    `json:"code"`
	Passage     string    `json:"passage"`
	Prompt      string    `json:"prompt"`
	AnswerKpID  int64    `json:"answerKpId"`
	AnswerWord  string    `json:"answerWord"`
	OptionKpIDs []int64  `json:"optionKpIds"`
	ContentHash string   `json:"contentHash"`
}

type sentenceSpec struct {
	Code, Text, Target string
	Tokens            []string
}

type passageSpec struct {
	Code, Passage, Prompt, Answer string
	Options                      []string
}

var curatedSentences = []sentenceSpec{
	{Code: "this-is-an-apple", Text: "This is an apple", Target: "apple", Tokens: []string{"This", "is", "an", "apple"}},
	{Code: "this-is-a-cat", Text: "This is a cat", Target: "cat", Tokens: []string{"This", "is", "a", "cat"}},
	{Code: "this-is-a-dog", Text: "This is a dog", Target: "dog", Tokens: []string{"This", "is", "a", "dog"}},
	{Code: "this-is-a-bird", Text: "This is a bird", Target: "bird", Tokens: []string{"This", "is", "a", "bird"}},
	{Code: "this-is-a-banana", Text: "This is a banana", Target: "banana", Tokens: []string{"This", "is", "a", "banana"}},
	{Code: "this-is-a-pencil", Text: "This is a pencil", Target: "pencil", Tokens: []string{"This", "is", "a", "pencil"}},
	{Code: "i-like-the-dog", Text: "I like the dog", Target: "dog", Tokens: []string{"I", "like", "the", "dog"}},
	{Code: "she-has-a-cat", Text: "She has a cat", Target: "cat", Tokens: []string{"She", "has", "a", "cat"}},
	{Code: "we-see-a-bird", Text: "We see a bird", Target: "bird", Tokens: []string{"We", "see", "a", "bird"}},
	{Code: "he-has-a-ball", Text: "He has a ball", Target: "ball", Tokens: []string{"He", "has", "a", "ball"}},
}

var curatedPassages = []passageSpec{
	{Code: "lucy-apple", Passage: "Lucy has a red apple. She puts it on the table.", Prompt: "Lucy 把什么放在桌上？", Answer: "apple", Options: []string{"apple", "dog", "cat", "bird"}},
	{Code: "tom-dog", Passage: "Tom has a small dog. He plays with it in the park.", Prompt: "Tom 有什么？", Answer: "dog", Options: []string{"cat", "dog", "apple", "bird"}},
	{Code: "anna-cat", Passage: "Anna has a black cat. It sleeps on the sofa.", Prompt: "Anna 有什么？", Answer: "cat", Options: []string{"cat", "dog", "apple", "bird"}},
	{Code: "sam-bird", Passage: "Sam sees a blue bird. It sits on the tree.", Prompt: "Sam 看见了什么？", Answer: "bird", Options: []string{"bird", "apple", "dog", "cat"}},
	{Code: "mia-banana", Passage: "Mia likes the yellow banana. She eats it after lunch.", Prompt: "Mia 喜欢什么？", Answer: "banana", Options: []string{"cake", "milk", "banana", "sun"}},
	{Code: "lily-pencil", Passage: "Lily has a new pencil. She writes with it at school.", Prompt: "Lily 用什么写字？", Answer: "pencil", Options: []string{"pencil", "bag", "ball", "book"}},
}

func ContentDigest(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:12])
}

func sentenceSpeechPublicURL(id int64) string {
	return fmt.Sprintf("%s/api/v1/english/sentences/%d/speech.mp3", publicBase(), id)
}

func wordIDByText(db *gorm.DB) (map[string]int64, error) {
	type row struct {
		KpID     int64  `gorm:"column:kp_id"`
		WordText string `gorm:"column:word_text"`
	}
	var rows []row
	err := db.Raw(`
		SELECT kp.id AS kp_id, LOWER(COALESCE(NULLIF(ea.word_text,''), kp.title)) AS word_text
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		JOIN english_assets ea ON ea.kp_id = kp.id
		WHERE s.code = 'english'`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[strings.ToLower(strings.TrimSpace(r.WordText))] = r.KpID
	}
	return out, nil
}

func EnsureQuestionMaterials(db *gorm.DB) error {
	words, err := wordIDByText(db)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, spec := range curatedSentences {
		kpID := words[strings.ToLower(spec.Target)]
		if kpID == 0 {
			continue
		}
		if strings.Join(spec.Tokens, " ") != spec.Text {
			return fmt.Errorf("英语句子素材 %s 词卡与完整句不一致", spec.Code)
		}
		tokens, err := json.Marshal(spec.Tokens)
		if err != nil {
			return err
		}
		hash := ContentDigest(spec.Code, spec.Text, strings.Join(spec.Tokens, " "), spec.Target)
		if err := upsertSentence(db, Sentence{
			Code: spec.Code, Text: spec.Text, TokensJSON: string(tokens),
			TargetKpID: kpID, ContentHash: hash, UpdatedAt: now,
		}); err != nil {
			return err
		}
	}
	for _, spec := range curatedPassages {
		answerID := words[strings.ToLower(spec.Answer)]
		if answerID == 0 {
			continue
		}
		if !strings.Contains(strings.ToLower(spec.Passage), strings.ToLower(spec.Answer)) {
			return fmt.Errorf("英语短文素材 %s 的答案无法从短文得出", spec.Code)
		}
		ids := make([]int64, 0, len(spec.Options))
		ok := true
		for _, option := range spec.Options {
			id := words[strings.ToLower(option)]
			if id == 0 {
				ok = false
				break
			}
			ids = append(ids, id)
		}
		if !ok || len(ids) < 4 {
			continue
		}
		raw, err := json.Marshal(ids)
		if err != nil {
			return err
		}
		hash := ContentDigest(spec.Code, spec.Passage, spec.Prompt, spec.Answer, strings.Join(spec.Options, ","))
		if err := upsertPassage(db, Passage{
			Code: spec.Code, Passage: spec.Passage, Prompt: spec.Prompt,
			AnswerKpID: answerID, OptionKpIDsJSON: string(raw), ContentHash: hash, UpdatedAt: now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func upsertSentence(db *gorm.DB, row Sentence) error {
	var existing Sentence
	err := db.Where("code = ?", row.Code).Take(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&row).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&existing).Updates(map[string]any{
		"text": row.Text, "tokens_json": row.TokensJSON, "target_kp_id": row.TargetKpID,
		"content_hash": row.ContentHash, "updated_at": row.UpdatedAt,
	}).Error
}

func upsertPassage(db *gorm.DB, row Passage) error {
	var existing Passage
	err := db.Where("code = ?", row.Code).Take(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&row).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&existing).Updates(map[string]any{
		"passage": row.Passage, "prompt": row.Prompt, "answer_kp_id": row.AnswerKpID,
		"option_kp_ids_json": row.OptionKpIDsJSON, "content_hash": row.ContentHash, "updated_at": row.UpdatedAt,
	}).Error
}

func (s *Service) ListSentences() ([]SentenceDTO, error) {
	if err := EnsureQuestionMaterials(s.db); err != nil {
		return nil, err
	}
	var rows []Sentence
	if err := s.db.Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	words := map[int64]string{}
	var assets []Asset
	if err := s.db.Find(&assets).Error; err != nil {
		return nil, err
	}
	for _, a := range assets {
		words[a.KpID] = a.WordText
	}
	out := make([]SentenceDTO, 0, len(rows))
	for _, row := range rows {
		var tokens []string
		_ = json.Unmarshal([]byte(row.TokensJSON), &tokens)
		out = append(out, SentenceDTO{
			ID: row.ID, Code: row.Code, Text: row.Text, Tokens: tokens,
			TargetKpID: row.TargetKpID, TargetWord: words[row.TargetKpID],
			SpeechAudioURL: row.SpeechAudioURL, ContentHash: row.ContentHash,
		})
	}
	return out, nil
}

func (s *Service) ListPassages() ([]PassageDTO, error) {
	if err := EnsureQuestionMaterials(s.db); err != nil {
		return nil, err
	}
	var rows []Passage
	if err := s.db.Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	words := map[int64]string{}
	var assets []Asset
	if err := s.db.Find(&assets).Error; err != nil {
		return nil, err
	}
	for _, a := range assets {
		words[a.KpID] = a.WordText
	}
	out := make([]PassageDTO, 0, len(rows))
	for _, row := range rows {
		var ids []int64
		_ = json.Unmarshal([]byte(row.OptionKpIDsJSON), &ids)
		out = append(out, PassageDTO{
			ID: row.ID, Code: row.Code, Passage: row.Passage, Prompt: row.Prompt,
			AnswerKpID: row.AnswerKpID, AnswerWord: words[row.AnswerKpID],
			OptionKpIDs: ids, ContentHash: row.ContentHash,
		})
	}
	return out, nil
}

func (s *Service) SentenceSpeechMP3(ctx context.Context, idOrCode string) ([]byte, error) {
	row, err := s.findSentence(idOrCode)
	if err != nil {
		return nil, err
	}
	if s.store == nil || strings.TrimSpace(row.SpeechAudioURL) == "" {
		return nil, fmt.Errorf("整句读音尚未准备")
	}
	data, err := s.store.GetBytes(ctx, s.store.EnglishSentenceSpeechKey(row.ID))
	if err != nil || len(data) == 0 {
		return nil, fmt.Errorf("整句读音尚未准备")
	}
	return data, nil
}

func (s *Service) StoreSentenceSpeech(ctx context.Context, idOrCode string, mp3 []byte) (SentenceDTO, error) {
	row, err := s.findSentence(idOrCode)
	if err != nil {
		return SentenceDTO{}, err
	}
	if s.store == nil {
		return SentenceDTO{}, fmt.Errorf("语音存储未就绪（检查 MinIO）")
	}
	if len(mp3) < 16 {
		return SentenceDTO{}, fmt.Errorf("整句读音无效")
	}
	if _, err := s.store.PutBytes(ctx, storage.EnglishSentenceSpeechObjectKey(row.ID), mp3, "audio/mpeg"); err != nil {
		return SentenceDTO{}, err
	}
	row.SpeechAudioURL = sentenceSpeechPublicURL(row.ID)
	row.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(&row).Error; err != nil {
		return SentenceDTO{}, err
	}
	var tokens []string
	_ = json.Unmarshal([]byte(row.TokensJSON), &tokens)
	return SentenceDTO{
		ID: row.ID, Code: row.Code, Text: row.Text, Tokens: tokens,
		TargetKpID: row.TargetKpID, SpeechAudioURL: row.SpeechAudioURL, ContentHash: row.ContentHash,
	}, nil
}

func (s *Service) findSentence(idOrCode string) (Sentence, error) {
	var row Sentence
	idOrCode = strings.TrimSpace(idOrCode)
	if idOrCode == "" {
		return Sentence{}, gorm.ErrRecordNotFound
	}
	q := s.db
	if idOrCode[0] >= '0' && idOrCode[0] <= '9' {
		q = q.Where("id = ?", idOrCode)
	} else {
		q = q.Where("code = ?", idOrCode)
	}
	if err := q.First(&row).Error; err != nil {
		return Sentence{}, err
	}
	return row, nil
}
