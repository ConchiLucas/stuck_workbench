package knowledge

import (
	"testing"

	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/stretchr/testify/require"
)

func TestLogicAttemptUsesLinkedPlanItemNotLatest(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN selected TEXT NOT NULL DEFAULT ''`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subjects(id,code,name,icon,order_no) VALUES (8,'logic','逻辑','🧩',7)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO modules(id,subject_id,code,name,order_no) VALUES (8,8,'playground','逻辑练习',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO knowledge_points(id,module_id,code,title,order_no) VALUES (80,8,'logic-classify-animal','动物里的例外',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO questions(id,kp_id,code,type,stem,options,answer,difficulty) VALUES (801,80,'classify','practice','选','[]','{}',1)`).Error)
	first := `{"schema":1,"kind":"classify","prompt":"哪个不属于这一类？","rule":{"type":"odd-one-out","dimension":"kingdom","inGroup":"animal","explain":"动物"},"objects":[{"id":"cat","caption":"猫","glyph":"cat","attrs":{"category":"animal"}},{"id":"car","caption":"汽车","glyph":"car","attrs":{"category":"vehicle"}}],"options":["cat","car"],"answerId":"car"}`
	second := `{"schema":1,"kind":"classify","prompt":"哪个不属于这一类？","rule":{"type":"odd-one-out","dimension":"kingdom","inGroup":"animal","explain":"动物"},"objects":[{"id":"cat","caption":"猫","glyph":"cat","attrs":{"category":"animal"}},{"id":"car","caption":"汽车","glyph":"car","attrs":{"category":"vehicle"}}],"options":["car","cat"],"answerId":"car"}`
	require.NoError(t, db.Exec(`
		INSERT INTO study_plans(id,child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count) VALUES
		 (81,1,'2026-09-10',1,'logic','done',1,1,0),
		 (82,1,'2026-09-11',1,'logic','done',1,1,1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO plan_items(id,plan_id,seq,kp_id,question_id,status,tries,cost_ms,picks,question_snapshot) VALUES
		 (810,81,1,80,801,'wrong',1,800,?,?),
		 (811,82,1,80,801,'correct',1,800,?,?)`, `{"selectedId":"cat"}`, first, `{"selectedId":"car"}`, second).Error)
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at,plan_item_id,selected) VALUES
		 (810,1,80,801,0,800,'quiz','lg-first','2026-09-10 10:00:00',810,?),
		 (811,1,80,801,1,800,'quiz','lg-second','2026-09-11 10:00:00',811,?)`, `{"selectedId":"cat"}`, `{"selectedId":"car"}`).Error)
	s := New(db)
	wrong, err := s.Attempt(1, 810)
	require.NoError(t, err)
	require.Equal(t, "instance_snapshot", wrong.QuestionFidelity)
	require.Contains(t, wrong.Response.SelectedOptionID, "cat")
	require.Equal(t, []string{"cat", "car"}, wrong.LogicExample.Options)
	later, err := s.Attempt(1, 811)
	require.NoError(t, err)
	require.Contains(t, later.Response.SelectedOptionID, "car")
	require.Equal(t, []string{"car", "cat"}, later.LogicExample.Options)
}

func TestLogicUnlinkedAttemptCannotInventHistory(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subjects(id,code,name,icon,order_no) VALUES (8,'logic','逻辑','🧩',7)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO modules(id,subject_id,code,name,order_no) VALUES (8,8,'playground','逻辑练习',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO knowledge_points(id,module_id,code,title,order_no) VALUES (80,8,'logic-classify-animal','动物里的例外',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO questions(id,kp_id,code,type,stem,options,answer,difficulty) VALUES (801,80,'classify','practice','选','[]','{}',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at) VALUES
		 (910,1,80,801,0,800,'quiz','lg-unlinked','2026-09-14 10:00:00')`).Error)
	s := New(db)
	got, err := s.Attempt(1, 910)
	require.NoError(t, err)
	require.Contains(t, got.EvidenceReasonCodes, "unlinked_plan_item")
	require.Nil(t, got.LogicExample)
}
