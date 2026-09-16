// Package pinyintask reads content assets and owns only frozen question packages.
package pinyintask

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io"
	"math/rand/v2"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	db         *gorm.DB
	contentURL string
	client     *http.Client
}

func New(db *gorm.DB, url string) *Service {
	return &Service{db: db, contentURL: strings.TrimRight(url, "/"), client: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

type taskRow struct {
	ID        int64 `gorm:"primaryKey"`
	Title     string
	Count     int
	TypesJSON string
	ItemsJSON string `gorm:"type:text"`
	CreatedAt time.Time
}

func (taskRow) TableName() string { return "pinyin_question_tasks" }

type mediaRow struct {
	SHA256 string `gorm:"primaryKey;size:64"`
	Data   []byte
}

func (mediaRow) TableName() string { return "pinyin_question_task_media" }
func Migrate(db *gorm.DB) error    { return db.AutoMigrate(&taskRow{}, &mediaRow{}) }

type letter struct {
	KpID       int64 `gorm:"primaryKey;column:kp_id"`
	Letter     string
	ModuleCode string
	ModuleName string
	SoloText   string
	WordText   string
}

func (letter) TableName() string { return "pinyin_assets" }

type syllable struct {
	ID           int64
	InitialText  string
	FinalText    string
	SyllableText string
	SpeechURL    string
	Enabled      bool
}

func (syllable) TableName() string { return "pinyin_syllable_assets" }

type Option struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	SpeechURL string `json:"speechUrl,omitempty"`
}
type Visual struct {
	Kind    string `json:"kind"`
	Text    string `json:"text,omitempty"`
	Initial string `json:"initial,omitempty"`
	Final   string `json:"final,omitempty"`
}
type Item struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"`
	Stem           string            `json:"stem"`
	TargetID       int64             `json:"targetId"`
	SourceTable    string            `json:"sourceTable"`
	ModuleCode     string            `json:"moduleCode"`
	ModuleName     string            `json:"moduleName"`
	SpeechURL      string            `json:"speechUrl,omitempty"`
	Visual         Visual            `json:"visual"`
	Options        []Option          `json:"options"`
	AnswerOptionID string            `json:"answerOptionId"`
	MediaSHA256    map[string]string `json:"mediaSHA256"`
}
type Task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Count     int       `json:"count"`
	Types     []string  `json:"types"`
	Groups    []string  `json:"groups"`
	Items     []Item    `json:"items,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}
type CreateInput struct {
	Title string   `json:"title"`
	Types []string `json:"types"`
	Count int      `json:"count"`
}

var typeNames = map[string]string{"listen": "听音选字母", "inword": "字中找拼音", "shape": "看形认读", "blend": "声韵拼读"}
var assetPath = regexp.MustCompile(`^/api/v1/pinyin/(items/[1-9][0-9]*/speech/(solo|word)\.mp3|syllables/[1-9][0-9]*/speech\.mp3(\?v=[a-f0-9]{16})?)$`)
var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func (s *Service) Create(ctx context.Context, in CreateInput) (Task, error) {
	if in.Count < 1 || in.Count > 40 || len(in.Types) == 0 || len(in.Types) > 4 || in.Count < len(in.Types) {
		return Task{}, errors.New("请选择题型，题数须为1至40且不少于所选题型数")
	}
	seen := map[string]bool{}
	for _, typ := range in.Types {
		if typeNames[typ] == "" || seen[typ] {
			return Task{}, errors.New("拼音题型无效或重复")
		}
		seen[typ] = true
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "拼音练习"
	}
	if len([]rune(title)) > 80 {
		return Task{}, errors.New("任务名称最多80字")
	}
	var letters []letter
	var syllables []syllable
	if seen["listen"] || seen["inword"] || seen["shape"] {
		if err := s.db.WithContext(ctx).Order("kp_id").Find(&letters).Error; err != nil {
			return Task{}, err
		}
	}
	if seen["blend"] {
		if err := s.db.WithContext(ctx).Where("enabled = ? AND speech_url <> ''", true).Order("id").Find(&syllables).Error; err != nil {
			return Task{}, err
		}
	}
	items := make([]Item, 0, in.Count)
	used := map[string]map[int64]bool{}
	cache := map[string]mediaRow{}
	for i := 0; i < in.Count; i++ {
		typ := in.Types[i%len(in.Types)]
		if used[typ] == nil {
			used[typ] = map[int64]bool{}
		}
		q, err := generate(typ, letters, syllables, used[typ])
		if err != nil {
			return Task{}, err
		}
		used[typ][q.TargetID] = true
		q.ID = fmt.Sprintf("pinyin-%d", i+1)
		q.MediaSHA256 = map[string]string{}
		freeze := func(url *string) error {
			if *url == "" {
				return nil
			}
			source := *url
			media, ok := cache[source]
			if !ok {
				var err error
				media, err = s.fetchAudio(ctx, source)
				if err != nil {
					return err
				}
				cache[source] = media
			}
			q.MediaSHA256[source] = media.SHA256
			*url = "/api/v1/pinyin/task-media/" + media.SHA256 + ".mp3"
			return nil
		}
		if err := freeze(&q.SpeechURL); err != nil {
			return Task{}, err
		}
		for j := range q.Options {
			if err := freeze(&q.Options[j].SpeechURL); err != nil {
				return Task{}, err
			}
		}
		items = append(items, q)
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
func generate(typ string, letters []letter, syllables []syllable, used map[int64]bool) (Item, error) {
	q := Item{Type: typ}
	eligible := []letter{}
	if typ == "blend" {
		if len(syllables) < 4 {
			return q, errors.New("声韵拼读至少需要4份已启用的音节音频素材")
		}
		pool := append([]syllable(nil), syllables...)
		rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
		target := pool[0]
		for _, v := range pool {
			if !used[v.ID] {
				target = v
				break
			}
		}
		q.TargetID = target.ID
		q.SourceTable = "pinyin_syllable_assets"
		q.ModuleCode = "blend"
		q.ModuleName = "声韵拼读"
		q.Stem = "把声母和韵母拼在一起"
		q.Visual = Visual{Kind: "blend", Initial: target.InitialText, Final: target.FinalText}
		q.AnswerOptionID = strconv.FormatInt(target.ID, 10)
		q.Options = append(q.Options, Option{ID: q.AnswerOptionID, Label: target.SyllableText, SpeechURL: target.SpeechURL})
		labels := map[string]bool{target.SyllableText: true}
		for _, v := range pool {
			if v.ID != target.ID && !labels[v.SyllableText] {
				q.Options = append(q.Options, Option{ID: strconv.FormatInt(v.ID, 10), Label: v.SyllableText, SpeechURL: v.SpeechURL})
				labels[v.SyllableText] = true
				if len(q.Options) == 4 {
					break
				}
			}
		}
	} else {
		groups := map[string][]letter{}
		for _, v := range letters {
			if strings.TrimSpace(v.Letter) == "" || strings.TrimSpace(v.SoloText) == "" || (typ == "inword" && strings.TrimSpace(v.WordText) == "") {
				continue
			}
			groups[v.ModuleCode] = append(groups[v.ModuleCode], v)
		}
		for _, v := range letters {
			if len(groups[v.ModuleCode]) >= 4 {
				for _, a := range groups[v.ModuleCode] {
					if a.KpID == v.KpID {
						eligible = append(eligible, a)
						break
					}
				}
			}
		}
		if len(eligible) < 4 {
			return q, fmt.Errorf("%s需要同分组至少4份完整拼音素材", typeNames[typ])
		}
		rand.Shuffle(len(eligible), func(i, j int) { eligible[i], eligible[j] = eligible[j], eligible[i] })
		target := eligible[0]
		for _, v := range eligible {
			if !used[v.KpID] {
				target = v
				break
			}
		}
		q.TargetID = target.KpID
		q.SourceTable = "pinyin_assets"
		q.ModuleCode = target.ModuleCode
		q.ModuleName = target.ModuleName
		if q.ModuleName == "" {
			q.ModuleName = target.ModuleCode
		}
		q.AnswerOptionID = strconv.FormatInt(target.KpID, 10)
		opts := append([]letter{target}, eligible...)
		labels := map[string]bool{}
		for _, v := range opts {
			if v.ModuleCode != target.ModuleCode || labels[v.Letter] {
				continue
			}
			labels[v.Letter] = true
			o := Option{ID: strconv.FormatInt(v.KpID, 10), Label: v.Letter}
			if typ == "shape" {
				o.SpeechURL = fmt.Sprintf("/api/v1/pinyin/items/%d/speech/solo.mp3", v.KpID)
			}
			q.Options = append(q.Options, o)
			if len(q.Options) == 4 {
				break
			}
		}
		switch typ {
		case "listen":
			q.Stem = "听一听，选出你听到的拼音"
			q.Visual = Visual{Kind: "sound"}
			q.SpeechURL = fmt.Sprintf("/api/v1/pinyin/items/%d/speech/solo.mp3", target.KpID)
		case "inword":
			q.Stem = "听一听，这个字里藏着哪个拼音？"
			q.Visual = Visual{Kind: "char", Text: target.WordText}
			q.SpeechURL = fmt.Sprintf("/api/v1/pinyin/items/%d/speech/word.mp3", target.KpID)
		case "shape":
			q.Stem = "看一看，选出四线格里拼音的读音"
			q.Visual = Visual{Kind: "glyph", Text: target.Letter}
		}
	}
	if len(q.Options) != 4 {
		return q, errors.New("素材不足以生成4个不同选项")
	}
	rand.Shuffle(len(q.Options), func(i, j int) { q.Options[i], q.Options[j] = q.Options[j], q.Options[i] })
	return q, nil
}
func (s *Service) fetchAudio(ctx context.Context, path string) (mediaRow, error) {
	if !assetPath.MatchString(path) {
		return mediaRow{}, errors.New("音频必须引用素材后台拼音媒体接口")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.contentURL+path, nil)
	if err != nil {
		return mediaRow{}, err
	}
	res, err := s.client.Do(req)
	if err != nil {
		return mediaRow{}, fmt.Errorf("读取拼音音频失败: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return mediaRow{}, fmt.Errorf("拼音音频素材暂不可用（%s，HTTP %d）", path, res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
	if err != nil {
		return mediaRow{}, err
	}
	if len(data) < 3 || len(data) > 8<<20 || !(string(data[:3]) == "ID3" || data[0] == 0xff && data[1]&0xe0 == 0xe0) {
		return mediaRow{}, errors.New("拼音音频素材不是有效的MP3")
	}
	sum := sha256.Sum256(data)
	return mediaRow{SHA256: hex.EncodeToString(sum[:]), Data: data}, nil
}
func decode(row taskRow) (Task, error) {
	t := Task{ID: row.ID, Title: row.Title, Count: row.Count, CreatedAt: row.CreatedAt}
	if err := json.Unmarshal([]byte(row.TypesJSON), &t.Types); err != nil {
		return t, err
	}
	err := json.Unmarshal([]byte(row.ItemsJSON), &t.Items)
	seen := map[string]bool{}
	for _, q := range t.Items {
		if !seen[q.ModuleName] {
			t.Groups = append(t.Groups, q.ModuleName)
			seen[q.ModuleName] = true
		}
	}
	return t, err
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
	if err := s.db.Select("id", "title", "count", "types_json", "items_json", "created_at").Order("id DESC").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Task, 0, len(rows))
	for _, r := range rows {
		v, e := decode(r)
		if e != nil {
			return nil, e
		}
		v.Items = nil
		out = append(out, v)
	}
	return out, nil
}
func (s *Service) Audio(hash string) ([]byte, error) {
	if !digestPattern.MatchString(hash) {
		return nil, gorm.ErrRecordNotFound
	}
	var row mediaRow
	err := s.db.First(&row, "sha256 = ?", hash).Error
	return row.Data, err
}
