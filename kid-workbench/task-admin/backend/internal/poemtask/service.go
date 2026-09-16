package poemtask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/conchi/study-learning/poemcontent"
	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	contentURL string
}

func New(db *gorm.DB, contentURL string) *Service {
	return &Service{db: db, contentURL: strings.TrimRight(contentURL, "/")}
}

type taskRow struct {
	ID        int64 `gorm:"primaryKey"`
	Title     string
	Count     int
	TypesJSON string
	ItemsJSON string `gorm:"type:text"`
	CreatedAt time.Time
}

func (taskRow) TableName() string { return "poem_question_tasks" }

func Migrate(db *gorm.DB) error {
	if err := poemcontent.MigrateMedia(db); err != nil {
		return err
	}
	return db.AutoMigrate(&taskRow{})
}

type Item struct {
	ID                string                  `json:"id"`
	Kind              string                   `json:"kind"`
	SkillCode         string                   `json:"skillCode"`
	TargetID          int64                   `json:"targetId"`
	SourceID          int64                   `json:"sourceId"`
	SourceTable       string                  `json:"sourceTable"`
	ModuleCode        string                   `json:"moduleCode"`
	ModuleName        string                   `json:"moduleName"`
	SourceContentHash string                  `json:"sourceContentHash"`
	Example           poemcontent.PoemExample `json:"example"`
	MediaSHA256       map[string]string        `json:"mediaSHA256"`
}

type Task struct {
	ID        int64     `json:"id"`
	Title     string     `json:"title"`
	Count     int        `json:"count"`
	Types     []string   `json:"types"`
	Groups    []string   `json:"groups,omitempty"`
	Items     []Item     `json:"items,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type CreateInput struct {
	Title string   `json:"title"`
	Types []string `json:"types"`
	Count int      `json:"count"`
}

type stored struct {
	QuestionID int64
	KpID       int64
	Code       string
	Stem       string
	Options    string
	Answer     string
	Visual     string
	Speech     string
	Payload    string
	Title      string
	WorkCode   string
	ModuleCode string
	ModuleName string
}

var typeNames = map[string]string{
	poemcontent.KindTitle:   "选诗名",
	poemcontent.KindFill:    "补字",
	poemcontent.KindCouplet: "选下一句",
	poemcontent.KindRecite:  "排顺序",
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Task, error) {
	if in.Count < 1 || in.Count > 40 || len(in.Types) == 0 || len(in.Types) > 4 || in.Count < len(in.Types) {
		return Task{}, errors.New("请选择题型，题数须为1至40且不少于所选题型数")
	}
	seen := map[string]bool{}
	kinds := make([]string, 0, len(in.Types))
	for _, raw := range in.Types {
		kind := poemcontent.KindForSkill(raw)
		if typeNames[kind] == "" || seen[kind] {
			return Task{}, errors.New("古诗题型无效或重复")
		}
		seen[kind] = true
		kinds = append(kinds, kind)
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "古诗练习"
	}
	if len([]rune(title)) > 80 {
		return Task{}, errors.New("任务名称最多80字")
	}
	rows, err := s.loadStored()
	if err != nil {
		return Task{}, err
	}
	pools := map[string][]stored{}
	for _, row := range rows {
		kind := poemcontent.KindForSkill(row.Code)
		if typeNames[kind] == "" {
			continue
		}
		pools[kind] = append(pools[kind], row)
	}
	for _, kind := range kinds {
		if len(pools[kind]) == 0 {
			return Task{}, fmt.Errorf("古诗素材不足：缺少%s", typeNames[kind])
		}
	}
	items := make([]Item, 0, in.Count)
	used := map[string]map[int64]bool{}
	for i := 0; i < in.Count; i++ {
		kind := kinds[i%len(kinds)]
		if used[kind] == nil {
			used[kind] = map[int64]bool{}
		}
		row, ok := nextStored(pools[kind], used[kind])
		if !ok {
			return Task{}, fmt.Errorf("%s 可用素材不足", typeNames[kind])
		}
		used[kind][row.QuestionID] = true
		work := poemcontent.ParseWork(row.KpID, row.WorkCode, row.Title, row.Payload)
		snap, err := poemcontent.SnapshotFromLiveQuestion(poemcontent.LiveQuestion{
			Code: row.Code, Stem: row.Stem, Options: row.Options, Answer: row.Answer,
			Visual: row.Visual, Speech: row.Speech, Payload: row.Payload, TargetKpID: row.KpID, Work: work,
		})
		if err != nil {
			return Task{}, err
		}
		snap.Example = poemcontent.ShuffleExample(snap.Example, int64(i+1)+row.QuestionID)
		if s.contentURL != "" {
			if err := poemcontent.FreezeMedia(ctx, s.db, s.contentURL, &snap); err != nil {
				return Task{}, err
			}
		} else if poemcontent.NeedsSpeech(snap.Example.Kind) && !poemcontent.HasFrozenMedia(snap.Example) {
			return Task{}, errors.New("古诗题目缺少可冻结的读音")
		}
		if err := poemcontent.Validate(snap.Example); err != nil {
			return Task{}, err
		}
		hash := poemcontent.SourceHash(fmt.Sprintf("%d", row.KpID), row.Code, row.Stem, row.Options, row.Answer)
		items = append(items, Item{
			ID: fmt.Sprintf("poem-%d", i+1), Kind: kind, SkillCode: kind, TargetID: row.KpID, SourceID: row.KpID,
			SourceTable: "knowledge_points", ModuleCode: row.ModuleCode, ModuleName: row.ModuleName,
			SourceContentHash: hash, Example: snap.Example, MediaSHA256: snap.MediaSHA256,
		})
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return Task{}, err
	}
	types, _ := json.Marshal(kinds)
	row := taskRow{Title: title, Count: in.Count, TypesJSON: string(types), ItemsJSON: string(raw)}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return Task{}, err
	}
	return decode(row)
}

func nextStored(pool []stored, used map[int64]bool) (stored, bool) {
	for _, row := range pool {
		if !used[row.QuestionID] {
			return row, true
		}
	}
	return stored{}, false
}

func (s *Service) loadStored() ([]stored, error) {
	var rows []stored
	err := s.db.Raw(`
		SELECT q.id AS question_id, q.kp_id, q.code, q.stem, q.options, q.answer, q.visual,
		       COALESCE(q.speech,'') AS speech, COALESCE(kp.payload,'') AS payload, kp.title AS title,
		       kp.code AS work_code, m.code AS module_code, m.name AS module_name
		FROM questions q
		JOIN knowledge_points kp ON kp.id = q.kp_id
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects sub ON sub.id = m.subject_id
		WHERE sub.code = 'poem' AND q.code IN ('title','fill','couplet','recite','nextline')
		ORDER BY m.order_no, kp.order_no, q.id`).Scan(&rows).Error
	return rows, err
}

func decode(row taskRow) (Task, error) {
	task := Task{ID: row.ID, Title: row.Title, Count: row.Count, CreatedAt: row.CreatedAt}
	_ = json.Unmarshal([]byte(row.TypesJSON), &task.Types)
	err := json.Unmarshal([]byte(row.ItemsJSON), &task.Items)
	seen := map[string]bool{}
	for _, item := range task.Items {
		if name := item.ModuleName; name != "" && !seen[name] {
			seen[name] = true
			task.Groups = append(task.Groups, name)
		}
	}
	return task, err
}

func (s *Service) Get(id int64) (Task, error) {
	var row taskRow
	if err := s.db.First(&row, id).Error; err != nil {
		return Task{}, err
	}
	return decode(row)
}

func (s *Service) List() ([]Task, error) {
	var rows []taskRow
	if err := s.db.Order("id DESC").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	tasks := make([]Task, 0, len(rows))
	for _, r := range rows {
		t, e := decode(r)
		if e != nil {
			return nil, e
		}
		t.Items = nil
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *Service) Media(file string) ([]byte, string, error) {
	return poemcontent.MediaBytes(s.db, file)
}
