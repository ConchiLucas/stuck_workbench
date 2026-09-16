package chengyutask

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/conchi/study-learning/chengyucontent"
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

func (taskRow) TableName() string { return "chengyu_question_tasks" }

func Migrate(db *gorm.DB) error {
	if err := chengyucontent.MigrateMedia(db); err != nil {
		return err
	}
	return db.AutoMigrate(&taskRow{})
}

type Item struct {
	ID                string                    `json:"id"`
	Kind              string                    `json:"kind"`
	SkillCode         string                    `json:"skillCode"`
	TargetID          int64                     `json:"targetId"`
	SourceID          int64                     `json:"sourceId"`
	SourceTable       string                    `json:"sourceTable"`
	ModuleCode        string                    `json:"moduleCode"`
	ModuleName        string                    `json:"moduleName"`
	SourceContentHash string                    `json:"sourceContentHash"`
	Example           chengyucontent.ChengyuExample `json:"example"`
	MediaSHA256       map[string]string         `json:"mediaSHA256"`
}

type Task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Count     int       `json:"count"`
	Types     []string  `json:"types"`
	Groups    []string  `json:"groups,omitempty"`
	Items     []Item    `json:"items,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type CreateInput struct {
	Title string   `json:"title"`
	Types []string `json:"types"`
	Count int      `json:"count"`
}

type material struct {
	KpID       int64
	Title      string
	Pinyin     string
	Meaning    string
	Example    string
	Wrong      []string
	ModuleCode string
	ModuleName string
	HasSpeech  bool
}

var typeNames = map[string]string{
	"meaning": "听释义",
	"pick":    "选成语",
	"pinyin":  "看拼音",
	"example": "看句子",
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Task, error) {
	if in.Count < 1 || in.Count > 40 || len(in.Types) == 0 || len(in.Types) > 4 || in.Count < len(in.Types) {
		return Task{}, errors.New("请选择题型，题数须为1至40且不少于所选题型数")
	}
	seen := map[string]bool{}
	for _, typ := range in.Types {
		if typeNames[typ] == "" || seen[typ] {
			return Task{}, errors.New("成语题型无效或重复")
		}
		seen[typ] = true
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "成语练习"
	}
	if len([]rune(title)) > 80 {
		return Task{}, errors.New("任务名称最多80字")
	}
	mats, err := s.loadMaterials()
	if err != nil {
		return Task{}, err
	}
	needMeaning := 0
	for _, typ := range in.Types {
		if typ == "meaning" {
			needMeaning++
		}
	}
	repeats := (in.Count + len(in.Types) - 1) / len(in.Types)
	if needMeaning > 0 && len(spokenMaterials(mats)) < 4 {
		return Task{}, errors.New("成语素材不足：至少需要4条已有成语读音")
	}
	if len(mats) < 4 {
		return Task{}, errors.New("成语素材不足：每个模块至少需要4条成语才能组成选项")
	}
	_ = repeats
	items := make([]Item, 0, in.Count)
	used := map[string]map[int64]bool{}
	for i := 0; i < in.Count; i++ {
		kind := in.Types[i%len(in.Types)]
		if used[kind] == nil {
			used[kind] = map[int64]bool{}
		}
		item, err := generate(kind, mats, used[kind], int64(i+1))
		if err != nil {
			return Task{}, err
		}
		used[kind][item.SourceID] = true
		item.ID = fmt.Sprintf("chengyu-%d", i+1)
		item.MediaSHA256 = map[string]string{}
		snap := chengyucontent.HistorySnapshot{Schema: 1, Kind: item.Kind, SkillCode: item.SkillCode, Example: item.Example, ResponseKind: "choice"}
		if err := chengyucontent.FreezeMedia(ctx, s.db, s.contentURL, &snap); err != nil {
			return Task{}, err
		}
		item.Example = snap.Example
		item.MediaSHA256 = snap.MediaSHA256
		if item.MediaSHA256 == nil {
			item.MediaSHA256 = map[string]string{}
		}
		if err := chengyucontent.Validate(item.Example); err != nil {
			return Task{}, err
		}
		items = append(items, item)
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return Task{}, err
	}
	types, _ := json.Marshal(in.Types)
	row := taskRow{Title: title, Count: in.Count, TypesJSON: string(types), ItemsJSON: string(raw)}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return Task{}, err
	}
	return decode(row)
}

func (s *Service) loadMaterials() ([]material, error) {
	type row struct {
		KpID                   int64
		Title, Payload, ModuleCode, ModuleName, SpeechSHA string
	}
	var rows []row
	err := s.db.Raw(`
		SELECT kp.id AS kp_id, kp.title AS title, COALESCE(kp.payload,'') AS payload,
		       m.code AS module_code, m.name AS module_name, COALESCE(cs.sha256,'') AS speech_sha
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects sub ON sub.id = m.subject_id
		LEFT JOIN chengyu_item_speech cs ON cs.kp_id = kp.id AND cs.kind = 'chengyu'
		WHERE sub.code = 'chengyu'
		ORDER BY m.order_no, kp.order_no, kp.id`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]material, 0, len(rows))
	for _, r := range rows {
		var p struct {
			Kind    string   `json:"kind"`
			Pinyin  string   `json:"pinyin"`
			Meaning string   `json:"meaning"`
			Example string   `json:"example"`
			Wrong   []string `json:"wrong"`
		}
		_ = json.Unmarshal([]byte(r.Payload), &p)
		if p.Kind != "chengyu" || strings.TrimSpace(p.Meaning) == "" {
			continue
		}
		wrong := make([]string, 0, len(p.Wrong))
		for _, item := range p.Wrong {
			item = strings.TrimSpace(item)
			if item != "" && item != p.Meaning {
				wrong = append(wrong, item)
			}
		}
		out = append(out, material{
			KpID: r.KpID, Title: strings.TrimSpace(r.Title), Pinyin: strings.TrimSpace(p.Pinyin),
			Meaning: strings.TrimSpace(p.Meaning), Example: strings.TrimSpace(p.Example), Wrong: wrong,
			ModuleCode: r.ModuleCode, ModuleName: r.ModuleName, HasSpeech: r.SpeechSHA != "",
		})
	}
	return out, nil
}

func spokenMaterials(mats []material) []material {
	out := make([]material, 0, len(mats))
	for _, m := range mats {
		if m.HasSpeech {
			out = append(out, m)
		}
	}
	return out
}

func generate(kind string, mats []material, used map[int64]bool, seed int64) (Item, error) {
	pool := mats
	if kind == "meaning" {
		pool = spokenMaterials(mats)
	}
	var target *material
	for i := range pool {
		if !used[pool[i].KpID] {
			item := pool[i]
			if kind == "example" {
				if _, err := chengyucontent.FirstBlank(item.Example, item.Title); err != nil {
					continue
				}
			}
			if kind == "pinyin" && strings.TrimSpace(item.Pinyin) == "" {
				continue
			}
			target = &item
			break
		}
	}
	if target == nil {
		return Item{}, fmt.Errorf("%s 可用成语不足", typeNames[kind])
	}
	example, err := buildExample(kind, *target, mats, seed)
	if err != nil {
		return Item{}, err
	}
	hash := sha256.Sum256([]byte(strings.Join([]string{strconv.FormatInt(target.KpID, 10), target.Title, target.Pinyin, target.Meaning, target.Example, strings.Join(target.Wrong, ",")}, "|")))
	return Item{
		Kind: kind, SkillCode: kind, TargetID: target.KpID, SourceID: target.KpID, SourceTable: "knowledge_points",
		ModuleCode: target.ModuleCode, ModuleName: target.ModuleName, SourceContentHash: hex.EncodeToString(hash[:]),
		Example: example,
	}, nil
}

func buildExample(kind string, target material, mats []material, seed int64) (chengyucontent.ChengyuExample, error) {
	example := chengyucontent.ChengyuExample{
		Kind: kind, Chengyu: target.Title, Pinyin: target.Pinyin, Meaning: target.Meaning, Example: target.Example,
	}
	rng := rand.New(rand.NewSource(seed*97 + target.KpID))
	switch kind {
	case "meaning":
		example.Stem = "这个成语是什么意思？"
		example.Speech = target.Title
		example.SpeechURL = fmt.Sprintf("/api/v1/chengyu/items/%d/speech.mp3", target.KpID)
		example.Options = meaningOptions(target, rng)
		example.AnswerID = "label:" + target.Meaning
	case "pick":
		example.Stem = "看意思，点出这个成语"
		example.Prompt = target.Meaning
		example.Options = idiomOptions(target, mats, rng)
		example.AnswerID = optionID(target)
	case "pinyin":
		example.Prompt = target.Pinyin
		example.Options = idiomOptions(target, mats, rng)
		example.AnswerID = optionID(target)
	case "example":
		blank, err := chengyucontent.FirstBlank(target.Example, target.Title)
		if err != nil {
			return chengyucontent.ChengyuExample{}, err
		}
		example.Blank = &blank
		example.Prompt = blank.Blanked
		example.Options = idiomOptions(target, mats, rng)
		example.AnswerID = optionID(target)
	}
	if len(example.Options) < 2 {
		return chengyucontent.ChengyuExample{}, fmt.Errorf("%s 选项不足", typeNames[kind])
	}
	return example, nil
}

func optionID(m material) string {
	if m.KpID > 0 {
		return strconv.FormatInt(m.KpID, 10)
	}
	return "label:" + m.Title
}

func meaningOptions(target material, rng *rand.Rand) []chengyucontent.Choice {
	labels := append([]string{target.Meaning}, target.Wrong...)
	out := make([]chengyucontent.Choice, 0, 4)
	seen := map[string]bool{}
	for _, label := range labels {
		if seen[label] || label == "" {
			continue
		}
		seen[label] = true
		out = append(out, chengyucontent.Choice{ID: "label:" + label, Label: label})
		if len(out) == 4 {
			break
		}
	}
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func idiomOptions(target material, mats []material, rng *rand.Rand) []chengyucontent.Choice {
	others := make([]material, 0, len(mats))
	for _, m := range mats {
		if m.KpID != target.KpID && m.Title != target.Title {
			others = append(others, m)
		}
	}
	rng.Shuffle(len(others), func(i, j int) { others[i], others[j] = others[j], others[i] })
	out := []chengyucontent.Choice{{ID: optionID(target), Label: target.Title}}
	same := make([]material, 0)
	rest := make([]material, 0)
	for _, m := range others {
		if m.ModuleCode == target.ModuleCode {
			same = append(same, m)
		} else {
			rest = append(rest, m)
		}
	}
	for _, pool := range [][]material{same, rest} {
		for _, m := range pool {
			if len(out) == 4 {
				break
			}
			out = append(out, chengyucontent.Choice{ID: optionID(m), Label: m.Title})
		}
	}
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
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

func (s *Service) Media(hash string) ([]byte, error) {
	return chengyucontent.MediaBytes(s.db, hash+".mp3")
}
