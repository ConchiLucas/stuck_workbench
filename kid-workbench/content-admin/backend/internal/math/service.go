package math

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/conchi/study-content-admin/internal/configclient"
	"github.com/conchi/study-content-admin/internal/storage"
	"github.com/conchi/study-content-admin/internal/tts"
)

type Asset struct {
	KpID           int64     `gorm:"column:kp_id;primaryKey" json:"kpId"`
	Title          string    `gorm:"column:title" json:"title"`
	Kind           string    `gorm:"column:kind" json:"kind"`
	Payload        string    `gorm:"column:payload" json:"payload"`
	Difficulty     int       `gorm:"column:difficulty" json:"difficulty"`
	ModuleCode     string    `gorm:"column:module_code" json:"moduleCode"`
	ModuleName     string    `gorm:"column:module_name" json:"moduleName"`
	ModuleOrder    int       `gorm:"column:module_order" json:"moduleOrder"`
	KpOrder        int       `gorm:"column:kp_order" json:"kpOrder"`
	GlyphImageURL  string    `gorm:"column:glyph_image_url" json:"glyphImageUrl"`
	SpeechAudioURL string    `gorm:"column:speech_audio_url" json:"speechAudioUrl"`
	SpeechText     string    `gorm:"column:speech_text" json:"speechText"`
	Enabled        bool      `gorm:"column:enabled;default:true" json:"enabled"`
	SyncedAt       time.Time `gorm:"column:synced_at" json:"syncedAt"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (Asset) TableName() string { return "math_assets" }

type ItemDTO struct {
	KpID           int64  `json:"kpId"`
	Title          string `json:"title"`
	Kind           string `json:"kind"`
	Payload        string `json:"payload"`
	Difficulty     int    `json:"difficulty"`
	ModuleCode     string `json:"moduleCode"`
	ModuleName     string `json:"moduleName"`
	ModuleOrder    int    `json:"moduleOrder"`
	KpOrder        int    `json:"kpOrder"`
	GlyphImageURL  string `json:"glyphImageUrl"`
	SpeechAudioURL string `json:"speechAudioUrl"`
	SpeechText     string `json:"speechText"`
	Enabled        bool   `json:"enabled"`
}

type GroupDTO struct {
	ModuleCode  string    `json:"moduleCode"`
	ModuleName  string    `json:"moduleName"`
	ModuleOrder int       `json:"moduleOrder"`
	Items       []ItemDTO `json:"items"`
}

type ListResult struct {
	View   string     `json:"view"`
	Total  int        `json:"total"`
	Groups []GroupDTO `json:"groups,omitempty"`
	Items  []ItemDTO  `json:"items,omitempty"`
}

type SyncResult struct {
	Upserted int `json:"upserted"`
	Total    int `json:"total"`
}

type BatchResult struct {
	Generated int      `json:"generated"`
	Skipped   int      `json:"skipped"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
}

type QuestionSpeechDTO struct {
	QuestionID int64  `json:"questionId"`
	KpID       int64  `json:"kpId"`
	Code       string `json:"code"`
	SpeechText string `json:"speechText"`
	MediaURL   string `json:"mediaUrl"`
}

type QuestionSpeechStatus struct {
	ModuleCode string `json:"moduleCode"`
	Total      int    `json:"total"`
	Ready      int    `json:"ready"`
	Missing    int    `json:"missing"`
}

type ObjectPutter interface {
	PutBytes(ctx context.Context, objectKey string, data []byte, contentType string) (string, error)
	GetBytes(ctx context.Context, relativeOrFullKey string) ([]byte, error)
}

type GlyphRenderer interface {
	RenderMathPNG(text string) ([]byte, error)
	RenderMathNumberPNG(text string) ([]byte, error)
	RenderMathShapePNG(shapeTitle string) ([]byte, error)
}

type VoiceModelSource interface {
	LoadVoiceModels(ctx context.Context) (configclient.AIConfiguration, error)
}

type SpeechSynthesizer interface {
	Synthesize(ctx context.Context, provider configclient.AIProvider, text string) ([]byte, error)
}

type Service struct {
	db       *gorm.DB
	store    ObjectPutter
	renderer GlyphRenderer
	voices   VoiceModelSource
	speech   SpeechSynthesizer
}

func NewService(db *gorm.DB, store ObjectPutter, renderer GlyphRenderer, voices VoiceModelSource, speech SpeechSynthesizer) *Service {
	return &Service{db: db, store: store, renderer: renderer, voices: voices, speech: speech}
}

type kpRow struct {
	ID          int64
	Title       string
	Payload     string
	Difficulty  int
	KpOrder     int
	ModuleCode  string
	ModuleName  string
	ModuleOrder int
}

func (s *Service) Sync() (SyncResult, error) {
	var rows []kpRow
	err := s.db.Raw(`
		SELECT kp.id AS id, kp.title AS title, kp.payload AS payload, kp.difficulty AS difficulty,
		       kp.order_no AS kp_order,
		       m.code AS module_code, m.name AS module_name, m.order_no AS module_order
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = ?
		ORDER BY m.order_no, kp.order_no, kp.id
	`, "math").Scan(&rows).Error
	if err != nil {
		return SyncResult{}, err
	}

	now := time.Now().UTC()
	upserted := 0
	keep := make(map[int64]struct{}, len(rows))
	for _, row := range rows {
		keep[row.ID] = struct{}{}
		kind := payloadKind(row.Payload)
		speech := buildSpeechText(kind, row.Title, row.Payload)
		var existing Asset
		findErr := s.db.First(&existing, "kp_id = ?", row.ID).Error
		if findErr == gorm.ErrRecordNotFound {
			asset := Asset{
				KpID:        row.ID,
				Title:       row.Title,
				Kind:        kind,
				Payload:     row.Payload,
				Difficulty:  row.Difficulty,
				ModuleCode:  row.ModuleCode,
				ModuleName:  row.ModuleName,
				ModuleOrder: row.ModuleOrder,
				KpOrder:     row.KpOrder,
				SpeechText:  speech,
				Enabled:     true,
				SyncedAt:    now,
				UpdatedAt:   now,
			}
			if err := s.db.Create(&asset).Error; err != nil {
				return SyncResult{}, err
			}
			upserted++
			continue
		}
		if findErr != nil {
			return SyncResult{}, findErr
		}
		existing.Title = row.Title
		existing.Kind = kind
		existing.Payload = row.Payload
		existing.Difficulty = row.Difficulty
		existing.ModuleCode = row.ModuleCode
		existing.ModuleName = row.ModuleName
		existing.ModuleOrder = row.ModuleOrder
		existing.KpOrder = row.KpOrder
		existing.SpeechText = speech
		existing.SyncedAt = now
		existing.UpdatedAt = now
		if err := s.db.Save(&existing).Error; err != nil {
			return SyncResult{}, err
		}
		upserted++
	}

	var stale []Asset
	if err := s.db.Find(&stale).Error; err != nil {
		return SyncResult{}, err
	}
	for _, a := range stale {
		if _, ok := keep[a.KpID]; ok {
			continue
		}
		if err := s.db.Delete(&Asset{}, "kp_id = ?", a.KpID).Error; err != nil {
			return SyncResult{}, err
		}
	}

	return SyncResult{Upserted: upserted, Total: len(rows)}, nil
}

func (s *Service) List(view string) (ListResult, error) {
	if view == "" {
		view = "groups"
	}
	var assets []Asset
	if err := s.db.Order("module_order ASC, kp_order ASC, kp_id ASC").Find(&assets).Error; err != nil {
		return ListResult{}, err
	}

	dtos := make([]ItemDTO, 0, len(assets))
	for _, a := range assets {
		dtos = append(dtos, toDTO(a))
	}

	out := ListResult{View: view, Total: len(dtos)}
	if view == "table" {
		out.Items = dtos
		return out, nil
	}

	byMod := map[string]*GroupDTO{}
	order := []string{}
	for _, dto := range dtos {
		g, ok := byMod[dto.ModuleCode]
		if !ok {
			g = &GroupDTO{
				ModuleCode:  dto.ModuleCode,
				ModuleName:  dto.ModuleName,
				ModuleOrder: dto.ModuleOrder,
				Items:       []ItemDTO{},
			}
			byMod[dto.ModuleCode] = g
			order = append(order, dto.ModuleCode)
		}
		g.Items = append(g.Items, dto)
	}
	out.Groups = make([]GroupDTO, 0, len(order))
	for _, code := range order {
		out.Groups = append(out.Groups, *byMod[code])
	}
	return out, nil
}

func (s *Service) SetEnabled(ctx context.Context, kpID int64, enabled bool) (ItemDTO, error) {
	var asset Asset
	if err := s.db.WithContext(ctx).First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return ItemDTO{}, err
	}
	if err := s.db.WithContext(ctx).Model(&asset).Updates(map[string]any{"enabled": enabled, "updated_at": time.Now().UTC()}).Error; err != nil {
		return ItemDTO{}, err
	}
	if err := s.db.WithContext(ctx).First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return ItemDTO{}, err
	}
	return toDTO(asset), nil
}

func (s *Service) GenerateGlyph(ctx context.Context, kpID int64) (ItemDTO, error) {
	if s.store == nil || s.renderer == nil {
		return ItemDTO{}, fmt.Errorf("字图生成未就绪（检查 MinIO 与字体）")
	}
	var asset Asset
	if err := s.db.First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return ItemDTO{}, err
	}
	title := strings.TrimSpace(asset.Title)
	if title == "" {
		return ItemDTO{}, fmt.Errorf("题目标题为空")
	}

	var png []byte
	var err error
	switch {
	case asset.ModuleCode == "shape" || isShapeKind(asset):
		png, err = s.renderer.RenderMathShapePNG(title)
	case asset.Kind == "number" || asset.ModuleCode == "num20":
		png, err = s.renderer.RenderMathNumberPNG(title)
	default:
		png, err = s.renderer.RenderMathPNG(title)
	}
	if err != nil {
		return ItemDTO{}, err
	}
	if _, err := s.store.PutBytes(ctx, storage.MathGlyphObjectKey(kpID), png, "image/png"); err != nil {
		return ItemDTO{}, err
	}
	asset.GlyphImageURL = glyphPublicURL(kpID)
	asset.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(&asset).Error; err != nil {
		return ItemDTO{}, err
	}
	return toDTO(asset), nil
}

func (s *Service) GlyphPNG(ctx context.Context, kpID int64) ([]byte, error) {
	if s.store == nil {
		return nil, fmt.Errorf("字图存储未就绪")
	}
	var asset Asset
	if err := s.db.First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return nil, err
	}
	if strings.TrimSpace(asset.GlyphImageURL) == "" {
		return nil, fmt.Errorf("尚未生成字图")
	}
	return s.store.GetBytes(ctx, storage.MathGlyphObjectKey(kpID))
}

func (s *Service) BatchGenerateGlyphs(ctx context.Context, moduleCode string) (BatchResult, error) {
	moduleCode = strings.TrimSpace(moduleCode)
	if moduleCode == "" {
		return BatchResult{}, fmt.Errorf("moduleCode 不能为空")
	}
	if s.store == nil || s.renderer == nil {
		return BatchResult{}, fmt.Errorf("字图生成未就绪（检查 MinIO 与字体）")
	}
	var assets []Asset
	q := s.db.Where("module_code = ?", moduleCode)
	// Force regenerate for shapes / 认数字 so style changes (数学本 → 白底田字格) replace old files.
	forceAll := moduleCode == "shape"
	if !forceAll {
		q = q.Where("glyph_image_url = '' OR glyph_image_url IS NULL")
	}
	if err := q.Order("module_order ASC, kp_order ASC, kp_id ASC").Find(&assets).Error; err != nil {
		return BatchResult{}, err
	}
	out := BatchResult{}
	for _, asset := range assets {
		if !forceAll && strings.TrimSpace(asset.GlyphImageURL) != "" {
			out.Skipped++
			continue
		}
		if _, err := s.GenerateGlyph(ctx, asset.KpID); err != nil {
			out.Failed++
			if len(out.Errors) < 10 {
				out.Errors = append(out.Errors, fmt.Sprintf("%s(%d): %v", asset.Title, asset.KpID, err))
			}
			continue
		}
		out.Generated++
	}
	return out, nil
}

func (s *Service) SpeechMP3(ctx context.Context, kpID int64) ([]byte, error) {
	var asset Asset
	if err := s.db.First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return nil, err
	}
	key := storage.MathSpeechObjectKey(kpID)
	if s.store != nil && strings.TrimSpace(asset.SpeechAudioURL) != "" {
		if data, err := s.store.GetBytes(ctx, key); err == nil && len(data) > 0 {
			return data, nil
		}
	}
	text := strings.TrimSpace(asset.SpeechText)
	if text == "" {
		text = buildSpeechText(asset.Kind, asset.Title, asset.Payload)
	}
	if text == "" {
		return nil, fmt.Errorf("朗读文本为空")
	}
	mp3, err := s.synthesize(ctx, text)
	if err != nil {
		return nil, err
	}
	if err := s.storeSpeech(ctx, &asset, mp3); err != nil {
		return nil, err
	}
	return mp3, nil
}

func (s *Service) RegenerateSpeech(ctx context.Context, kpID int64) (ItemDTO, error) {
	var asset Asset
	if err := s.db.First(&asset, "kp_id = ?", kpID).Error; err != nil {
		return ItemDTO{}, err
	}
	text := strings.TrimSpace(asset.SpeechText)
	if text == "" {
		text = buildSpeechText(asset.Kind, asset.Title, asset.Payload)
		asset.SpeechText = text
	}
	if text == "" {
		return ItemDTO{}, fmt.Errorf("朗读文本为空")
	}
	mp3, err := s.synthesize(ctx, text)
	if err != nil {
		return ItemDTO{}, err
	}
	if err := s.storeSpeech(ctx, &asset, mp3); err != nil {
		return ItemDTO{}, err
	}
	return toDTO(asset), nil
}

type questionSpeechRow struct {
	QuestionID int64
	KpID       int64
	Code       string
	Speech     string
	MediaURL   string
}

func (s *Service) RegenerateQuestionSpeech(ctx context.Context, questionID int64) (QuestionSpeechDTO, error) {
	var row questionSpeechRow
	query := s.db.Raw(`
		SELECT q.id AS question_id, q.kp_id AS kp_id, q.code AS code,
		       q.speech AS speech, q.media_url AS media_url
		FROM questions q
		JOIN knowledge_points kp ON kp.id = q.kp_id
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE q.id = ? AND s.code = 'math'`, questionID).Scan(&row)
	if query.Error != nil {
		return QuestionSpeechDTO{}, query.Error
	}
	if query.RowsAffected == 0 {
		return QuestionSpeechDTO{}, gorm.ErrRecordNotFound
	}

	var speechPayload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(row.Speech), &speechPayload); err != nil {
		return QuestionSpeechDTO{}, fmt.Errorf("题目朗读配置无效: %w", err)
	}
	text := strings.TrimSpace(speechPayload.Text)
	if text == "" {
		return QuestionSpeechDTO{}, fmt.Errorf("题目朗读文本为空")
	}
	mp3, err := s.synthesize(ctx, text)
	if err != nil {
		return QuestionSpeechDTO{}, err
	}
	if s.store == nil {
		return QuestionSpeechDTO{}, fmt.Errorf("语音存储未就绪（检查 MinIO）")
	}
	canonicalKey := storage.MathQuestionSpeechObjectKey(questionID)
	_, err = s.store.PutBytes(ctx, canonicalKey, mp3, "audio/mpeg")
	if err != nil {
		return QuestionSpeechDTO{}, err
	}
	if err := s.db.Model(&struct{ ID int64 }{}).Table("questions").
		Where("id = ?", questionID).Update("media_url", canonicalKey).Error; err != nil {
		return QuestionSpeechDTO{}, err
	}
	return QuestionSpeechDTO{
		QuestionID: row.QuestionID,
		KpID:       row.KpID,
		Code:       row.Code,
		SpeechText: text,
		MediaURL:   canonicalKey,
	}, nil
}

func (s *Service) QuestionSpeechStatus(moduleCode string) (QuestionSpeechStatus, error) {
	moduleCode = strings.TrimSpace(moduleCode)
	if moduleCode == "" {
		return QuestionSpeechStatus{}, fmt.Errorf("moduleCode 不能为空")
	}
	rows, err := s.questionSpeechRows(moduleCode)
	if err != nil {
		return QuestionSpeechStatus{}, err
	}
	status := QuestionSpeechStatus{ModuleCode: moduleCode, Total: len(rows)}
	for _, row := range rows {
		if strings.TrimSpace(row.MediaURL) == storage.MathQuestionSpeechObjectKey(row.QuestionID) {
			status.Ready++
		}
	}
	status.Missing = status.Total - status.Ready
	return status, nil
}

func (s *Service) BatchGenerateQuestionSpeech(ctx context.Context, moduleCode string) (BatchResult, error) {
	moduleCode = strings.TrimSpace(moduleCode)
	if moduleCode == "" {
		return BatchResult{}, fmt.Errorf("moduleCode 不能为空")
	}
	if s.store == nil || s.voices == nil || s.speech == nil {
		return BatchResult{}, fmt.Errorf("TTS 未就绪（检查 MinIO 与语音配置）")
	}
	rows, err := s.questionSpeechRows(moduleCode)
	if err != nil {
		return BatchResult{}, err
	}
	out := BatchResult{}
	needDelay := false
	for _, row := range rows {
		if strings.TrimSpace(row.MediaURL) == storage.MathQuestionSpeechObjectKey(row.QuestionID) {
			out.Skipped++
			continue
		}
		if needDelay {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(400 * time.Millisecond):
			}
		}
		if _, err := s.RegenerateQuestionSpeech(ctx, row.QuestionID); err != nil {
			out.Failed++
			if len(out.Errors) < 10 {
				out.Errors = append(out.Errors, fmt.Sprintf("question(%d): %v", row.QuestionID, err))
			}
		} else {
			out.Generated++
		}
		needDelay = true
	}
	return out, nil
}

func (s *Service) questionSpeechRows(moduleCode string) ([]questionSpeechRow, error) {
	var rows []questionSpeechRow
	err := s.db.Raw(`
		SELECT q.id AS question_id, q.kp_id AS kp_id, q.code AS code,
		       q.speech AS speech, q.media_url AS media_url
		FROM questions q
		JOIN knowledge_points kp ON kp.id = q.kp_id
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'math' AND m.code = ?
		ORDER BY m.order_no, kp.order_no, kp.id, q.id`, moduleCode).Scan(&rows).Error
	return rows, err
}

func (s *Service) BatchGenerateSpeech(ctx context.Context, moduleCode string) (BatchResult, error) {
	moduleCode = strings.TrimSpace(moduleCode)
	if moduleCode == "" {
		return BatchResult{}, fmt.Errorf("moduleCode 不能为空")
	}
	if s.store == nil || s.voices == nil || s.speech == nil {
		return BatchResult{}, fmt.Errorf("TTS 未就绪（检查 MinIO 与语音配置）")
	}
	var assets []Asset
	if err := s.db.Where("module_code = ?", moduleCode).
		Order("module_order ASC, kp_order ASC, kp_id ASC").
		Find(&assets).Error; err != nil {
		return BatchResult{}, err
	}
	out := BatchResult{}
	needDelay := false
	for i := range assets {
		asset := &assets[i]
		if strings.TrimSpace(asset.SpeechAudioURL) != "" {
			out.Skipped++
			continue
		}
		text := strings.TrimSpace(asset.SpeechText)
		if text == "" {
			text = buildSpeechText(asset.Kind, asset.Title, asset.Payload)
			asset.SpeechText = text
		}
		if text == "" {
			out.Skipped++
			continue
		}
		if needDelay {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(400 * time.Millisecond):
			}
		}
		mp3, err := s.synthesize(ctx, text)
		if err != nil {
			out.Failed++
			if len(out.Errors) < 10 {
				out.Errors = append(out.Errors, fmt.Sprintf("%s(%d): %v", asset.Title, asset.KpID, err))
			}
			needDelay = true
			continue
		}
		if err := s.storeSpeech(ctx, asset, mp3); err != nil {
			out.Failed++
			if len(out.Errors) < 10 {
				out.Errors = append(out.Errors, fmt.Sprintf("%s(%d): %v", asset.Title, asset.KpID, err))
			}
			needDelay = true
			continue
		}
		out.Generated++
		needDelay = true
	}
	return out, nil
}

func (s *Service) synthesize(ctx context.Context, text string) ([]byte, error) {
	if s.voices == nil || s.speech == nil {
		return nil, fmt.Errorf("TTS 未就绪（检查共享配置中心）")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("朗读文本为空")
	}
	ai, err := s.voices.LoadVoiceModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("加载语音配置失败: %w", err)
	}
	order, err := tts.SynthesisOrder(ai)
	if err != nil {
		return nil, err
	}
	var last error
	for _, provider := range order {
		provider = mathTTSProvider(provider)
		mp3, synthErr := s.speech.Synthesize(ctx, provider, text)
		if synthErr == nil {
			return mp3, nil
		}
		last = synthErr
	}
	if last == nil {
		return nil, fmt.Errorf("配置中心没有可用的 TTS Provider（需 AUDIO_TTS）")
	}
	return nil, last
}

func mathTTSProvider(provider configclient.AIProvider) configclient.AIProvider {
	out := tts.WithLanguage(provider, "zh")
	id := strings.ToLower(strings.TrimSpace(provider.ID) + " " + provider.Type)
	if strings.Contains(id, "mimo") {
		out = tts.WithOption(out, "style", "用清晰、平稳、适合幼儿算术课的语速朗读算式或图形名称。")
	}
	return out
}

func (s *Service) storeSpeech(ctx context.Context, asset *Asset, mp3 []byte) error {
	if s.store == nil {
		return fmt.Errorf("语音存储未就绪（检查 MinIO）")
	}
	if _, err := s.store.PutBytes(ctx, storage.MathSpeechObjectKey(asset.KpID), mp3, "audio/mpeg"); err != nil {
		return err
	}
	asset.SpeechAudioURL = speechPublicURL(asset.KpID)
	asset.UpdatedAt = time.Now().UTC()
	return s.db.Save(asset).Error
}

func isShapeKind(a Asset) bool {
	return a.ModuleCode == "shape" || (a.Kind == "" && a.ModuleCode == "shape")
}

func publicBase() string {
	base := strings.TrimRight(os.Getenv("APP_PUBLIC_BASE"), "/")
	if base == "" {
		base = "http://localhost:19091"
	}
	return base
}

func glyphPublicURL(kpID int64) string {
	return fmt.Sprintf("%s/api/v1/math/items/%d/glyph.png", publicBase(), kpID)
}

func speechPublicURL(kpID int64) string {
	return fmt.Sprintf("%s/api/v1/math/items/%d/speech.mp3", publicBase(), kpID)
}

func payloadKind(raw string) string {
	var p struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal([]byte(raw), &p)
	return p.Kind
}

type mathPayload struct {
	Kind string `json:"kind"`
	A    int    `json:"a"`
	B    int    `json:"b"`
	N    int    `json:"n"`
}

func buildSpeechText(kind, title, payload string) string {
	var p mathPayload
	_ = json.Unmarshal([]byte(payload), &p)
	if kind == "" {
		kind = p.Kind
	}
	switch kind {
	case "add":
		return fmt.Sprintf("%s加%s", cnNumber(p.A), cnNumber(p.B))
	case "sub":
		return fmt.Sprintf("%s减%s", cnNumber(p.A), cnNumber(p.B))
	case "compare":
		return fmt.Sprintf("%s和%s比大小", cnNumber(p.A), cnNumber(p.B))
	case "number":
		return cnNumber(p.N)
	default:
		// 图形等：直接读标题
		return strings.TrimSpace(title)
	}
}

var cnDigits = []string{"零", "一", "二", "三", "四", "五", "六", "七", "八", "九", "十"}

func cnNumber(n int) string {
	if n < 0 {
		return fmt.Sprintf("%d", n)
	}
	if n < len(cnDigits) {
		return cnDigits[n]
	}
	if n < 20 {
		return "十" + cnDigits[n-10]
	}
	if n == 20 {
		return "二十"
	}
	return fmt.Sprintf("%d", n)
}

func toDTO(a Asset) ItemDTO {
	return ItemDTO{
		KpID:           a.KpID,
		Title:          a.Title,
		Kind:           a.Kind,
		Payload:        a.Payload,
		Difficulty:     a.Difficulty,
		ModuleCode:     a.ModuleCode,
		ModuleName:     a.ModuleName,
		ModuleOrder:    a.ModuleOrder,
		KpOrder:        a.KpOrder,
		GlyphImageURL:  withCacheBust(a.GlyphImageURL, a.UpdatedAt),
		SpeechAudioURL: withCacheBust(a.SpeechAudioURL, a.UpdatedAt),
		SpeechText:     a.SpeechText,
		Enabled:        a.Enabled,
	}
}

func withCacheBust(url string, updatedAt time.Time) string {
	if url == "" || updatedAt.IsZero() {
		return url
	}
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%sv=%d", url, sep, updatedAt.Unix())
}
