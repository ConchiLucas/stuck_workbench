package knowledge

import (
	"fmt"
	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func fixedNow() time.Time { return time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC) }
func TestAnalysisRetainsRetryFacts(t *testing.T) {
	n := fixedNow()
	rows := []Evidence{{AttemptID: 1, KpID: 10, SkillCode: "write_char", OccurredAt: n.Add(-time.Hour), InstanceKey: "item:1", Assistance: "none", IsCorrect: false}, {AttemptID: 2, KpID: 10, SkillCode: "write_char", OccurredAt: n, InstanceKey: "item:1", Assistance: "hinted", IsCorrect: true}}
	c := Analyze(rows, n)
	require.Len(t, c, 1)
	require.Len(t, c[0].Evidence, 2)
	require.Equal(t, "assisted_completion", c[0].Evidence[1].Role)
	rows[0].IsCorrect = true
	rows[1].IsCorrect = false
	rows[1].Assistance = "none"
	c = Analyze(rows, n)
	require.Len(t, c, 1)
	require.Equal(t, 1, c[0].WrongCount)
	require.Equal(t, 0, c[0].WrongInitialCount)
}
func TestFollowUpUnknownAndFiltering(t *testing.T) {
	db := testdb.Open(t)
	s := New(db)
	s.Now = fixedNow
	p, e := s.Attempts(1, Filter{FollowUpState: "no_later_practice", Limit: 1, WrongOnly: true})
	require.NoError(t, e)
	require.Len(t, p.Items, 1)
	require.True(t, p.HasMore)
	p, e = s.Attempts(1, Filter{FollowUpState: "insufficient_evidence", WrongOnly: true})
	require.NoError(t, e)
	require.Len(t, p.Items, 1)
	require.Equal(t, int64(1), p.Items[0].AttemptID)
}
func TestSkillGapCandidate(t *testing.T) {
	db := testdb.Open(t)
	s := New(db)
	s.Now = fixedNow
	p, e := s.Candidates(1, Filter{Subject: "literacy"})
	require.NoError(t, e)
	require.Len(t, p.Items, 1)
	require.Equal(t, "observed_wrong", p.Items[0].ReasonCode)
}
func TestInvalidPinyinReceiptNotIndependent(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`CREATE TABLE pinyin_quiz_instances(id TEXT PRIMARY KEY,child_id INTEGER,kp_id INTEGER,public_snapshot TEXT,answer_option_id TEXT); CREATE TABLE pinyin_answer_receipts(child_id INTEGER,client_id TEXT,instance_id TEXT,attempt_id INTEGER,skill_code TEXT,selected_option_id TEXT); INSERT INTO pinyin_quiz_instances VALUES('bad',1,20,'bad','x'); INSERT INTO pinyin_answer_receipts VALUES(1,'a1','bad',1,'listen','x')`).Error)
	s := New(db)
	p, e := s.Summary(1)
	require.NoError(t, e)
	require.Equal(t, 0, p.Stats.IndependentAttempts)
}

func TestGroupsKeepAllHistoryAndRetries(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`UPDATE attempts SET created_at='2025-01-01'`).Error)
	s := New(db)
	s.Now = fixedNow
	reader, ok := any(s).(interface {
		Groups(int64, Filter) (Page[Candidate], error)
	})
	require.True(t, ok, "historical groups reader is required")
	p, e := reader.Groups(1, Filter{Subject: "english"})
	require.NoError(t, e)
	require.Len(t, p.Items, 1)
	require.Equal(t, 2, p.Items[0].WrongCount)
	require.Len(t, p.Items[0].Evidence, 2)
}
func TestScienceReceiptMapsItsOwnTry(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`UPDATE subjects SET code='science' WHERE id=2;UPDATE questions SET code='recognize' WHERE id=201;
 ALTER TABLE plan_items ADD COLUMN question_stem TEXT; ALTER TABLE plan_items ADD COLUMN question_options TEXT; ALTER TABLE plan_items ADD COLUMN option_order TEXT;
 UPDATE plan_items SET question_stem='哪一个',question_options='[{"label":"A"},{"label":"B"},{"label":"C"}]',question_answer='{"index":1}',option_order='2,0,1',picks='0,2',status='completed';
 CREATE TABLE science_attempt_receipts(child_id INTEGER,client_id TEXT,kp_id INTEGER,question_id INTEGER,plan_id INTEGER,item_id INTEGER,result_json TEXT);
 INSERT INTO science_attempt_receipts VALUES(1,'a1',20,201,9,90,'{"tries":1}'),(1,'a2',20,201,9,90,'{"tries":2}');`).Error)
	s := New(db)
	s.Now = fixedNow
	a, e := s.Attempt(1, 1)
	require.NoError(t, e)
	require.Equal(t, "verified_order", a.SelectionFidelity)
	require.Equal(t, "reference-2", a.Response.SelectedOptionID)
	require.Equal(t, "reference-1", a.Question.AnswerOptionID)
	a, e = s.Attempt(1, 2)
	require.NoError(t, e)
	require.Equal(t, "reference-1", a.Response.SelectedOptionID)
	require.NoError(t, db.Exec(`UPDATE science_attempt_receipts SET result_json='{"tries":8}' WHERE client_id='a1'`).Error)
	a, e = s.Attempt(1, 1)
	require.NoError(t, e)
	require.Equal(t, "missing", a.SelectionFidelity)
}
func TestSkillGapWithoutReliableStrengthDoesNotInventCandidate(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`UPDATE attempts SET is_correct=1 WHERE id=3`).Error)
	s := New(db)
	s.Now = fixedNow
	p, e := s.Candidates(1, Filter{Subject: "literacy"})
	require.NoError(t, e)
	require.Empty(t, p.Items)
}

func TestUnknownChoiceContractDoesNotProveIndependence(t *testing.T) {
	r := Evidence{KpID: 10, SubjectCode: "literacy", Assistance: "unknown", Media: map[string]string{}}
	decodeVersion(&r, versionRow{QuestionType: "future_choice", SkillCode: "future_choice", SelectedOptionID: "b", SnapshotJSON: `{"schemaVersion":1,"subjectCode":"literacy","kpId":10,"questionType":"future_choice","skillCode":"future_choice","answerOptionId":"a","options":[{"id":"a"},{"id":"b"}]}`})
	require.Equal(t, "unknown", r.Assistance)
}
func TestInvalidSelectionDoesNotProveIndependence(t *testing.T) {
	r := Evidence{KpID: 10, SubjectCode: "literacy", Assistance: "unknown", Media: map[string]string{}}
	decodeVersion(&r, versionRow{QuestionType: "glyph_sense", SkillCode: "glyph_sense", SelectedOptionID: "missing", SnapshotJSON: `{"schemaVersion":1,"subjectCode":"literacy","kpId":10,"questionType":"glyph_sense","skillCode":"glyph_sense","answerOptionId":"a","options":[{"id":"a"},{"id":"b"}]}`})
	require.Equal(t, "unknown", r.Assistance)
}
func TestAnalysisRestrictsKnownInstancesButKeepsTheirRetries(t *testing.T) {
	rows := []Evidence{}
	for i := 1; i <= 21; i++ {
		rows = append(rows, Evidence{AttemptID: int64(i), KpID: 10, SkillCode: "write_char", InstanceKey: fmt.Sprintf("item:%d", i), Assistance: "none", IsCorrect: i != 1, OccurredAt: fixedNow().Add(time.Duration(i-22) * time.Hour)})
	}
	require.Empty(t, Analyze(rows, fixedNow()), "oldest instance is outside the recent twenty")
	rows = append(rows, Evidence{AttemptID: 22, KpID: 10, SkillCode: "write_char", InstanceKey: "item:21", Assistance: "hinted", IsCorrect: true, OccurredAt: fixedNow()})
	c := Analyze(rows, fixedNow())
	require.Len(t, c, 1)
	require.Equal(t, "assisted_completion", c[0].ReasonCode)
	require.Len(t, c[0].Evidence, 21)
}

func TestFollowUpUsesLaterInstanceInitialAndMastery(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`UPDATE subjects SET code='pinyin' WHERE id=2;UPDATE modules SET code='syllables' WHERE id=2;UPDATE questions SET code='blend' WHERE id=201;
 CREATE TABLE pinyin_quiz_instances(id TEXT PRIMARY KEY,child_id INTEGER,kp_id INTEGER,public_snapshot TEXT,answer_option_id TEXT);
 CREATE TABLE pinyin_answer_receipts(child_id INTEGER,client_id TEXT,instance_id TEXT,attempt_id INTEGER,skill_code TEXT,selected_option_id TEXT);
 INSERT INTO pinyin_quiz_instances VALUES('first',1,20,'{"kpId":20,"options":[{"id":"a","label":"a"},{"id":"b","label":"b"}]}','a'),('second',1,20,'{"kpId":20,"options":[{"id":"a","label":"a"},{"id":"b","label":"b"}]}','a'),('third',1,20,'{"kpId":20,"options":[{"id":"a","label":"a"},{"id":"b","label":"b"}]}','a');
 INSERT INTO pinyin_answer_receipts VALUES(1,'a1','first',1,'blend','b'),(1,'a2','second',2,'blend','b');
 INSERT INTO attempts VALUES(4,1,20,201,1,100,'quiz','a4','2026-09-07 11:00:00');INSERT INTO pinyin_answer_receipts VALUES(1,'a4','second',4,'blend','a');`).Error)
	s := New(db)
	s.Now = fixedNow
	a, e := s.Attempt(1, 1)
	require.NoError(t, e)
	require.Equal(t, "still_wrong", a.FollowUpState)
	pending, e := s.Attempts(1, Filter{Subject: "pinyin", From: "2026-09-07", To: "2026-09-07", WrongOnly: true, FollowUpState: "needs_practice"})
	require.NoError(t, e)
	require.Len(t, pending.Items, 2, "same-instance correction must not resolve the errors")

	require.NoError(t, db.Exec(`INSERT INTO attempts VALUES(5,1,20,201,1,100,'quiz','a5','2026-09-08 11:00:00');INSERT INTO pinyin_answer_receipts VALUES(1,'a5','third',5,'blend','a');`).Error)
	a, e = s.Attempt(1, 1)
	require.NoError(t, e)
	require.Equal(t, "answered_correctly_later", a.FollowUpState)
	pending, e = s.Attempts(1, Filter{Subject: "pinyin", From: "2026-09-07", To: "2026-09-07", WrongOnly: true, FollowUpState: "needs_practice"})
	require.NoError(t, e)
	require.Empty(t, pending.Items, "later independent success outside the window resolves both errors")
	require.NoError(t, db.Exec(`INSERT INTO attempts VALUES(6,1,20,201,1,100,'quiz','unknown','2026-09-09 11:00:00')`).Error)
	pending, e = s.Attempts(1, Filter{Subject: "pinyin", From: "2026-09-07", To: "2026-09-07", WrongOnly: true, FollowUpState: "needs_practice"})
	require.NoError(t, e)
	require.Empty(t, pending.Items, "newer unknown success does not erase independently verified success")
	require.NoError(t, db.Exec(`DELETE FROM attempts WHERE id=6`).Error)

	require.NoError(t, db.Exec(`INSERT INTO mastery_skills(child_id,kp_id,skill_code,status) VALUES(1,20,'blend','mastered')`).Error)
	a, e = s.Attempt(1, 1)
	require.NoError(t, e)
	require.Equal(t, "mastered_later", a.FollowUpState)
	a, e = s.Attempt(1, 5)
	require.NoError(t, e)
	require.Equal(t, "no_later_practice", a.FollowUpState)
}
func TestRetryInsideWindowDoesNotBecomeFirstAnswer(t *testing.T) {
	rows := []Evidence{{AttemptID: 1, KpID: 10, SkillCode: "write_char", InstanceKey: "item:1", Assistance: "none", IsCorrect: false, OccurredAt: fixedNow().AddDate(0, 0, -31)}, {AttemptID: 2, KpID: 10, SkillCode: "write_char", InstanceKey: "item:1", Assistance: "none", IsCorrect: false, OccurredAt: fixedNow()}}
	c := Analyze(rows, fixedNow())
	require.Len(t, c, 1)
	require.Equal(t, 0, c[0].IndependentInstanceCount)
	require.Equal(t, 1, c[0].WrongCount)
}

func TestAttemptsByIDsScopesChildAndRealFacts(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`INSERT INTO children(id,name) VALUES(2,'other');INSERT INTO attempts VALUES(4,2,20,201,0,100,'quiz','other','2026-09-08'),(5,1,20,201,1,100,'parent_mark','mark','2026-09-08')`).Error)
	reader, ok := any(New(db)).(interface {
		AttemptsByIDs(int64, []int64) ([]Evidence, error)
	})
	require.True(t, ok, "bounded fact reader is required")
	rows, e := reader.AttemptsByIDs(1, []int64{1, 1, 4, 5})
	require.NoError(t, e)
	require.Len(t, rows, 1)
	require.Equal(t, int64(1), rows[0].AttemptID)
	require.Equal(t, "unknown", rows[0].Assistance)
	rows, e = reader.AttemptsByIDs(1, nil)
	require.NoError(t, e)
	require.Empty(t, rows)
}

func TestPinyinBlendPreservesVisualWithoutAnswerInStem(t *testing.T) {
	r := Evidence{KpID: 40, SubjectCode: "pinyin", ModuleCode: "syllables", Assistance: "unknown", Media: map[string]string{}}
	decodePinyin(&r, pinyinRow{InstanceID: "blend-1", SkillCode: "blend", SelectedOptionID: "b", AnswerOptionID: "a", PublicSnapshot: `{"kpId":40,"stem":"声母和韵母合起来怎么读？","visual":{"kind":"blend","initial":"b","final":"a","syllable":"ba"},"options":[{"id":"a","label":"ba"},{"id":"b","label":"pa"}]}`})
	require.NotNil(t, r.Question)
	require.JSONEq(t, `{"kind":"blend","initial":"b","final":"a","syllable":"ba"}`, string(r.Question.Visual))
	require.Equal(t, "声母和韵母合起来怎么读？", r.Question.Stem.Text)
}

func TestStrengthEvidenceRequiresRecentIndependentMasteredSkill(t *testing.T) {
	p := Point{KpID: 10, Skills: []Skill{{SkillCode: "sense_char", MasteryStatus: "mastered"}}}
	rows := []Evidence{{AttemptID: 1, KpID: 10, SkillCode: "sense_char", IsCorrect: true, Assistance: "none", InstanceKey: "item:1", OccurredAt: fixedNow().Add(-time.Hour)}}
	ev := supportingStrength(p, "glyph_sense", rows, fixedNow())
	require.Len(t, ev, 1)
	require.Equal(t, "supporting_strength", ev[0].Role)
	for _, kind := range []string{"old", "hinted", "unknown", "not_mastered"} {
		r := append([]Evidence(nil), rows...)
		point := p
		point.Skills = append([]Skill(nil), p.Skills...)
		switch kind {
		case "old":
			r[0].OccurredAt = fixedNow().AddDate(0, 0, -31)
		case "not_mastered":
			point.Skills[0].MasteryStatus = "learning"
		default:
			r[0].Assistance = kind
		}
		require.Empty(t, supportingStrength(point, "glyph_sense", r, fixedNow()), kind)
	}
}
func TestAnalysisSQLWindowOrdersFirstAnswerNotRetry(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`CREATE TABLE pinyin_quiz_instances(id TEXT PRIMARY KEY,child_id INTEGER,kp_id INTEGER,public_snapshot TEXT,answer_option_id TEXT); CREATE TABLE pinyin_answer_receipts(child_id INTEGER,client_id TEXT,instance_id TEXT,attempt_id INTEGER,skill_code TEXT,selected_option_id TEXT)`).Error)
	for i := 1; i <= 21; i++ {
		id := int64(i + 10)
		at := fixedNow().Add(time.Duration(i-22) * time.Hour)
		require.NoError(t, db.Exec("INSERT INTO attempts VALUES(?,1,20,201,0,100,'quiz',?,?)", id, fmt.Sprint(id), at).Error)
		require.NoError(t, db.Exec(`INSERT INTO pinyin_quiz_instances VALUES(?,1,20,'{"kpId":20,"options":[{"id":"a"},{"id":"b"}]}','a')`, fmt.Sprint(i)).Error)
		require.NoError(t, db.Exec("INSERT INTO pinyin_answer_receipts VALUES(1,?,?,?,'listen','b')", fmt.Sprint(id), fmt.Sprint(i), id).Error)
	}
	require.NoError(t, db.Exec("INSERT INTO attempts VALUES(100,1,20,201,0,100,'quiz','retry',?)", fixedNow()).Error)
	require.NoError(t, db.Exec("INSERT INTO pinyin_answer_receipts VALUES(1,'retry','1',100,'listen','b')").Error)
	s := New(db)
	rows, _, e := s.analysisEvidence(1, Filter{Skill: "listen"}, 100, fixedNow())
	require.NoError(t, e)
	ids := map[int64]bool{}
	for _, r := range rows {
		ids[r.AttemptID] = true
	}
	require.True(t, ids[12], "second instance must remain in latest twenty first answers")
	require.False(t, ids[11], "retry must not move oldest first answer into the window")
}

func TestGapCandidateIncludesStrengthEvenWithSkillFilter(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`UPDATE subjects SET code='pinyin' WHERE id=1;UPDATE modules SET code='letters' WHERE id=1;
 UPDATE mastery_skills SET skill_code='listen' WHERE kp_id=10 AND skill_code='glyph_sense';UPDATE mastery_skills SET skill_code='shape' WHERE kp_id=10 AND skill_code='write_char';
 CREATE TABLE pinyin_quiz_instances(id TEXT PRIMARY KEY,child_id INTEGER,kp_id INTEGER,public_snapshot TEXT,answer_option_id TEXT);
 CREATE TABLE pinyin_answer_receipts(child_id INTEGER,client_id TEXT,instance_id TEXT,attempt_id INTEGER,skill_code TEXT,selected_option_id TEXT);
 INSERT INTO pinyin_quiz_instances VALUES('weak',1,10,'{"kpId":10,"options":[{"id":"a"},{"id":"b"}]}','a'),('strong',1,10,'{"kpId":10,"options":[{"id":"a"},{"id":"b"}]}','a');
 INSERT INTO pinyin_answer_receipts VALUES(1,'a3','weak',3,'shape','b');
 INSERT INTO attempts VALUES(10,1,10,NULL,1,100,'quiz','strong','2026-09-08 11:00:00');
 INSERT INTO pinyin_answer_receipts VALUES(1,'strong','strong',10,'listen','a');`).Error)
	s := New(db)
	s.Now = fixedNow
	p, e := s.Candidates(1, Filter{Subject: "pinyin", Skill: "shape"})
	require.NoError(t, e)
	require.Len(t, p.Items, 1)
	require.Equal(t, "practiced_skill_gap", p.Items[0].ReasonCode)
	require.Contains(t, p.Items[0].Evidence, CandidateEvidence{10, "supporting_strength", SourceRef{Kind: "pinyin_instance", InstanceID: "strong", ClientID: "strong"}})
}

func TestPinyinConflictingResultKeepsFactWithoutIndependentCredit(t *testing.T) {
	r := Evidence{KpID: 40, SubjectCode: "pinyin", ModuleCode: "syllables", IsCorrect: true, Assistance: "unknown", Media: map[string]string{}}
	decodePinyin(&r, pinyinRow{InstanceID: "i", SkillCode: "blend", SelectedOptionID: "b", AnswerOptionID: "a", PublicSnapshot: `{"kpId":40,"options":[{"id":"a"},{"id":"b"}]}`})
	require.True(t, r.IsCorrect, "stored result remains a fact")
	require.Equal(t, "unknown", r.Assistance)
	require.Equal(t, "b", r.Response.SelectedOptionID)
	require.Contains(t, r.EvidenceReasonCodes, "result_conflict")
}

func TestVersionConflictingChoiceKeepsSceneAndBlocksReview(t *testing.T) {
	r := Evidence{KpID: 10, SubjectCode: "literacy", IsCorrect: true, Assistance: "unknown", Media: map[string]string{}}
	decodeVersion(&r, versionRow{QuestionType: "glyph_sense", SkillCode: "glyph_sense", PlanStatus: "done", SelectedOptionID: "b", SnapshotJSON: `{"schemaVersion":1,"subjectCode":"literacy","kpId":10,"questionType":"glyph_sense","skillCode":"glyph_sense","answerOptionId":"a","options":[{"id":"a"},{"id":"b"}]}`})
	require.True(t, r.IsCorrect)
	require.Equal(t, "b", r.Response.SelectedOptionID)
	require.Equal(t, "unknown", r.Assistance)
	require.False(t, r.ReviewEligible)
	require.Contains(t, r.EvidenceReasonCodes, "result_conflict")
}
func TestWritingZeroHintsDoesNotReplaceUnknownEvaluation(t *testing.T) {
	for _, tc := range []struct{ payload, eval, want string }{
		{`{"strokes":[[]],"hintsUsed":0}`, `{}`, "unknown"},
		{`{"strokes":[[]],"hintsUsed":0}`, `{"assistance":"none"}`, "none"},
		{`{"strokes":[[]],"hintsUsed":1}`, `{}`, "hinted"},
		{`{"strokes":[[]],"hintsUsed":1}`, `bad`, "hinted"},
		{`{"strokes":[[]],"hintsUsed":-1}`, `{"assistance":"hinted"}`, "unknown"},
	} {
		r := Evidence{KpID: 10, SubjectCode: "literacy", Media: map[string]string{}}
		decodeVersion(&r, versionRow{QuestionType: "write_char", SkillCode: "write_char", PlanStatus: "done", AnswerPayloadJSON: tc.payload, EvaluationJSON: tc.eval, SnapshotJSON: `{"schemaVersion":2,"subjectCode":"literacy","kpId":10,"questionType":"write_char","skillCode":"write_char","interaction":"handwriting"}`})
		require.Equal(t, tc.want, r.Assistance, tc.payload+tc.eval)
	}
}

func TestWritingOutcomeConflictPreservesFactButNotIndependentEvidence(t *testing.T) {
	r := Evidence{AttemptID: 1, KpID: 10, SubjectCode: "literacy", IsCorrect: true, Assistance: "unknown", Media: map[string]string{}, ReviewBlockReasons: []string{"evidence_incomplete"}}
	decodeVersion(&r, versionRow{PlanItemID: 1, QuestionType: "write_char", SkillCode: "write_char", PlanStatus: "done", SnapshotJSON: `{"schemaVersion":2,"subjectCode":"literacy","kpId":10,"questionType":"write_char","skillCode":"write_char","interaction":"handwriting"}`, AnswerPayloadJSON: `{"strokes":[],"hintsUsed":0}`, EvaluationJSON: `{"outcome":"not_passed","assistance":"none"}`})
	require.True(t, r.IsCorrect)
	require.Equal(t, "unknown", r.Assistance)
	require.False(t, r.ReviewEligible)
	require.Contains(t, r.EvidenceReasonCodes, "result_conflict")
}
