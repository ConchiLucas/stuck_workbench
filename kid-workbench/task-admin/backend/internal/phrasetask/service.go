package phrasetask

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/conchi/study-learning/phrasecontent"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	db         *gorm.DB
	contentURL string
	client     *http.Client
}

func New(db *gorm.DB, contentURL string) *Service {
	return &Service{db: db, contentURL: strings.TrimRight(contentURL, "/"), client: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

type taskRow struct {
	ID        int64 `gorm:"primaryKey"`
	Title     string
	Count     int
	TypesJSON string
	ItemsJSON string `gorm:"type:text"`
	CreatedAt time.Time
}

func (taskRow) TableName() string { return "phrase_question_tasks" }

type mediaRow struct {
	SHA256 string `gorm:"primaryKey;size:64"`
	Kind   string
	Data   []byte
}

func (mediaRow) TableName() string { return "phrase_question_task_media" }

func Migrate(db *gorm.DB) error { return db.AutoMigrate(&taskRow{}, &mediaRow{}) }

type Item struct {
	ID                string                   `json:"id"`
	Kind              string                   `json:"kind"`
	SkillCode         string                   `json:"skillCode"`
	TargetID          int64                    `json:"targetId"`
	SourceID          int64                    `json:"sourceId"`
	SourceTable       string                   `json:"sourceTable"`
	ModuleCode        string                   `json:"moduleCode"`
	ModuleName        string                   `json:"moduleName"`
	SourceContentHash string                   `json:"sourceContentHash"`
	Example           phrasecontent.PhraseExample `json:"example"`
	MediaSHA256       map[string]string        `json:"mediaSHA256"`
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
	Zh         string
	Wrong      []string
	Scene      string
	ReplyTo    string
	ModuleCode string
	ModuleName string
	HasSpeech  bool
}

var typeNames = map[string]string{
	"listen_zh": "听一听",
	"listen_en": "选句子",
	"scene":     "什么时候说",
	"reply":     "问与答",
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Task, error) {
	if in.Count < 1 || in.Count > 40 || len(in.Types) == 0 || len(in.Types) > 4 || in.Count < len(in.Types) {
		return Task{}, errors.New("请选择题型，题数须为1至40且不少于所选题型数")
	}
	seen := map[string]bool{}
	for _, typ := range in.Types {
		if typeNames[typ] == "" || seen[typ] {
			return Task{}, errors.New("短句题型无效或重复")
		}
		seen[typ] = true
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "短句练习"
	}
	if len([]rune(title)) > 80 {
		return Task{}, errors.New("任务名称最多80字")
	}
	mats, err := s.loadMaterials()
	if err != nil {
		return Task{}, err
	}
	needListen, needScene, needReply := 0, 0, 0
	for _, typ := range in.Types {
		switch typ {
		case "scene":
			needScene++
		case "reply":
			needReply++
		default:
			needListen++
		}
	}
	repeats := (in.Count + len(in.Types) - 1) / len(in.Types)
	spoken := spokenMaterials(mats)
	if needListen > 0 && len(spoken) < 4 {
		return Task{}, errors.New("短句素材不足：至少需要4句已有整句读音的短句")
	}
	if needScene > 0 && len(sceneMaterials(mats)) < repeats {
		return Task{}, errors.New("短句素材不足：需要带场景的短句")
	}
	if needReply > 0 && len(replyMaterials(mats)) < repeats {
		return Task{}, errors.New("短句素材不足：需要问句与答句对应且问句已有整句读音")
	}
	items := make([]Item, 0, in.Count)
	used := map[string]map[int64]bool{}
	cache := map[string]mediaRow{}
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
		item.ID = fmt.Sprintf("phrase-%d", i+1)
		item.MediaSHA256 = map[string]string{}
		if err := s.freeze(ctx, &item, cache); err != nil {
			return Task{}, err
		}
		if err := phrasecontent.Validate(item.Example); err != nil {
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
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, media := range cache {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&media).Error; err != nil {
				return err
			}
		}
		return tx.Create(&row).Error
	})
	if err != nil {
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
		       m.code AS module_code, m.name AS module_name, COALESCE(ps.sha256,'') AS speech_sha
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects sub ON sub.id = m.subject_id
		LEFT JOIN phrase_item_speech ps ON ps.kp_id = kp.id
		WHERE sub.code = 'phrase'
		ORDER BY m.order_no, kp.order_no, kp.id`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]material, 0, len(rows))
	for _, r := range rows {
		var p struct {
			Kind    string   `json:"kind"`
			Zh      string   `json:"zh"`
			Wrong   []string `json:"wrong"`
			Scene   string   `json:"scene"`
			ReplyTo string   `json:"replyTo"`
		}
		_ = json.Unmarshal([]byte(r.Payload), &p)
		if p.Kind != "phrase" || strings.TrimSpace(p.Zh) == "" {
			continue
		}
		wrong := make([]string, 0, len(p.Wrong))
		for _, item := range p.Wrong {
			item = strings.TrimSpace(item)
			if item != "" && item != p.Zh {
				wrong = append(wrong, item)
			}
		}
		out = append(out, material{
			KpID: r.KpID, Title: strings.TrimSpace(r.Title), Zh: strings.TrimSpace(p.Zh), Wrong: wrong,
			Scene: strings.TrimSpace(p.Scene), ReplyTo: strings.TrimSpace(p.ReplyTo),
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

func sceneMaterials(mats []material) []material {
	out := make([]material, 0, len(mats))
	for _, m := range mats {
		if m.Scene != "" && len(m.Title) > 0 {
			out = append(out, m)
		}
	}
	return out
}

func replyMaterials(mats []material) []material {
	byTitle := map[string]material{}
	for _, m := range mats {
		byTitle[m.Title] = m
	}
	out := make([]material, 0)
	for _, m := range mats {
		q, ok := byTitle[m.ReplyTo]
		if m.ReplyTo != "" && ok && q.HasSpeech {
			out = append(out, m)
		}
	}
	return out
}

func generate(kind string, mats []material, used map[int64]bool, seed int64) (Item, error) {
	pool := mats
	switch kind {
	case "listen_zh", "listen_en":
		pool = spokenMaterials(mats)
	case "scene":
		pool = sceneMaterials(mats)
	case "reply":
		pool = replyMaterials(mats)
	}
	var target *material
	for i := range pool {
		if !used[pool[i].KpID] {
			target = &pool[i]
			break
		}
	}
	if target == nil {
		return Item{}, fmt.Errorf("%s 可用短句不足", typeNames[kind])
	}
	example, err := buildExample(kind, *target, mats, seed)
	if err != nil {
		return Item{}, err
	}
	hash := sha256.Sum256([]byte(strings.Join([]string{strconv.FormatInt(target.KpID, 10), target.Title, target.Zh, target.Scene, target.ReplyTo, strings.Join(target.Wrong, ",")}, "|")))
	return Item{
		Kind: kind, SkillCode: kind, TargetID: target.KpID, SourceID: target.KpID, SourceTable: "knowledge_points",
		ModuleCode: target.ModuleCode, ModuleName: target.ModuleName, SourceContentHash: hex.EncodeToString(hash[:]),
		Example: example,
	}, nil
}

func buildExample(kind string, target material, mats []material, seed int64) (phrasecontent.PhraseExample, error) {
	byTitle := map[string]material{}
	byZh := map[string]material{}
	for _, m := range mats {
		byTitle[m.Title] = m
		byZh[m.Zh] = m
	}
	example := phrasecontent.PhraseExample{
		Kind: kind, Phrase: target.Title, MeaningZh: target.Zh, Scene: target.Scene, ReplyTo: target.ReplyTo,
	}
	rng := rand.New(rand.NewSource(seed*97 + target.KpID))
	switch kind {
	case "listen_zh":
		example.Stem = "听一听，选出中文意思"
		example.Speech = target.Title
		example.SpeechURL = fmt.Sprintf("/api/v1/phrase/items/%d/speech.mp3", target.KpID)
		example.Options = zhOptions(target, byZh, rng)
	case "listen_en":
		example.Stem = "听一听，点出这句英语"
		example.Speech = target.Title
		example.SpeechURL = fmt.Sprintf("/api/v1/phrase/items/%d/speech.mp3", target.KpID)
		example.Options = enOptions(target, mats, rng)
	case "scene":
		example.Stem = "这种时候该说哪一句？"
		example.Prompt = target.Scene
		example.Options = enOptions(target, mats, rng)
	case "reply":
		q := byTitle[target.ReplyTo]
		example.Stem = "对方说了这句话，你怎么答？"
		example.Prompt = target.ReplyTo
		example.Speech = target.ReplyTo
		example.SpeechURL = fmt.Sprintf("/api/v1/phrase/items/%d/speech.mp3", q.KpID)
		example.Options = enOptions(target, mats, rng)
	}
	if len(example.Options) < 2 {
		return phrasecontent.PhraseExample{}, fmt.Errorf("%s 选项不足", typeNames[kind])
	}
	example.AnswerID = optionID(target)
	return example, nil
}

func optionID(m material) string {
	if m.KpID > 0 {
		return strconv.FormatInt(m.KpID, 10)
	}
	return "label:" + m.Title
}

func zhOptions(target material, byZh map[string]material, rng *rand.Rand) []phrasecontent.Choice {
	labels := append([]string{target.Zh}, target.Wrong...)
	out := make([]phrasecontent.Choice, 0, 4)
	seen := map[string]bool{}
	for _, label := range labels {
		if seen[label] || label == "" {
			continue
		}
		seen[label] = true
		id := "label:" + label
		if m, ok := byZh[label]; ok {
			id = optionID(m)
		} else if label == target.Zh {
			id = optionID(target)
		}
		out = append(out, phrasecontent.Choice{ID: id, Label: label})
		if len(out) == 4 {
			break
		}
	}
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func enOptions(target material, mats []material, rng *rand.Rand) []phrasecontent.Choice {
	others := make([]material, 0, len(mats))
	for _, m := range mats {
		if m.KpID != target.KpID && m.Title != target.Title {
			others = append(others, m)
		}
	}
	rng.Shuffle(len(others), func(i, j int) { others[i], others[j] = others[j], others[i] })
	out := []phrasecontent.Choice{{ID: optionID(target), Label: target.Title}}
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
			out = append(out, phrasecontent.Choice{ID: optionID(m), Label: m.Title})
		}
	}
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func (s *Service) freeze(ctx context.Context, item *Item, cache map[string]mediaRow) error {
	if item.Example.SpeechURL == "" {
		if phrasecontent.NeedsSpeech(item.Kind) {
			return errors.New("短句题目缺少整句读音")
		}
		return nil
	}
	source := item.Example.SpeechURL
	media, ok := cache[source]
	if !ok {
		var err error
		media, err = s.fetchMedia(ctx, source)
		if err != nil {
			return err
		}
		cache[source] = media
	}
	item.MediaSHA256[source] = media.SHA256
	item.Example.SpeechURL = phrasecontent.MediaPrefix + media.SHA256 + ".mp3"
	return nil
}

func (s *Service) fetchMedia(ctx context.Context, source string) (mediaRow, error) {
	if !(strings.HasPrefix(source, "/api/v1/phrase/items/") && strings.HasSuffix(source, "/speech.mp3")) || strings.Contains(source, "..") {
		return mediaRow{}, errors.New("媒体必须引用素材后台短句整句读音")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.contentURL+source, nil)
	if err != nil {
		return mediaRow{}, err
	}
	res, err := s.client.Do(req)
	if err != nil {
		return mediaRow{}, fmt.Errorf("读取短句读音失败: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return mediaRow{}, fmt.Errorf("短句读音暂不可用（%s，HTTP %d）", source, res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
	if err != nil {
		return mediaRow{}, err
	}
	if len(data) == 0 || len(data) > 8<<20 {
		return mediaRow{}, errors.New("短句读音无效")
	}
	if !(bytes.HasPrefix(data, []byte("ID3")) || (len(data) >= 2 && data[0] == 255 && data[1]&224 == 224)) {
		return mediaRow{}, errors.New("短句读音不是有效的MP3")
	}
	sum := sha256.Sum256(data)
	return mediaRow{SHA256: hex.EncodeToString(sum[:]), Kind: "audio", Data: data}, nil
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
	if len(hash) != 64 {
		return nil, gorm.ErrRecordNotFound
	}
	var row mediaRow
	err := s.db.First(&row, "sha256 = ?", hash).Error
	return row.Data, err
}
