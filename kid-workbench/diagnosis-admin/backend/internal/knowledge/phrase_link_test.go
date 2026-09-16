package knowledge

import (
	"testing"

	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/conchi/study-learning/phrasecontent"
	"github.com/stretchr/testify/require"
)

func TestPhraseAttemptUsesLinkedPlanItemNotLatest(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subjects(id,code,name,icon,order_no) VALUES (4,'phrase','英语短句','💬',4)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO modules(id,subject_id,code,name,order_no) VALUES (4,4,'greet','问候',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO knowledge_points(id,module_id,code,title,order_no) VALUES (40,4,'gm','Good morning.',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO questions(id,kp_id,code,type,stem,options,answer,difficulty) VALUES (401,40,'listen_zh','choice','听','[]','{}',1)`).Error)
	first := `{"schema":1,"kind":"listen_zh","skillCode":"listen_zh","responseKind":"choice","selected":"2","example":{"kind":"listen_zh","speech":"Good morning.","speechUrl":"/api/v1/phrase/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"1","label":"早上好。"},{"id":"2","label":"下午好。"},{"id":"3","label":"晚上好。"},{"id":"4","label":"晚安。"}],"answerId":"1"}}`
	second := `{"schema":1,"kind":"listen_zh","skillCode":"listen_zh","responseKind":"choice","selected":"1","example":{"kind":"listen_zh","speech":"Good morning.","speechUrl":"/api/v1/phrase/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"4","label":"晚安。"},{"id":"1","label":"早上好。"},{"id":"3","label":"晚上好。"},{"id":"2","label":"下午好。"}],"answerId":"1"}}`
	updated := `{"schema":1,"kind":"listen_zh","skillCode":"listen_zh","responseKind":"choice","selected":"1","example":{"kind":"listen_zh","speech":"Good morning.","speechUrl":"/api/v1/phrase/task-media/ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff.mp3","options":[{"id":"4","label":"晚安。"},{"id":"1","label":"早上好。"},{"id":"3","label":"晚上好。"},{"id":"2","label":"下午好。"}],"answerId":"1"}}`
	require.NoError(t, db.Exec(`
		INSERT INTO study_plans(id,child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count) VALUES
		 (41,1,'2026-09-10',1,'phrase','done',1,1,0),
		 (42,1,'2026-09-11',1,'phrase','done',1,1,1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO plan_items(id,plan_id,seq,kp_id,question_id,status,tries,cost_ms,picks,question_snapshot) VALUES
		 (410,41,1,40,401,'wrong',1,800,'2',?),
		 (411,42,1,40,401,'correct',1,800,'1',?)`, first, second).Error)
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at,plan_item_id) VALUES
		 (410,1,40,401,0,800,'quiz','ph-first','2026-09-10 10:00:00',410),
		 (411,1,40,401,1,800,'quiz','ph-second','2026-09-11 10:00:00',411)`).Error)
	require.NoError(t, db.Exec(`UPDATE plan_items SET question_snapshot=? WHERE id=411`, updated).Error)
	s := New(db)
	wrong, err := s.Attempt(1, 410)
	require.NoError(t, err)
	require.Equal(t, "instance_snapshot", wrong.QuestionFidelity)
	require.Equal(t, "2", wrong.Response.SelectedOptionID)
	require.Equal(t, "下午好。", wrong.PhraseExample.Options[1].Label)
	require.Equal(t, "/api/v1/phrase/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3", wrong.PhraseExample.SpeechURL)
	require.Equal(t, []string{"1", "2", "3", "4"}, phraseOptionIDs(wrong.PhraseExample.Options))
	later, err := s.Attempt(1, 411)
	require.NoError(t, err)
	require.Equal(t, "1", later.Response.SelectedOptionID)
	require.Equal(t, []string{"4", "1", "3", "2"}, phraseOptionIDs(later.PhraseExample.Options))
	require.Contains(t, later.PhraseExample.SpeechURL, "fff")
}

func TestPhraseSamePlanItemKeepsEachAttemptSelection(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN selected TEXT NOT NULL DEFAULT ''`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subjects(id,code,name,icon,order_no) VALUES (4,'phrase','英语短句','💬',4)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO modules(id,subject_id,code,name,order_no) VALUES (4,4,'greet','问候',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO knowledge_points(id,module_id,code,title,order_no) VALUES (40,4,'gm','Good morning.',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO questions(id,kp_id,code,type,stem,options,answer,difficulty) VALUES (401,40,'listen_zh','choice','听','[]','{}',1)`).Error)
	snap := `{"schema":1,"kind":"listen_zh","skillCode":"listen_zh","responseKind":"choice","example":{"kind":"listen_zh","speech":"Good morning.","speechUrl":"/api/v1/phrase/items/40/speech.mp3","options":[{"id":"40","label":"早上好。"},{"id":"41","label":"下午好。"},{"id":"42","label":"晚上好。"},{"id":"43","label":"晚安。"}],"answerId":"40"}}`
	require.NoError(t, db.Exec(`
		INSERT INTO study_plans(id,child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count) VALUES
		 (51,1,'2026-09-14',1,'phrase','done',1,1,1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO plan_items(id,plan_id,seq,kp_id,question_id,status,tries,cost_ms,picks,question_snapshot) VALUES (510,51,1,40,401,'correct',2,1600,'40',?)`, snap).Error)
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at,plan_item_id,selected) VALUES
		 (510,1,40,401,0,800,'quiz','ph-same-1','2026-09-14 10:00:00',510,'41'),
		 (511,1,40,401,1,800,'quiz','ph-same-2','2026-09-14 10:01:00',510,'40')`).Error)
	s := New(db)
	wrong, err := s.Attempt(1, 510)
	require.NoError(t, err)
	right, err := s.Attempt(1, 511)
	require.NoError(t, err)
	require.Equal(t, "41", wrong.Response.SelectedOptionID)
	require.Equal(t, "40", right.Response.SelectedOptionID)
	require.Equal(t, wrong.PhraseExample.SpeechURL, right.PhraseExample.SpeechURL)
}

func TestPhraseUnlinkedAttemptCannotInventHistory(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subjects(id,code,name,icon,order_no) VALUES (4,'phrase','英语短句','💬',4)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO modules(id,subject_id,code,name,order_no) VALUES (4,4,'greet','问候',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO knowledge_points(id,module_id,code,title,order_no) VALUES (40,4,'gm','Good morning.',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO questions(id,kp_id,code,type,stem,options,answer,difficulty) VALUES (401,40,'listen_zh','choice','听','[]','{}',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at) VALUES
		 (610,1,40,401,0,800,'quiz','ph-unlinked','2026-09-14 10:00:00')`).Error)
	s := New(db)
	got, err := s.Attempt(1, 610)
	require.NoError(t, err)
	require.Contains(t, got.EvidenceReasonCodes, "unlinked_plan_item")
	require.Nil(t, got.PhraseExample)
	require.False(t, got.IsCorrect == true)
}

func phraseOptionIDs(opts []phrasecontent.Choice) []string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.ID)
	}
	return out
}
