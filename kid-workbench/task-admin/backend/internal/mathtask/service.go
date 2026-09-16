package mathtask

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

	"github.com/conchi/study-learning/mathcontent"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrMaterials = errors.New("已发布算术素材暂不可用")

type Service struct {
	db         *gorm.DB
	contentURL string
	client     *http.Client
}

func New(db *gorm.DB, contentURL string) *Service {
	return &Service{db: db, contentURL: strings.TrimRight(contentURL, "/"), client: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

type taskRow struct {
	ID          int64 `gorm:"primaryKey"`
	Title       string
	Status      string `gorm:"index"`
	RangeMax    int
	Count       int
	ItemsJSON   string `gorm:"type:text"`
	CreatedAt   time.Time
	PublishedAt *time.Time
}

func (taskRow) TableName() string { return "math_question_tasks" }

type mediaRow struct {
	SHA256 string `gorm:"primaryKey;size:64"`
	Kind   string
	Data   []byte
}

func (mediaRow) TableName() string { return "math_question_task_media" }
func Migrate(db *gorm.DB) error    { return db.AutoMigrate(&taskRow{}, &mediaRow{}) }

type Item struct {
	Sequence       int                    `json:"sequence"`
	SourceID       string                 `json:"sourceId"`
	SourceRevision int                    `json:"sourceRevision"`
	Source         mathcontent.MathDetail `json:"source"`
	Detail         mathcontent.MathDetail `json:"detail"`
	MediaSHA256    map[string]string      `json:"mediaSHA256,omitempty"`
}
type Task struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Status      string     `json:"status"`
	RangeMax    int        `json:"rangeMax"`
	Count       int        `json:"count"`
	Titles      []string   `json:"titles,omitempty"`
	Groups      []string   `json:"groups,omitempty"`
	Items       []Item     `json:"items,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
}
type CreateInput struct {
	Title     string   `json:"title"`
	DetailIDs []string `json:"detailIds"`
	RangeMax  int      `json:"rangeMax"`
	Count     int      `json:"count"`
}

var sourceAudio = regexp.MustCompile(`^/api/v1/math/detail-audio/[a-f0-9]{64}\.mp3$`)
var sourceImage = regexp.MustCompile(`^/api/v1/math/detail-image/[a-f0-9]{64}\.(png|jpe?g|webp)$`)
var frozenMedia = regexp.MustCompile(`^/api/v1/math/task-media/[a-f0-9]{64}\.(mp3|png|jpe?g|webp)$`)
var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var pngMagic = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
var groupNames = map[string]string{"addition": "加法", "subtraction": "减法", "shape": "图形"}

func (s *Service) Materials(ctx context.Context) (mathcontent.Catalog, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.contentURL+"/api/v1/math/details/published", nil)
	if err != nil {
		return mathcontent.Catalog{}, ErrMaterials
	}
	res, err := s.client.Do(req)
	if err != nil {
		return mathcontent.Catalog{}, ErrMaterials
	}
	defer res.Body.Close()
	var catalog mathcontent.Catalog
	if res.StatusCode != 200 {
		return catalog, ErrMaterials
	}
	if err = json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(&catalog); err != nil || catalog.SchemaVersion != 1 {
		return mathcontent.Catalog{}, ErrMaterials
	}
	seen := map[string]bool{}
	for _, d := range catalog.Items {
		if mathcontent.Validate(d) != nil || seen[d.ID] {
			return mathcontent.Catalog{}, ErrMaterials
		}
		seen[d.ID] = true
	}
	return catalog, nil
}
func (s *Service) Create(ctx context.Context, in CreateInput) (Task, error) {
	if in.Count < 1 || in.Count > 100 {
		return Task{}, errors.New("题目数量须为1至100")
	}
	if in.RangeMax != 5 && in.RangeMax != 10 && in.RangeMax != 20 {
		return Task{}, errors.New("数值范围须为5、10或20以内")
	}
	if len(in.DetailIDs) == 0 || len(in.DetailIDs) > 12 || in.Count < len(in.DetailIDs) {
		return Task{}, errors.New("请选择题型，题数不能少于所选题型数")
	}
	catalog, err := s.Materials(ctx)
	if err != nil {
		return Task{}, err
	}
	byID := map[string]mathcontent.MathDetail{}
	for _, d := range catalog.Items {
		byID[d.ID] = d
	}
	selected := []mathcontent.MathDetail{}
	seen := map[string]bool{}
	for _, id := range in.DetailIDs {
		d, ok := byID[id]
		if !ok {
			return Task{}, fmt.Errorf("题型 %s 缺少已发布素材", id)
		}
		if seen[id] {
			return Task{}, errors.New("题型不能重复")
		}
		seen[id] = true
		selected = append(selected, d)
	}
	items := make([]Item, 0, in.Count)
	cache := map[string]mediaRow{}
	for i := 0; i < in.Count; i++ {
		source := selected[i%len(selected)]
		detail, err := generate(source, in.RangeMax, i)
		if err != nil {
			return Task{}, err
		}
		item := Item{Sequence: i + 1, SourceID: source.ID, SourceRevision: source.Revision, Source: source, Detail: detail, MediaSHA256: map[string]string{}}
		if err = s.freezeItem(ctx, &item, cache); err != nil {
			return Task{}, err
		}
		items = append(items, item)
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return Task{}, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "算术练习"
	}
	if len([]rune(title)) > 80 {
		return Task{}, errors.New("标题最多80个字")
	}
	row := taskRow{Title: title, Status: "draft", RangeMax: in.RangeMax, Count: in.Count, ItemsJSON: string(raw)}
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
func (s *Service) freezeItem(ctx context.Context, item *Item, cache map[string]mediaRow) error {
	e := &item.Detail.Example
	if e.Kind == "audio-shape" && strings.TrimSpace(e.AudioURL) == "" {
		return errors.New("听音图形缺少题目音频")
	}
	freeze := func(url *string, image bool) error {
		if url == nil || strings.TrimSpace(*url) == "" {
			return nil
		}
		if frozenMedia.MatchString(*url) {
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
		if item.MediaSHA256 == nil {
			item.MediaSHA256 = map[string]string{}
		}
		item.MediaSHA256[source] = media.SHA256
		ext := path.Ext(source)
		if ext == "" {
			ext = ".bin"
		}
		*url = "/api/v1/math/task-media/" + media.SHA256 + ext
		return nil
	}
	if err := freeze(&e.AudioURL, false); err != nil {
		return err
	}
	if e.Kind == "audio-shape" && e.AudioURL == "" {
		return errors.New("听音图形缺少有效题目音频")
	}
	if err := freeze(&e.ObjectImageURL, true); err != nil {
		return err
	}
	for key, href := range e.ShapeImageURLs {
		value := href
		if err := freeze(&value, true); err != nil {
			return err
		}
		e.ShapeImageURLs[key] = value
	}
	return nil
}
func (s *Service) fetchMedia(ctx context.Context, source string, image bool) (mediaRow, error) {
	if image {
		if !sourceImage.MatchString(source) {
			return mediaRow{}, errors.New("图片必须引用素材后台算术媒体接口")
		}
	} else if !sourceAudio.MatchString(source) {
		return mediaRow{}, errors.New("音频必须引用素材后台算术媒体接口")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.contentURL+source, nil)
	if err != nil {
		return mediaRow{}, err
	}
	res, err := s.client.Do(req)
	if err != nil {
		return mediaRow{}, fmt.Errorf("读取算术%s失败: %w", mediaKind(image), err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return mediaRow{}, fmt.Errorf("算术%s素材暂不可用（%s，HTTP %d）", mediaKind(image), source, res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
	if err != nil {
		return mediaRow{}, err
	}
	if len(data) == 0 || len(data) > 8<<20 {
		return mediaRow{}, fmt.Errorf("算术%s素材无效", mediaKind(image))
	}
	if image {
		if !validSourceImage(source, data) {
			return mediaRow{}, errors.New("算术图片素材不是有效图片")
		}
		sum := sha256.Sum256(data)
		return mediaRow{SHA256: hex.EncodeToString(sum[:]), Kind: "image", Data: data}, nil
	}
	if len(data) < 3 || !(string(data[:3]) == "ID3" || data[0] == 0xff && data[1]&0xe0 == 0xe0) {
		return mediaRow{}, errors.New("算术音频素材不是有效的MP3")
	}
	sum := sha256.Sum256(data)
	return mediaRow{SHA256: hex.EncodeToString(sum[:]), Kind: "audio", Data: data}, nil
}

func mediaKind(image bool) string {
	if image {
		return "图片"
	}
	return "音频"
}

func validSourceImage(source string, data []byte) bool {
	switch strings.ToLower(path.Ext(source)) {
	case ".png":
		return len(data) >= 8 && bytes.Equal(data[:8], pngMagic)
	case ".jpg", ".jpeg":
		return len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff
	case ".webp":
		return len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"
	default:
		return false
	}
}
func decode(row taskRow) (Task, error) {
	task := Task{ID: row.ID, Title: row.Title, Status: row.Status, RangeMax: row.RangeMax, Count: row.Count, CreatedAt: row.CreatedAt, PublishedAt: row.PublishedAt}
	err := json.Unmarshal([]byte(row.ItemsJSON), &task.Items)
	summarize(&task)
	return task, err
}
func summarize(task *Task) {
	seenTitle, seenGroup := map[string]bool{}, map[string]bool{}
	task.Titles, task.Groups = nil, nil
	for _, item := range task.Items {
		title := item.Detail.Title
		if title == "" {
			title = item.Source.Title
		}
		if title != "" && !seenTitle[title] {
			seenTitle[title] = true
			task.Titles = append(task.Titles, title)
		}
		if name := groupNames[item.Detail.GroupID]; name != "" && !seenGroup[name] {
			seenGroup[name] = true
			task.Groups = append(task.Groups, name)
		}
	}
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
func (s *Service) Publish(id int64) (Task, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var row taskRow
		if err := tx.First(&row, id).Error; err != nil {
			return err
		}
		task, err := decode(row)
		if err != nil {
			return err
		}
		if len(task.Items) == 0 || len(task.Items) != task.Count {
			return errors.New("题目快照不完整")
		}
		for i, item := range task.Items {
			if item.Sequence != i+1 || item.SourceID != item.Source.ID || item.SourceRevision != item.Source.Revision || item.Detail.ID != item.SourceID || item.Detail.Revision != item.SourceRevision {
				return errors.New("素材修订引用不完整")
			}
			if err := mathcontent.Validate(item.Source); err != nil {
				return err
			}
			if err := validateGenerated(item.Detail, task.RangeMax); err != nil {
				return err
			}
		}
		if row.Status == "published" {
			return nil
		}
		if row.Status != "draft" {
			return errors.New("仅草稿可以发布")
		}
		return tx.Model(&taskRow{}).Where("id = ? AND status = ?", id, "draft").Updates(map[string]any{"status": "published", "published_at": time.Now()}).Error
	})
	if err != nil {
		return Task{}, err
	}
	return s.Get(id)
}

func (s *Service) Media(hash string) ([]byte, error) {
	if !digestPattern.MatchString(hash) {
		return nil, gorm.ErrRecordNotFound
	}
	var row mediaRow
	err := s.db.First(&row, "sha256 = ?", hash).Error
	return row.Data, err
}
