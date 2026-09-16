package chengyu

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
	Pinyin  string   `json:"pinyin"`
	Meaning string   `json:"meaning"`
	Example string   `json:"example"`
	Wrong   []string `json:"wrong"`
}

func ParsePayload(raw string) Payload {
	var p Payload
	_ = json.Unmarshal([]byte(raw), &p)
	p.Kind = strings.TrimSpace(p.Kind)
	p.Pinyin = strings.TrimSpace(p.Pinyin)
	p.Meaning = strings.TrimSpace(p.Meaning)
	p.Example = strings.TrimSpace(p.Example)
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
	KpID                 int64    `json:"kpId"`
	Title                string   `json:"title"`
	Pinyin               string   `json:"pinyin"`
	Meaning              string   `json:"meaning"`
	Example              string   `json:"example"`
	Wrong                []string `json:"wrong"`
	Difficulty           int      `json:"difficulty"`
	ModuleCode           string   `json:"moduleCode"`
	ModuleName           string   `json:"moduleName"`
	ModuleOrder          int      `json:"moduleOrder"`
	KpOrder              int      `json:"kpOrder"`
	HasChengyuSpeech     bool     `json:"hasChengyuSpeech"`
	ChengyuSpeechURL     string   `json:"chengyuSpeechUrl,omitempty"`
	ChengyuSpeechSHA256  string   `json:"chengyuSpeechSha256,omitempty"`
	HasMeaningSpeech     bool     `json:"hasMeaningSpeech"`
	MeaningSpeechURL     string   `json:"meaningSpeechUrl,omitempty"`
	MeaningSpeechSHA256  string   `json:"meaningSpeechSha256,omitempty"`
	HasExampleSpeech     bool     `json:"hasExampleSpeech"`
	ExampleSpeechURL     string   `json:"exampleSpeechUrl,omitempty"`
	ExampleSpeechSHA256  string   `json:"exampleSpeechSha256,omitempty"`
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
	Kind   string `gorm:"primaryKey;column:kind;size:16"`
	Text   string `gorm:"column:text"`
	SHA256 string `gorm:"column:sha256"`
	Data   []byte `gorm:"column:data"`
}

func (speechRow) TableName() string { return "chengyu_item_speech" }

var speechKinds = map[string]string{
	"chengyu": "speech.mp3",
	"meaning": "meaning.mp3",
	"example": "example.mp3",
}

type kpRow struct {
	ID            int64
	Title         string
	Payload       string
	Difficulty    int
	KpOrder       int
	ModuleCode    string
	ModuleName    string
	ModuleOrder   int
	ChengyuSHA    string
	MeaningSHA    string
	ExampleSHA    string
}

func (s *Service) List(view string) (ListResult, error) {
	if view == "" {
		view = "groups"
	}
	var rows []kpRow
	err := s.db.Raw(`
		SELECT kp.id AS id, kp.title AS title, kp.payload AS payload, kp.difficulty AS difficulty,
		       kp.order_no AS kp_order, m.code AS module_code, m.name AS module_name, m.order_no AS module_order,
		       COALESCE(cs.sha256,'') AS chengyu_sha, COALESCE(ms.sha256,'') AS meaning_sha, COALESCE(es.sha256,'') AS example_sha
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		LEFT JOIN chengyu_item_speech cs ON cs.kp_id = kp.id AND cs.kind = 'chengyu'
		LEFT JOIN chengyu_item_speech ms ON ms.kp_id = kp.id AND ms.kind = 'meaning'
		LEFT JOIN chengyu_item_speech es ON es.kp_id = kp.id AND es.kind = 'example'
		WHERE s.code = ?
		ORDER BY m.order_no, kp.order_no, kp.id
	`, "chengyu").Scan(&rows).Error
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
			Pinyin:      content.Pinyin,
			Meaning:     content.Meaning,
			Example:     content.Example,
			Wrong:       content.Wrong,
			Difficulty:  diff,
			ModuleCode:  row.ModuleCode,
			ModuleName:  row.ModuleName,
			ModuleOrder: row.ModuleOrder,
			KpOrder:     row.KpOrder,
			HasChengyuSpeech: row.ChengyuSHA != "",
			HasMeaningSpeech: row.MeaningSHA != "",
			HasExampleSpeech: row.ExampleSHA != "",
		}
		if item.HasChengyuSpeech {
			item.ChengyuSpeechURL = fmt.Sprintf("/api/v1/chengyu/items/%d/speech.mp3", row.ID)
			item.ChengyuSpeechSHA256 = row.ChengyuSHA
		}
		if item.HasMeaningSpeech {
			item.MeaningSpeechURL = fmt.Sprintf("/api/v1/chengyu/items/%d/meaning.mp3", row.ID)
			item.MeaningSpeechSHA256 = row.MeaningSHA
		}
		if item.HasExampleSpeech {
			item.ExampleSpeechURL = fmt.Sprintf("/api/v1/chengyu/items/%d/example.mp3", row.ID)
			item.ExampleSpeechSHA256 = row.ExampleSHA
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

func (s *Service) SpeechMP3(ctx context.Context, kpID int64, kind string) ([]byte, error) {
	if speechKinds[kind] == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var row speechRow
	if err := s.db.WithContext(ctx).Where("kp_id = ? AND kind = ?", kpID, kind).First(&row).Error; err != nil {
		return nil, err
	}
	if len(row.Data) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return row.Data, nil
}

func (s *Service) StoreSpeech(ctx context.Context, kpID int64, kind string, mp3 []byte) (ItemDTO, error) {
	if speechKinds[kind] == "" {
		return ItemDTO{}, errors.New("未知的成语音频种类")
	}
	if !isMP3(mp3) {
		return ItemDTO{}, errors.New("成语音频必须是 MP3")
	}
	var title string
	err := s.db.WithContext(ctx).Raw(`
		SELECT kp.title FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects sub ON sub.id = m.subject_id
		WHERE sub.code = 'chengyu' AND kp.id = ?`, kpID).Scan(&title).Error
	if err != nil {
		return ItemDTO{}, err
	}
	if strings.TrimSpace(title) == "" {
		return ItemDTO{}, gorm.ErrRecordNotFound
	}
	sum := sha256.Sum256(mp3)
	row := speechRow{KpID: kpID, Kind: kind, Text: title, SHA256: hex.EncodeToString(sum[:]), Data: mp3}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "kp_id"}, {Name: "kind"}},
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
