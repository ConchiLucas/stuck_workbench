package poem

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/conchi/study-learning/poemcontent"
	"gorm.io/gorm"
)

type Asset struct {
	KpID        int64     `gorm:"column:kp_id;primaryKey" json:"kpId"`
	Code        string    `gorm:"column:code" json:"code"`
	Title       string    `gorm:"column:title" json:"title"`
	Author      string    `gorm:"column:author" json:"author"`
	Dynasty     string    `gorm:"column:dynasty" json:"dynasty"`
	Edition     string    `gorm:"column:edition" json:"edition"`
	WorkID      string    `gorm:"column:work_id" json:"workId"`
	Line1       string    `gorm:"column:line1" json:"line1"`
	Line2       string    `gorm:"column:line2" json:"line2"`
	LinesJSON   string    `gorm:"column:lines_json" json:"-"`
	Difficulty  int       `gorm:"column:difficulty" json:"difficulty"`
	ModuleCode  string    `gorm:"column:module_code" json:"moduleCode"`
	ModuleName  string    `gorm:"column:module_name" json:"moduleName"`
	ModuleOrder int       `gorm:"column:module_order" json:"moduleOrder"`
	KpOrder     int       `gorm:"column:kp_order" json:"kpOrder"`
	SyncedAt    time.Time `gorm:"column:synced_at" json:"syncedAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (Asset) TableName() string { return "poem_assets" }

type Payload struct {
	Kind     string          `json:"kind"`
	WorkID   string          `json:"workId"`
	Author   string          `json:"author"`
	Dynasty  string          `json:"dynasty"`
	Edition  string          `json:"edition"`
	Line1    string          `json:"line1"`
	Line2    string          `json:"line2"`
	Lines    json.RawMessage `json:"lines"`
}

func ParsePayload(raw string) Payload {
	var p Payload
	_ = json.Unmarshal([]byte(raw), &p)
	p.Author = strings.TrimSpace(p.Author)
	p.Dynasty = strings.TrimSpace(p.Dynasty)
	p.Edition = strings.TrimSpace(p.Edition)
	p.WorkID = strings.TrimSpace(p.WorkID)
	p.Line1 = strings.TrimSpace(p.Line1)
	p.Line2 = strings.TrimSpace(p.Line2)
	return p
}

func payloadLines(p Payload) []string {
	w := poemcontent.ParseWork(0, p.WorkID, "", mustJSON(p))
	return poemcontent.Texts(w)
}

func mustJSON(p Payload) string {
	raw, _ := json.Marshal(p)
	return string(raw)
}

type ItemDTO struct {
	KpID        int64             `json:"kpId"`
	Code        string            `json:"code"`
	WorkID      string            `json:"workId"`
	Title       string            `json:"title"`
	Author      string            `json:"author"`
	Dynasty     string            `json:"dynasty"`
	Edition     string            `json:"edition"`
	Line1       string            `json:"line1"`
	Line2       string            `json:"line2"`
	Lines       []string          `json:"lines"`
	LineItems   []poemcontent.Line `json:"lineItems"`
	SpeechOrds  []int             `json:"speechOrds"`
	Difficulty  int              `json:"difficulty"`
	ModuleCode  string            `json:"moduleCode"`
	ModuleName  string            `json:"moduleName"`
	ModuleOrder int               `json:"moduleOrder"`
	KpOrder     int               `json:"kpOrder"`
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

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type kpRow struct {
	ID          int64
	Code        string
	Title       string
	Payload     string
	Difficulty  int
	KpOrder     int
	ModuleCode  string
	ModuleName  string
	ModuleOrder int
}

func (s *Service) Sync() (SyncResult, error) {
	if err := s.db.AutoMigrate(&Asset{}); err != nil {
		return SyncResult{}, err
	}
	var rows []kpRow
	err := s.db.Raw(`
		SELECT kp.id AS id, kp.code AS code, kp.title AS title, kp.payload AS payload, kp.difficulty AS difficulty,
		       kp.order_no AS kp_order, m.code AS module_code, m.name AS module_name, m.order_no AS module_order
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = ?
		ORDER BY m.order_no, kp.order_no, kp.id
	`, "poem").Scan(&rows).Error
	if err != nil {
		return SyncResult{}, err
	}

	now := time.Now().UTC()
	upserted := 0
	for _, row := range rows {
		work := poemcontent.ParseWork(row.ID, row.Code, row.Title, row.Payload)
		content := ParsePayload(row.Payload)
		if work.WorkID == "" {
			work.WorkID = strings.TrimSpace(row.Code)
		}
		if work.Dynasty == "" {
			work.Dynasty = content.Dynasty
		}
		if work.Edition == "" {
			work.Edition = content.Edition
		}
		if work.Edition == "" {
			work.Edition = poemcontent.Edition
		}
		texts := poemcontent.Texts(work)
		linesJSON, _ := json.Marshal(work.Lines)
		diff := row.Difficulty
		if diff < 1 {
			diff = 1
		}
		line1, line2 := "", ""
		if len(texts) > 0 {
			line1 = texts[0]
		}
		if len(texts) > 1 {
			line2 = texts[1]
		}
		var existing Asset
		findErr := s.db.First(&existing, "kp_id = ?", row.ID).Error
		asset := Asset{
			KpID: row.ID, Code: row.Code, Title: row.Title, Author: work.Author, Dynasty: work.Dynasty,
			Edition: work.Edition, WorkID: work.WorkID, Line1: line1, Line2: line2, LinesJSON: string(linesJSON),
			Difficulty: diff, ModuleCode: row.ModuleCode, ModuleName: row.ModuleName, ModuleOrder: row.ModuleOrder,
			KpOrder: row.KpOrder, SyncedAt: now, UpdatedAt: now,
		}
		if findErr == gorm.ErrRecordNotFound {
			if err := s.db.Create(&asset).Error; err != nil {
				return SyncResult{}, err
			}
			upserted++
			continue
		}
		if findErr != nil {
			return SyncResult{}, findErr
		}
		asset.SyncedAt = now
		if err := s.db.Save(&asset).Error; err != nil {
			return SyncResult{}, err
		}
		upserted++
	}
	if err := poemcontent.EnsureMaterials(s.db); err != nil {
		return SyncResult{}, err
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

func toDTO(a Asset) ItemDTO {
	raw, _ := json.Marshal(map[string]any{
		"workId": a.WorkID, "author": a.Author, "dynasty": a.Dynasty, "edition": a.Edition,
		"lines": json.RawMessage(orJSON(a.LinesJSON)),
	})
	work := poemcontent.ParseWork(a.KpID, a.Code, a.Title, string(raw))
	if work.WorkID == "" {
		work.WorkID = a.WorkID
		if work.WorkID == "" {
			work.WorkID = a.Code
		}
	}
	texts := poemcontent.Texts(work)
	ords := make([]int, 0, len(work.Lines))
	for _, line := range work.Lines {
		ords = append(ords, line.Ord)
	}
	return ItemDTO{
		KpID: a.KpID, Code: a.Code, WorkID: work.WorkID, Title: a.Title, Author: a.Author, Dynasty: work.Dynasty,
		Edition: work.Edition, Line1: a.Line1, Line2: a.Line2, Lines: texts, LineItems: work.Lines, SpeechOrds: ords,
		Difficulty: a.Difficulty, ModuleCode: a.ModuleCode, ModuleName: a.ModuleName, ModuleOrder: a.ModuleOrder, KpOrder: a.KpOrder,
	}
}

func orJSON(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "[]"
	}
	return raw
}

func (s *Service) Speech(kpID int64, ord int) ([]byte, string, error) {
	return poemcontent.ItemSpeech(s.db, kpID, ord)
}
