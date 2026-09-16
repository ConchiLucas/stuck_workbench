package pinyin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SoloSyllable is the textbook 呼读音 (toned pinyin) used when audio-cmn has no HSK single-char file.
var SoloSyllable = map[string]string{
	"b": "bo1", "p": "po1", "m": "mo1", "f": "fo2", "d": "de2", "t": "te4",
	"n": "ne1", "l": "le4", "g": "ge1", "k": "ke1", "h": "he1",
	"j": "ji1", "q": "qi1", "x": "xi1", "zh": "zhi1", "ch": "chi1", "sh": "shi1",
	"r": "ri4", "z": "zi1", "c": "ci1", "s": "si1", "y": "yi1", "w": "wu1",
	"a": "a1", "o": "o1", "e": "e2", "i": "yi1", "u": "wu1", "ü": "yu2",
	"ai": "ai1", "ei": "ei1", "ui": "wei1", "ao": "ao2", "ou": "ou1", "iu": "you1",
	"ie": "ye1", "üe": "yue1", "er": "er2", "an": "an1", "en": "en1", "in": "yin1",
	"un": "wen1", "ün": "yun1", "ang": "ang2", "eng": "eng1",
}

// WordSyllable maps 例字 without an HSK single-char file to a toned syllable recording.
var WordSyllable = map[string]string{
	"妈": "ma1", "爸": "ba4", "波": "bo1", "哥": "ge1", "豆": "dou4",
	"叶": "ye4", "姐": "jie3", "爷": "ye2", "儿": "er2", "耳": "er3", "尔": "er3",
	"林": "lin2", "民": "min2", "村": "cun1", "孙": "sun1", "朋": "peng2",
}

func packMP3(root, kind, name string) (string, []byte, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, fmt.Errorf("空文件名")
	}
	rel := filepath.Join(kind, "cmn-"+name+".mp3")
	path := filepath.Join(root, rel)
	data, err := os.ReadFile(path)
	if err != nil {
		return rel, nil, err
	}
	if len(data) == 0 {
		return rel, nil, fmt.Errorf("%s 为空", rel)
	}
	return rel, data, nil
}

func (s *Service) humanSoloMP3(root string, asset Asset) ([]byte, error) {
	if _, data, err := packMP3(root, "hsk", asset.SoloText); err == nil {
		return data, nil
	}
	syl := SoloSyllable[strings.TrimSpace(asset.Letter)]
	if syl == "" {
		return nil, fmt.Errorf("没有 %s 的真人单读", asset.Letter)
	}
	_, data, err := packMP3(root, "syllabs", syl)
	return data, err
}

func (s *Service) humanWordMP3(root, word string) ([]byte, error) {
	if _, data, err := packMP3(root, "hsk", word); err == nil {
		return data, nil
	}
	syl := WordSyllable[strings.TrimSpace(word)]
	if syl == "" {
		return nil, fmt.Errorf("没有 %s 的真人例字录音", word)
	}
	_, data, err := packMP3(root, "syllabs", syl)
	return data, err
}

// ImportHumanPack overwrites solo/例字 speech from a local audio-cmn tree (hsk/ + syllabs/).
func (s *Service) ImportHumanPack(ctx context.Context, moduleCode, packDir string) (BatchResult, error) {
	moduleCode = strings.TrimSpace(moduleCode)
	packDir = strings.TrimSpace(packDir)
	if moduleCode == "" {
		return BatchResult{}, fmt.Errorf("moduleCode 不能为空")
	}
	if packDir == "" {
		return BatchResult{}, fmt.Errorf("真人包目录未配置")
	}
	if s.store == nil {
		return BatchResult{}, fmt.Errorf("语音存储未就绪（检查 MinIO）")
	}
	info, err := os.Stat(packDir)
	if err != nil || !info.IsDir() {
		return BatchResult{}, fmt.Errorf("真人包目录无效: %s", packDir)
	}
	var assets []Asset
	if err := s.db.Where("module_code = ?", moduleCode).
		Order("module_order ASC, kp_order ASC, kp_id ASC").
		Find(&assets).Error; err != nil {
		return BatchResult{}, err
	}
	out := BatchResult{}
	for i := range assets {
		asset := &assets[i]
		kinds := []string{"solo"}
		words := ParseWordExamples(asset.WordExamples, asset.WordText)
		for index := range words {
			kinds = append(kinds, SpeechKindForExampleIndex(index))
		}
		for _, kind := range kinds {
			var mp3 []byte
			var loadErr error
			if kind == "solo" {
				if strings.TrimSpace(asset.SoloText) == "" {
					out.Skipped++
					continue
				}
				mp3, loadErr = s.humanSoloMP3(packDir, *asset)
			} else {
				index, ok := ParseExampleSpeechKind(kind)
				if !ok || index < 0 || index >= len(words) {
					out.Skipped++
					continue
				}
				mp3, loadErr = s.humanWordMP3(packDir, words[index])
			}
			if loadErr != nil || len(mp3) == 0 {
				out.Failed++
				if len(out.Errors) < 10 {
					out.Errors = append(out.Errors, fmt.Sprintf("%s/%s(%d): %v", asset.Letter, kind, asset.KpID, loadErr))
				}
				continue
			}
			if err := s.storeSpeech(ctx, asset, kind, mp3); err != nil {
				out.Failed++
				if len(out.Errors) < 10 {
					out.Errors = append(out.Errors, fmt.Sprintf("%s/%s(%d): %v", asset.Letter, kind, asset.KpID, err))
				}
				continue
			}
			out.Generated++
		}
	}
	return out, nil
}
