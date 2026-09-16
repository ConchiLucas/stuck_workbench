package englishtask

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/conchi/study-learning/englishcontent"
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

func (taskRow) TableName() string { return "english_question_tasks" }

type mediaRow struct {
	SHA256 string `gorm:"primaryKey;size:64"`
	Kind   string
	Data   []byte
}

func (mediaRow) TableName() string { return "english_question_task_media" }
func Migrate(db *gorm.DB) error  { return db.AutoMigrate(&taskRow{}, &mediaRow{}) }

type Item struct {
	ID                string                       `json:"id"`
	Kind              string                       `json:"kind"`
	SkillCode         string                       `json:"skillCode"`
	TargetID          int64                       `json:"targetId"`
	SourceID          int64                       `json:"sourceId"`
	SourceTable       string                       `json:"sourceTable"`
	ModuleCode        string                       `json:"moduleCode"`
	ModuleName        string                       `json:"moduleName"`
	SourceRevision    int                         `json:"sourceRevision,omitempty"`
	SourceContentHash string                       `json:"sourceContentHash"`
	Example           englishcontent.EnglishExample `json:"example"`
	MediaSHA256       map[string]string             `json:"mediaSHA256"`
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

type word struct {
	KpID        int64
	WordText    string
	MeaningZh   string
	ModuleCode  string
	ModuleName  string
	ContentHash string
}

var typeNames = map[string]string{
	"audio-choice": "听音选词",
	"image-text":   "看图选词",
	"card-builder": "组句子",
	"input-gap":    "写单词",
	"reading-qa":   "读一读",
}
var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var pngMagic = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

func (s *Service) Create(ctx context.Context, in CreateInput) (Task, error) {
	if in.Count < 1 || in.Count > 40 || len(in.Types) == 0 || len(in.Types) > 5 || in.Count < len(in.Types) {
		return Task{}, errors.New("请选择题型，题数须为1至40且不少于所选题型数")
	}
	seen := map[string]bool{}
	for _, typ := range in.Types {
		if typeNames[typ] == "" || seen[typ] {
			return Task{}, errors.New("英语题型无效或重复")
		}
		seen[typ] = true
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "英语练习"
	}
	if len([]rune(title)) > 80 {
		return Task{}, errors.New("任务名称最多80字")
	}
	mats, err := s.loadMaterials(ctx)
	if err != nil {
		return Task{}, err
	}
	needWords := false
	needSentences, needPassages := 0, 0
	for _, typ := range in.Types {
		switch typ {
		case "card-builder":
			needSentences++
		case "reading-qa":
			needPassages++
		default:
			needWords = true
		}
	}
	repeats := (in.Count + len(in.Types) - 1) / len(in.Types)
	if needWords && len(mats.words) < 4 {
		return Task{}, errors.New("英语素材不足：至少需要4个同时具备义图和读音的单词")
	}
	if needSentences > 0 && len(mats.sentences) < repeats {
		return Task{}, errors.New("英语组句子素材不足：需要已准备完整句子和整句读音")
	}
	if needPassages > 0 && len(mats.passages) < repeats {
		return Task{}, errors.New("英语读一读素材不足：需要已准备短文、问题和选项")
	}
	items := make([]Item, 0, in.Count)
	used := map[string]map[int64]bool{}
	cache := map[string]mediaRow{}
	for i := 0; i < in.Count; i++ {
		kind := in.Types[i%len(in.Types)]
		if used[kind] == nil {
			used[kind] = map[int64]bool{}
		}
		item, err := generate(kind, mats, used[kind])
		if err != nil {
			return Task{}, err
		}
		used[kind][item.SourceID] = true
		item.ID = fmt.Sprintf("english-%d", i+1)
		item.MediaSHA256 = map[string]string{}
		if err := s.freeze(ctx, &item, cache); err != nil {
			return Task{}, err
		}
		if err := englishcontent.Validate(item.Example); err != nil {
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

func (s *Service) loadWords(ctx context.Context) ([]word, error) {
	type row struct {
		KpID       int64  `gorm:"column:kp_id"`
		WordText   string `gorm:"column:word_text"`
		Payload    string `gorm:"column:payload"`
		ModuleCode string `gorm:"column:module_code"`
		ModuleName string `gorm:"column:module_name"`
		SenseURL   string `gorm:"column:sense_image_url"`
		SpeechURL  string `gorm:"column:speech_audio_url"`
	}
	var rows []row
	err := s.db.WithContext(ctx).Raw(`
		SELECT kp.id AS kp_id, COALESCE(NULLIF(ea.word_text,''), kp.title) AS word_text, COALESCE(kp.payload,'') AS payload,
		       m.code AS module_code, m.name AS module_name, COALESCE(ea.sense_image_url,'') AS sense_image_url,
		       COALESCE(ea.speech_audio_url,'') AS speech_audio_url
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		JOIN english_assets ea ON ea.kp_id = kp.id
		WHERE s.code = 'english'
		  AND COALESCE(ea.sense_image_url,'') <> ''
		  AND COALESCE(ea.speech_audio_url,'') <> ''
		ORDER BY kp.id`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]word, 0, len(rows))
	for _, r := range rows {
		text := strings.TrimSpace(r.WordText)
		if text == "" || !isEnglishWord(text) {
			continue
		}
		meaning := meaningZh(r.Payload)
		if meaning == "" {
			meaning = text
		}
		out = append(out, word{
			KpID: r.KpID, WordText: text, MeaningZh: meaning, ModuleCode: r.ModuleCode, ModuleName: r.ModuleName,
			ContentHash: wordContentHash(r.KpID, text, meaning, r.SenseURL, r.SpeechURL),
		})
	}
	return out, nil
}

func isEnglishWord(text string) bool {
	for _, r := range text {
		if unicode.IsLetter(r) && r < unicode.MaxASCII {
			return true
		}
	}
	return false
}

func meaningZh(payload string) string {
	var p struct {
		MeaningZh string `json:"meaningZh"`
	}
	_ = json.Unmarshal([]byte(payload), &p)
	return strings.TrimSpace(p.MeaningZh)
}

func (s *Service) freeze(ctx context.Context, item *Item, cache map[string]mediaRow) error {
	freeze := func(url *string, image bool) error {
		if url == nil || *url == "" {
			return nil
		}
		source := *url
		media, ok := cache[source]
		if !ok {
			var err error
			media, err = s.fetchMedia(ctx, source, image)
			if err != nil {
				return err
			}
			cache[source] = media
		}
		item.MediaSHA256[source] = media.SHA256
		*url = "/api/v1/english/task-media/" + media.SHA256 + mediaExt(media.Kind, source)
		return nil
	}
	if err := freeze(&item.Example.SpeechURL, false); err != nil {
		return err
	}
	if err := freeze(&item.Example.Cue, true); err != nil {
		return err
	}
	for i := range item.Example.Options {
		if err := freeze(&item.Example.Options[i].Picture, true); err != nil {
			return err
		}
	}
	if (item.Kind == "audio-choice" || item.Kind == "card-builder" || item.Kind == "input-gap") && item.Example.SpeechURL == "" {
		return errors.New("英语题目缺少有效读音")
	}
	if (item.Kind == "image-text" || item.Kind == "reading-qa") && len(item.Example.Options) > 0 {
		for _, o := range item.Example.Options {
			if o.Picture == "" {
				return errors.New("选项缺少有效义图")
			}
		}
	}
	return nil
}

func mediaExt(kind, source string) string {
	switch kind {
	case "jpeg":
		return ".jpg"
	case "webp":
		return ".webp"
	case "png":
		return ".png"
	case "audio":
		return ".mp3"
	}
	ext := strings.ToLower(path.Ext(source))
	if ext == "" {
		return ".bin"
	}
	return ext
}

func (s *Service) fetchMedia(ctx context.Context, source string, image bool) (mediaRow, error) {
	allowed := strings.HasPrefix(source, "/api/v1/english/words/") || strings.HasPrefix(source, "/api/v1/english/sentences/")
	if !allowed || strings.Contains(source, "..") {
		return mediaRow{}, errors.New("媒体必须引用素材后台英语单词或句子接口")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.contentURL+source, nil)
	if err != nil {
		return mediaRow{}, err
	}
	res, err := s.client.Do(req)
	if err != nil {
		return mediaRow{}, fmt.Errorf("读取英语%s失败: %w", mediaKind(image), err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return mediaRow{}, fmt.Errorf("英语%s素材暂不可用（%s，HTTP %d）", mediaKind(image), source, res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
	if err != nil {
		return mediaRow{}, err
	}
	if len(data) == 0 || len(data) > 8<<20 {
		return mediaRow{}, fmt.Errorf("英语%s素材无效", mediaKind(image))
	}
	kind, err := detectMedia(data, image)
	if err != nil {
		return mediaRow{}, err
	}
	sum := sha256.Sum256(data)
	return mediaRow{SHA256: hex.EncodeToString(sum[:]), Kind: kind, Data: data}, nil
}

func detectMedia(data []byte, image bool) (string, error) {
	if image {
		if len(data) >= 8 && bytes.Equal(data[:8], pngMagic) {
			return "png", nil
		}
		if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
			return "jpeg", nil
		}
		if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
			return "webp", nil
		}
		return "", errors.New("英语义图不是有效图片")
	}
	if len(data) >= 3 && (string(data[:3]) == "ID3" || data[0] == 0xff && data[1]&0xe0 == 0xe0) {
		return "audio", nil
	}
	return "", errors.New("英语读音不是有效的MP3")
}

func mediaKind(image bool) string {
	if image {
		return "义图"
	}
	return "读音"
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
	if !digestPattern.MatchString(hash) {
		return nil, gorm.ErrRecordNotFound
	}
	var row mediaRow
	err := s.db.First(&row, "sha256 = ?", hash).Error
	return row.Data, err
}
