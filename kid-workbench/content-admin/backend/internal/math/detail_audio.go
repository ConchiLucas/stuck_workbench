package math

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/conchi/study-learning/mathcontent"
	"regexp"
	"strings"
)

func (s *Service) GenerateDetailAudio(ctx context.Context, id string, revision int) (mathcontent.MathDetail, error) {
	var row detailDraft
	var d mathcontent.MathDetail
	if e := s.db.WithContext(ctx).Where("id=?", id).First(&row).Error; e != nil {
		return d, e
	}
	if row.Revision != revision {
		return d, ErrDetailConflict
	}
	if e := json.Unmarshal([]byte(row.Content), &d); e != nil {
		return d, e
	}
	text := detailSpeechText(d)
	mp3, e := s.synthesize(ctx, text)
	if e != nil {
		return d, e
	}
	if len(mp3) == 0 {
		return d, fmt.Errorf("生成音频为空")
	}
	if s.store == nil {
		return d, fmt.Errorf("语音存储未就绪")
	}
	hash := sha256.Sum256(mp3)
	name := hex.EncodeToString(hash[:]) + ".mp3"
	if _, e = s.store.PutBytes(ctx, "math/detail-audio/"+name, mp3, "audio/mpeg"); e != nil {
		return d, e
	}
	d.Example.AudioURL = "/api/v1/math/detail-audio/" + name
	return s.SaveDetail(ctx, d)
}

func detailSpeechText(d mathcontent.MathDetail) string {
	text := d.Example.Prompt
	if d.Example.Kind == "shape-name" {
		text = "这是什么图形？"
	}
	return strings.NewReplacer("+", "加", "−", "减", "-", "减", "=", "等于", "□", "空格", "?", "几").Replace(text)
}

var detailAudioName = regexp.MustCompile(`^[a-f0-9]{64}\.mp3$`)

func (s *Service) DetailAudio(ctx context.Context, file string) ([]byte, error) {
	if !detailAudioName.MatchString(file) || s.store == nil {
		return nil, fmt.Errorf("音频不存在")
	}
	return s.store.GetBytes(ctx, "math/detail-audio/"+file)
}

func (s *Service) verifyDetailAudio(ctx context.Context, d mathcontent.MathDetail) error {
	if d.Example.AudioURL == "" {
		return nil
	}
	file := strings.TrimPrefix(d.Example.AudioURL, "/api/v1/math/detail-audio/")
	data, e := s.DetailAudio(ctx, file)
	if e != nil || len(data) == 0 {
		return fmt.Errorf("题干音频不存在，请重新生成后发布")
	}
	sum := sha256.Sum256(data)
	if file != hex.EncodeToString(sum[:])+".mp3" {
		return fmt.Errorf("题干音频校验失败，请重新生成")
	}
	return nil
}
