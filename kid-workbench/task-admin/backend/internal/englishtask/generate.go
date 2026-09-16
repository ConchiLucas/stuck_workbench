package englishtask

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/conchi/study-learning/englishcontent"
)

type sentence struct {
	ID          int64
	Code        string
	Text        string
	Tokens      []string
	TargetKpID  int64
	ContentHash string
	ModuleCode  string
	ModuleName  string
}

type passage struct {
	ID          int64
	Code        string
	Passage     string
	Prompt      string
	AnswerKpID  int64
	OptionKpIDs []int64
	ContentHash string
}

type materials struct {
	words      []word
	sentences  []sentence
	passages   []passage
	wordsByID  map[int64]word
}

func (s *Service) loadMaterials(ctx context.Context) (materials, error) {
	words, err := s.loadWords(ctx)
	if err != nil {
		return materials{}, err
	}
	byID := map[int64]word{}
	for _, w := range words {
		byID[w.KpID] = w
	}
	sentences, err := s.loadSentences(byID)
	if err != nil {
		return materials{}, err
	}
	passages, err := s.loadPassages(byID)
	if err != nil {
		return materials{}, err
	}
	return materials{words: words, sentences: sentences, passages: passages, wordsByID: byID}, nil
}

func (s *Service) loadSentences(words map[int64]word) ([]sentence, error) {
	if !s.db.Migrator().HasTable("english_sentences") {
		return nil, nil
	}
	type row struct {
		ID, TargetKpID          int64
		Code, Text string
		TokensJSON string `gorm:"column:tokens_json"`
		Hash       string `gorm:"column:content_hash"`
		SpeechURL               string `gorm:"column:speech_url"`
		ModuleCode, ModuleName  string
	}
	var rows []row
	err := s.db.Raw(`
		SELECT es.id, es.code, es.text, es.tokens_json, es.target_kp_id, es.content_hash,
		       COALESCE(es.speech_audio_url,'') AS speech_url,
		       COALESCE(m.code,'') AS module_code, COALESCE(m.name,'') AS module_name
		FROM english_sentences es
		JOIN knowledge_points kp ON kp.id = es.target_kp_id
		JOIN modules m ON m.id = kp.module_id
		WHERE COALESCE(es.speech_audio_url,'') <> ''
		ORDER BY es.id`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]sentence, 0, len(rows))
	for _, r := range rows {
		if _, ok := words[r.TargetKpID]; !ok {
			continue
		}
		var tokens []string
		if json.Unmarshal([]byte(r.TokensJSON), &tokens) != nil || len(tokens) < 2 {
			continue
		}
		if strings.Join(tokens, " ") != r.Text {
			continue
		}
		out = append(out, sentence{
			ID: r.ID, Code: r.Code, Text: r.Text, Tokens: tokens, TargetKpID: r.TargetKpID,
			ContentHash: r.Hash, ModuleCode: r.ModuleCode, ModuleName: r.ModuleName,
		})
	}
	return out, nil
}

func (s *Service) loadPassages(words map[int64]word) ([]passage, error) {
	if !s.db.Migrator().HasTable("english_passages") {
		return nil, nil
	}
	type row struct {
		ID, AnswerKpID int64
		Code, Passage, Prompt string
		Hash                   string `gorm:"column:content_hash"`
		OptionJSON             string `gorm:"column:option_kp_ids_json"`
	}
	var rows []row
	err := s.db.Raw(`
		SELECT id, code, passage, prompt, answer_kp_id, option_kp_ids_json, content_hash
		FROM english_passages ORDER BY id`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]passage, 0, len(rows))
	for _, r := range rows {
		answer, ok := words[r.AnswerKpID]
		if !ok {
			continue
		}
		if !strings.Contains(strings.ToLower(r.Passage), strings.ToLower(answer.WordText)) {
			continue
		}
		var ids []int64
		if json.Unmarshal([]byte(r.OptionJSON), &ids) != nil || len(ids) < 4 {
			continue
		}
		ready := true
		for _, id := range ids {
			if _, ok := words[id]; !ok {
				ready = false
				break
			}
		}
		if !ready {
			continue
		}
		out = append(out, passage{
			ID: r.ID, Code: r.Code, Passage: r.Passage, Prompt: r.Prompt,
			AnswerKpID: r.AnswerKpID, OptionKpIDs: ids, ContentHash: r.Hash,
		})
	}
	return out, nil
}

func generate(kind string, mats materials, used map[int64]bool) (Item, error) {
	switch kind {
	case "card-builder":
		return generateSentence(mats, used)
	case "reading-qa":
		return generatePassage(mats, used)
	case "audio-choice", "image-text", "input-gap":
		return generateWord(kind, mats, used)
	default:
		return Item{}, errors.New("英语题型无效")
	}
}

func generateWord(kind string, mats materials, used map[int64]bool) (Item, error) {
	pool := append([]word(nil), mats.words...)
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	target, ok := pickWord(pool, used)
	if !ok {
		return Item{}, errors.New("英语素材不足：无法选出未使用的单词")
	}
	others := make([]word, 0, 3)
	for _, w := range pool {
		if w.KpID != target.KpID && len(others) < 3 {
			others = append(others, w)
		}
	}
	if len(others) < 3 {
		return Item{}, errors.New("英语素材不足：无法组成四个选项")
	}
	item := Item{
		Kind: kind, SkillCode: englishcontent.SkillForKind(kind), TargetID: target.KpID, SourceID: target.KpID,
		SourceTable: "english_assets", ModuleCode: target.ModuleCode, ModuleName: target.ModuleName,
		SourceContentHash: target.ContentHash,
	}
	choices := append([]word{target}, others...)
	rand.Shuffle(len(choices), func(i, j int) { choices[i], choices[j] = choices[j], choices[i] })
	sense := func(w word) string { return fmt.Sprintf("/api/v1/english/words/%d/sense.png", w.KpID) }
	speech := func(w word) string { return fmt.Sprintf("/api/v1/english/words/%d/speech.mp3", w.KpID) }
	switch kind {
	case "audio-choice":
		opts := make([]englishcontent.Choice, 0, 4)
		for _, w := range choices {
			opts = append(opts, englishcontent.Choice{ID: strconv.FormatInt(w.KpID, 10), Label: w.MeaningZh, Picture: sense(w)})
		}
		item.Example = englishcontent.EnglishExample{Kind: kind, Speech: target.WordText, SpeechURL: speech(target), Options: opts, AnswerID: strconv.FormatInt(target.KpID, 10)}
	case "image-text":
		opts := make([]englishcontent.Choice, 0, 4)
		for _, w := range choices {
			opts = append(opts, englishcontent.Choice{ID: strconv.FormatInt(w.KpID, 10), Label: w.WordText, Picture: sense(w)})
		}
		item.Example = englishcontent.EnglishExample{Kind: kind, Prompt: "哪一张图是 " + target.WordText + "？", Options: opts, AnswerID: strconv.FormatInt(target.KpID, 10)}
	case "input-gap":
		item.Example = englishcontent.EnglishExample{Kind: kind, Prompt: "写出这个单词", Speech: target.WordText, SpeechURL: speech(target), Cue: sense(target), Answer: target.WordText}
	}
	return item, nil
}

func generateSentence(mats materials, used map[int64]bool) (Item, error) {
	pool := append([]sentence(nil), mats.sentences...)
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	var picked *sentence
	for i := range pool {
		if !used[pool[i].ID] {
			picked = &pool[i]
			break
		}
	}
	if picked == nil {
		return Item{}, errors.New("英语组句子素材不足：需要已准备完整句子和整句读音")
	}
	target, ok := mats.wordsByID[picked.TargetKpID]
	if !ok {
		return Item{}, errors.New("英语组句子素材缺少目标单词")
	}
	bank := append([]string(nil), picked.Tokens...)
	rand.Shuffle(len(bank), func(i, j int) { bank[i], bank[j] = bank[j], bank[i] })
	return Item{
		Kind: "card-builder", SkillCode: englishcontent.SkillForKind("card-builder"),
		TargetID: picked.TargetKpID, SourceID: picked.ID, SourceTable: "english_sentences",
		ModuleCode: target.ModuleCode, ModuleName: target.ModuleName, SourceContentHash: picked.ContentHash,
		Example: englishcontent.EnglishExample{
			Kind: "card-builder", Prompt: "把单词排成一句话", Speech: picked.Text,
			SpeechURL: fmt.Sprintf("/api/v1/english/sentences/%d/speech.mp3", picked.ID),
			Bank: bank, Answer: picked.Text,
		},
	}, nil
}

func generatePassage(mats materials, used map[int64]bool) (Item, error) {
	pool := append([]passage(nil), mats.passages...)
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	var picked *passage
	for i := range pool {
		if !used[pool[i].ID] {
			picked = &pool[i]
			break
		}
	}
	if picked == nil {
		return Item{}, errors.New("英语读一读素材不足：需要已准备短文、问题和选项")
	}
	answer, ok := mats.wordsByID[picked.AnswerKpID]
	if !ok {
		return Item{}, errors.New("英语读一读素材缺少答案单词")
	}
	opts := make([]englishcontent.Choice, 0, len(picked.OptionKpIDs))
	for _, id := range picked.OptionKpIDs {
		w, ok := mats.wordsByID[id]
		if !ok {
			return Item{}, errors.New("英语读一读选项缺少义图或读音")
		}
		opts = append(opts, englishcontent.Choice{ID: strconv.FormatInt(w.KpID, 10), Label: w.MeaningZh, Picture: fmt.Sprintf("/api/v1/english/words/%d/sense.png", w.KpID)})
	}
	rand.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
	return Item{
		Kind: "reading-qa", SkillCode: englishcontent.SkillForKind("reading-qa"),
		TargetID: picked.AnswerKpID, SourceID: picked.ID, SourceTable: "english_passages",
		ModuleCode: answer.ModuleCode, ModuleName: answer.ModuleName, SourceContentHash: picked.ContentHash,
		Example: englishcontent.EnglishExample{
			Kind: "reading-qa", Prompt: picked.Prompt, Passage: picked.Passage,
			Options: opts, AnswerID: strconv.FormatInt(picked.AnswerKpID, 10),
		},
	}, nil
}

func pickWord(pool []word, used map[int64]bool) (word, bool) {
	for _, w := range pool {
		if !used[w.KpID] {
			return w, true
		}
	}
	if len(pool) == 0 {
		return word{}, false
	}
	return pool[0], true
}

func wordContentHash(kpID int64, parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(append([]string{strconv.FormatInt(kpID, 10)}, parts...), "|")))
	return hex.EncodeToString(sum[:12])
}
