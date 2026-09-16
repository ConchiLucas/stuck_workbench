package practice_test

import (
	"context"
	"encoding/json"
	"github.com/conchi/literacy-server/internal/plan"
	"github.com/conchi/literacy-server/internal/practice"
	"github.com/conchi/study-learning/mastery"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestPublishedTaskClaimAndReceipts(t *testing.T) {
	ctx := context.Background()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, sql := range practiceSchema() {
		require.NoError(t, database.Exec(sql).Error)
	}
	for _, sql := range []string{
		`ALTER TABLE study_plans ADD COLUMN source_question_task_id INTEGER`, `ALTER TABLE study_plans ADD COLUMN source_question_task_revision_id INTEGER`, `ALTER TABLE study_plans ADD COLUMN task_claim_key TEXT`, `ALTER TABLE plan_items ADD COLUMN question_version_id INTEGER`,
		`CREATE TABLE study_plan_task_claims(child_id INTEGER,claim_key TEXT,task_id INTEGER,revision_id INTEGER,plan_id INTEGER,PRIMARY KEY(child_id,claim_key))`,
		`CREATE TABLE question_task_revisions(id INTEGER PRIMARY KEY,task_id INTEGER)`,
		`INSERT INTO question_task_revisions VALUES(1,1)`,
		`CREATE TABLE question_tasks(id INTEGER PRIMARY KEY,title TEXT,subject_code TEXT,kind TEXT,status TEXT,source_mode TEXT,published_revision_id INTEGER,target_child_id INTEGER)`,
		`CREATE TABLE question_versions(id INTEGER PRIMARY KEY,revision_id INTEGER,seq INTEGER,kp_id INTEGER,question_type TEXT,skill_code TEXT,snapshot_json TEXT)`,
		`INSERT INTO question_tasks VALUES(1,'识字任务','literacy','practice','published','material_template',1,NULL)`,
		`CREATE TABLE question_attempt_receipts(id INTEGER PRIMARY KEY AUTOINCREMENT,attempt_id INTEGER UNIQUE,child_id INTEGER,client_id TEXT,plan_id INTEGER,plan_item_id INTEGER,question_version_id INTEGER,kp_id INTEGER,skill_code TEXT,question_type TEXT,selected_option_id TEXT,display_index INTEGER,original_index INTEGER,is_correct BOOLEAN,cost_ms INTEGER,response_json TEXT,created_at DATETIME,UNIQUE(child_id,client_id))`,
	} {
		require.NoError(t, database.Exec(sql).Error)
	}
	snapshot := `{"schemaVersion":1,"subjectCode":"literacy","kpId":100,"targetText":"一","questionType":"glyph_sense","skillCode":"glyph_sense","templateVersion":"literacy-choice-v1","prompt":"看字选义","stem":{"text":"一"},"options":[{"id":"kp:100","kpId":100,"text":"一"},{"id":"kp:101","kpId":101,"text":"二"},{"id":"kp:102","kpId":102,"text":"三"},{"id":"kp:103","kpId":103,"text":"四"}],"answerOptionId":"kp:100","explanation":"secret answer"}`
	require.NoError(t, database.Exec(`INSERT INTO question_versions VALUES(1,1,1,100,'glyph_sense','glyph_sense',?)`, snapshot).Error)
	plans := plan.NewService(database)
	claimed, err := plans.ClaimTask(ctx, 1, 1, plan.ClaimInput{RevisionID: 1, ClaimKey: "claim1"})
	require.NoError(t, err)
	require.Len(t, claimed.Items, 1)
	safe, err := json.Marshal(claimed)
	require.NoError(t, err)
	require.NotContains(t, string(safe), "answerOptionId")
	require.NotContains(t, string(safe), "secret answer")
	again, err := plans.ClaimTask(ctx, 1, 1, plan.ClaimInput{RevisionID: 1, ClaimKey: "claim1"})
	require.NoError(t, err)
	require.Equal(t, claimed.Plan.ID, again.Plan.ID)
	_, err = plans.ClaimTask(ctx, 1, 1, plan.ClaimInput{RevisionID: 2, ClaimKey: "claim1"})
	require.Error(t, err)
	var row struct{ QuestionID *int64 }
	require.NoError(t, database.Table("plan_items").Select("question_id").Where("id = ?", claimed.Items[0].ID).Take(&row).Error)
	require.Nil(t, row.QuestionID)
	require.NoError(t, database.Table("plan_items").Where("id = ?", claimed.Items[0].ID).Updates(map[string]any{"question_answer": `{"index":3}`, "question_options": `[{"label":"edited"}]`}).Error)
	stable, err := plans.Get(ctx, 1, claimed.Plan.ID)
	require.NoError(t, err)
	require.NotContains(t, string(stable.Items[0].Question.Options), "edited")
	order, err := plan.ParseOrder(claimed.Items[0].OptionOrder)
	require.NoError(t, err)
	correct := 0
	for i, v := range order {
		if v == 0 {
			correct = i
		}
	}
	wrong := (correct + 1) % 4
	svc := practice.NewService(database, mastery.Config{BaseMasterStreak: 2, MinAccuracy: .8, EaseMin: 1.3, EaseMax: 2.8, MaxIntervalDays: 60})
	first, err := svc.Answer(ctx, 1, claimed.Plan.ID, claimed.Items[0].ID, practice.AnswerInput{ClientID: "try1", OptionIndex: wrong, CostMs: 100})
	require.NoError(t, err)
	require.False(t, first.Correct)
	_, err = svc.Answer(ctx, 1, claimed.Plan.ID, claimed.Items[0].ID, practice.AnswerInput{ClientID: "try1", OptionIndex: correct})
	require.ErrorIs(t, err, practice.ErrIdempotencyConflict)
	_, err = svc.Answer(ctx, 1, claimed.Plan.ID, claimed.Items[0].ID, practice.AnswerInput{ClientID: "try2", OptionIndex: correct, CostMs: 100})
	require.NoError(t, err)
	replay, err := svc.Answer(ctx, 1, claimed.Plan.ID, claimed.Items[0].ID, practice.AnswerInput{ClientID: "try1", OptionIndex: wrong, CostMs: 100})
	require.NoError(t, err)
	firstJSON, _ := json.Marshal(first)
	replayJSON, _ := json.Marshal(replay)
	require.JSONEq(t, string(firstJSON), string(replayJSON))
	continued, err := plans.ClaimTask(ctx, 1, 1, plan.ClaimInput{RevisionID: 1, ClaimKey: "claim2"})
	require.NoError(t, err)
	require.Equal(t, claimed.Plan.ID, continued.Plan.ID)
	_, err = svc.Finish(ctx, 1, claimed.Plan.ID)
	require.NoError(t, err)
	replayPlan, err := plans.ClaimTask(ctx, 1, 1, plan.ClaimInput{RevisionID: 1, ClaimKey: "claim2"})
	require.NoError(t, err)
	require.Equal(t, claimed.Plan.ID, replayPlan.Plan.ID)
	next, err := plans.ClaimTask(ctx, 1, 1, plan.ClaimInput{RevisionID: 1, ClaimKey: "claim3"})
	require.NoError(t, err)
	require.NotEqual(t, claimed.Plan.ID, next.Plan.ID)
	var count int64
	require.NoError(t, database.Table("question_attempt_receipts").Count(&count).Error)
	require.EqualValues(t, 2, count)
	require.NoError(t, database.Exec(`CREATE TRIGGER fail_receipt BEFORE INSERT ON question_attempt_receipts BEGIN SELECT RAISE(ABORT,'receipt failure'); END`).Error)
	_, err = svc.Answer(ctx, 1, next.Plan.ID, next.Items[0].ID, practice.AnswerInput{ClientID: "rollback", OptionIndex: 0, CostMs: 100})
	require.Error(t, err)
	require.NoError(t, database.Table("attempts").Count(&count).Error)
	require.EqualValues(t, 2, count)
	var tries int
	require.NoError(t, database.Table("plan_items").Select("tries").Where("id = ?", next.Items[0].ID).Scan(&tries).Error)
	require.Zero(t, tries)
	require.NoError(t, database.Exec(`INSERT INTO children(id,name) VALUES(2,'另一孩子')`).Error)
	require.NoError(t, database.Table("question_tasks").Where("id=1").Update("target_child_id", 1).Error)
	available, err := plans.ListTasks(ctx, 2)
	require.NoError(t, err)
	require.Empty(t, available.Items)
	_, err = plans.ClaimTask(ctx, 2, 1, plan.ClaimInput{RevisionID: 1, ClaimKey: "forbidden"})
	require.ErrorIs(t, err, plan.ErrTaskNotAvailable)
	require.NoError(t, database.Table("question_tasks").Where("id=1").Update("status", "draft").Error)
	_, err = plans.ClaimTask(ctx, 1, 1, plan.ClaimInput{RevisionID: 1, ClaimKey: "after-withdraw"})
	require.ErrorIs(t, err, plan.ErrTaskNotAvailable)
	resumed, err := plans.Get(ctx, 1, next.Plan.ID)
	require.NoError(t, err)
	require.Len(t, resumed.Items, 1)

}
