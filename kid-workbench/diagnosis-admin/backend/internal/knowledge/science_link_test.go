package knowledge

import (
	"strings"
	"testing"

	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/conchi/study-learning/sciencecontent"
	"github.com/stretchr/testify/require"
)

func TestScienceAttemptUsesLinkedPlanItemNotLatest(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subjects(id,code,name,icon,order_no) VALUES (5,'science','科普','🔬',5)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO modules(id,subject_id,code,name,order_no) VALUES (5,5,'observe','观察与过程',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO knowledge_points(id,module_id,code,title,order_no) VALUES (50,5,'duck','鸭子的脚掌',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO questions(id,kp_id,code,type,stem,options,answer,difficulty) VALUES (501,50,'choice','practice','选','[]','{}',1)`).Error)
	first := `{"schema":1,"kind":"choice","skillCode":"choice","responseKind":"choice","selected":"cat","example":{"kind":"choice","prompt":"哪种动物的脚掌最适合在水里游泳？","options":[{"id":"cat","label":"猫"},{"id":"duck","label":"鸭子"},{"id":"rabbit","label":"兔子"}],"answerId":"duck"}}`
	second := `{"schema":1,"kind":"choice","skillCode":"choice","responseKind":"choice","selected":"duck","example":{"kind":"choice","prompt":"哪种动物的脚掌最适合在水里游泳？","options":[{"id":"rabbit","label":"兔子"},{"id":"duck","label":"鸭子"},{"id":"cat","label":"猫"}],"answerId":"duck"}}`
	updated := `{"schema":1,"kind":"choice","skillCode":"choice","responseKind":"choice","selected":"duck","example":{"kind":"choice","prompt":"哪种动物的脚掌最适合在水里游泳？","options":[{"id":"rabbit","label":"兔子"},{"id":"duck","label":"鸭子"},{"id":"cat","label":"猫"}],"answerId":"duck","imageUrl":"/api/v1/science/task-media/ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff.png"}}`
	require.NoError(t, db.Exec(`
		INSERT INTO study_plans(id,child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count) VALUES
		 (51,1,'2026-09-10',1,'science','done',1,1,0),
		 (52,1,'2026-09-11',1,'science','done',1,1,1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO plan_items(id,plan_id,seq,kp_id,question_id,status,tries,cost_ms,picks,question_snapshot) VALUES
		 (510,51,1,50,501,'wrong',1,800,'cat',?),
		 (511,52,1,50,501,'correct',1,800,'duck',?)`, first, second).Error)
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at,plan_item_id) VALUES
		 (510,1,50,501,0,800,'quiz','sc-first','2026-09-10 10:00:00',510),
		 (511,1,50,501,1,800,'quiz','sc-second','2026-09-11 10:00:00',511)`).Error)
	require.NoError(t, db.Exec(`UPDATE plan_items SET question_snapshot=? WHERE id=511`, updated).Error)
	s := New(db)
	wrong, err := s.Attempt(1, 510)
	require.NoError(t, err)
	require.Equal(t, "instance_snapshot", wrong.QuestionFidelity)
	require.Equal(t, "cat", wrong.Response.SelectedOptionID)
	require.Equal(t, []string{"cat", "duck", "rabbit"}, scienceOptionIDs(wrong.ScienceExample.Options))
	later, err := s.Attempt(1, 511)
	require.NoError(t, err)
	require.Equal(t, "duck", later.Response.SelectedOptionID)
	require.Equal(t, []string{"rabbit", "duck", "cat"}, scienceOptionIDs(later.ScienceExample.Options))
	require.Contains(t, later.ScienceExample.ImageURL, "fff")
}

func TestScienceSamePlanItemKeepsEachAttemptSelection(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN selected TEXT NOT NULL DEFAULT ''`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subjects(id,code,name,icon,order_no) VALUES (5,'science','科普','🔬',5)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO modules(id,subject_id,code,name,order_no) VALUES (5,5,'observe','观察与过程',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO knowledge_points(id,module_id,code,title,order_no) VALUES (50,5,'organs','器官本领',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO questions(id,kp_id,code,type,stem,options,answer,difficulty) VALUES (501,50,'match','practice','连','[]','{}',1)`).Error)
	snap := `{"schema":1,"kind":"match","skillCode":"match","responseKind":"match","example":{"kind":"match","prompt":"把器官和它们的本领连在一起。","matchSources":[{"id":"ear","label":"耳朵"},{"id":"eye","label":"眼睛"},{"id":"nose","label":"鼻子"}],"matchTargets":[{"id":"see","label":"看"},{"id":"hear","label":"听"},{"id":"smell","label":"闻"}],"matchAnswers":{"eye":"see","ear":"hear","nose":"smell"}}}`
	require.NoError(t, db.Exec(`
		INSERT INTO study_plans(id,child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count) VALUES
		 (61,1,'2026-09-14',1,'science','done',1,1,1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO plan_items(id,plan_id,seq,kp_id,question_id,status,tries,cost_ms,picks,question_snapshot) VALUES (610,61,1,50,501,'correct',2,1600,'',?)`, snap).Error)
	wrongSel := `{"kind":"match","pairs":{"eye":"hear","ear":"see","nose":"smell"}}`
	rightSel := `{"kind":"match","pairs":{"eye":"see","ear":"hear","nose":"smell"}}`
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at,plan_item_id,selected) VALUES
		 (610,1,50,501,0,800,'quiz','sc-same-1','2026-09-14 10:00:00',610,?),
		 (611,1,50,501,1,800,'quiz','sc-same-2','2026-09-14 10:01:00',610,?)`, wrongSel, rightSel).Error)
	s := New(db)
	wrong, err := s.Attempt(1, 610)
	require.NoError(t, err)
	right, err := s.Attempt(1, 611)
	require.NoError(t, err)
	require.Equal(t, sciencecontent.DecodeInput(wrongSel, "match").Pairs, sciencecontent.DecodeInput(wrong.Response.SelectedOptionID, "match").Pairs)
	require.Equal(t, sciencecontent.DecodeInput(rightSel, "match").Pairs, sciencecontent.DecodeInput(right.Response.SelectedOptionID, "match").Pairs)
	require.Contains(t, strings.Join(sciencecontent.ErrorFacts(*wrong.ScienceExample, sciencecontent.DecodeInput(wrong.Response.Value, "match")), "\n"), "眼睛")
	require.Empty(t, sciencecontent.ErrorFacts(*right.ScienceExample, sciencecontent.DecodeInput(right.Response.Value, "match")))
}

func TestScienceUnlinkedAttemptCannotInventHistory(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subjects(id,code,name,icon,order_no) VALUES (5,'science','科普','🔬',5)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO modules(id,subject_id,code,name,order_no) VALUES (5,5,'observe','观察与过程',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO knowledge_points(id,module_id,code,title,order_no) VALUES (50,5,'duck','鸭子的脚掌',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO questions(id,kp_id,code,type,stem,options,answer,difficulty) VALUES (501,50,'choice','practice','选','[]','{}',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at) VALUES
		 (710,1,50,501,0,800,'quiz','sc-unlinked','2026-09-14 10:00:00')`).Error)
	s := New(db)
	got, err := s.Attempt(1, 710)
	require.NoError(t, err)
	require.Contains(t, got.EvidenceReasonCodes, "unlinked_plan_item")
	require.Nil(t, got.ScienceExample)
}

func scienceOptionIDs(opts []sciencecontent.Choice) []string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.ID)
	}
	return out
}
