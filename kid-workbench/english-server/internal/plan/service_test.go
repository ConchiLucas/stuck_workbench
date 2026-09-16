package plan_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conchi/english-server/internal/plan"
	"github.com/conchi/study-learning/englishcontent"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreatePlanIsStableEnglishScoped(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:english_plan?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, sql := range []string{`CREATE TABLE children(id INTEGER PRIMARY KEY,name TEXT)`, `CREATE TABLE subjects(id INTEGER PRIMARY KEY,code TEXT)`, `CREATE TABLE modules(id INTEGER PRIMARY KEY,subject_id INTEGER,code TEXT,order_no INTEGER)`, `CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY,module_id INTEGER,title TEXT,payload TEXT,order_no INTEGER)`, `CREATE TABLE english_assets(kp_id INTEGER PRIMARY KEY,glyph_image_url TEXT,sense_image_url TEXT,speech_audio_url TEXT)`, `CREATE TABLE mastery_states(child_id INTEGER,kp_id INTEGER,status TEXT,due_at DATETIME)`, `CREATE TABLE questions(id INTEGER PRIMARY KEY,kp_id INTEGER,code TEXT,type TEXT,stem TEXT,options TEXT,answer TEXT,visual TEXT,speech TEXT,difficulty INTEGER)`, `CREATE TABLE study_plans(id INTEGER PRIMARY KEY AUTOINCREMENT,child_id INTEGER,plan_date TEXT,seq_no INTEGER,subject_code TEXT,status TEXT,target_count INTEGER,done_count INTEGER DEFAULT 0,correct_count INTEGER DEFAULT 0,stars INTEGER DEFAULT 0,duration_sec INTEGER DEFAULT 0,created_at DATETIME,started_at DATETIME,completed_at DATETIME)`, `CREATE TABLE plan_items(id INTEGER PRIMARY KEY AUTOINCREMENT,plan_id INTEGER,seq INTEGER,kp_id INTEGER,question_id INTEGER,bucket TEXT,status TEXT,tries INTEGER DEFAULT 0,cost_ms INTEGER DEFAULT 0,picks TEXT DEFAULT '',option_order TEXT DEFAULT '',question_stem TEXT DEFAULT '',question_options TEXT DEFAULT '',question_answer TEXT DEFAULT '',question_visual TEXT DEFAULT '',question_speech TEXT DEFAULT '',question_snapshot TEXT DEFAULT '{}',explanation TEXT DEFAULT '',content_snapshot_version INTEGER DEFAULT 0,answered_at DATETIME)`, `INSERT INTO children VALUES(1,'安安')`, `INSERT INTO subjects VALUES(1,'english'),(2,'pinyin')`, `INSERT INTO modules VALUES(10,1,'animals',1),(20,2,'initials',1)`, `INSERT INTO knowledge_points VALUES(100,10,'cat','{"meaningZh":"猫"}',1),(101,10,'dog','{"meaningZh":"狗"}',2),(102,10,'bird','{"meaningZh":"鸟"}',3),(103,10,'fish','{"meaningZh":"鱼"}',4),(200,20,'b','{}',1)`, `INSERT INTO english_assets VALUES(100,'g','s','a'),(101,'g','s','a'),(102,'g','s','a'),(103,'g','s','a')`, `INSERT INTO questions VALUES(1000,100,'listen','choice','听','[{"kpId":100,"label":"cat"},{"kpId":101,"label":"dog"},{"kpId":102,"label":"bird"},{"kpId":103,"label":"fish"}]','{"index":0}','{}','{"text":"cat"}',1),(1001,100,'picture','choice','图','[{"kpId":100,"assetKind":"sense"},{"kpId":101,"assetKind":"sense"},{"kpId":102,"assetKind":"sense"},{"kpId":103,"assetKind":"sense"}]','{"index":0}','{}','{"text":"cat"}',1),(2000,200,'listen','choice','bad','[]','{"index":0}','{}','{"text":"b"}',1)`} {
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
	s := plan.NewService(db, mediaServer.URL)
	created, err := s.Create(context.Background(), 1, plan.CreateInput{Mode: "module", ModuleCode: "animals", Count: 2})
	require.NoError(t, err)
	require.Equal(t, "english", created.Plan.SubjectCode)
	require.Len(t, created.Items, 2)
	require.NotEmpty(t, created.Items[0].MeaningZh)
	require.NoError(t, db.Table("questions").Where("id=?", created.Items[0].Question.ID).Updates(map[string]any{"stem": "changed", "options": "[]"}).Error)
	again, err := s.Get(context.Background(), 1, created.Plan.ID)
	require.NoError(t, err)
	require.Equal(t, created.Items[0].Question.Stem, again.Items[0].Question.Stem)
	require.Equal(t, string(created.Items[0].Question.Options), string(again.Items[0].Question.Options))
	require.NotEmpty(t, again.Items[0].OptionOrder)
	var snap englishcontent.HistorySnapshot
	require.NoError(t, json.Unmarshal([]byte(mustSnapshot(t, db, created.Items[0].ID)), &snap))
	require.Equal(t, 1, snap.Schema)
	require.Contains(t, []string{"audio-choice", "image-text"}, snap.Kind)
	require.GreaterOrEqual(t, len(snap.Example.Options), 2)
	require.NotEmpty(t, snap.Example.AnswerID)
	require.NotEmpty(t, snap.Example.SpeechURL)
	require.Empty(t, snap.Selected)
	require.True(t, englishcontent.HasFrozenMedia(snap.Example))
	require.NoError(t, englishcontent.VerifyMedia(db, snap.Example))
	var before, after int64
	require.NoError(t, db.Table("study_plans").Count(&before).Error)
	mediaServer.Close()
	_, err = s.Create(context.Background(), 1, plan.CreateInput{Count: 2})
	require.Error(t, err)
	require.NoError(t, db.Table("study_plans").Count(&after).Error)
	require.Equal(t, before, after, "media failure must roll back plan")
}

func mustSnapshot(t *testing.T, db *gorm.DB, itemID int64) string {
	t.Helper()
	var raw string
	require.NoError(t, db.Table("plan_items").Select("question_snapshot").Where("id=?", itemID).Scan(&raw).Error)
	require.NotEmpty(t, raw)
	return raw
}
