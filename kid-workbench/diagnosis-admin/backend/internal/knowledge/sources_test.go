package knowledge

import (
	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSourcePinyinAndFrozenVersion(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`CREATE TABLE pinyin_quiz_instances(id TEXT PRIMARY KEY,child_id INTEGER,kp_id INTEGER,public_snapshot TEXT,answer_option_id TEXT);CREATE TABLE pinyin_answer_receipts(child_id INTEGER,client_id TEXT,instance_id TEXT,attempt_id INTEGER,skill_code TEXT,selected_option_id TEXT); INSERT INTO subjects(id,code,name,order_no)VALUES(4,'pinyin','拼音',4);INSERT INTO modules(id,subject_id,code,name,order_no)VALUES(4,4,'syllables','音节',1);INSERT INTO knowledge_points(id,module_id,code,title,order_no)VALUES(40,4,'ba','ba',1);INSERT INTO attempts(id,child_id,kp_id,is_correct,cost_ms,source,client_id,created_at)VALUES(40,1,40,0,1000,'quiz','py1','2026-09-10 10:00:00'); INSERT INTO pinyin_quiz_instances VALUES('pyi',1,40,'{"kpId":40,"stem":"听音选音节","options":[{"id":"ba","label":"ba"},{"id":"pa","label":"pa"}]}','ba');INSERT INTO pinyin_answer_receipts VALUES(1,'py1','pyi',40,'blend','pa')`).Error)
	s := New(db)
	p, e := s.Attempt(1, 40)
	require.NoError(t, e)
	require.Equal(t, "blend", p.SkillCode)
	require.Equal(t, "pa", p.Response.SelectedOptionID)
	require.Equal(t, "instance_snapshot", p.QuestionFidelity)
	require.False(t, p.ReviewEligible)
	summary, e := s.Summary(1)
	require.NoError(t, e)
	require.Equal(t, 1, summary.Stats.IndependentAttempts)
	page, e := s.Candidates(1, Filter{Limit: 20})
	require.NoError(t, e)
	require.NotEmpty(t, page.Items)
}
func TestCursorDoesNotLoseEqualTimestampOrAcceptChangedFilter(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`UPDATE attempts SET created_at='2026-09-07 10:00:00'`).Error)
	s := New(db)
	s.Now = func() time.Time { return time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC) }
	p, e := s.Attempts(1, Filter{Limit: 1})
	require.NoError(t, e)
	seen := map[int64]bool{}
	for {
		require.Len(t, p.Items, 1)
		require.False(t, seen[p.Items[0].AttemptID])
		seen[p.Items[0].AttemptID] = true
		if !p.HasMore {
			break
		}
		p, e = s.Attempts(1, Filter{Limit: 1, Cursor: p.NextCursor})
		require.NoError(t, e)
	}
	require.Len(t, seen, 3)
}
