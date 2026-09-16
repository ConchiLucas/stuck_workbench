package quiz

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-learning/model"
	"github.com/conchi/study-learning/pinyincatalog"
	"github.com/conchi/study-learning/pinyincontract"
	"gorm.io/gorm"
)

var (
	ErrNotFound       = errors.New("孩子或题目不存在")
	ErrInvalidRequest = errors.New("作答请求或选项无效")
	ErrConflict       = errors.New("此请求或题目已提交其他作答")
	ErrExpired        = errors.New("题目已过期，请重新练习")
)

// GenerateForChild persists only an answer-free public snapshot and private key.
// No attempt or learning state is written until Answer accepts a response.
func (s *Service) GenerateForChild(ctx context.Context, childID int64, kind string, excluded []int64) (pinyincontract.GeneratedQuestion, error) {
	kind = strings.TrimSpace(kind)
	switch kind {
	case "listen", "inword", "shape", "blend":
	default:
		return pinyincontract.GeneratedQuestion{}, ErrInvalidType
	}
	if err := s.db.WithContext(ctx).Select("id").First(&model.Child{}, childID).Error; err != nil {
		return pinyincontract.GeneratedQuestion{}, notFound(err)
	}
	// Letter practice remains available when syllable content has not been installed.
	if err := pinyincatalog.Sync(ctx, s.db); err != nil && (kind == "blend" || !errors.Is(err, pinyincatalog.ErrUnavailable)) {
		return pinyincontract.GeneratedQuestion{}, err
	}
	q, err := s.Generate(ctx, kind, excluded)
	if err != nil {
		return pinyincontract.GeneratedQuestion{}, err
	}
	kpID := q.TargetID
	if kind == "blend" {
		var link model.PinyinSyllableLink
		if err := s.db.WithContext(ctx).Where("asset_id = ? AND enabled = ?", q.TargetID, true).First(&link).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return pinyincontract.GeneratedQuestion{}, ErrNoMaterial
			}
			return pinyincontract.GeneratedQuestion{}, err
		}
		kpID = link.KpID
	}
	// Asset IDs for letters are learning IDs; verify module ownership before issuing.
	var scope struct{ SubjectCode, ModuleCode string }
	if err := s.db.WithContext(ctx).Table("knowledge_points kp").Select("sub.code AS subject_code,m.code AS module_code").Joins("JOIN modules m ON m.id=kp.module_id").Joins("JOIN subjects sub ON sub.id=m.subject_id").Where("kp.id = ?", kpID).Take(&scope).Error; err != nil {
		return pinyincontract.GeneratedQuestion{}, notFound(err)
	}
	allowed := false
	for _, skill := range mastery.SkillsFor(scope.SubjectCode, scope.ModuleCode) {
		allowed = allowed || skill == kind
	}
	if scope.SubjectCode != "pinyin" || !allowed {
		return pinyincontract.GeneratedQuestion{}, ErrNoMaterial
	}
	idBytes := make([]byte, 24)
	if _, err = rand.Read(idBytes); err != nil {
		return pinyincontract.GeneratedQuestion{}, err
	}
	now := time.Now().UTC()
	out := pinyincontract.GeneratedQuestion{InstanceID: hex.EncodeToString(idBytes), Type: kind, TargetID: q.TargetID, KpID: kpID, ExpiresAt: now.Add(24 * time.Hour), Stem: q.Stem, SpeechText: q.SpeechText, SpeechURL: q.SpeechURL, Visual: pinyincontract.Visual(q.Visual), Options: make([]pinyincontract.Option, 0, len(q.Options))}
	if kind == "blend" {
		out.Visual.Syllable = ""
	}
	for index, o := range q.Options {
		out.Options = append(out.Options, pinyincontract.Option{ID: "option-" + strconv.Itoa(index), Label: o.Label, SpeechText: o.SpeechText, SpeechURL: o.SpeechURL})
	}
	if q.AnswerIndex < 0 || q.AnswerIndex >= len(out.Options) {
		return pinyincontract.GeneratedQuestion{}, ErrNoMaterial
	}
	data, err := json.Marshal(out)
	if err != nil {
		return pinyincontract.GeneratedQuestion{}, err
	}
	row := model.PinyinQuizInstance{ID: out.InstanceID, ChildID: childID, KpID: kpID, SkillCode: kind, SnapshotVersion: 1, PublicSnapshot: string(data), AnswerOptionID: out.Options[q.AnswerIndex].ID, CreatedAt: now, ExpiresAt: out.ExpiresAt}
	if err = s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return pinyincontract.GeneratedQuestion{}, err
	}
	return out, nil
}

func (s *Service) GetInstance(ctx context.Context, childID int64, id string) (pinyincontract.InstanceSnapshot, error) {
	var row model.PinyinQuizInstance
	if err := s.db.WithContext(ctx).Where("id = ? AND child_id = ?", id, childID).First(&row).Error; err != nil {
		return pinyincontract.InstanceSnapshot{}, notFound(err)
	}
	var out pinyincontract.InstanceSnapshot
	if err := json.Unmarshal([]byte(row.PublicSnapshot), &out.GeneratedQuestion); err != nil {
		return out, err
	}
	var receipt model.PinyinAnswerReceipt
	err := s.db.WithContext(ctx).Where("child_id = ? AND instance_id = ?", childID, id).First(&receipt).Error
	if err == nil {
		out.AcceptedResult = &pinyincontract.AnswerResult{}
		err = json.Unmarshal([]byte(receipt.ResponseSnapshot), out.AcceptedResult)
		return out, err
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return out, err
	}
	if !time.Now().Before(row.ExpiresAt) {
		return pinyincontract.InstanceSnapshot{}, ErrExpired
	}
	return out, nil
}
func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
