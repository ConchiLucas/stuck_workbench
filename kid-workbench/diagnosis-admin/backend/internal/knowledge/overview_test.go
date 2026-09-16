package knowledge

import (
	"encoding/json"
	"github.com/conchi/study-diagnosis-admin/internal/testdb"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSummaryAbilityCountsAndHistoricalUnknown(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`UPDATE mastery_skills SET due_at='2020-01-01' WHERE skill_code='glyph_sense'; INSERT INTO knowledge_points(id,module_id,title) VALUES(11,1,'未练'); INSERT INTO subjects(id,code,name,order_no) VALUES(4,'poem','古诗',4); INSERT INTO modules(id,subject_id,code) VALUES(4,4,'poem'); INSERT INTO knowledge_points(id,module_id,title) VALUES(40,4,'诗'); INSERT INTO attempts(id,child_id,kp_id,is_correct,source,created_at) VALUES(4,1,40,0,'quiz','2026-09-01');`).Error)
	summary, err := New(db).Summary(1)
	require.NoError(t, err)
	b, err := json.Marshal(summary)
	require.NoError(t, err)
	var wire struct {
		Subjects []struct {
			Code               string
			WrongCount         int
			UnmappedPointCount int
			Abilities          map[string]int
		}
	}
	require.NoError(t, json.Unmarshal(b, &wire))
	require.Equal(t, map[string]int{"mastered": 0, "learning": 0, "shaky": 2, "unpracticed": 1, "unknown": 0}, wire.Subjects[0].Abilities)
	require.Equal(t, 1, wire.Subjects[0].WrongCount)
	require.Equal(t, map[string]int{"mastered": 0, "learning": 0, "shaky": 0, "unpracticed": 0, "unknown": 0}, wire.Subjects[2].Abilities)
	require.Equal(t, 1, wire.Subjects[2].UnmappedPointCount)
	require.NoError(t, db.Exec(`DELETE FROM mastery_skills WHERE kp_id=10; UPDATE mastery_states SET status='mastered' WHERE kp_id=10`).Error)
	summary, err = New(db).Summary(1)
	require.NoError(t, err)
	b, _ = json.Marshal(summary)
	require.NoError(t, json.Unmarshal(b, &wire))
	require.Equal(t, 3, wire.Subjects[0].Abilities["unknown"])
	require.Equal(t, 0, wire.Subjects[0].Abilities["unpracticed"])
}

func TestSummaryUnknownSkillIsNotUnpracticedAndKeepsLegacyCounts(t *testing.T) {
	db := testdb.Open(t)
	require.NoError(t, db.Exec(`UPDATE mastery_skills SET status='future_state' WHERE kp_id=20; DELETE FROM mastery_skills WHERE kp_id=10 AND skill_code='write_char'; INSERT INTO attempts(id,child_id,kp_id,question_id,is_correct,source,created_at) VALUES(4,2,10,102,0,'quiz','2026-09-01'),(5,1,10,102,0,'parent_mark','2026-09-01')`).Error)
	out, err := New(db).Summary(1)
	require.NoError(t, err)
	require.Equal(t, AbilityCounts{Mastered: 1, Unpracticed: 1, Unknown: 1}, out.Subjects[0].Abilities)
	require.Equal(t, AbilityCounts{Unpracticed: 4, Unknown: 1}, out.Subjects[1].Abilities)
	require.Equal(t, 1, out.Subjects[0].MasteredAbilityCount)
	require.Equal(t, 1, out.Subjects[0].WrongCount)
	require.Equal(t, 2, out.Subjects[1].WrongCount)
	require.Equal(t, 3, out.WrongCount)
}

func TestSummaryUnattributedPracticeDoesNotInventUnpracticedSkills(t *testing.T) {
	for _, tc := range []struct {
		name, question, overall string
		explicit                bool
		want                    AbilityCounts
	}{
		{"missing question and absent overall", "NULL", "DELETE FROM mastery_states WHERE kp_id=10", false, AbilityCounts{Unknown: 3}},
		{"unmapped question and not started", "103", "UPDATE mastery_states SET status='not_started' WHERE kp_id=10", false, AbilityCounts{Unknown: 3}},
		{"mapped practice retains known unpracticed", "102", "DELETE FROM mastery_states WHERE kp_id=10", false, AbilityCounts{Unknown: 1, Unpracticed: 2}},
		{"unmapped practice preserves explicit skill states", "NULL", "DELETE FROM mastery_states WHERE kp_id=10", true, AbilityCounts{Mastered: 1, Unpracticed: 1, Unknown: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := testdb.Open(t)
			require.NoError(t, db.Exec(`DELETE FROM mastery_skills WHERE kp_id=10; INSERT INTO questions(id,kp_id,code,type) VALUES(103,10,'old_unknown','choice')`).Error)
			require.NoError(t, db.Exec(tc.overall).Error)
			require.NoError(t, db.Exec("UPDATE attempts SET question_id="+tc.question+" WHERE id=3").Error)
			if tc.explicit {
				require.NoError(t, db.Exec(`INSERT INTO mastery_skills(child_id,kp_id,skill_code,status) VALUES(1,10,'glyph_sense','mastered'),(1,10,'sense_char','not_started')`).Error)
			}
			out, err := New(db).Summary(1)
			require.NoError(t, err)
			require.Equal(t, tc.want, out.Subjects[0].Abilities)
		})
	}
}
