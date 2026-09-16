package knowledge

import (
	"encoding/json"
	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPointCategoriesContract(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`INSERT INTO knowledge_points(id,module_id,title) VALUES(11,1,'未练'); UPDATE mastery_skills SET due_at='2020-01-01' WHERE skill_code='glyph_sense'`).Error)
	s := New(db)
	summary, e := s.Summary(1)
	require.NoError(t, e)
	b, _ := json.Marshal(summary)
	var wire map[string]any
	require.NoError(t, json.Unmarshal(b, &wire))
	require.Equal(t, float64(3), wire["totalCount"])
	require.Equal(t, map[string]any{"complete": float64(0), "partial": float64(1), "weak": float64(1), "learning": float64(0), "unknown": float64(0), "unpracticed": float64(1), "reviewDue": float64(1)}, wire["pointCounts"])
	for state, want := range map[string]int{"complete": 0, "partial": 1, "weak": 1, "unpracticed": 1, "review_due": 1, "mastered": 1} {
		p, e := s.Points(1, Filter{State: state})
		require.NoError(t, e)
		require.Len(t, p.Items, want, state)
	}
	require.NoError(t, db.Exec(`UPDATE mastery_skills SET status='mastered' WHERE kp_id=10; INSERT INTO mastery_skills(child_id,kp_id,skill_code,status) VALUES(1,10,'sense_char','mastered'); DELETE FROM mastery_skills WHERE kp_id=20; UPDATE mastery_states SET status='mastered' WHERE kp_id=20`).Error)
	p, e := New(db).Points(1, Filter{State: "complete"})
	require.NoError(t, e)
	require.Len(t, p.Items, 1)
	p, e = New(db).Points(1, Filter{State: "unknown"})
	require.NoError(t, e)
	require.Len(t, p.Items, 1)
}
func TestWrongShanghaiDatesAndUnresolvedContract(t *testing.T) {
	db := testdb.Open(t)
	s := New(db)
	s.Now = func() time.Time { return time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC) }
	require.NoError(t, db.Exec(`UPDATE attempts SET created_at='2026-09-07 15:59:59' WHERE id=1; UPDATE attempts SET created_at='2026-09-07 16:00:00' WHERE id=2; UPDATE attempts SET created_at='2026-09-08 16:00:00' WHERE id=3`).Error)
	var f Filter
	require.NoError(t, json.Unmarshal([]byte(`{"From":"2026-09-08","To":"2026-09-08","WrongOnly":true,"FollowUpState":"needs_practice"}`), &f))
	p, e := s.Attempts(1, f)
	require.NoError(t, e)
	require.Len(t, p.Items, 1)
	require.Equal(t, int64(2), p.Items[0].AttemptID)
	g, e := s.Groups(1, f)
	require.NoError(t, e)
	require.Len(t, g.Items, 1)
	require.Equal(t, 1, g.Items[0].WrongCount)
}

func TestCalendarFactsZeroFillAndConservativeMastery(t *testing.T) {
	db := testdb.Open(t)
	s := New(db)
	s.Now = func() time.Time { return time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC) }
	require.NoError(t, db.Exec(`INSERT INTO subjects(id,code,name,order_no) VALUES(4,'poem','古诗',4);INSERT INTO modules(id,subject_id,code) VALUES(4,4,'poem');INSERT INTO knowledge_points(id,module_id,title) VALUES(40,4,'诗');INSERT INTO mastery_states(child_id,kp_id,status,mastered_at) VALUES(1,40,'shaky','2026-09-07 16:00:00'); UPDATE mastery_skills SET status='mastered',mastered_at='2026-09-01' WHERE kp_id=10;INSERT INTO mastery_skills(child_id,kp_id,skill_code,status,mastered_at) VALUES(1,10,'sense_char','mastered','2026-09-02');UPDATE mastery_states SET status='mastered',mastered_at='2026-09-01' WHERE kp_id=10;UPDATE attempts SET created_at='2026-09-07 16:00:00' WHERE id=1;INSERT INTO attempts(id,child_id,kp_id,is_correct,source,created_at) VALUES(4,2,20,0,'quiz','2026-09-08'),(5,1,20,0,'parent_mark','2026-09-08'),(6,1,30,0,'quiz','2026-09-08'),(7,1,40,1,'quiz','2026-09-14')`).Error)
	out, e := s.Calendar(1, "2026-09")
	require.NoError(t, e)
	require.Len(t, out.Days, 30)
	require.Len(t, out.Trend, 7)
	require.Equal(t, "2026-09-13", out.Today)
	require.Equal(t, 1, out.Days[7].Total)
	require.Equal(t, 1, out.Days[7].Mastered)
	require.Equal(t, 1, out.MasteryDateUnknownCount)
	require.Zero(t, out.Days[13].Total)
	total := 0
	for _, d := range out.Days {
		total += d.Total
	}
	require.Equal(t, 3, total)
	p, e := s.Points(1, Filter{MasteredOn: "2026-09-08"})
	require.NoError(t, e)
	require.Len(t, p.Items, 1)
	require.Equal(t, int64(40), p.Items[0].KpID)
	empty, e := s.Calendar(1, "2020-02")
	require.NoError(t, e)
	require.Len(t, empty.Days, 29)
	require.Equal(t, out.Trend, empty.Trend)
}

func TestPinyinMilestoneSurvivesRegressionAndDoesNotDoubleCount(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`UPDATE subjects SET code='pinyin' WHERE id=2;CREATE TABLE pinyin_mastery_milestones(child_id INTEGER,kp_id INTEGER,rule_version INTEGER,first_completed_at DATETIME);INSERT INTO pinyin_mastery_milestones VALUES(1,20,2,'2026-09-08 16:00:00'),(2,20,2,'2026-09-01'),(1,20,1,'2026-09-02')`).Error)
	s := New(db)
	s.Now = fixedNow
	p, e := s.Point(1, 20)
	require.NoError(t, e)
	require.NotNil(t, p.FirstMasteredAt)
	require.Equal(t, "2026-09-09", *p.FirstMasteredAt)
	c, e := s.Calendar(1, "2026-09")
	require.NoError(t, e)
	require.Equal(t, 1, c.Days[8].Mastered)
}

func TestWrongDateCursorFingerprint(t *testing.T) {
	s := New(testdb.Open(t))
	s.Now = fixedNow
	f := Filter{From: "2026-09-07", To: "2026-09-08", WrongOnly: true, Limit: 1, FollowUpState: "needs_practice"}
	p, e := s.Attempts(1, f)
	require.NoError(t, e)
	require.True(t, p.HasMore)
	f.Cursor = p.NextCursor
	f.From = "2026-09-08"
	_, e = s.Attempts(1, f)
	require.Error(t, e)
	f.From = "2026-09-07"
	p, e = s.Attempts(1, f)
	require.NoError(t, e)
	require.Len(t, p.Items, 1)
	for _, f := range []Filter{{From: "2026-02-30"}, {From: "2026-09-09", To: "2026-09-08"}, {MasteredOn: "2026-9-08"}, {Limit: 101}} {
		_, e = s.Attempts(1, f)
		require.Error(t, e)
	}
}

func TestPointDateRevisionInvalidatesCursor(t *testing.T) {
	db := testdb.Open(t)
	s := New(db)
	s.Now = fixedNow
	p, e := s.Points(1, Filter{State: "all", Limit: 1})
	require.NoError(t, e)
	require.True(t, p.HasMore)
	require.NoError(t, db.Exec(`CREATE TABLE pinyin_mastery_milestones(child_id INTEGER,kp_id INTEGER,rule_version INTEGER,first_completed_at DATETIME);UPDATE subjects SET code='pinyin' WHERE id=2;INSERT INTO pinyin_mastery_milestones VALUES(1,20,2,'2026-09-08')`).Error)
	// Establish a cursor after the directory revision changed, then change only its date.
	p, e = s.Points(1, Filter{State: "all", Limit: 1})
	require.NoError(t, e)
	require.NoError(t, db.Exec(`UPDATE pinyin_mastery_milestones SET first_completed_at='2026-09-09'`).Error)
	_, e = s.Points(1, Filter{State: "all", Limit: 1, Cursor: p.NextCursor})
	require.Error(t, e)
}
