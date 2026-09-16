package studyplan

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	learningmodel "github.com/conchi/study-learning/model"
	"github.com/conchi/study-learning/poemcontent"
	"gorm.io/gorm"
)

var (
	ErrChildNotFound = errors.New("child not found")
	ErrPlanNotFound  = errors.New("plan not found")
	ErrNoQuestions   = errors.New("no poem questions")
	ErrInvalidMode   = errors.New("invalid plan mode")
)

type Service struct {
	db         *gorm.DB
	contentURL string
}

func NewService(db *gorm.DB, contentURL ...string) *Service {
	base := ""
	if len(contentURL) > 0 {
		base = strings.TrimRight(contentURL[0], "/")
	}
	return &Service{db: db, contentURL: base}
}

func (s *Service) Create(ctx context.Context, childID int64, input CreateInput) (Detail, error) {
	count := input.Count
	if count <= 0 {
		count = 8
	}
	if count > 20 {
		count = 20
	}
	var childCount int64
	if err := s.db.WithContext(ctx).Table("children").Where("id = ?", childID).Count(&childCount).Error; err != nil {
		return Detail{}, err
	}
	if childCount == 0 {
		return Detail{}, ErrChildNotFound
	}
	mode := strings.TrimSpace(input.Mode)
	if mode == "" {
		mode = "daily"
	}
	if mode != "daily" && mode != "review" {
		return Detail{}, ErrInvalidMode
	}

	row := StudyPlan{ChildID: childID, PlanDate: time.Now().Format("2006-01-02"), SubjectCode: "poem", Status: "pending", CreatedAt: time.Now()}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var candidates []questionCandidate
		codes := planQuestionCodes(input.Types)
		query := tx.Table("questions q").
			Select("q.id, q.kp_id, q.code, q.stem, q.options, q.answer, q.visual, COALESCE(q.speech,'') AS speech, COALESCE(kp.payload,'') AS payload, kp.code AS work_code, kp.title AS title").
			Joins("JOIN knowledge_points kp ON kp.id = q.kp_id").
			Joins("JOIN modules m ON m.id = kp.module_id").
			Joins("JOIN subjects s ON s.id = m.subject_id").
			Where("s.code = ? AND q.code IN ?", "poem", codes)
		if mode == "review" {
			query = query.Where(`EXISTS (SELECT 1 FROM attempts a WHERE a.child_id = ? AND a.kp_id = kp.id AND a.is_correct = ?) OR EXISTS (SELECT 1 FROM mastery_states ms WHERE ms.child_id = ? AND ms.kp_id = kp.id AND (ms.status = 'review_due' OR (ms.status = 'mastered' AND ms.due_at <= ?)))`, childID, false, childID, time.Now())
		}
		if err := query.Order("m.order_no, kp.order_no, q.id").Scan(&candidates).Error; err != nil {
			return err
		}
		if len(candidates) == 0 {
			return ErrNoQuestions
		}
		if len(input.Types) > 0 {
			candidates = pickByKinds(candidates, input.Types, count)
			if len(candidates) == 0 {
				return ErrNoQuestions
			}
		} else if len(candidates) > count {
			candidates = candidates[:count]
		}
		row.TargetCount = len(candidates)
		if err := tx.Table("study_plans").Where("child_id = ? AND plan_date = ?", childID, row.PlanDate).
			Select("COALESCE(MAX(seq_no), 0)").Scan(&row.SeqNo).Error; err != nil {
			return err
		}
		row.SeqNo++
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		for index, candidate := range candidates {
			work := poemcontent.ParseWork(candidate.KpID, candidate.WorkCode, candidate.Title, candidate.Payload)
			snap, err := poemcontent.SnapshotFromLiveQuestion(poemcontent.LiveQuestion{
				Code: candidate.Code, Stem: candidate.Stem, Options: candidate.Options, Answer: candidate.Answer,
				Visual: candidate.Visual, Speech: candidate.Speech, Payload: candidate.Payload,
				TargetKpID: candidate.KpID, Work: work,
			})
			if err != nil {
				return err
			}
			snap.Example = poemcontent.ShuffleExample(snap.Example, row.ID+candidate.ID)
			if s.contentURL != "" {
				if err := poemcontent.FreezeMedia(ctx, tx, s.contentURL, &snap); err != nil {
					return err
				}
			} else if poemcontent.NeedsSpeech(snap.Example.Kind) {
				return errors.New("缺少古诗读音冻结地址")
			}
			raw, err := poemcontent.HistoryBytes(snap)
			if err != nil {
				return err
			}
			item := learningmodel.PlanItem{
				PlanID: row.ID, Seq: index + 1, KpID: candidate.KpID, QuestionID: candidate.ID,
				Bucket: "new", Status: "pending", OptionOrder: poemcontent.OrderFromExample(snap.Example),
				ContentSnapshotVersion: 1, QuestionSnapshot: string(raw),
			}
			item.QuestionStem, item.QuestionOptions, item.QuestionAnswer = candidate.Stem, candidate.Options, candidate.Answer
			item.QuestionVisual = candidate.Visual
			item.Explanation = snap.Example.Explanation
			if snap.Example.SpeechURL != "" {
				speech, _ := json.Marshal(map[string]string{"url": snap.Example.SpeechURL, "text": snap.Example.SpeechText})
				item.QuestionSpeech = string(speech)
			}
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

func (s *Service) Get(ctx context.Context, childID, planID int64) (Detail, error) {
	var row StudyPlan
	err := s.db.WithContext(ctx).Where("id = ? AND child_id = ? AND subject_code = ?", planID, childID, "poem").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Detail{}, ErrPlanNotFound
	}
	if err != nil {
		return Detail{}, err
	}

	type itemRow struct {
		ID, KpID, QuestionID                                       int64
		Seq, Tries                                                int
		Bucket, Status, Picks, OptionOrder                        string
		Title, Code, Type, Stem, Options, Visual, Speech, Snapshot string
	}
	var rows []itemRow
	if err := s.db.WithContext(ctx).Table("plan_items pi").
		Select(`pi.id, pi.kp_id, pi.question_id, pi.seq, pi.tries, pi.bucket, pi.status, pi.picks, pi.option_order, pi.question_snapshot AS snapshot,
			kp.title, q.code, q.type,
			CASE WHEN pi.content_snapshot_version > 0 THEN pi.question_stem ELSE q.stem END AS stem,
			CASE WHEN pi.content_snapshot_version > 0 THEN pi.question_options ELSE q.options END AS options,
			CASE WHEN pi.content_snapshot_version > 0 THEN pi.question_visual ELSE q.visual END AS visual,
			CASE WHEN pi.content_snapshot_version > 0 THEN pi.question_speech ELSE q.speech END AS speech`).
		Joins("JOIN questions q ON q.id = pi.question_id").
		Joins("JOIN knowledge_points kp ON kp.id = pi.kp_id").
		Where("pi.plan_id = ?", row.ID).Order("pi.seq").Scan(&rows).Error; err != nil {
		return Detail{}, err
	}
	items := make([]Item, 0, len(rows))
	for _, value := range rows {
		item := Item{ID: value.ID, Seq: value.Seq, KpID: value.KpID, Title: value.Title,
			Bucket: value.Bucket, Status: value.Status, Tries: value.Tries, Picks: value.Picks, OptionOrder: value.OptionOrder,
			Question: Question{ID: value.QuestionID, Code: value.Code, Type: value.Type, Stem: value.Stem,
				Options: rawJSON(value.Options, "[]"), Visual: rawJSON(value.Visual, "{}"), Speech: rawJSON(value.Speech, "{}")}}
		applyPoemSnapshot(&item, value.Snapshot)
		items = append(items, item)
	}
	return Detail{Plan: row, Items: items}, nil
}

func (s *Service) Start(ctx context.Context, childID, planID int64) (Detail, error) {
	result := s.db.WithContext(ctx).Model(&StudyPlan{}).
		Where("id = ? AND child_id = ? AND subject_code = ?", planID, childID, "poem").
		Where("status = ?", "pending").Updates(map[string]any{"status": "active", "started_at": time.Now()})
	if result.Error != nil {
		return Detail{}, result.Error
	}
	return s.Get(ctx, childID, planID)
}

func (s *Service) FrozenMedia(ctx context.Context, file string) ([]byte, string, error) {
	return poemcontent.MediaBytes(s.db.WithContext(ctx), file)
}

type questionCandidate struct {
	ID, KpID                                              int64
	Code, Stem, Options, Answer, Visual, Speech, Payload, WorkCode, Title string
}

func pickByKinds(candidates []questionCandidate, types []string, count int) []questionCandidate {
	pools := map[string][]questionCandidate{}
	for _, row := range candidates {
		kind := poemcontent.KindForSkill(row.Code)
		if kind == "" {
			continue
		}
		pools[kind] = append(pools[kind], row)
	}
	wanted := make([]string, 0, len(types))
	seen := map[string]bool{}
	for _, raw := range types {
		kind := poemcontent.KindForSkill(raw)
		if kind == "" || seen[kind] {
			continue
		}
		seen[kind] = true
		wanted = append(wanted, kind)
	}
	if len(wanted) == 0 {
		if len(candidates) > count {
			return candidates[:count]
		}
		return candidates
	}
	out := make([]questionCandidate, 0, count)
	used := map[int64]bool{}
	for len(out) < count {
		added := false
		for _, kind := range wanted {
			if len(out) >= count {
				break
			}
			for _, row := range pools[kind] {
				if used[row.ID] {
					continue
				}
				used[row.ID] = true
				out = append(out, row)
				added = true
				break
			}
		}
		if !added {
			break
		}
	}
	return out
}

func planQuestionCodes(types []string) []string {
	if len(types) == 0 {
		return []string{poemcontent.KindTitle, poemcontent.KindFill, poemcontent.KindCouplet, poemcontent.KindRecite}
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(types))
	for _, raw := range types {
		kind := poemcontent.KindForSkill(raw)
		if kind == "" || seen[kind] {
			continue
		}
		seen[kind] = true
		out = append(out, kind)
	}
	if len(out) == 0 {
		return []string{poemcontent.KindTitle, poemcontent.KindFill, poemcontent.KindCouplet, poemcontent.KindRecite}
	}
	return out
}

func applyPoemSnapshot(item *Item, raw string) {
	converted, err := poemcontent.PlanExampleFromSnapshot(raw, "")
	if err != nil {
		return
	}
	ex := converted.Example
	item.Example = &ex
	item.Question.Code = ex.Kind
	if ex.Prompt != "" {
		item.Question.Stem = ex.Prompt
	}
	switch poemcontent.KindForSkill(ex.Kind) {
	case poemcontent.KindRecite:
		if opts, err := json.Marshal(map[string]any{"items": ex.SequenceItems, "displayOrder": ex.SequenceDisplayOrder}); err == nil {
			item.Question.Options = opts
		}
	default:
		if opts, err := json.Marshal(ex.Options); err == nil {
			item.Question.Options = opts
		}
	}
}

func rawJSON(value, fallback string) json.RawMessage {
	if !json.Valid([]byte(value)) {
		value = fallback
	}
	return json.RawMessage(value)
}
