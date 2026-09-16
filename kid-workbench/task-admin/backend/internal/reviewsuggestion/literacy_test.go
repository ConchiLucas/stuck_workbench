package reviewsuggestion

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/conchi/study-task-admin/internal/generation"
	"github.com/conchi/study-task-admin/internal/taskgen"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
	"time"
)

type testMaterials struct {
	freezeCount int
	cancel      func()
}

func (m *testMaterials) List(ctx context.Context, module string) ([]generation.Material, error) {
	out := []generation.Material{}
	for i := 0; i < 8; i++ {
		rev := fmt.Sprintf("r%d", i)
		out = append(out, generation.Material{KpID: int64(103 + i), Text: fmt.Sprintf("字%d", i), ModuleCode: module, RevisionID: rev, SourceRevision: rev, Glyph: &generation.MediaRef{RevisionID: rev, Kind: "glyph", SHA256: rev}, Sense: &generation.MediaRef{RevisionID: rev, Kind: "sense", SHA256: rev}, Speech: &generation.MediaRef{RevisionID: rev, Kind: "speech", SHA256: rev}, Capabilities: map[string]generation.Capability{"glyph_sense": {Ready: true}}})
	}
	return out, nil
}
func (m *testMaterials) Freeze(ctx context.Context, rows []generation.Material) ([]generation.Material, error) {
	m.freezeCount++
	if m.cancel != nil {
		m.cancel()
	}
	return rows, nil
}
func literacyFixture(t *testing.T) (*Service, *testMaterials, Input) {
	t.Helper()
	return literacyFixtureService(t, fixture(t))
}
func literacyFixtureService(t *testing.T, s *Service) (*Service, *testMaterials, Input) {
	t.Helper()
	g := s.DB
	require.NoError(t, db.Migrate(g))
	require.NoError(t, taskgen.Migrate(g))
	m := &testMaterials{}
	s.Generation = taskgen.New(g, m)
	for _, sql := range []string{
		`CREATE TABLE study_plans(id INTEGER PRIMARY KEY,child_id INTEGER,subject_code TEXT,status TEXT,source_question_task_id INTEGER,source_question_task_revision_id INTEGER)`,
		`CREATE TABLE plan_items(id INTEGER PRIMARY KEY,plan_id INTEGER,kp_id INTEGER,question_version_id INTEGER)`,
		`CREATE TABLE question_attempt_receipts(id INTEGER PRIMARY KEY,attempt_id INTEGER,child_id INTEGER,plan_id INTEGER,plan_item_id INTEGER,question_version_id INTEGER,kp_id INTEGER,question_type TEXT,selected_option_id TEXT,evaluation_json TEXT,is_correct BOOLEAN)`,
		`INSERT INTO modules VALUES(2,1,'g2')`, `INSERT INTO knowledge_points VALUES(104,2)`,
		`INSERT INTO attempts VALUES(2,7,104,1,false,'quiz','c2','2026-09-12 01:01:00')`,
	} {
		require.NoError(t, g.Exec(sql).Error)
	}
	in := input()
	in.Targets = nil
	for i := 0; i < 2; i++ {
		id := int64(i + 1)
		kp := int64(103 + i)
		module := fmt.Sprintf("g%d", i+1)
		rows, _ := m.List(context.Background(), module)
		q, e := generation.Build(rows[i], "glyph_sense", rows, rand.New(rand.NewSource(id)))
		require.NoError(t, e)
		wrongOption := ""
		for _, o := range q.Options {
			if o.ID != q.AnswerOptionID {
				wrongOption = o.ID
				break
			}
		}
		b, _ := json.Marshal(q)
		rev := taskgen.Revision{TaskID: 100 + id, RevisionNo: 1, SpecJSON: "{}"}
		require.NoError(t, g.Create(&rev).Error)
		v := taskgen.QuestionVersion{RevisionID: rev.ID, Seq: 1, KpID: kp, QuestionType: "glyph_sense", SnapshotJSON: string(b), Fingerprint: generation.Fingerprint(q)}
		require.NoError(t, g.Create(&v).Error)
		require.NoError(t, g.Exec("INSERT INTO study_plans VALUES(?,7,'literacy','done',?,?)", id, 100+id, rev.ID).Error)
		require.NoError(t, g.Exec("INSERT INTO plan_items VALUES(?,?,?,?)", id, id, kp, v.ID).Error)
		require.NoError(t, g.Exec("INSERT INTO question_attempt_receipts VALUES(?,?,7,?,?,?,?, 'glyph_sense',?,'{}',false)", id, id, id, id, v.ID, kp, wrongOption).Error)
		in.Targets = append(in.Targets, TargetInput{Key: fmt.Sprintf("%d:glyph_sense", kp), KpID: kp, QuestionType: "glyph_sense", ReasonCode: "observed_wrong", Mode: "original_only", RequestedCount: 1, Evidence: []EvidenceInput{{AttemptID: id, Role: "target_error", Source: Source{Kind: "literacy_version", ReceiptID: id, QuestionVersionID: v.ID}}}})
	}
	return s, m, in
}
func TestCrossPlanPartitionsCommitRevisionAndMapping(t *testing.T) {
	s, m, in := literacyFixture(t)
	ctx := context.Background()
	a, _, e := s.Save(ctx, 7, "save", in)
	require.NoError(t, e)
	require.Zero(t, m.freezeCount)
	require.Empty(t, a.Partitions)
	a, _, e = s.Command(ctx, 7, a.ID, "generate", "go", CommandInput{ExpectedRowVersion: 1})
	require.NoError(t, e)
	require.Len(t, a.Partitions, 2)
	require.NoError(t, s.Tick(ctx))
	a, e = s.Get(ctx, 7, a.ID)
	require.NoError(t, e)
	require.Equal(t, 1, a.TaskCount)
	require.NoError(t, s.Tick(ctx))
	a, e = s.Get(ctx, 7, a.ID)
	require.NoError(t, e)
	require.Equal(t, "succeeded", *a.GenerationStatus)
	require.Equal(t, 2, a.GeneratedCount)
	links, e := s.Tasks(ctx, 7, a.ID)
	require.NoError(t, e)
	require.Len(t, links, 2)
	for _, link := range links {
		task, e := s.Generation.Get(ctx, link.TaskID)
		require.NoError(t, e)
		require.Nil(t, task.ParentTaskID)
		require.Equal(t, *task.ActiveRevisionID, link.GeneratedRevisionID)
		require.Contains(t, string(link.TargetMap), "original")
	}
	require.NoError(t, s.Tick(ctx))
	again, e := s.Tasks(ctx, 7, a.ID)
	require.NoError(t, e)
	require.Equal(t, links, again)
	_, _, e = s.Command(ctx, 7, a.ID, "retry", "retry", CommandInput{ExpectedRowVersion: a.RowVersion, PartitionKeys: []string{a.Partitions[0].PartitionKey}})
	require.ErrorContains(t, e, "successful")
}
func TestIncompleteSourceBlocksOnlyItsPartition(t *testing.T) {
	s, _, in := literacyFixture(t)
	require.NoError(t, s.DB.Exec("UPDATE study_plans SET status='active' WHERE id=2").Error)
	ctx := context.Background()
	a, _, e := s.Save(ctx, 7, "save", in)
	require.NoError(t, e)
	a, _, e = s.Command(ctx, 7, a.ID, "generate", "go", CommandInput{ExpectedRowVersion: 1})
	require.NoError(t, e)
	require.NoError(t, s.Tick(ctx))
	require.NoError(t, s.Tick(ctx))
	a, e = s.Get(ctx, 7, a.ID)
	require.NoError(t, e)
	require.Equal(t, "partial", *a.GenerationStatus)
	require.Equal(t, 1, a.TaskCount)
	require.Contains(t, string(a.Partitions[1].Error), "source_plan_incomplete")
	require.NoError(t, s.DB.Exec("UPDATE study_plans SET status='done' WHERE id=2").Error)
	first := a.Partitions[0].TaskID
	a, _, e = s.Command(ctx, 7, a.ID, "retry", "retry", CommandInput{ExpectedRowVersion: a.RowVersion, PartitionKeys: []string{a.Partitions[1].PartitionKey}})
	require.NoError(t, e)
	require.NoError(t, s.Tick(ctx))
	a, e = s.Get(ctx, 7, a.ID)
	require.NoError(t, e)
	require.Equal(t, "succeeded", *a.GenerationStatus)
	require.Equal(t, first, a.Partitions[0].TaskID)
}
func TestCancellationDuringFreezePreventsTaskCommit(t *testing.T) {
	s, m, in := literacyFixture(t)
	in.Targets = in.Targets[:1]
	in.Targets[0].Mode = "mixed"
	in.Targets[0].RequestedCount = 2
	ctx := context.Background()
	a, _, e := s.Save(ctx, 7, "save", in)
	require.NoError(t, e)
	a, _, e = s.Command(ctx, 7, a.ID, "generate", "go", CommandInput{ExpectedRowVersion: 1})
	require.NoError(t, e)
	m.cancel = func() {
		_, _, e := s.Command(ctx, 7, a.ID, "cancel", "cancel", CommandInput{ExpectedRowVersion: a.RowVersion})
		require.NoError(t, e)
	}
	require.NoError(t, s.Tick(ctx))
	a, e = s.Get(ctx, 7, a.ID)
	require.NoError(t, e)
	require.Equal(t, "cancelled", *a.GenerationStatus)
	require.Zero(t, a.TaskCount)
}
func TestMixedGeneratesExactOriginalVariantRatio(t *testing.T) {
	s, m, in := literacyFixture(t)
	in.Targets = in.Targets[:1]
	in.Targets[0].Mode = "mixed"
	in.Targets[0].RequestedCount = 3
	ctx := context.Background()
	a, _, e := s.Save(ctx, 7, "save", in)
	require.NoError(t, e)
	require.Zero(t, m.freezeCount)
	_, _, e = s.Command(ctx, 7, a.ID, "generate", "go", CommandInput{ExpectedRowVersion: 1})
	require.NoError(t, e)
	require.NoError(t, s.Tick(ctx))
	a, e = s.Get(ctx, 7, a.ID)
	require.NoError(t, e)
	require.Equal(t, "succeeded", *a.GenerationStatus)
	require.Equal(t, 3, a.GeneratedCount)
	links, e := s.Tasks(ctx, 7, a.ID)
	require.NoError(t, e)
	var mapping []taskgen.EvidenceTargetMap
	require.NoError(t, json.Unmarshal(links[0].TargetMap, &mapping))
	require.Equal(t, "original", mapping[0].Kind)
	require.Equal(t, "variant", mapping[1].Kind)
	require.Equal(t, "variant", mapping[2].Kind)
}
func TestEvidenceAndTaskPagesUseStableCursor(t *testing.T) {
	s, _, in := literacyFixture(t)
	ctx := context.Background()
	a, _, e := s.Save(ctx, 7, "save", in)
	require.NoError(t, e)
	page, e := s.EvidencePage(ctx, 7, a.ID, 0, 1)
	require.NoError(t, e)
	require.True(t, page.HasMore)
	require.Len(t, page.Items.([]Evidence), 1)
	cursor := page.Items.([]Evidence)[0].ID
	next, e := s.EvidencePage(ctx, 7, a.ID, cursor, 1)
	require.NoError(t, e)
	require.False(t, next.HasMore)
	require.Greater(t, next.Items.([]Evidence)[0].ID, cursor)
}
func TestRejectForgedPlanTaskOwnerAndUnknownType(t *testing.T) {
	for _, kind := range []string{"task_owner", "type"} {
		t.Run(kind, func(t *testing.T) {
			s, _, in := literacyFixture(t)
			if kind == "task_owner" {
				require.NoError(t, s.DB.Exec("UPDATE study_plans SET source_question_task_id=999 WHERE id=1").Error)
			} else {
				in.Targets[0].QuestionType = "invented"
				in.Targets[0].Key = "103:invented"
				require.NoError(t, s.DB.Exec("UPDATE question_attempt_receipts SET question_type='invented' WHERE id=1").Error)
				require.NoError(t, s.DB.Exec("UPDATE question_versions SET question_type='invented' WHERE id=1").Error)
			}
			_, _, e := s.Save(context.Background(), 7, "key", in)
			require.Error(t, e)
		})
	}
}
func TestReceiptSelectionMustExistInFrozenSnapshot(t *testing.T) {
	s, _, in := literacyFixture(t)
	require.NoError(t, s.DB.Exec("UPDATE question_attempt_receipts SET selected_option_id='kp:99999' WHERE id=1").Error)
	_, _, e := s.Save(context.Background(), 7, "bad", in)
	require.Error(t, e)
}
func TestTransientGenerationStopsAfterThreeAttempts(t *testing.T) {
	s, _, in := literacyFixture(t)
	s.Generation = nil
	ctx := context.Background()
	a, _, e := s.Save(ctx, 7, "save", in)
	require.NoError(t, e)
	_, _, e = s.Command(ctx, 7, a.ID, "generate", "go", CommandInput{ExpectedRowVersion: 1})
	require.NoError(t, e)
	for i := 0; i < 6; i++ {
		require.NoError(t, s.DB.Model(&Partition{}).Where("state='queued'").Update("next_run_at", time.Now().UTC().Add(-time.Minute)).Error)
		require.NoError(t, s.Tick(ctx))
	}
	a, e = s.Get(ctx, 7, a.ID)
	require.NoError(t, e)
	require.Equal(t, "failed", *a.GenerationStatus)
	for _, p := range a.Partitions {
		require.Equal(t, 3, p.AttemptCount)
	}
	require.Zero(t, a.TaskCount)
}
func TestLeaseRenewalRequiresSameActiveOwner(t *testing.T) {
	s, _, in := literacyFixture(t)
	ctx := context.Background()
	a, _, e := s.Save(ctx, 7, "save", in)
	require.NoError(t, e)
	a, _, e = s.Command(ctx, 7, a.ID, "generate", "go", CommandInput{ExpectedRowVersion: 1})
	require.NoError(t, e)
	p := a.Partitions[0]
	require.NoError(t, s.DB.Model(&Partition{}).Where("id=?", p.ID).Updates(map[string]any{"state": "running", "lease_owner": "owner", "lease_until": time.Now().UTC().Add(time.Second)}).Error)
	ok, e := s.renewLease(ctx, p.ID, "other")
	require.NoError(t, e)
	require.False(t, ok)
	ok, e = s.renewLease(ctx, p.ID, "owner")
	require.NoError(t, e)
	require.True(t, ok)
	_, _, e = s.Command(ctx, 7, a.ID, "cancel", "cancel", CommandInput{ExpectedRowVersion: a.RowVersion})
	require.NoError(t, e)
	ok, e = s.renewLease(ctx, p.ID, "owner")
	require.NoError(t, e)
	require.False(t, ok)
}
func TestTaskMappingFailureRollsBackAndExhaustsRetries(t *testing.T) {
	s, _, in := literacyFixture(t)
	in.Targets = in.Targets[:1]
	require.NoError(t, s.DB.Exec(`CREATE TRIGGER fail_review_link BEFORE INSERT ON review_suggestion_tasks BEGIN SELECT RAISE(ABORT,'simulated mapping failure'); END`).Error)
	ctx := context.Background()
	a, _, e := s.Save(ctx, 7, "save", in)
	require.NoError(t, e)
	_, _, e = s.Command(ctx, 7, a.ID, "generate", "go", CommandInput{ExpectedRowVersion: 1})
	require.NoError(t, e)
	for i := 0; i < 3; i++ {
		require.NoError(t, s.DB.Model(&Partition{}).Where("state='queued'").Update("next_run_at", time.Now().UTC().Add(-time.Minute)).Error)
		require.NoError(t, s.Tick(ctx))
	}
	a, e = s.Get(ctx, 7, a.ID)
	require.NoError(t, e)
	require.Equal(t, "failed", *a.GenerationStatus)
	var n int64
	require.NoError(t, s.DB.Table("question_tasks").Count(&n).Error)
	require.Zero(t, n)
	require.Zero(t, a.TaskCount)
}
func TestTaskPagesExposeCurrentStatusWithoutChangingGeneratedRevision(t *testing.T) {
	s, _, in := literacyFixture(t)
	in.Targets = in.Targets[:1]
	ctx := context.Background()
	a, _, e := s.Save(ctx, 7, "save", in)
	require.NoError(t, e)
	_, _, e = s.Command(ctx, 7, a.ID, "generate", "go", CommandInput{ExpectedRowVersion: 1})
	require.NoError(t, e)
	require.NoError(t, s.Tick(ctx))
	links, e := s.Tasks(ctx, 7, a.ID)
	require.NoError(t, e)
	require.NoError(t, s.DB.Model(&taskgen.Task{}).Where("id=?", links[0].TaskID).Updates(map[string]any{"status": "published", "active_revision_id": 999}).Error)
	page, e := s.TasksPage(ctx, 7, a.ID, 0, 20)
	require.NoError(t, e)
	link := page.Items.([]TaskLink)[0]
	require.Equal(t, "published", link.TaskStatus)
	require.EqualValues(t, 999, *link.ActiveRevisionID)
	require.Equal(t, links[0].GeneratedRevisionID, link.GeneratedRevisionID)
}
func TestCancellationRetainsRunResultSnapshot(t *testing.T) {
	s, _, in := literacyFixture(t)
	ctx := context.Background()
	a, _, e := s.Save(ctx, 7, "save", in)
	require.NoError(t, e)
	a, _, e = s.Command(ctx, 7, a.ID, "generate", "go", CommandInput{ExpectedRowVersion: 1})
	require.NoError(t, e)
	_, _, e = s.Command(ctx, 7, a.ID, "cancel", "cancel", CommandInput{ExpectedRowVersion: a.RowVersion})
	require.NoError(t, e)
	var run Run
	require.NoError(t, s.DB.Where("suggestion_id=?", a.ID).Take(&run).Error)
	require.Contains(t, run.ResultJSON, "cancelled")
}
