package pinyin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

type SyllableUpdate struct {
	SpeechText string `json:"speechText"`
	Enabled    bool   `json:"enabled"`
}

func (s *Service) ListSyllables(ctx context.Context) ([]SyllableAsset, error) {
	items := []SyllableAsset{}
	err := s.db.WithContext(ctx).Order("id").Find(&items).Error
	return items, err
}

// Review eligibility is controlled by the material owner; learning facts are untouched.
func (s *Service) UpdateSyllable(ctx context.Context, id int64, update SyllableUpdate) (SyllableAsset, error) {
	var item SyllableAsset
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return item, err
	}
	text := strings.TrimSpace(update.SpeechText)
	if text == "" || len([]rune(text)) > 20 {
		return item, fmt.Errorf("例字须为 1–20 个字符")
	}
	item.SpeechText = text
	item.Enabled = update.Enabled
	err := s.db.WithContext(ctx).Model(&item).Updates(map[string]any{"speech_text": text, "enabled": update.Enabled}).Error
	return item, err
}
func syllableObjectKey(id int64) string { return fmt.Sprintf("pinyin/syllables/%d/speech.mp3", id) }
func syllablePublicURL(id int64) string {
	return fmt.Sprintf("/api/v1/pinyin/syllables/%d/speech.mp3", id)
}

// SyllableSpeechMP3 is a read-only recording endpoint. It never synthesizes audio.
func (s *Service) SyllableSpeechMP3(ctx context.Context, id int64, versions ...string) ([]byte, error) {
	var item SyllableAsset
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	if item.SpeechURL == "" {
		return nil, fmt.Errorf("音节录音缺失，请在素材后台导入真人录音")
	}
	if s.store == nil {
		return nil, fmt.Errorf("语音存储未就绪")
	}
	key := syllableObjectKey(id)
	if len(versions) > 0 && versions[0] != "" {
		if !regexp.MustCompile(`^[a-f0-9]{16}$`).MatchString(versions[0]) {
			return nil, fmt.Errorf("无效录音版本")
		}
		key = fmt.Sprintf("pinyin/syllables/%d/speech-%s.mp3", id, versions[0])
	}
	data, err := s.store.GetBytes(ctx, key)
	if err != nil || len(data) == 0 {
		return nil, fmt.Errorf("音节录音不可用，请重新导入真人录音")
	}
	return data, nil
}

func recordingName(item SyllableAsset) (string, error) {
	if item.Tone < 1 || item.Tone > 5 {
		return "", fmt.Errorf("无效声调")
	}
	replacer := strings.NewReplacer("ā", "a", "á", "a", "ǎ", "a", "à", "a", "ē", "e", "é", "e", "ě", "e", "è", "e", "ī", "i", "í", "i", "ǐ", "i", "ì", "i", "ō", "o", "ó", "o", "ǒ", "o", "ò", "o", "ū", "u", "ú", "u", "ǔ", "u", "ù", "u", "ǖ", "v", "ǘ", "v", "ǚ", "v", "ǜ", "v", "ü", "v")
	text := replacer.Replace(strings.ToLower(strings.TrimSpace(item.SyllableText)))
	if text == "" {
		return "", fmt.Errorf("音节为空")
	}
	for _, r := range text {
		if r < 'a' || r > 'z' {
			return "", fmt.Errorf("无效音节")
		}
	}
	return fmt.Sprintf("%s%d", text, item.Tone), nil
}

// ImportSyllableHumanPack imports the exact toned syllable, never a homograph's ambiguous reading.
func (s *Service) ImportSyllableHumanPack(ctx context.Context, root string) (BatchResult, error) {
	if strings.TrimSpace(root) == "" {
		return BatchResult{}, fmt.Errorf("真人包目录未配置")
	}
	if s.store == nil {
		return BatchResult{}, fmt.Errorf("语音存储未就绪")
	}
	items, err := s.ListSyllables(ctx)
	if err != nil {
		return BatchResult{}, err
	}
	out := BatchResult{}
	for _, item := range items {
		name, loadErr := recordingName(item)
		var data []byte
		if loadErr == nil {
			_, data, loadErr = packMP3(root, "syllabs", name)
		}
		if loadErr == nil {
			_, loadErr = s.store.PutBytes(ctx, syllableObjectKey(item.ID), data, "audio/mpeg")
		}
		version := ""
		if loadErr == nil {
			sum := sha256.Sum256(data)
			version = hex.EncodeToString(sum[:])[:16]
			_, loadErr = s.store.PutBytes(ctx, fmt.Sprintf("pinyin/syllables/%d/speech-%s.mp3", item.ID, version), data, "audio/mpeg")
		}
		if loadErr == nil {
			loadErr = s.db.WithContext(ctx).Model(&item).Update("speech_url", syllablePublicURL(item.ID)+"?v="+version).Error
		}
		if loadErr != nil {
			out.Failed++
			if len(out.Errors) < 10 {
				out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", item.SyllableText, loadErr))
			}
			continue
		}
		out.Generated++
	}
	return out, nil
}
