package knowledge

import (
	"testing"

	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/conchi/study-learning/englishcontent"
	"github.com/stretchr/testify/require"
)

func TestEnglishAttemptUsesLinkedPlanItemNotLatest(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	first := `{"schema":1,"kind":"audio-choice","skillCode":"listen","responseKind":"choice","selected":"2","example":{"kind":"audio-choice","speech":"apple","speechUrl":"/api/v1/english/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"1","label":"苹果","picture":"/one.jpg"},{"id":"2","label":"小狗","picture":"/two.jpg"},{"id":"3","label":"小猫","picture":"/three.jpg"},{"id":"4","label":"小鸟","picture":"/four.jpg"}],"answerId":"1"}}`
	second := `{"schema":1,"kind":"audio-choice","skillCode":"listen","responseKind":"choice","selected":"1","example":{"kind":"audio-choice","speech":"apple","speechUrl":"/api/v1/english/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"4","label":"小鸟","picture":"/four.jpg"},{"id":"1","label":"苹果","picture":"/one.jpg"},{"id":"3","label":"小猫","picture":"/three.jpg"},{"id":"2","label":"小狗","picture":"/two.jpg"}],"answerId":"1"}}`
	updated := `{"schema":1,"kind":"audio-choice","skillCode":"listen","responseKind":"choice","selected":"1","example":{"kind":"audio-choice","speech":"apple","speechUrl":"/api/v1/english/task-media/ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff.mp3","options":[{"id":"4","label":"小鸟","picture":"/four.jpg"},{"id":"1","label":"苹果","picture":"/new.jpg"},{"id":"3","label":"小猫","picture":"/three.jpg"},{"id":"2","label":"小狗","picture":"/two.jpg"}],"answerId":"1"}}`
	require.NoError(t, db.Exec(`
		INSERT INTO study_plans(id,child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count) VALUES
		 (11,1,'2026-09-10',1,'english','done',1,1,0),
		 (12,1,'2026-09-11',1,'english','done',1,1,1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO plan_items(id,plan_id,seq,kp_id,question_id,status,tries,cost_ms,picks,question_snapshot) VALUES
		 (110,11,1,20,201,'wrong',1,800,'2',?),
		 (111,12,1,20,201,'correct',1,800,'1',?)`, first, second).Error)
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at,plan_item_id) VALUES
		 (110,1,20,201,0,800,'quiz','first','2026-09-10 10:00:00',110),
		 (111,1,20,201,1,800,'quiz','second','2026-09-11 10:00:00',111)`).Error)
	require.NoError(t, db.Exec(`UPDATE plan_items SET question_snapshot=? WHERE id=111`, updated).Error)
	s := New(db)
	wrong, err := s.Attempt(1, 110)
	require.NoError(t, err)
	require.Equal(t, "instance_snapshot", wrong.QuestionFidelity)
	require.Equal(t, "2", wrong.Response.SelectedOptionID)
	require.Equal(t, "小狗", wrong.EnglishExample.Options[1].Label)
	require.Equal(t, "/api/v1/english/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3", wrong.EnglishExample.SpeechURL)
	require.Equal(t, []string{"1", "2", "3", "4"}, optionIDs(wrong.EnglishExample.Options))
	later, err := s.Attempt(1, 111)
	require.NoError(t, err)
	require.Equal(t, "1", later.Response.SelectedOptionID)
	require.Equal(t, []string{"4", "1", "3", "2"}, optionIDs(later.EnglishExample.Options))
	require.Contains(t, later.EnglishExample.SpeechURL, "fff")
}

func TestEnglishSamePlanItemKeepsEachAttemptSelection(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN selected TEXT NOT NULL DEFAULT ''`).Error)
	snap := `{"schema":1,"kind":"audio-choice","skillCode":"listen","responseKind":"choice","example":{"kind":"audio-choice","speech":"apple","speechUrl":"/api/v1/english/words/20/speech.mp3","options":[{"id":"20","label":"苹果","picture":"/one.jpg"},{"id":"21","label":"小狗","picture":"/two.jpg"},{"id":"22","label":"小猫","picture":"/three.jpg"},{"id":"23","label":"小鸟","picture":"/four.jpg"}],"answerId":"20"}}`
	require.NoError(t, db.Exec(`
		INSERT INTO study_plans(id,child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count) VALUES
		 (21,1,'2026-09-14',1,'english','done',1,1,1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO plan_items(id,plan_id,seq,kp_id,question_id,status,tries,cost_ms,picks,question_snapshot) VALUES
		 (210,21,1,20,201,'correct',2,800,?,?)`, `[{"clientId":"first","optionIndex":1},{"clientId":"second","optionIndex":0}]`, snap).Error)
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at,plan_item_id,selected) VALUES
		 (210,1,20,201,0,800,'quiz','first','2026-09-14 10:00:00',210,'21'),
		 (211,1,20,201,1,800,'quiz','second','2026-09-14 10:01:00',210,'20')`).Error)
	s := New(db)
	wrong, err := s.Attempt(1, 210)
	require.NoError(t, err)
	require.Equal(t, "21", wrong.Response.SelectedOptionID)
	require.Equal(t, "小狗", selectedLabel(wrong.EnglishExample.Options, wrong.Response.SelectedOptionID))
	later, err := s.Attempt(1, 211)
	require.NoError(t, err)
	require.Equal(t, "20", later.Response.SelectedOptionID)
	require.Equal(t, "苹果", selectedLabel(later.EnglishExample.Options, later.Response.SelectedOptionID))
	require.Equal(t, wrong.EnglishExample.SpeechURL, later.EnglishExample.SpeechURL)
}

func TestEnglishUnlinkedAttemptIsNotInvented(t *testing.T) {
	db := testdb.Open(t)
	later := `{"schema":1,"kind":"audio-choice","skillCode":"listen","responseKind":"choice","selected":"1","example":{"kind":"audio-choice","speech":"apple","speechUrl":"/api/v1/english/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"1","label":"苹果","picture":"/one.jpg"},{"id":"2","label":"小狗","picture":"/two.jpg"},{"id":"3","label":"小猫","picture":"/three.jpg"},{"id":"4","label":"小鸟","picture":"/four.jpg"}],"answerId":"1"}}`
	require.NoError(t, db.Exec(`
		INSERT INTO study_plans(id,child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count)
		VALUES (13,1,'2026-09-12',1,'english','done',1,1,1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO plan_items(id,plan_id,seq,kp_id,question_id,status,tries,cost_ms,picks,question_snapshot)
		VALUES (112,13,1,20,201,'correct',1,800,'1',?)`, later).Error)
	s := New(db)
	got, err := s.Attempt(1, 1)
	require.NoError(t, err)
	require.Contains(t, got.EvidenceReasonCodes, "unlinked_plan_item")
	require.Nil(t, got.EnglishExample)
}

func TestEnglishUnlinkedAttemptIgnoresLaterPlanItemEvenWithColumn(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	later := `{"schema":1,"kind":"audio-choice","skillCode":"listen","responseKind":"choice","selected":"1","example":{"kind":"audio-choice","speech":"apple","speechUrl":"/api/v1/english/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"1","label":"苹果","picture":"/one.jpg"},{"id":"2","label":"小狗","picture":"/two.jpg"},{"id":"3","label":"小猫","picture":"/three.jpg"},{"id":"4","label":"小鸟","picture":"/four.jpg"}],"answerId":"1"}}`
	require.NoError(t, db.Exec(`
		INSERT INTO study_plans(id,child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count)
		VALUES (13,1,'2026-09-12',1,'english','done',1,1,1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO plan_items(id,plan_id,seq,kp_id,question_id,status,tries,cost_ms,picks,question_snapshot)
		VALUES (112,13,1,20,201,'correct',1,800,'1',?)`, later).Error)
	s := New(db)
	got, err := s.Attempt(1, 1)
	require.NoError(t, err)
	require.Contains(t, got.EvidenceReasonCodes, "unlinked_plan_item")
	require.Nil(t, got.EnglishExample)
}

func TestEnglishCodeTypeSnapshotCannotInventHistory(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN selected TEXT NOT NULL DEFAULT ''`).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO study_plans(id,child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count)
		VALUES (31,1,'2026-09-14',1,'english','done',1,1,0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO plan_items(id,plan_id,seq,kp_id,question_id,status,tries,cost_ms,picks,question_snapshot) VALUES
		 (310,31,1,20,201,'wrong',1,800,'2','{"code":"listen","type":"choice"}')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at,plan_item_id,selected) VALUES
		 (310,1,20,201,0,800,'quiz','legacy-code-type','2026-09-14 10:00:00',310,'2')`).Error)
	s := New(db)
	got, err := s.Attempt(1, 310)
	require.NoError(t, err)
	require.Contains(t, got.EvidenceReasonCodes, "invalid_snapshot")
	require.Nil(t, got.EnglishExample)
	require.Empty(t, got.Response.SelectedOptionID)
}

func optionIDs(opts []englishcontent.Choice) []string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.ID)
	}
	return out
}

func selectedLabel(opts []englishcontent.Choice, id string) string {
	for _, o := range opts {
		if o.ID == id {
			return o.Label
		}
	}
	return ""
}
