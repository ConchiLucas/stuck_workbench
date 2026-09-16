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

	learningmodel "github.com/conchi/study-learning/model"
	"github.com/conchi/study-learning/phrasecontent"
	"gorm.io/gorm"
)

var (
	ErrChildNotFound = errors.New("child not found")
	ErrPlanNotFound  = errors.New("plan not found")
	ErrNoQuestions   = errors.New("no eligible phrase questions")
	ErrInvalidType   = errors.New("invalid phrase question type")
)

var phraseCodes = []string{"listen_zh", "listen_en", "scene", "reply"}

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
	ID, KpID                                                                              int64
	Code, Type, Stem, Options, Answer, Visual, Speech, ModuleCode, MasteryStatus, Payload, Phrase string
}

func (s *Service) Create(ctx context.Context, childID int64, input CreateInput) (Detail, error) {
	if input.Mode == "" {
		input.Mode = "today"
	}
	if input.Count <= 0 {
		input.Count = 4
	}
	if input.Count > 20 {
		input.Count = 20
	}
	codes, err := resolveCodes(input)
	if err != nil {
		return Detail{}, err
	}
	var n int64
	if err := s.db.WithContext(ctx).Table("children").Where("id=?", childID).Count(&n).Error; err != nil {
		return Detail{}, err
	}
	if n == 0 {
		return Detail{}, ErrChildNotFound
	}
	q := s.db.WithContext(ctx).Table("questions q").Select("q.id,q.kp_id,q.code,q.type,q.stem,q.options,q.answer,q.visual,q.speech,m.code AS module_code,COALESCE(ms.status,'') AS mastery_status,kp.payload,kp.title AS phrase").Joins("JOIN knowledge_points kp ON kp.id=q.kp_id").Joins("JOIN modules m ON m.id=kp.module_id").Joins("JOIN subjects sub ON sub.id=m.subject_id").Joins("LEFT JOIN mastery_states ms ON ms.kp_id=kp.id AND ms.child_id=?", childID).Where("sub.code=? AND q.code IN ?", "phrase", codes)
	var raw []candidate
	if err := q.Order("CASE WHEN ms.status='review_due' THEN 0 ELSE 1 END,m.order_no,kp.order_no,q.id").Scan(&raw).Error; err != nil {
		return Detail{}, err
	}
	candidates := make([]candidate, 0, input.Count)
	for _, c := range raw {
		if phrasecontent.NeedsSpeech(c.Code) && !validSpeech(c.Speech) {
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
	row := StudyPlan{ChildID: childID, PlanDate: time.Now().Format("2006-01-02"), SubjectCode: "phrase", Status: "pending", TargetCount: len(candidates), CreatedAt: time.Now()}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("study_plans").Where("child_id=? AND plan_date=?", childID, row.PlanDate).Select("COALESCE(MAX(seq_no),0)").Scan(&row.SeqNo).Error; err != nil {
			return err
		}
		row.SeqNo++
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		for i, c := range candidates {
			order := formatOrder(shuffledOrder(optionCount(c.Options), row.ID+c.ID))
			meta := phraseMeta(c.Payload)
			if strings.TrimSpace(c.Phrase) == "" {
				c.Phrase = meta.Zh
			}
			replyToKpID, err := s.lookupKpID(tx, meta.ReplyTo)
			if err != nil {
				return err
			}
			snap, err := phrasecontent.SnapshotFromLiveQuestion(phrasecontent.LiveQuestion{
				Code: c.Code, Stem: c.Stem, Options: c.Options, Answer: c.Answer, Speech: c.Speech, Visual: c.Visual,
				OptionOrder: order, Phrase: c.Phrase, MeaningZh: meta.Zh, Scene: meta.Scene, ReplyTo: meta.ReplyTo,
				TargetKpID: c.KpID, ReplyToKpID: replyToKpID,
			})
			if err != nil {
				return err
			}
			if s.contentURL != "" {
				if err := phrasecontent.FreezeMedia(ctx, tx, s.contentURL, &snap); err != nil {
					return err
				}
			}
			raw, err := phrasecontent.HistoryBytes(snap)
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

func resolveCodes(input CreateInput) ([]string, error) {
	switch input.Mode {
	case "today":
		return append([]string{}, phraseCodes...), nil
	case "type":
		if !isPhraseCode(input.QuestionCode) {
			return nil, ErrInvalidType
		}
		return []string{input.QuestionCode}, nil
	default:
		return nil, fmt.Errorf("invalid plan mode %q", input.Mode)
	}
}

func isPhraseCode(code string) bool {
	for _, c := range phraseCodes {
		if c == code {
			return true
		}
	}
	return false
}

func validSpeech(raw string) bool {
	var value struct {
		Text string `json:"text"`
	}
	return json.Unmarshal([]byte(raw), &value) == nil && strings.TrimSpace(value.Text) != ""
}

func (s *Service) Get(ctx context.Context, childID, planID int64) (Detail, error) {
	var row StudyPlan
	err := s.db.WithContext(ctx).Where("id=? AND child_id=? AND subject_code=?", planID, childID, "phrase").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Detail{}, ErrPlanNotFound
	}
	if err != nil {
		return Detail{}, err
	}
	type itemRow struct {
		ID, KpID, QuestionID                                                                                             int64
		Seq, Tries                                                                                                       int
		Bucket, Status, Picks, OptionOrder, Phrase, Payload, Code, Type, Stem, Options, Visual, Speech, QuestionSnapshot string
	}
	var rows []itemRow
	if err := s.db.WithContext(ctx).Table("plan_items pi").Select("pi.id,pi.kp_id,pi.question_id,pi.seq,pi.tries,pi.bucket,pi.status,pi.picks,pi.option_order,kp.title AS phrase,kp.payload,q.code,q.type,pi.question_snapshot,COALESCE(NULLIF(pi.question_stem,''),q.stem) AS stem,COALESCE(NULLIF(pi.question_options,''),q.options) AS options,COALESCE(NULLIF(pi.question_visual,''),q.visual) AS visual,COALESCE(NULLIF(pi.question_speech,''),q.speech) AS speech").Joins("JOIN questions q ON q.id=pi.question_id").Joins("JOIN knowledge_points kp ON kp.id=pi.kp_id").Where("pi.plan_id=?", row.ID).Order("pi.seq").Scan(&rows).Error; err != nil {
		return Detail{}, err
	}
	items := make([]Item, 0, len(rows))
	for _, v := range rows {
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
		var meta struct {
			Zh      string `json:"zh"`
			Scene   string `json:"scene"`
			ReplyTo string `json:"replyTo"`
		}
		_ = json.Unmarshal([]byte(v.Payload), &meta)
		item := Item{
			ID: v.ID, Seq: v.Seq, KpID: v.KpID, Tries: v.Tries,
			Phrase: v.Phrase, MeaningZh: meta.Zh, Scene: meta.Scene, ReplyTo: meta.ReplyTo,
			Bucket: v.Bucket, Status: v.Status, Picks: v.Picks, OptionOrder: order,
			Question: Question{ID: v.QuestionID, Code: v.Code, Type: v.Type, Stem: v.Stem, Options: options, Visual: rawJSON(v.Visual, "{}"), Speech: rawJSON(v.Speech, "{}")},
		}
		applySnapshot(&item, v.QuestionSnapshot)
		items = append(items, item)
	}
	return Detail{row, items}, nil
}

func (s *Service) Start(ctx context.Context, childID, planID int64) (Detail, error) {
	r := s.db.WithContext(ctx).Model(&StudyPlan{}).Where("id=? AND child_id=? AND subject_code=?", planID, childID, "phrase").Where("status=?", "pending").Updates(map[string]any{"status": "doing", "started_at": time.Now()})
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

func rawJSON(v, f string) json.RawMessage {
	if !json.Valid([]byte(v)) {
		v = f
	}
	return json.RawMessage(v)
}

type phrasePayload struct {
	Zh      string `json:"zh"`
	Scene   string `json:"scene"`
	ReplyTo string `json:"replyTo"`
}

func phraseMeta(payload string) phrasePayload {
	var p phrasePayload
	_ = json.Unmarshal([]byte(payload), &p)
	return p
}

func (s *Service) lookupKpID(tx *gorm.DB, title string) (int64, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return 0, nil
	}
	var id int64
	err := tx.Raw(`SELECT kp.id FROM knowledge_points kp JOIN modules m ON m.id=kp.module_id JOIN subjects sub ON sub.id=m.subject_id WHERE sub.code='phrase' AND kp.title=? ORDER BY kp.id LIMIT 1`, title).Scan(&id).Error
	return id, err
}

func applySnapshot(item *Item, raw string) {
	converted, err := phrasecontent.PlanExampleFromSnapshot(raw, "")
	if err != nil {
		var snap struct{ Code, Kind string }
		if json.Unmarshal([]byte(raw), &snap) == nil {
			if snap.Kind != "" {
				item.Question.Code = snap.Kind
			} else if snap.Code != "" {
				item.Question.Code = snap.Code
			}
		}
		return
	}
	ex := converted.Example
	item.Question.Code = ex.Kind
	if ex.Stem != "" {
		item.Question.Stem = ex.Stem
	}
	if opts, err := json.Marshal(ex.Options); err == nil {
		item.Question.Options = opts
	}
	kind := "scene"
	if ex.Kind == "reply" {
		kind = "prompt"
	}
	if visual, err := json.Marshal(map[string]string{"kind": kind, "text": ex.Prompt}); err == nil {
		item.Question.Visual = visual
	}
	speech := map[string]string{"text": ex.Speech, "lang": "en-US"}
	if ex.SpeechURL != "" {
		speech["url"] = ex.SpeechURL
	}
	if encoded, err := json.Marshal(speech); err == nil {
		item.Question.Speech = encoded
	}
}

func (s *Service) FrozenSpeech(ctx context.Context, file string) ([]byte, error) {
	return phrasecontent.MediaBytes(s.db.WithContext(ctx), file)
}
