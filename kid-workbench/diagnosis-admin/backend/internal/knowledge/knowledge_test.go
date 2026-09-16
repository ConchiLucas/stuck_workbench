package knowledge

import (
	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestFactsExcludeManualMarksAndKeepEarlierErrors(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,cost_ms,source,client_id,created_at) VALUES (4,1,20,201,1,10,'parent_mark','m','2026-09-08'),(5,1,20,201,1,10,'quiz','a5','2026-09-08'); UPDATE plan_items SET status='correct'`).Error)
	s := New(db)
	s.Now = func() time.Time { return time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC) }
	page, err := s.Attempts(1, Filter{Limit: 2, WrongOnly: true})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	require.True(t, page.HasMore)
	next, err := s.Attempts(1, Filter{Limit: 2, WrongOnly: true, Cursor: page.NextCursor})
	require.NoError(t, err)
	require.Len(t, next.Items, 1)
	summary, err := s.Summary(1)
	require.NoError(t, err)
	require.Equal(t, 4, summary.Stats.ObservedAttempts)
	require.Equal(t, 3, summary.WrongCount)
	require.Equal(t, 2, summary.PracticedCount)
	_, err = s.Attempts(2, Filter{Limit: 2, WrongOnly: true, Cursor: page.NextCursor})
	require.Error(t, err)
}

func TestModuleSkillsAndNoPracticeIsNotWeak(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`INSERT INTO subjects(id,code,name,order_no) VALUES(4,'pinyin','拼音',4),(5,'math','算数',5); INSERT INTO modules(id,subject_id,code,name,order_no) VALUES(4,4,'syllables','音节',1),(5,5,'shape','图形',1); INSERT INTO knowledge_points(id,module_id,code,title,order_no) VALUES(40,4,'ba','ba',1),(50,5,'circle','圆',1)`).Error)
	s := New(db)
	p, err := s.Point(1, 40)
	require.NoError(t, err)
	require.Len(t, p.Skills, 1)
	require.Equal(t, "blend", p.Skills[0].SkillCode)
	require.Nil(t, p.Stats.Accuracy)
	p, err = s.Point(1, 50)
	require.NoError(t, err)
	require.Len(t, p.Skills, 2)
	require.Equal(t, "find", p.Skills[0].SkillCode)
	p, err = s.Point(1, 10)
	require.NoError(t, err)
	require.Len(t, p.Skills, 3)
	require.Equal(t, "not_started", p.Skills[1].MasteryStatus)
}

func TestAnalysisRequiresIndependentInstances(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	input := []Evidence{{AttemptID: 1, KpID: 10, SkillCode: "glyph_sense", IsCorrect: false, Assistance: "none", InstanceKey: "item:1", SelectedSemanticID: "kp:20", OccurredAt: now.Add(-time.Hour)}}
	groups := Analyze(input, now)
	require.Len(t, groups, 1)
	require.Equal(t, "observed_wrong", groups[0].ReasonCode)
	input = append(input, Evidence{AttemptID: 2, KpID: 10, SkillCode: "glyph_sense", IsCorrect: false, Assistance: "none", InstanceKey: "item:1", SelectedSemanticID: "kp:20", OccurredAt: now.Add(-time.Minute)})
	groups = Analyze(input, now)
	require.Equal(t, "observed_wrong", groups[0].ReasonCode)
	input[1].InstanceKey = "item:2"
	groups = Analyze(input, now)
	require.Equal(t, "repeated_confusion", groups[0].ReasonCode)
}
