package phrase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Payload struct {
	Kind    string   `json:"kind"`
	Zh      string   `json:"zh"`
	Wrong   []string `json:"wrong"`
	Scene   string   `json:"scene"`
	ReplyTo string   `json:"replyTo"`
}

func ParsePayload(raw string) Payload {
	var p Payload
	_ = json.Unmarshal([]byte(raw), &p)
	p.Kind = strings.TrimSpace(p.Kind)
	p.Zh = strings.TrimSpace(p.Zh)
	p.Scene = strings.TrimSpace(p.Scene)
	p.ReplyTo = strings.TrimSpace(p.ReplyTo)
	wrong := make([]string, 0, len(p.Wrong))
	for _, item := range p.Wrong {
		item = strings.TrimSpace(item)
		if item != "" {
			wrong = append(wrong, item)
		}
	}
	p.Wrong = wrong
	return p
}

type ItemDTO struct {
	KpID           int64    `json:"kpId"`
	Title          string   `json:"title"`
	Zh             string   `json:"zh"`
	Wrong          []string `json:"wrong"`
	Scene          string   `json:"scene"`
	ReplyTo        string   `json:"replyTo"`
	Difficulty     int      `json:"difficulty"`
	ModuleCode     string   `json:"moduleCode"`
	ModuleName     string   `json:"moduleName"`
	ModuleOrder    int      `json:"moduleOrder"`
	KpOrder        int      `json:"kpOrder"`
	HasSpeech      bool     `json:"hasSpeech"`
	SpeechAudioURL string   `json:"speechAudioUrl,omitempty"`
	SpeechSHA256   string   `json:"speechSha256,omitempty"`
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

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type speechRow struct {
	KpID   int64  `gorm:"primaryKey;column:kp_id"`
	Text   string `gorm:"column:text"`
	SHA256 string `gorm:"column:sha256"`
	Data   []byte `gorm:"column:data"`
}

func (speechRow) TableName() string { return "phrase_item_speech" }

type kpRow struct {
	ID          int64
	Title       string
	Payload     string
	Difficulty  int
	KpOrder     int
	ModuleCode  string
	ModuleName  string
	ModuleOrder int
	SpeechSHA   string
}

func (s *Service) List(view string) (ListResult, error) {
	if view == "" {
		view = "groups"
	}
	var rows []kpRow
	err := s.db.Raw(`
		SELECT kp.id AS id, kp.title AS title, kp.payload AS payload, kp.difficulty AS difficulty,
		       kp.order_no AS kp_order, m.code AS module_code, m.name AS module_name, m.order_no AS module_order,
		       COALESCE(ps.sha256,'') AS speech_sha
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects sub ON sub.id = m.subject_id
		LEFT JOIN phrase_item_speech ps ON ps.kp_id = kp.id
		WHERE sub.code = ?
		ORDER BY m.order_no, kp.order_no, kp.id
	`, "phrase").Scan(&rows).Error
	if err != nil {
		return ListResult{}, err
	}

	dtos := make([]ItemDTO, 0, len(rows))
	for _, row := range rows {
		content := ParsePayload(row.Payload)
		diff := row.Difficulty
		if diff < 1 {
			diff = 1
		}
		item := ItemDTO{
			KpID:        row.ID,
			Title:       row.Title,
			Zh:          content.Zh,
			Wrong:       content.Wrong,
			Scene:       content.Scene,
			ReplyTo:     content.ReplyTo,
			Difficulty:  diff,
			ModuleCode:  row.ModuleCode,
			ModuleName:  row.ModuleName,
			ModuleOrder: row.ModuleOrder,
			KpOrder:     row.KpOrder,
			HasSpeech:   row.SpeechSHA != "",
		}
		if item.HasSpeech {
			item.SpeechAudioURL = fmt.Sprintf("/api/v1/phrase/items/%d/speech.mp3", row.ID)
			item.SpeechSHA256 = row.SpeechSHA
		}
		dtos = append(dtos, item)
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

func (s *Service) SpeechMP3(ctx context.Context, kpID int64) ([]byte, error) {
	var row speechRow
	if err := s.db.WithContext(ctx).Where("kp_id = ?", kpID).First(&row).Error; err != nil {
		return nil, err
	}
	if len(row.Data) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return row.Data, nil
}

func (s *Service) StoreSpeech(ctx context.Context, kpID int64, mp3 []byte) (ItemDTO, error) {
	if !isMP3(mp3) {
		return ItemDTO{}, errors.New("整句读音必须是 MP3")
	}
	var title string
	err := s.db.WithContext(ctx).Raw(`
		SELECT kp.title FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects sub ON sub.id = m.subject_id
		WHERE sub.code = 'phrase' AND kp.id = ?`, kpID).Scan(&title).Error
	if err != nil {
		return ItemDTO{}, err
	}
	if strings.TrimSpace(title) == "" {
		return ItemDTO{}, gorm.ErrRecordNotFound
	}
	sum := sha256.Sum256(mp3)
	row := speechRow{KpID: kpID, Text: title, SHA256: hex.EncodeToString(sum[:]), Data: mp3}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "kp_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"text", "sha256", "data"}),
	}).Create(&row).Error; err != nil {
		return ItemDTO{}, err
	}
	list, err := s.List("table")
	if err != nil {
		return ItemDTO{}, err
	}
	for _, item := range list.Items {
		if item.KpID == kpID {
			return item, nil
		}
	}
	return ItemDTO{}, gorm.ErrRecordNotFound
}

func isMP3(data []byte) bool {
	return bytes.HasPrefix(data, []byte("ID3")) || (len(data) >= 2 && data[0] == 255 && data[1]&224 == 224)
}
