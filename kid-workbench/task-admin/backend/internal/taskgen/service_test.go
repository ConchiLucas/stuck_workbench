package taskgen

import (
	"context"
	"fmt"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/conchi/study-task-admin/internal/generation"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

type fakeMaterials struct{}

func (fakeMaterials) List(context.Context, string) ([]generation.Material, error) {
	var out []generation.Material
	for i, ch := range []string{"春", "雨", "花", "草", "山", "水", "日", "月"} {
		rev := fmt.Sprintf("r%d", i)
		out = append(out, generation.Material{KpID: int64(i + 1), Text: ch, ModuleCode: "g1", ModuleName: "第一组", RevisionID: rev, SourceRevision: rev, Glyph: &generation.MediaRef{RevisionID: rev, Kind: "glyph", SHA256: rev}, Sense: &generation.MediaRef{RevisionID: rev, Kind: "sense", SHA256: rev}, Speech: &generation.MediaRef{RevisionID: rev, Kind: "speech", SHA256: rev}, Capabilities: map[string]generation.Capability{"glyph_sense": {Ready: true}, "sense_char": {Ready: true}}})
	}
	return out, nil
}
func (fakeMaterials) Freeze(_ context.Context, m []generation.Material) ([]generation.Material, error) {
	return m, nil
}
func setup(t *testing.T) *Service {
	t.Helper()
	g, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(g))
	require.NoError(t, Migrate(g))
	require.NoError(t, Migrate(g))
	return New(g, fakeMaterials{})
}
func testSpec() generation.Spec {
	return generation.Spec{SubjectCode: "literacy", Scope: generation.Scope{ModuleCodes: []string{"g1"}, KpIDs: []int64{1, 2, 3, 4}}, TargetCount: 8, TypeCounts: map[string]int{"glyph_sense": 4, "sense_char": 4}}
}
func TestGenerateIdempotentPublishAndWithdraw(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	task, err := s.Create(ctx, "春天练习", testSpec())
	require.NoError(t, err)
	require.Equal(t, int64(1), task.RowVersion)
	a, err := s.Generate(ctx, task.ID, task.RowVersion, "create-1", 0)
	require.NoError(t, err)
	require.Len(t, a.Items, 8)
	b, err := s.Generate(ctx, task.ID, task.RowVersion, "create-1", 0)
	require.NoError(t, err)
	require.Equal(t, a.ActiveRevisionID, b.ActiveRevisionID)
	_, err = s.Publish(ctx, a.ID, a.RowVersion, false)
	require.Error(t, err)
	pub, err := s.Publish(ctx, a.ID, a.RowVersion, true)
	require.NoError(t, err)
	require.Equal(t, "published", pub.Status)
	_, err = s.Generate(ctx, pub.ID, pub.RowVersion, "again", 0)
	require.Error(t, err)
	withdrawn, err := s.Transition(ctx, pub.ID, pub.RowVersion, "draft")
	require.NoError(t, err)
	require.Equal(t, "draft", withdrawn.Status)
	require.Error(t, s.Delete(ctx, withdrawn.ID))
	require.Equal(t, pub.PublishedRevisionID, withdrawn.PublishedRevisionID)
}
func TestRevisionConflictAndStableSnapshots(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	task, err := s.Create(ctx, "练习", testSpec())
	require.NoError(t, err)
	a, err := s.Generate(ctx, task.ID, 1, "generate", 0)
	require.NoError(t, err)
	_, err = s.Update(ctx, a.ID, 1, "改名", testSpec())
	require.Error(t, err)
	b, err := s.Generate(ctx, a.ID, a.RowVersion, "replace", 1)
	require.NoError(t, err)
	require.NotEqual(t, a.ActiveRevisionID, b.ActiveRevisionID)
	old, err := s.Revision(ctx, a.ID, *a.ActiveRevisionID)
	require.NoError(t, err)
	require.Equal(t, a.Items[0].Snapshot, old.Items[0].Snapshot)
	require.NotEqual(t, a.Items[0].Fingerprint, b.Items[0].Fingerprint)
	for i := 1; i < 8; i++ {
		require.Equal(t, a.Items[i].Fingerprint, b.Items[i].Fingerprint)
	}
}

func TestReviewCannotLoseItsEvidenceThroughRegularGeneration(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	spec := testSpec()
	spec.Kind = "review"
	child := int64(1)
	task, e := s.create(ctx, "复习", spec, &child, nil, nil)
	require.NoError(t, e)
	_, e = s.Generate(ctx, task.ID, 1, "regular", 0)
	require.Error(t, e)
}

type missingMedia struct{ fakeMaterials }

func (missingMedia) Verify(context.Context, []generation.Snapshot) error {
	return fmt.Errorf("media missing")
}
func TestPublishChecksFrozenMedia(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	task, e := s.Create(ctx, "练习", testSpec())
	require.NoError(t, e)
	task, e = s.Generate(ctx, task.ID, 1, "gen", 0)
	require.NoError(t, e)
	s.Materials = missingMedia{}
	_, e = s.Publish(ctx, task.ID, task.RowVersion, true)
	require.Error(t, e)
}

func TestRegenerateSearchesPastRepeatedFirstCombination(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	require.NoError(t, s.DB.Callback().Create().Before("gorm:create").Register("test:fixed_seed", func(tx *gorm.DB) {
		if run, ok := tx.Statement.Dest.(*Run); ok {
			run.Seed = 11
		}
	}))
	spec := testSpec()
	spec.Scope.KpIDs = []int64{1}
	spec.TargetCount = 1
	spec.TypeCounts = map[string]int{"glyph_sense": 1}
	task, e := s.Create(ctx, "重复采样", spec)
	require.NoError(t, e)
	first, e := s.Generate(ctx, task.ID, 1, "first", 0)
	require.NoError(t, e)
	second, e := s.Generate(ctx, task.ID, first.RowVersion, "second", 0)
	require.NoError(t, e)
	require.NotEqual(t, first.Items[0].Fingerprint, second.Items[0].Fingerprint)
	r, e := s.Revision(ctx, task.ID, *second.ActiveRevisionID)
	require.NoError(t, e)
	materials, e := s.Materials.List(ctx, "g1")
	require.NoError(t, e)
	replayed, e := generation.Generate(second.Spec, materials, r.Seed)
	require.NoError(t, e)
	require.Equal(t, second.Items[0].Fingerprint, generation.Fingerprint(replayed[0]))
}

type failingFreezeMaterials struct{ fakeMaterials }

func (failingFreezeMaterials) Freeze(context.Context, []generation.Material) ([]generation.Material, error) {
	return nil, fmt.Errorf("素材冻结失败：来源修订发生变化")
}

func TestFailedFreezePreservesExistingAndPublishedRevision(t *testing.T) {
	for _, replaceSeq := range []int{0, 1} {
		t.Run(fmt.Sprintf("replace_%d", replaceSeq), func(t *testing.T) {
			s := setup(t)
			ctx := context.Background()
			task, err := s.Create(ctx, "保持原题包", testSpec())
			require.NoError(t, err)
			task, err = s.Generate(ctx, task.ID, task.RowVersion, "first", 0)
			require.NoError(t, err)
			task, err = s.Publish(ctx, task.ID, task.RowVersion, true)
			require.NoError(t, err)
			task, err = s.Transition(ctx, task.ID, task.RowVersion, "draft")
			require.NoError(t, err)
			before, err := s.Revision(ctx, task.ID, *task.ActiveRevisionID)
			require.NoError(t, err)
			s.Materials = failingFreezeMaterials{}
			_, err = s.Generate(ctx, task.ID, task.RowVersion, "failed-freeze", replaceSeq)
			require.ErrorContains(t, err, "素材冻结失败")
			after, err := s.Get(ctx, task.ID)
			require.NoError(t, err)
			require.Equal(t, task.ActiveRevisionID, after.ActiveRevisionID)
			require.Equal(t, task.PublishedRevisionID, after.PublishedRevisionID)
			require.Equal(t, task.RowVersion, after.RowVersion)
			require.Equal(t, task.Items, after.Items)
			require.Len(t, after.Revisions, 1)
			require.Contains(t, after.LastError, "素材冻结失败")
			historical, err := s.Revision(ctx, task.ID, *task.PublishedRevisionID)
			require.NoError(t, err)
			require.Equal(t, before, historical)
			var run Run
			require.NoError(t, s.DB.Where("task_id=? AND key=?", task.ID, "failed-freeze").First(&run).Error)
			require.Equal(t, "failed", run.State)
			require.Nil(t, run.RevisionID)
		})
	}
}

type writingMaterials struct{ fakeMaterials }

func (writingMaterials) List(ctx context.Context, module string) ([]generation.Material, error) {
	ms, e := (fakeMaterials{}).List(ctx, module)
	for i := range ms {
		ms[i].Sense = nil
		ms[i].RevisionID = fmt.Sprintf("%064x", i+1)
		ms[i].Speech = &generation.MediaRef{RevisionID: ms[i].RevisionID, Kind: "speech", SHA256: fmt.Sprintf("%064x", i+101)}
		ms[i].WritingTemplate = &generation.MediaRef{RevisionID: ms[i].RevisionID, Kind: "writing_template", SHA256: fmt.Sprintf("%064x", 99)}
		ms[i].Capabilities = map[string]generation.Capability{"write_char": {Ready: true}}
	}
	return ms, e
}
func TestGenerateWritingWithoutChoiceAssets(t *testing.T) {
	s := setup(t)
	s.Materials = writingMaterials{}
	spec := testSpec()
	spec.TargetCount = 4
	spec.TypeCounts = map[string]int{"write_char": 4}
	task, e := s.Create(context.Background(), "听写", spec)
	require.NoError(t, e)
	got, e := s.Generate(context.Background(), task.ID, task.RowVersion, "write", 0)
	require.NoError(t, e)
	require.Len(t, got.Items, 4)
}
