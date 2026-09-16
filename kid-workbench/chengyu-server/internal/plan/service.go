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

	"github.com/conchi/chengyu-server/internal/quiz"
	"github.com/conchi/study-learning/chengyucontent"
	learningmodel "github.com/conchi/study-learning/model"
	"gorm.io/gorm"
)

var (
	ErrChildNotFound = errors.New("child not found")
	ErrPlanNotFound  = errors.New("plan not found")
	ErrNoQuestions   = errors.New("no eligible chengyu questions")
	ErrInvalidType   = errors.New("invalid chengyu question type")
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

type kpRow struct {
	ID, QuestionID                                          int64
	Title, Payload, ModuleCode, MasteryStatus, QuestionCode string
	Stem, Options, Answer, Visual, Speech                   string
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
	var kps []kpRow
	if err := s.db.WithContext(ctx).Table("knowledge_points kp").Select("kp.id,kp.title,kp.payload,m.code AS module_code,COALESCE(ms.status,'') AS mastery_status").Joins("JOIN modules m ON m.id=kp.module_id").Joins("JOIN subjects sub ON sub.id=m.subject_id").Joins("LEFT JOIN mastery_states ms ON ms.kp_id=kp.id AND ms.child_id=?", childID).Where("sub.code=?", "chengyu").Order("CASE WHEN ms.status='review_due' THEN 0 ELSE 1 END,m.order_no,kp.order_no,kp.id").Scan(&kps).Error; err != nil {
		return Detail{}, err
	}
	siblingsByModule := map[string][]quiz.Sibling{}
	for _, row := range kps {
		siblingsByModule[row.ModuleCode] = append(siblingsByModule[row.ModuleCode], quiz.Sibling{KpID: row.ID, Title: row.Title})
	}
	code := codes[0]
	items := make([]builtItem, 0, input.Count)
	for _, row := range kps {
		built, ok := s.buildItem(ctx, row, code, siblingsByModule[row.ModuleCode])
		if !ok {
			continue
		}
		items = append(items, built)
		if len(items) == input.Count {
			break
		}
	}
	if len(items) == 0 {
		return Detail{}, ErrNoQuestions
	}
	planRow := StudyPlan{ChildID: childID, PlanDate: time.Now().Format("2006-01-02"), SubjectCode: "chengyu", Status: "pending", TargetCount: len(items), CreatedAt: time.Now()}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("study_plans").Where("child_id=? AND plan_date=?", childID, planRow.PlanDate).Select("COALESCE(MAX(seq_no),0)").Scan(&planRow.SeqNo).Error; err != nil {
			return err
		}
		planRow.SeqNo++
		if err := tx.Create(&planRow).Error; err != nil {
			return err
		}
		for i, item := range items {
			order := formatOrder(shuffledOrder(optionCount(item.snap.Options), planRow.ID+item.kpID))
			snap, err := chengyucontent.SnapshotFromLiveQuestion(chengyucontent.LiveQuestion{
				Code: item.snap.Code, Stem: item.snap.Stem, Options: item.snap.Options, Answer: item.snap.Answer,
				Speech: item.snap.Speech, Visual: item.snap.Visual, OptionOrder: order,
				Chengyu: item.title, Pinyin: item.pinyin, Meaning: item.meaning, Example: item.example, TargetKpID: item.kpID,
			})
			if err != nil {
				return err
			}
			if s.contentURL != "" {
				if err := chengyucontent.FreezeMedia(ctx, tx, s.contentURL, &snap); err != nil {
					return err
				}
			}
			raw, err := chengyucontent.HistoryBytes(snap)
			if err != nil {
				return err
			}
			bucket := "new"
			if item.mastery == "review_due" || item.mastery == "mastered" {
				bucket = "review"
			}
			row := learningmodel.PlanItem{
				PlanID: planRow.ID, Seq: i + 1, KpID: item.kpID, QuestionID: item.questionID, Bucket: bucket, Status: "pending",
				OptionOrder: order, QuestionStem: item.snap.Stem, QuestionOptions: item.snap.Options, QuestionAnswer: item.snap.Answer,
				QuestionVisual: item.snap.Visual, QuestionSpeech: item.snap.Speech, QuestionSnapshot: string(raw), ContentSnapshotVersion: 1,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Detail{}, err
	}
	return s.Get(ctx, childID, planRow.ID)
}

type builtItem struct {
	kpID, questionID                   int64
	mastery, title, pinyin, meaning, example string
	snap                               quiz.Snapshot
}

func (s *Service) buildItem(ctx context.Context, row kpRow, code string, siblings []quiz.Sibling) (builtItem, bool) {
	payload, ok := quiz.ParsePayload(row.Payload)
	if !ok {
		return builtItem{}, false
	}
	if code == "meaning" && !s.hasChengyuSpeech(ctx, row.ID) && s.contentURL != "" {
		return builtItem{}, false
	}
	var qid int64
	if s.db.WithContext(ctx).Table("questions").Where("kp_id=? AND code=?", row.ID, code).Select("id").Scan(&qid).Error != nil {
		return builtItem{}, false
	}
	if qid == 0 {
		if s.db.WithContext(ctx).Table("questions").Where("kp_id=?", row.ID).Select("id").Limit(1).Scan(&qid).Error != nil || qid == 0 {
			return builtItem{}, false
		}
	}
	snap, ok := quiz.Build(code, row.Title, payload, siblings, row.ID, row.ID)
	if !ok {
		return builtItem{}, false
	}
	return builtItem{kpID: row.ID, questionID: qid, mastery: row.MasteryStatus, title: row.Title, pinyin: payload.Pinyin, meaning: payload.Meaning, example: payload.Example, snap: snap}, true
}

func (s *Service) hasChengyuSpeech(ctx context.Context, kpID int64) bool {
	if !s.db.Migrator().HasTable("chengyu_item_speech") {
		return false
	}
	var n int64
	if err := s.db.WithContext(ctx).Table("chengyu_item_speech").Where("kp_id=? AND kind=?", kpID, "chengyu").Count(&n).Error; err != nil {
		return false
	}
	return n > 0
}

func resolveCodes(input CreateInput) ([]string, error) {
	switch input.Mode {
	case "today":
		return append([]string{}, quiz.Codes...), nil
	case "type":
		if !quiz.IsCode(input.QuestionCode) {
			return nil, ErrInvalidType
		}
		return []string{input.QuestionCode}, nil
	default:
		return nil, fmt.Errorf("invalid plan mode %q", input.Mode)
	}
}

func (s *Service) Get(ctx context.Context, childID, planID int64) (Detail, error) {
	var row StudyPlan
	err := s.db.WithContext(ctx).Where("id=? AND child_id=? AND subject_code=?", planID, childID, "chengyu").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Detail{}, ErrPlanNotFound
	}
	if err != nil {
		return Detail{}, err
	}
	type itemRow struct {
		ID, KpID, QuestionID                                                                                             int64
		Seq, Tries                                                                                                       int
		Bucket, Status, Picks, OptionOrder, Chengyu, Payload, Code, Type, Stem, Options, Visual, Speech, QuestionSnapshot string
	}
	var rows []itemRow
	if err := s.db.WithContext(ctx).Table("plan_items pi").Select("pi.id,pi.kp_id,pi.question_id,pi.seq,pi.tries,pi.bucket,pi.status,pi.picks,pi.option_order,kp.title AS chengyu,kp.payload,q.code,q.type,pi.question_snapshot,COALESCE(NULLIF(pi.question_stem,''),q.stem) AS stem,COALESCE(NULLIF(pi.question_options,''),q.options) AS options,COALESCE(NULLIF(pi.question_visual,''),q.visual) AS visual,COALESCE(NULLIF(pi.question_speech,''),q.speech) AS speech").Joins("JOIN questions q ON q.id=pi.question_id").Joins("JOIN knowledge_points kp ON kp.id=pi.kp_id").Where("pi.plan_id=?", row.ID).Order("pi.seq").Scan(&rows).Error; err != nil {
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
		payload, _ := quiz.ParsePayload(v.Payload)
		item := Item{
			ID: v.ID, Seq: v.Seq, KpID: v.KpID, Tries: v.Tries,
			Chengyu: v.Chengyu, Pinyin: payload.Pinyin, Meaning: payload.Meaning, Example: payload.Example,
			Bucket: v.Bucket, Status: v.Status, Picks: v.Picks, OptionOrder: order,
			Question: Question{ID: v.QuestionID, Code: v.Code, Type: v.Type, Stem: v.Stem, Options: options, Visual: rawJSON(v.Visual, "{}"), Speech: rawJSON(v.Speech, "{}")},
		}
		applySnapshot(&item, v.QuestionSnapshot)
		items = append(items, item)
	}
	return Detail{row, items}, nil
}

func (s *Service) Start(ctx context.Context, childID, planID int64) (Detail, error) {
	r := s.db.WithContext(ctx).Model(&StudyPlan{}).Where("id=? AND child_id=? AND subject_code=?", planID, childID, "chengyu").Where("status=?", "pending").Updates(map[string]any{"status": "doing", "started_at": time.Now()})
	if r.Error != nil {
		return Detail{}, r.Error
	}
	return s.Get(ctx, childID, planID)
}

func (s *Service) FrozenSpeech(ctx context.Context, file string) ([]byte, error) {
	return chengyucontent.MediaBytes(s.db.WithContext(ctx), file)
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

func applySnapshot(item *Item, raw string) {
	converted, err := chengyucontent.PlanExampleFromSnapshot(raw, "")
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
	item.Question.Stem = ex.Stem
	if opts, err := json.Marshal(ex.Options); err == nil {
		item.Question.Options = opts
	}
	prompt := ex.Prompt
	kind := ex.Kind
	if ex.Kind == "example" && ex.Blank != nil {
		prompt = ex.Blank.Blanked
		kind = "example"
	}
	visual := map[string]any{"kind": kind, "text": prompt}
	if ex.Blank != nil {
		visual["full"] = ex.Blank.Full
		visual["blanked"] = ex.Blank.Blanked
		visual["target"] = ex.Blank.Target
		visual["start"] = ex.Blank.Start
		visual["length"] = ex.Blank.Length
	}
	if encoded, err := json.Marshal(visual); err == nil {
		item.Question.Visual = encoded
	}
	speech := map[string]string{"text": ex.Speech, "lang": "zh-CN"}
	if ex.SpeechURL != "" {
		speech["url"] = ex.SpeechURL
	}
	if encoded, err := json.Marshal(speech); err == nil {
		item.Question.Speech = encoded
	}
}
