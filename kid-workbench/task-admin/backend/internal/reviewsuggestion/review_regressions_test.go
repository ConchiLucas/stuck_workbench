package reviewsuggestion

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/conchi/study-task-admin/internal/generation"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
	"time"
)

func TestConfusionCannotUseOutsideWindowEvidence(t *testing.T) {
	for _, old := range []bool{true, false} {
		t.Run(fmt.Sprint(old), func(t *testing.T) {
			s := fixture(t)
			g := s.DB
			require.NoError(t, g.Exec(`UPDATE subjects SET code='pinyin'; CREATE TABLE pinyin_quiz_instances(id TEXT PRIMARY KEY,child_id INTEGER,kp_id INTEGER,skill_code TEXT,public_snapshot TEXT,answer_option_id TEXT); CREATE TABLE pinyin_answer_receipts(child_id INTEGER,client_id TEXT,instance_id TEXT,attempt_id INTEGER,skill_code TEXT,selected_option_id TEXT)`).Error)
			in := input()
			in.SubjectCode = "pinyin"
			target := &in.Targets[0]
			target.Key = "103:listen"
			target.QuestionType = "listen"
			target.ReasonCode = "repeated_confusion"
			target.Evidence = nil
			n := 22
			if old {
				n = 2
			}
			for i := 1; i <= n; i++ {
				at := in.AnalysisAsOf.Add(time.Duration(i-n-1) * time.Minute)
				if old {
					at = in.AnalysisAsOf.AddDate(0, 0, -31)
				}
				correct := i > 2
				client := fmt.Sprintf("c%d", i)
				if i == 1 {
					require.NoError(t, g.Exec("UPDATE attempts SET created_at=?,is_correct=? WHERE id=1", at, correct).Error)
				} else {
					require.NoError(t, g.Exec("INSERT INTO attempts VALUES(?,7,103,NULL,?,'quiz',?,?)", i, correct, client, at).Error)
				}
				instance := fmt.Sprintf("i%d", i)
				require.NoError(t, g.Exec(`INSERT INTO pinyin_quiz_instances VALUES(?,7,103,'listen','{"kpId":103,"options":[{"id":"option-1","label":"a"},{"id":"option-2","label":"b"}]}','option-2')`, instance).Error)
				require.NoError(t, g.Exec("INSERT INTO pinyin_answer_receipts VALUES(7,?,?,?,'listen','option-1')", client, instance, i).Error)
				role := "target_error"
				if correct {
					role = "target_observation"
					require.NoError(t, g.Exec("UPDATE pinyin_answer_receipts SET selected_option_id='option-2' WHERE attempt_id=?", i).Error)
				}
				target.Evidence = append(target.Evidence, EvidenceInput{AttemptID: int64(i), Role: role, Source: Source{Kind: "pinyin_instance", InstanceID: instance, ClientID: client}})
			}
			_, _, e := s.Save(context.Background(), 7, "outside", in)
			require.Error(t, e)
		})
	}
}
func TestPracticedSkillGapRequiresMasteredIndependentStrength(t *testing.T) {
	s, m, in := literacyFixture(t)
	g := s.DB
	rows, _ := m.List(context.Background(), "g2")
	q, e := generation.Build(rows[0], "sense_char", rows, rand.New(rand.NewSource(2)))
	require.NoError(t, e)
	b, _ := json.Marshal(q)
	require.NoError(t, g.Exec("UPDATE question_versions SET kp_id=103,question_type='sense_char',snapshot_json=? WHERE id=2", string(b)).Error)
	require.NoError(t, g.Exec("UPDATE attempts SET kp_id=103,is_correct=true WHERE id=2").Error)
	require.NoError(t, g.Exec("UPDATE plan_items SET kp_id=103 WHERE id=2").Error)
	require.NoError(t, g.Exec("UPDATE question_attempt_receipts SET kp_id=103,is_correct=true,question_type='sense_char',selected_option_id=? WHERE id=2", q.AnswerOptionID).Error)
	require.NoError(t, g.Exec(`CREATE TABLE mastery_skills(child_id INTEGER,kp_id INTEGER,skill_code TEXT,status TEXT,attempts INTEGER); INSERT INTO mastery_skills VALUES(7,103,'sense_char','mastered',5),(7,103,'glyph_sense','shaky',2)`).Error)
	strength := in.Targets[1].Evidence[0]
	strength.Role = "supporting_strength"
	in.Targets = in.Targets[:1]
	in.Targets[0].ReasonCode = "practiced_skill_gap"
	in.Targets[0].Evidence = append(in.Targets[0].Evidence, strength)
	a, _, e := s.Save(context.Background(), 7, "gap", in)
	require.NoError(t, e)
	require.Contains(t, a.Targets[0].ReasonText, "sense_char")
	require.NoError(t, g.Exec("UPDATE mastery_skills SET status='learning' WHERE skill_code='sense_char'").Error)
	_, _, e = s.Save(context.Background(), 7, "not-mastered", in)
	require.Error(t, e)
}
func TestHandwritingAssistanceUsesPayloadAndPreservesUnknown(t *testing.T) {
	for _, kind := range []string{"payload_hint", "missing", "invalid", "result_conflict"} {
		t.Run(kind, func(t *testing.T) {
			s, _, in := literacyFixture(t)
			g := s.DB
			require.NoError(t, g.Exec("ALTER TABLE question_attempt_receipts ADD COLUMN answer_payload_json TEXT").Error)
			h := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			q := generation.Snapshot{SchemaVersion: 2, SubjectCode: "literacy", KpID: 103, TargetText: "一", QuestionType: "write_char", SkillCode: "write_char", Interaction: "handwriting", ResponseSchemaVersion: 2, EvaluationPolicyVersion: "ink-match-v1", WritingTemplate: &generation.MediaRef{RevisionID: h, Kind: "writing_template", SHA256: h}, Stem: generation.Stem{Audio: &generation.MediaRef{RevisionID: h, Kind: "speech", SHA256: h}}, MaterialRevisionIDs: []string{h}}
			b, _ := json.Marshal(q)
			require.NoError(t, g.Exec("UPDATE question_versions SET question_type='write_char',snapshot_json=? WHERE id=1", string(b)).Error)
			require.NoError(t, g.Exec("UPDATE attempts SET is_correct=true WHERE id=1").Error)
			payload, eval := "{}", "{}"
			if kind == "payload_hint" {
				payload = `{"hintsUsed":1}`
				eval = `{"assistance":"none"}`
			} else if kind == "result_conflict" {
				payload = `{"hintsUsed":0}`
				eval = `{"outcome":"not_passed","assistance":"none"}`
			} else if kind == "invalid" {
				eval = `{"assistance":"unrecognized"}`
			}
			require.NoError(t, g.Exec("UPDATE question_attempt_receipts SET question_type='write_char',is_correct=true,answer_payload_json=?,evaluation_json=? WHERE id=1", payload, eval).Error)
			target := in.Targets[0]
			target.QuestionType = "write_char"
			target.Key = "103:write_char"
			ev := target.Evidence[0]
			ev.Role = "target_observation"
			v, e := verifyEvidence(g, 7, target, ev, in.AnalysisAsOf)
			if kind == "result_conflict" {
				require.Error(t, e)
			} else if kind == "payload_hint" {
				require.Error(t, e)
				ev.Role = "assisted_completion"
				v, e = verifyEvidence(g, 7, target, ev, in.AnalysisAsOf)
				require.NoError(t, e)
				require.Equal(t, "hinted", v.Assistance)
			} else {
				require.NoError(t, e)
				require.Equal(t, "unknown", v.Assistance)
			}
		})
	}
}
func TestPlansIncludeEditedRevisionWithoutOriginalAttribution(t *testing.T) {
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
	require.NoError(t, s.DB.Exec("INSERT INTO study_plans VALUES(20,7,'literacy','done',?,999)", links[0].TaskID).Error)
	page, e := s.PlansPage(ctx, 7, a.ID, 0, 30)
	require.NoError(t, e)
	require.Len(t, page.Items.([]PlanLink), 1)
	p := page.Items.([]PlanLink)[0]
	require.EqualValues(t, 999, p.ActualPlanRevisionID)
	require.True(t, p.RevisionChanged)
	require.Equal(t, "revision_changed", p.AttributionStatus)
	require.Equal(t, links[0].GeneratedRevisionID, p.GeneratedRevisionID)
}
func TestSupersedesRequiresSameChildAndDoesNotChangeContentIdentity(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	in := input()
	a, _, e := s.Save(ctx, 7, "a", in)
	require.NoError(t, e)
	in.SupersedesID = &a.ID
	b, replay, e := s.Save(ctx, 7, "b", in)
	require.NoError(t, e)
	require.True(t, replay)
	require.Equal(t, a.ID, b.ID)
	other := Suggestion{ChildID: 8, ContentHash: "other", Lifecycle: "open", RowVersion: 1}
	require.NoError(t, s.DB.Create(&other).Error)
	in.SupersedesID = &other.ID
	_, _, e = s.Save(ctx, 7, "bad-child", in)
	require.Error(t, e)
}
func TestGeneratedTaskLinksBackToSuggestion(t *testing.T) {
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
	task, e := s.Generation.Get(ctx, links[0].TaskID)
	require.NoError(t, e)
	require.NotNil(t, task.SourceReviewSuggestionID)
	require.Equal(t, a.ID, *task.SourceReviewSuggestionID)
}
func TestReviewReferencesRestrictSourceDeletion(t *testing.T) {
	s, _, in := literacyFixture(t)
	require.NoError(t, Migrate(s.DB))
	a, _, e := s.Save(context.Background(), 7, "save", in)
	require.NoError(t, e)
	for _, sql := range []string{"DELETE FROM review_suggestions WHERE id=" + fmt.Sprint(a.ID), "DELETE FROM review_suggestion_targets WHERE suggestion_id=" + fmt.Sprint(a.ID), "DELETE FROM attempts WHERE id=1", "DELETE FROM children WHERE id=7", "DELETE FROM question_versions WHERE id=1", "DELETE FROM plan_items WHERE id=1", "DELETE FROM study_plans WHERE id=1"} {
		require.Error(t, s.DB.Exec(sql).Error, sql)
	}
	require.Error(t, s.DB.Create(&Target{SuggestionID: 99999, TargetKey: "dangling", KpID: 103}).Error)
}

func TestReferenceUpgradeBackfillsExistingEvidence(t *testing.T) {
	s, _, in := literacyFixture(t)
	_, _, e := s.Save(context.Background(), 7, "save", in)
	require.NoError(t, e)
	require.NoError(t, s.DB.Exec("UPDATE review_suggestion_evidence SET source_receipt_id=NULL,source_question_version_id=NULL,source_plan_id=NULL,source_plan_item_id=NULL").Error)
	require.NoError(t, Migrate(s.DB))
	require.Error(t, s.DB.Exec("DELETE FROM question_versions WHERE id=1").Error)
}

func TestPinyinInvalidSnapshotStaysUnknownAndCannotSupportStrongReason(t *testing.T) {
	for _, kind := range []string{"valid", "malformed", "wrong_kp", "missing_selection", "duplicate", "conflict"} {
		t.Run(kind, func(t *testing.T) {
			s := fixture(t)
			g := s.DB
			require.NoError(t, g.Exec(`UPDATE subjects SET code='pinyin';CREATE TABLE pinyin_quiz_instances(id TEXT PRIMARY KEY,child_id INTEGER,kp_id INTEGER,skill_code TEXT,public_snapshot TEXT,answer_option_id TEXT);CREATE TABLE pinyin_answer_receipts(child_id INTEGER,client_id TEXT,instance_id TEXT,attempt_id INTEGER,skill_code TEXT,selected_option_id TEXT)`).Error)
			snap := `{"kpId":103,"options":[{"id":"a","label":"a"},{"id":"b","label":"b"}]}`
			switch kind {
			case "malformed":
				snap = "broken"
			case "wrong_kp":
				snap = `{"kpId":999,"options":[{"id":"a"},{"id":"b"}]}`
			case "missing_selection":
				snap = `{"kpId":103,"options":[{"id":"a"}]}`
			case "duplicate":
				snap = `{"kpId":103,"options":[{"id":"a"},{"id":"b"},{"id":"b"}]}`
			case "conflict":
				require.NoError(t, g.Exec("UPDATE attempts SET is_correct=true WHERE id=1").Error)
			}
			require.NoError(t, g.Exec("INSERT INTO pinyin_quiz_instances VALUES('i',7,103,'listen',?,'a')", snap).Error)
			require.NoError(t, g.Exec("INSERT INTO pinyin_answer_receipts VALUES(7,'c1','i',1,'listen','b')").Error)
			in := input()
			in.SubjectCode = "pinyin"
			target := in.Targets[0]
			target.Key = "103:listen"
			target.QuestionType = "listen"
			ev := EvidenceInput{AttemptID: 1, Role: "target_observation", Source: Source{Kind: "pinyin_instance", InstanceID: "i", ClientID: "c1"}}
			v, e := verifyEvidence(g, 7, target, ev, in.AnalysisAsOf)
			require.NoError(t, e)
			if kind == "valid" {
				require.Equal(t, "none", v.Assistance)
			} else {
				require.Equal(t, "unknown", v.Assistance)
				require.Empty(t, v.Semantic)
			}
		})
	}
}
