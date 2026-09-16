package knowledge

import (
	"strings"
	"testing"

	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/conchi/study-learning/poemcontent"
	"github.com/stretchr/testify/require"
)

func TestPoemAttemptUsesLinkedPlanItemNotLatest(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subjects(id,code,name,icon,order_no) VALUES (6,'poem','古诗','🌸',6)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO modules(id,subject_id,code,name,order_no) VALUES (6,6,'poem50','必背古诗',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO knowledge_points(id,module_id,code,title,order_no) VALUES (60,6,'pm001','静夜思',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO questions(id,kp_id,code,type,stem,options,answer,difficulty) VALUES (601,60,'fill','practice','缺的字是哪个？','[]','{}',1)`).Error)
	first := `{"schema":1,"kind":"fill","skillCode":"fill","responseKind":"fill","selected":"char:天","example":{"kind":"fill","prompt":"缺的字是哪个？","line":"锄禾日当□","workId":"pm004","lineId":"pm004:L1","sourceLine":"锄禾日当午","gapIndexes":[4],"options":[{"id":"char:天","label":"天"},{"id":"char:午#4","label":"午"}],"answerId":"char:午#4"}}`
	second := `{"schema":1,"kind":"fill","skillCode":"fill","responseKind":"fill","selected":"char:午#4","example":{"kind":"fill","prompt":"缺的字是哪个？","line":"锄禾日当□","workId":"pm004","lineId":"pm004:L1","sourceLine":"锄禾日当午","gapIndexes":[4],"options":[{"id":"char:午#4","label":"午"},{"id":"char:天","label":"天"}],"answerId":"char:午#4"}}`
	require.NoError(t, db.Exec(`
		INSERT INTO study_plans(id,child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count) VALUES
		 (71,1,'2026-09-10',1,'poem','done',1,1,0),
		 (72,1,'2026-09-11',1,'poem','done',1,1,1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO plan_items(id,plan_id,seq,kp_id,question_id,status,tries,cost_ms,picks,question_snapshot) VALUES
		 (710,71,1,60,601,'wrong',1,800,'char:天',?),
		 (711,72,1,60,601,'correct',1,800,'char:午#4',?)`, first, second).Error)
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at,plan_item_id) VALUES
		 (710,1,60,601,0,800,'quiz','pm-first','2026-09-10 10:00:00',710),
		 (711,1,60,601,1,800,'quiz','pm-second','2026-09-11 10:00:00',711)`).Error)
	s := New(db)
	wrong, err := s.Attempt(1, 710)
	require.NoError(t, err)
	require.Equal(t, "instance_snapshot", wrong.QuestionFidelity)
	require.Equal(t, "char:天", wrong.Response.SelectedOptionID)
	require.Equal(t, []string{"char:天", "char:午#4"}, poemOptionIDs(wrong.PoemExample.Options))
	later, err := s.Attempt(1, 711)
	require.NoError(t, err)
	require.Equal(t, "char:午#4", later.Response.SelectedOptionID)
	require.Equal(t, []string{"char:午#4", "char:天"}, poemOptionIDs(later.PoemExample.Options))
}

func TestPoemSamePlanItemKeepsEachAttemptSelection(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN selected TEXT NOT NULL DEFAULT ''`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subjects(id,code,name,icon,order_no) VALUES (6,'poem','古诗','🌸',6)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO modules(id,subject_id,code,name,order_no) VALUES (6,6,'poem50','必背古诗',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO knowledge_points(id,module_id,code,title,order_no) VALUES (60,6,'pm001','静夜思',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO questions(id,kp_id,code,type,stem,options,answer,difficulty) VALUES (601,60,'recite','practice','按顺序点出这4句','[]','{}',1)`).Error)
	snap := `{"schema":1,"kind":"recite","skillCode":"recite","responseKind":"recite","example":{"kind":"recite","prompt":"按顺序点出这4句","line":"床前明月光","workId":"pm001","sequenceItems":[{"id":"pm001:L1","label":"床前明月光"},{"id":"pm001:L2","label":"疑是地上霜"},{"id":"pm001:L3","label":"举头望明月"},{"id":"pm001:L4","label":"低头思故乡"}],"sequenceDisplayOrder":["pm001:L2","pm001:L1","pm001:L4","pm001:L3"],"correctSequence":["pm001:L1","pm001:L2","pm001:L3","pm001:L4"]}}`
	require.NoError(t, db.Exec(`
		INSERT INTO study_plans(id,child_id,plan_date,seq_no,subject_code,status,target_count,done_count,correct_count) VALUES
		 (81,1,'2026-09-14',1,'poem','done',1,1,1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO plan_items(id,plan_id,seq,kp_id,question_id,status,tries,cost_ms,picks,question_snapshot) VALUES (810,81,1,60,601,'correct',2,1600,'',?)`, snap).Error)
	wrongSel := `{"kind":"recite","sequence":["pm001:L2","pm001:L1","pm001:L3","pm001:L4"]}`
	rightSel := `{"kind":"recite","sequence":["pm001:L1","pm001:L2","pm001:L3","pm001:L4"]}`
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at,plan_item_id,selected) VALUES
		 (810,1,60,601,0,800,'quiz','pm-same-1','2026-09-14 10:00:00',810,?),
		 (811,1,60,601,1,800,'quiz','pm-same-2','2026-09-14 10:01:00',810,?)`, wrongSel, rightSel).Error)
	s := New(db)
	wrong, err := s.Attempt(1, 810)
	require.NoError(t, err)
	right, err := s.Attempt(1, 811)
	require.NoError(t, err)
	require.Equal(t, poemcontent.DecodeInput(wrongSel, "recite").Sequence, poemcontent.DecodeInput(wrong.Response.SelectedOptionID, "recite").Sequence)
	require.Equal(t, poemcontent.DecodeInput(rightSel, "recite").Sequence, poemcontent.DecodeInput(right.Response.SelectedOptionID, "recite").Sequence)
	require.Contains(t, strings.Join(poemcontent.ErrorFacts(*wrong.PoemExample, poemcontent.DecodeInput(wrong.Response.Value, "recite")), "\n"), "疑是地上霜")
	require.Empty(t, poemcontent.ErrorFacts(*right.PoemExample, poemcontent.DecodeInput(right.Response.Value, "recite")))
}

func TestPoemUnlinkedAttemptCannotInventHistory(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subjects(id,code,name,icon,order_no) VALUES (6,'poem','古诗','🌸',6)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO modules(id,subject_id,code,name,order_no) VALUES (6,6,'poem50','必背古诗',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO knowledge_points(id,module_id,code,title,order_no) VALUES (60,6,'pm001','静夜思',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO questions(id,kp_id,code,type,stem,options,answer,difficulty) VALUES (601,60,'title','practice','这首诗叫什么？','[]','{}',1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at) VALUES
		 (910,1,60,601,0,800,'quiz','pm-unlinked','2026-09-14 10:00:00')`).Error)
	s := New(db)
	got, err := s.Attempt(1, 910)
	require.NoError(t, err)
	require.Contains(t, got.EvidenceReasonCodes, "unlinked_plan_item")
	require.Nil(t, got.PoemExample)
}

func poemOptionIDs(opts []poemcontent.Choice) []string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.ID)
	}
	return out
}
