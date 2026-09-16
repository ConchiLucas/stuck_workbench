package practice_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conchi/english-server/internal/plan"
	"github.com/conchi/english-server/internal/practice"
	"github.com/conchi/study-learning/englishcontent"
	"github.com/conchi/study-learning/mastery"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAnswerSupportsTwoTriesAndIdempotency(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:english_practice?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	require.NoError(t, err)
	for _, sql := range schema() {
		require.NoError(t, db.Exec(sql).Error)
	}
	require.NoError(t, englishcontent.MigrateMedia(db))
	mediaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".mp3") {
			w.Write([]byte("ID3test audio"))
		} else {
			w.Write([]byte("\x89PNG\r\n\x1a\ntest image"))
		}
	}))
	defer mediaServer.Close()
	detail, err := plan.NewService(db, mediaServer.URL).Create(context.Background(), 1, plan.CreateInput{Mode: "today", Count: 2})
	require.NoError(t, err)
	item := detail.Items[0]
	order, err := plan.ParseOrder(item.OptionOrder)
	require.NoError(t, err)
	correct := 0
	for display, original := range order {
		if original == 0 {
			correct = display
		}
	}
	wrong := (correct + 1) % len(order)
	createdSnap := itemSnapshot(t, db, item.ID)
	svc := practice.NewService(db, mastery.Config{BaseMasterStreak: 2, MinAccuracy: .8, ShakyMinAttempts: 3, ShakyAccuracy: .6, EaseMin: 1.3, EaseMax: 2.8, EaseUp: .1, EaseDown: .2, MaxIntervalDays: 60})
	first, err := svc.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "try-1", OptionIndex: wrong, CostMs: 1000})
	require.NoError(t, err)
	require.False(t, first.Correct)
	require.True(t, first.CanRetry)
	second, err := svc.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "try-2", OptionIndex: correct, CostMs: 800})
	require.NoError(t, err)
	require.True(t, second.Correct)
	require.False(t, second.CanRetry)
	_, err = svc.Answer(context.Background(), 1, detail.Plan.ID, item.ID, practice.AnswerInput{ClientID: "try-2", OptionIndex: correct})
	require.NoError(t, err)
	var attempts int64
	require.NoError(t, db.Table("attempts").Count(&attempts).Error)
	require.Equal(t, int64(2), attempts)
	var linked int64
	require.NoError(t, db.Table("attempts").Where("plan_item_id = ?", item.ID).Count(&linked).Error)
	require.Equal(t, int64(2), linked)
	var rows []struct {
		ClientID string
		Selected string
		Correct  bool
	}
	require.NoError(t, db.Table("attempts").Select("client_id, selected, is_correct AS correct").Where("plan_item_id = ?", item.ID).Order("id").Scan(&rows).Error)
	require.Len(t, rows, 2)
	require.Equal(t, "try-1", rows[0].ClientID)
	require.False(t, rows[0].Correct)
	require.NotEmpty(t, rows[0].Selected)
	require.Equal(t, "try-2", rows[1].ClientID)
	require.True(t, rows[1].Correct)
	require.NotEqual(t, rows[0].Selected, rows[1].Selected)
	firstSelected := rows[0].Selected
	require.NoError(t, db.Table("attempts").Select("client_id, selected, is_correct AS correct").Where("plan_item_id = ?", item.ID).Order("id").Scan(&rows).Error)
	require.Equal(t, firstSelected, rows[0].Selected)
	require.False(t, rows[0].Correct)
	raw := itemSnapshot(t, db, item.ID)
	var snap englishcontent.HistorySnapshot
	require.NoError(t, json.Unmarshal([]byte(raw), &snap))
	require.Equal(t, 1, snap.Schema)
	require.NotEmpty(t, snap.Example.AnswerID)
	wrongEx, err := englishcontent.PlanExampleFromSnapshot(raw, rows[0].Selected)
	require.NoError(t, err)
	require.Equal(t, rows[0].Selected, wrongEx.Selected)
	rightEx, err := englishcontent.PlanExampleFromSnapshot(raw, rows[1].Selected)
	require.NoError(t, err)
	require.Equal(t, rows[1].Selected, rightEx.Selected)
	require.Equal(t, snap.Example.AnswerID, rightEx.Selected)
	require.Equal(t, createdSnap, itemSnapshot(t, db, item.ID))
	var picks string
	require.NoError(t, db.Table("plan_items").Select("picks").Where("id=?", item.ID).Scan(&picks).Error)
	require.NotContains(t, picks, "[")
	require.Equal(t, rows[1].Selected, picks)
	var skills int64
	require.NoError(t, db.Table("mastery_skills").Where("skill_code IN ?", []string{"listen", "picture"}).Count(&skills).Error)
	require.Greater(t, skills, int64(0))
}

func itemSnapshot(t *testing.T, db *gorm.DB, itemID int64) string {
	t.Helper()
	var raw string
	require.NoError(t, db.Table("plan_items").Select("question_snapshot").Where("id=?", itemID).Scan(&raw).Error)
	require.NotEmpty(t, raw)
	return raw
}

func schema() []string {
	return []string{`CREATE TABLE children(id INTEGER PRIMARY KEY,name TEXT,grade TEXT DEFAULT '',avatar_url TEXT DEFAULT '',flowers INTEGER DEFAULT 0,created_at DATETIME)`, `CREATE TABLE subjects(id INTEGER PRIMARY KEY,code TEXT)`, `CREATE TABLE modules(id INTEGER PRIMARY KEY,subject_id INTEGER,code TEXT,order_no INTEGER)`, `CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY,module_id INTEGER,code TEXT,title TEXT,payload TEXT,difficulty INTEGER,order_no INTEGER)`, `CREATE TABLE english_assets(kp_id INTEGER PRIMARY KEY,glyph_image_url TEXT,sense_image_url TEXT,speech_audio_url TEXT)`, `CREATE TABLE questions(id INTEGER PRIMARY KEY,kp_id INTEGER,code TEXT,type TEXT,stem TEXT,options TEXT,answer TEXT,visual TEXT,speech TEXT,difficulty INTEGER)`, `CREATE TABLE study_plans(id INTEGER PRIMARY KEY AUTOINCREMENT,child_id INTEGER,plan_date TEXT,seq_no INTEGER,subject_code TEXT,status TEXT,target_count INTEGER,done_count INTEGER DEFAULT 0,correct_count INTEGER DEFAULT 0,stars INTEGER DEFAULT 0,duration_sec INTEGER DEFAULT 0,created_at DATETIME,started_at DATETIME,completed_at DATETIME)`, `CREATE TABLE plan_items(id INTEGER PRIMARY KEY AUTOINCREMENT,plan_id INTEGER,seq INTEGER,kp_id INTEGER,question_id INTEGER,bucket TEXT,status TEXT,tries INTEGER DEFAULT 0,cost_ms INTEGER DEFAULT 0,picks TEXT DEFAULT '',option_order TEXT DEFAULT '',question_stem TEXT DEFAULT '',question_options TEXT DEFAULT '',question_answer TEXT DEFAULT '',question_visual TEXT DEFAULT '',question_speech TEXT DEFAULT '',question_snapshot TEXT DEFAULT '{}',explanation TEXT DEFAULT '',content_snapshot_version INTEGER DEFAULT 0,answered_at DATETIME)`, `CREATE TABLE attempts(id INTEGER PRIMARY KEY AUTOINCREMENT,child_id INTEGER,kp_id INTEGER,question_id INTEGER,is_correct BOOLEAN,cost_ms INTEGER,source TEXT,client_id TEXT,created_at DATETIME,plan_item_id INTEGER,selected TEXT DEFAULT '')`, `CREATE UNIQUE INDEX idx_attempts_idem ON attempts(child_id,client_id)`, `CREATE TABLE mastery_states(child_id INTEGER,kp_id INTEGER,status TEXT,attempts INTEGER,correct INTEGER,streak INTEGER,best_streak INTEGER,ease REAL,interval_days INTEGER,due_at DATETIME,first_seen_at DATETIME,mastered_at DATETIME,updated_at DATETIME,PRIMARY KEY(child_id,kp_id))`, `CREATE TABLE mastery_skills(child_id INTEGER,kp_id INTEGER,skill_code TEXT,status TEXT,attempts INTEGER,correct INTEGER,streak INTEGER,best_streak INTEGER,ease REAL,interval_days INTEGER,due_at DATETIME,first_seen_at DATETIME,mastered_at DATETIME,updated_at DATETIME,PRIMARY KEY(child_id,kp_id,skill_code))`, `CREATE TABLE daily_stats(child_id INTEGER,stat_date TEXT,practice_sec INTEGER DEFAULT 0,attempts INTEGER DEFAULT 0,correct INTEGER DEFAULT 0,newly_mastered INTEGER DEFAULT 0,review_done INTEGER DEFAULT 0,checked_in BOOLEAN DEFAULT FALSE,PRIMARY KEY(child_id,stat_date))`, `CREATE TABLE flower_ledger(id INTEGER PRIMARY KEY AUTOINCREMENT,child_id INTEGER,delta INTEGER,reason TEXT,ref_type TEXT,ref_id INTEGER,created_at DATETIME)`, `INSERT INTO children(id,name,created_at) VALUES(1,'安安',CURRENT_TIMESTAMP)`, `INSERT INTO subjects VALUES(1,'english')`, `INSERT INTO modules VALUES(10,1,'animals',1)`, `INSERT INTO knowledge_points VALUES(100,10,'e1001','cat','{"meaningZh":"猫"}',1,1)`, `INSERT INTO english_assets VALUES(100,'g','s','a')`, `INSERT INTO questions VALUES(1000,100,'listen','choice','听','[{"kpId":100,"label":"cat"},{"kpId":101,"label":"dog"},{"kpId":102,"label":"bird"},{"kpId":103,"label":"fish"}]','{"index":0}','{}','{"text":"cat"}',1),(1001,100,'picture','choice','图','[{"kpId":100,"assetKind":"sense"},{"kpId":101,"assetKind":"sense"},{"kpId":102,"assetKind":"sense"},{"kpId":103,"assetKind":"sense"}]','{"index":0}','{}','{"text":"cat"}',1)`}
}
