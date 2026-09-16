package plan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/conchi/study-learning/englishcontent"
	learningmodel "github.com/conchi/study-learning/model"
	"gorm.io/gorm"
)

var (
	ErrChildNotFound = errors.New("child not found")
	ErrPlanNotFound  = errors.New("plan not found")
	ErrNoQuestions   = errors.New("no eligible english questions")
)

type Service struct {
	db         *gorm.DB
	contentURL string
}

func NewService(db *gorm.DB, contentURL ...string) *Service {
	base := ""
	if len(contentURL) > 0 {
		base = contentURL[0]
	}
	return &Service{db: db, contentURL: base}
}

type candidate struct {
	ID, KpID                                                                           int64
	Code, Type, Stem, Options, Answer, Visual, Speech, ModuleCode, MasteryStatus, Word string
}

func (s *Service) Create(ctx context.Context, childID int64, input CreateInput) (Detail, error) {
	if input.Mode == "" {
		input.Mode = "today"
	}
	if input.Count <= 0 {
		input.Count = 8
	}
	if input.Count > 20 {
		input.Count = 20
	}
	var n int64
	if err := s.db.WithContext(ctx).Table("children").Where("id=?", childID).Count(&n).Error; err != nil {
		return Detail{}, err
	}
	if n == 0 {
		return Detail{}, ErrChildNotFound
	}
	q := s.db.WithContext(ctx).Table("questions q").Select("q.id,q.kp_id,q.code,q.type,q.stem,q.options,q.answer,q.visual,q.speech,m.code AS module_code,COALESCE(ms.status,'') AS mastery_status,kp.title AS word").Joins("JOIN knowledge_points kp ON kp.id=q.kp_id").Joins("JOIN modules m ON m.id=kp.module_id").Joins("JOIN subjects sub ON sub.id=m.subject_id").Joins("LEFT JOIN mastery_states ms ON ms.kp_id=kp.id AND ms.child_id=?", childID).Joins("JOIN english_assets ea ON ea.kp_id=kp.id").Where("sub.code=? AND q.code IN ? AND COALESCE(ea.speech_audio_url,'')<>''", "english", []string{"listen", "picture"})
	switch input.Mode {
	case "module":
		if input.ModuleCode == "" {
			return Detail{}, ErrNoQuestions
		}
		q = q.Where("m.code=?", input.ModuleCode)
	case "review":
		q = q.Where("ms.status='review_due' OR (ms.status='mastered' AND ms.due_at<=CURRENT_TIMESTAMP)")
	case "today":
	default:
		return Detail{}, fmt.Errorf("invalid plan mode %q", input.Mode)
	}
	var raw []candidate
	if err := q.Order("CASE WHEN ms.status='review_due' THEN 0 ELSE 1 END,m.order_no,kp.order_no,q.id").Scan(&raw).Error; err != nil {
		return Detail{}, err
	}
	candidates := make([]candidate, 0, input.Count)
	for _, c := range raw {
		if !validSpeech(c.Speech) {
			continue
		}
		if c.Code == "picture" && !s.pictureReady(ctx, c.Options) {
			continue
		}
		candidates = append(candidates, c)
		if len(candidates) == input.Count {
			break
		}
	}
	if len(candidates) == 0 {
		return Detail{}, ErrNoQuestions
	}
	row := StudyPlan{ChildID: childID, PlanDate: time.Now().Format("2006-01-02"), SubjectCode: "english", Status: "pending", TargetCount: len(candidates), CreatedAt: time.Now()}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("study_plans").Where("child_id=? AND plan_date=?", childID, row.PlanDate).Select("COALESCE(MAX(seq_no),0)").Scan(&row.SeqNo).Error; err != nil {
			return err
		}
		row.SeqNo++
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		for i, c := range candidates {
			order := formatOrder(shuffledOrder(optionCount(c.Options), row.ID+c.ID))
			meanings, keys, err := optionMeta(tx, c.Options)
			if err != nil {
				return err
			}
			snap, err := englishcontent.SnapshotFromLiveQuestion(englishcontent.LiveQuestion{
				Code: c.Code, Stem: c.Stem, Options: c.Options, Answer: c.Answer, Speech: c.Speech,
				OptionOrder: order, Word: c.Word, TargetKpID: c.KpID, Meanings: meanings, AssetKeys: keys,
			})
			if err != nil {
				return err
			}
			if err := englishcontent.FreezeMedia(ctx, tx, s.contentURL, &snap); err != nil {
				return err
			}
			raw, err := englishcontent.HistoryBytes(snap)
			if err != nil {
				return err
			}
			bucket := "new"
			if c.MasteryStatus == "review_due" || c.MasteryStatus == "mastered" {
				bucket = "review"
			}
			item := learningmodel.PlanItem{PlanID: row.ID, Seq: i + 1, KpID: c.KpID, QuestionID: c.ID, Bucket: bucket, Status: "pending", OptionOrder: order, QuestionStem: c.Stem, QuestionOptions: c.Options, QuestionAnswer: c.Answer, QuestionVisual: c.Visual, QuestionSpeech: c.Speech, QuestionSnapshot: string(raw), ContentSnapshotVersion: 1}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Detail{}, err
	}
	return s.Get(ctx, childID, row.ID)
}
func validSpeech(raw string) bool {
	var value struct {
		Text string `json:"text"`
	}
	return json.Unmarshal([]byte(raw), &value) == nil && strings.TrimSpace(value.Text) != ""
}
func (s *Service) pictureReady(ctx context.Context, raw string) bool {
	var opts []struct {
		KpID      int64  `json:"kpId"`
		AssetKind string `json:"assetKind"`
	}
	if json.Unmarshal([]byte(raw), &opts) != nil || len(opts) != 4 {
		return false
	}
	ids := make([]int64, 0, 4)
	for _, o := range opts {
		if o.KpID == 0 || o.AssetKind != "sense" {
			return false
		}
		ids = append(ids, o.KpID)
	}
	var n int64
	if s.db.WithContext(ctx).Table("english_assets").Where("kp_id IN ? AND COALESCE(sense_image_url,'')<>''", ids).Count(&n).Error != nil {
		return false
	}
	return n == int64(len(ids))
}

func optionMeta(tx *gorm.DB, raw string) (map[int64]string, map[int64]englishcontent.AssetKey, error) {
	var opts []englishcontent.LiveOption
	if err := json.Unmarshal([]byte(raw), &opts); err != nil {
		return nil, nil, err
	}
	ids := make([]int64, 0, len(opts))
	for _, o := range opts {
		if o.KpID > 0 {
			ids = append(ids, o.KpID)
		}
	}
	meanings := map[int64]string{}
	keys := map[int64]englishcontent.AssetKey{}
	if len(ids) == 0 {
		return meanings, keys, nil
	}
	type row struct {
		ID      int64
		Payload string
		Speech  string
		Sense   string
	}
	var rows []row
	if err := tx.Raw(`SELECT kp.id, kp.payload, COALESCE(ea.speech_audio_url,'') AS speech, COALESCE(ea.sense_image_url,'') AS sense
 FROM knowledge_points kp LEFT JOIN english_assets ea ON ea.kp_id=kp.id WHERE kp.id IN ?`, ids).Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	for _, r := range rows {
		var meta struct {
			MeaningZh string `json:"meaningZh"`
		}
		_ = json.Unmarshal([]byte(r.Payload), &meta)
		meanings[r.ID] = strings.TrimSpace(meta.MeaningZh)
		keys[r.ID] = englishcontent.AssetKey{Speech: r.Speech, Sense: r.Sense}
	}
	return meanings, keys, nil
}
func (s *Service) Get(ctx context.Context, childID, planID int64) (Detail, error) {
	var row StudyPlan
	err := s.db.WithContext(ctx).Where("id=? AND child_id=? AND subject_code=?", planID, childID, "english").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Detail{}, ErrPlanNotFound
	}
	if err != nil {
		return Detail{}, err
	}
	type itemRow struct {
		ID, KpID, QuestionID                                                                                           int64
		Seq, Tries                                                                                                     int
		Bucket, Status, Picks, OptionOrder, Word, Payload, Code, Type, Stem, Options, Visual, Speech, QuestionSnapshot string
	}
	var rows []itemRow
	if err := s.db.WithContext(ctx).Table("plan_items pi").Select("pi.id,pi.kp_id,pi.question_id,pi.seq,pi.tries,pi.bucket,pi.status,pi.picks,pi.option_order,kp.title AS word,kp.payload,q.code,q.type,pi.question_snapshot,COALESCE(NULLIF(pi.question_stem,''),q.stem) AS stem,COALESCE(NULLIF(pi.question_options,''),q.options) AS options,COALESCE(NULLIF(pi.question_visual,''),q.visual) AS visual,COALESCE(NULLIF(pi.question_speech,''),q.speech) AS speech").Joins("JOIN questions q ON q.id=pi.question_id").Joins("JOIN knowledge_points kp ON kp.id=pi.kp_id").Where("pi.plan_id=?", row.ID).Order("pi.seq").Scan(&rows).Error; err != nil {
		return Detail{}, err
	}
	items := make([]Item, 0, len(rows))
	for _, v := range rows {
		var snap struct{ Code, Type string }
		if json.Unmarshal([]byte(v.QuestionSnapshot), &snap) == nil {
			if snap.Code != "" {
				v.Code = snap.Code
			}
			if snap.Type != "" {
				v.Type = snap.Type
			}
		}
		order := v.OptionOrder
		if order == "" {
			order = formatOrder(identityOrder(optionCount(v.Options)))
			if err := s.db.WithContext(ctx).Table("plan_items").Where("id=?", v.ID).Update("option_order", order).Error; err != nil {
				return Detail{}, err
			}
		}
		options, err := reorderOptions(v.Options, order)
		if err != nil {
			return Detail{}, err
		}
		if v.Code == "picture" {
			options = withAssetURLs(options)
		}
		var meta struct {
			MeaningZh string `json:"meaningZh"`
		}
		_ = json.Unmarshal([]byte(v.Payload), &meta)
		items = append(items, Item{v.ID, v.Seq, v.KpID, v.Word, meta.MeaningZh, v.Bucket, v.Status, v.Tries, v.Picks, order, Question{v.QuestionID, v.Code, v.Type, v.Stem, options, rawJSON(v.Visual, "{}"), rawJSON(v.Speech, "{}")}})
	}
	return Detail{row, items}, nil
}
func (s *Service) Start(ctx context.Context, childID, planID int64) (Detail, error) {
	r := s.db.WithContext(ctx).Model(&StudyPlan{}).Where("id=? AND child_id=? AND subject_code=?", planID, childID, "english").Where("status=?", "pending").Updates(map[string]any{"status": "doing", "started_at": time.Now()})
	if r.Error != nil {
		return Detail{}, r.Error
	}
	return s.Get(ctx, childID, planID)
}
func optionCount(raw string) int {
	var v []json.RawMessage
	if json.Unmarshal([]byte(raw), &v) != nil {
		return 0
	}
	return len(v)
}
func identityOrder(n int) []int {
	o := make([]int, n)
	for i := range o {
		o[i] = i
	}
	return o
}
func shuffledOrder(n int, seed int64) []int {
	o := identityOrder(n)
	rand.New(rand.NewSource(seed)).Shuffle(len(o), func(i, j int) { o[i], o[j] = o[j], o[i] })
	return o
}
func formatOrder(o []int) string {
	v := make([]string, len(o))
	for i, n := range o {
		v[i] = strconv.Itoa(n)
	}
	return strings.Join(v, ",")
}
func ParseOrder(s string) ([]int, error) {
	if s == "" {
		return nil, nil
	}
	p := strings.Split(s, ",")
	o := make([]int, len(p))
	for i, v := range p {
		n, e := strconv.Atoi(v)
		if e != nil {
			return nil, e
		}
		o[i] = n
	}
	return o, nil
}
func reorderOptions(raw, orderRaw string) (json.RawMessage, error) {
	var src []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &src); err != nil {
		return nil, err
	}
	order, err := ParseOrder(orderRaw)
	if err != nil {
		return nil, err
	}
	out := make([]json.RawMessage, len(order))
	for i, n := range order {
		if n < 0 || n >= len(src) {
			return nil, errors.New("option order out of range")
		}
		out[i] = src[n]
	}
	return json.Marshal(out)
}
func withAssetURLs(raw json.RawMessage) json.RawMessage {
	var opts []map[string]any
	if json.Unmarshal(raw, &opts) != nil {
		return raw
	}
	for _, o := range opts {
		if id, ok := o["kpId"].(float64); ok {
			o["assetUrl"] = fmt.Sprintf("/api/v1/english/words/%d/sense.png", int64(id))
		}
	}
	out, err := json.Marshal(opts)
	if err != nil {
		return raw
	}
	return out
}
func rawJSON(v, f string) json.RawMessage {
	if !json.Valid([]byte(v)) {
		v = f
	}
	return json.RawMessage(v)
}
