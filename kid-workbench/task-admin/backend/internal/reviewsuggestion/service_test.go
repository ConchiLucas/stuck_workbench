package reviewsuggestion

import (
	"context"
	"fmt"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) *Service {
	t.Helper()
	g, e := db.OpenSQLite(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	require.NoError(t, e)
	return fixtureDB(t, g)
}
func fixtureDB(t *testing.T, g *gorm.DB) *Service {
	t.Helper()
	for _, sql := range []string{
		`CREATE TABLE children(id INTEGER PRIMARY KEY)`, `INSERT INTO children VALUES(7),(8)`,
		`CREATE TABLE subjects(id INTEGER PRIMARY KEY,code TEXT)`,
		`CREATE TABLE modules(id INTEGER PRIMARY KEY,subject_id INTEGER,code TEXT)`,
		`CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY,module_id INTEGER)`,
		`CREATE TABLE questions(id INTEGER PRIMARY KEY,kp_id INTEGER,type TEXT,code TEXT)`,
		`CREATE TABLE attempts(id INTEGER PRIMARY KEY,child_id INTEGER,kp_id INTEGER,question_id INTEGER,is_correct BOOLEAN,source TEXT,client_id TEXT,created_at DATETIME)`,
		`INSERT INTO subjects VALUES(1,'literacy')`, `INSERT INTO modules VALUES(1,1,'g1')`,
		`INSERT INTO knowledge_points VALUES(103,1)`, `INSERT INTO questions VALUES(1,103,'choice','glyph_sense')`,
		`INSERT INTO attempts VALUES(1,7,103,1,false,'quiz','c1','2026-09-12 01:00:00')`,
	} {
		if g.Dialector.Name() == "postgres" {
			sql = strings.ReplaceAll(sql, "DATETIME", "TIMESTAMPTZ")
		}
		require.NoError(t, g.Exec(sql).Error)
	}
	require.NoError(t, Migrate(g))
	return New(g, nil, true)
}
func input() Input {
	return Input{SchemaVersion: 1, Title: "复习", AnalysisVersion: "knowledge-analysis-v1", AnalysisAsOf: time.Date(2026, 9, 12, 2, 0, 0, 0, time.UTC), SubjectCode: "literacy", Targets: []TargetInput{{Key: "103:glyph_sense", KpID: 103, QuestionType: "glyph_sense", ReasonCode: "observed_wrong", Mode: "mixed", RequestedCount: 2, Evidence: []EvidenceInput{{AttemptID: 1, Role: "target_error", Source: Source{Kind: "legacy_attempt"}}}}}}
}
func TestSaveImmutableAndReplayConflict(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	in := input()
	a, replay, e := s.Save(ctx, 7, "key", in)
	require.NoError(t, e)
	require.False(t, replay)
	require.Nil(t, a.GenerationStatus)
	require.EqualValues(t, 1, a.RowVersion)
	b, replay, e := s.Save(ctx, 7, "key", in)
	require.NoError(t, e)
	require.True(t, replay)
	require.Equal(t, a.ID, b.ID)
	b, replay, e = s.Save(ctx, 7, "other-key", in)
	require.NoError(t, e)
	require.True(t, replay)
	require.Equal(t, a.ID, b.ID)
	in.Targets[0].RequestedCount = 3
	_, _, e = s.Save(ctx, 7, "key", in)
	require.ErrorContains(t, e, "idempotency")
	var n int64
	require.NoError(t, s.DB.Table("review_suggestions").Count(&n).Error)
	require.EqualValues(t, 1, n)
}
func TestRejectEvidenceOwnershipSourceAndRole(t *testing.T) {
	for _, test := range []string{"child", "source", "correct", "key", "count", "reason"} {
		t.Run(test, func(t *testing.T) {
			s := fixture(t)
			in := input()
			child := int64(7)
			switch test {
			case "child":
				child = 8
			case "source":
				require.NoError(t, s.DB.Exec("UPDATE attempts SET source='parent_mark'").Error)
			case "correct":
				require.NoError(t, s.DB.Exec("UPDATE attempts SET is_correct=true").Error)
			case "key":
				in.Targets[0].Key = "103:audio_glyph"
			case "count":
				in.Targets[0].RequestedCount = 11
			case "reason":
				in.Targets[0].ReasonCode = "repeated_skill_error"
			}
			_, _, e := s.Save(context.Background(), child, "key", in)
			require.Error(t, e)
			var n int64
			require.NoError(t, s.DB.Table("review_suggestions").Count(&n).Error)
			require.Zero(t, n)
		})
	}
}
func TestCommandsVersionReplayAndUnsupportedGeneration(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	a, _, e := s.Save(ctx, 7, "save", input())
	require.NoError(t, e)
	_, _, e = s.Command(ctx, 7, a.ID, "generate", "gen", CommandInput{ExpectedRowVersion: 99})
	require.Error(t, e)
	b, replay, e := s.Command(ctx, 7, a.ID, "generate", "gen", CommandInput{ExpectedRowVersion: 1})
	require.NoError(t, e)
	require.False(t, replay)
	require.Equal(t, "queued", *b.GenerationStatus)
	c, replay, e := s.Command(ctx, 7, a.ID, "generate", "gen", CommandInput{ExpectedRowVersion: 1})
	require.NoError(t, e)
	require.True(t, replay)
	require.Equal(t, b.RowVersion, c.RowVersion)
	require.NoError(t, s.Tick(ctx))
	c, e = s.Get(ctx, 7, a.ID)
	require.NoError(t, e)
	require.Equal(t, "blocked", *c.GenerationStatus)
	c, _, e = s.Command(ctx, 7, a.ID, "archive", "archive", CommandInput{ExpectedRowVersion: c.RowVersion})
	require.NoError(t, e)
	require.Equal(t, "archived", c.Lifecycle)
}
func TestDisabledCommandsDoNotWrite(t *testing.T) {
	s := fixture(t)
	s.Enabled = false
	_, _, e := s.Save(context.Background(), 7, "key", input())
	require.Error(t, e)
	var n int64
	s.DB.Table("review_suggestions").Count(&n)
	require.Zero(t, n)
}

func TestOpenContentReuseIgnoresDisplayTitleAndAnalysisTime(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	in := input()
	a, _, e := s.Save(ctx, 7, "a", in)
	require.NoError(t, e)
	in.Title = "另一个标题"
	in.AnalysisAsOf = in.AnalysisAsOf.Add(time.Minute)
	b, replay, e := s.Save(ctx, 7, "b", in)
	require.NoError(t, e)
	require.True(t, replay)
	require.Equal(t, a.ID, b.ID)
	require.Equal(t, a.Title, b.Title)
}
func TestPinyinRepeatedReasonUsesCanonicalInstances(t *testing.T) {
	s := fixture(t)
	g := s.DB
	for _, sql := range []string{`UPDATE subjects SET code='pinyin'`, `CREATE TABLE pinyin_quiz_instances(id TEXT PRIMARY KEY,child_id INTEGER,kp_id INTEGER,skill_code TEXT,public_snapshot TEXT,answer_option_id TEXT)`, `CREATE TABLE pinyin_answer_receipts(child_id INTEGER,client_id TEXT,instance_id TEXT,attempt_id INTEGER,skill_code TEXT,selected_option_id TEXT)`, `INSERT INTO attempts VALUES(2,7,103,NULL,false,'quiz','c2','2026-09-12 01:01:00')`, `INSERT INTO attempts VALUES(3,7,103,NULL,true,'quiz','c3','2026-09-12 01:02:00')`} {
		require.NoError(t, g.Exec(sql).Error)
	}
	in := input()
	in.SubjectCode = "pinyin"
	target := &in.Targets[0]
	target.Key = "103:listen"
	target.QuestionType = "listen"
	target.ReasonCode = "repeated_skill_error"
	target.Evidence = nil
	for i := 1; i <= 3; i++ {
		instance := fmt.Sprintf("i%d", i)
		client := fmt.Sprintf("c%d", i)
		require.NoError(t, g.Exec(`INSERT INTO pinyin_quiz_instances VALUES(?,7,103,'listen','{"kpId":103,"options":[{"id":"option-1","label":"a"},{"id":"option-2","label":"b"}]}','option-2')`, instance).Error)
		require.NoError(t, g.Exec(`INSERT INTO pinyin_answer_receipts VALUES(7,?,?,?,'listen','option-1')`, client, instance, i).Error)
		role := "target_error"
		if i == 3 {
			role = "target_observation"
			require.NoError(t, g.Exec("UPDATE pinyin_answer_receipts SET selected_option_id='option-2' WHERE attempt_id=?", i).Error)
		}
		target.Evidence = append(target.Evidence, EvidenceInput{AttemptID: int64(i), Role: role, Source: Source{Kind: "pinyin_instance", ClientID: client, InstanceID: instance}})
	}
	a, _, e := s.Save(context.Background(), 7, "save", in)
	require.NoError(t, e)
	require.Contains(t, a.Targets[0].ReasonText, "3")
	require.NoError(t, g.Exec("UPDATE pinyin_quiz_instances SET public_snapshot='broken'").Error)
	_, _, e = s.Save(context.Background(), 7, "invalid-snapshots-strong", in)
	require.Error(t, e, "unknown assistance cannot support repeated-skill reasoning")
	target.ReasonCode = "observed_wrong"
	_, _, e = s.Save(context.Background(), 7, "invalid-snapshots-observations", in)
	require.NoError(t, e, "actual errors remain saveable as observations")
	target.ReasonCode = "repeated_skill_error"
	target.Evidence = target.Evidence[:2]
	_, _, e = s.Save(context.Background(), 7, "missing-observation", in)
	require.Error(t, e)
}
