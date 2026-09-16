package testdb

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func Open(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	require.NoError(t, err)
	for _, sql := range Schema() {
		require.NoError(t, db.Exec(sql).Error)
	}
	return db
}

func Schema() []string {
	return []string{
		`CREATE TABLE children(id INTEGER PRIMARY KEY,name TEXT,grade TEXT DEFAULT '',avatar_url TEXT DEFAULT '',flowers INTEGER DEFAULT 0,created_at DATETIME)`,
		`CREATE TABLE subjects(id INTEGER PRIMARY KEY,code TEXT)`,
		`CREATE TABLE modules(id INTEGER PRIMARY KEY,subject_id INTEGER,code TEXT,order_no INTEGER)`,
		`CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY,module_id INTEGER,code TEXT,title TEXT,payload TEXT,difficulty INTEGER,order_no INTEGER)`,
		`CREATE TABLE questions(id INTEGER PRIMARY KEY,kp_id INTEGER,code TEXT,type TEXT,stem TEXT,options TEXT,answer TEXT,visual TEXT,speech TEXT,difficulty INTEGER)`,
		`CREATE TABLE study_plans(id INTEGER PRIMARY KEY AUTOINCREMENT,child_id INTEGER,plan_date TEXT,seq_no INTEGER,subject_code TEXT,status TEXT,target_count INTEGER,done_count INTEGER DEFAULT 0,correct_count INTEGER DEFAULT 0,stars INTEGER DEFAULT 0,duration_sec INTEGER DEFAULT 0,created_at DATETIME,started_at DATETIME,completed_at DATETIME)`,
		`CREATE TABLE plan_items(id INTEGER PRIMARY KEY AUTOINCREMENT,plan_id INTEGER,seq INTEGER,kp_id INTEGER,question_id INTEGER,bucket TEXT,status TEXT,tries INTEGER DEFAULT 0,cost_ms INTEGER DEFAULT 0,picks TEXT DEFAULT '',option_order TEXT DEFAULT '',question_stem TEXT DEFAULT '',question_options TEXT DEFAULT '',question_answer TEXT DEFAULT '',question_visual TEXT DEFAULT '',question_speech TEXT DEFAULT '',question_snapshot TEXT DEFAULT '{}',explanation TEXT DEFAULT '',content_snapshot_version INTEGER DEFAULT 0,answered_at DATETIME)`,
		`CREATE TABLE attempts(id INTEGER PRIMARY KEY AUTOINCREMENT,child_id INTEGER,kp_id INTEGER,question_id INTEGER,plan_item_id INTEGER,selected TEXT DEFAULT '',is_correct BOOLEAN,cost_ms INTEGER,source TEXT,client_id TEXT,created_at DATETIME)`,
		`CREATE UNIQUE INDEX idx_attempts_idem ON attempts(child_id,client_id)`,
		`CREATE TABLE phrase_question_task_media(sha256 TEXT PRIMARY KEY,kind TEXT,data BLOB)`,
		`CREATE TABLE mastery_states(child_id INTEGER,kp_id INTEGER,status TEXT,attempts INTEGER,correct INTEGER,streak INTEGER,best_streak INTEGER,ease REAL,interval_days INTEGER,due_at DATETIME,first_seen_at DATETIME,mastered_at DATETIME,updated_at DATETIME,PRIMARY KEY(child_id,kp_id))`,
		`CREATE TABLE mastery_skills(child_id INTEGER,kp_id INTEGER,skill_code TEXT,status TEXT,attempts INTEGER,correct INTEGER,streak INTEGER,best_streak INTEGER,ease REAL,interval_days INTEGER,due_at DATETIME,first_seen_at DATETIME,mastered_at DATETIME,updated_at DATETIME,PRIMARY KEY(child_id,kp_id,skill_code))`,
		`CREATE TABLE daily_stats(child_id INTEGER,stat_date TEXT,practice_sec INTEGER DEFAULT 0,attempts INTEGER DEFAULT 0,correct INTEGER DEFAULT 0,newly_mastered INTEGER DEFAULT 0,review_done INTEGER DEFAULT 0,checked_in BOOLEAN DEFAULT FALSE,PRIMARY KEY(child_id,stat_date))`,
		`CREATE TABLE flower_ledger(id INTEGER PRIMARY KEY AUTOINCREMENT,child_id INTEGER,delta INTEGER,reason TEXT,ref_type TEXT,ref_id INTEGER,created_at DATETIME)`,
		`INSERT INTO children(id,name,flowers,created_at) VALUES(1,'安安',0,CURRENT_TIMESTAMP)`,
		`INSERT INTO subjects VALUES(1,'phrase'),(2,'pinyin')`,
		`INSERT INTO modules VALUES(10,1,'greet',1),(20,2,'initials',1)`,
		`INSERT INTO knowledge_points VALUES(100,10,'ph0001','Good morning.','{"kind":"phrase","zh":"早上好。","wrong":["下午好。","晚上好。","晚安。"],"scene":"早上见到老师"}',1,1),(101,10,'ph0002','I''m fine.','{"kind":"phrase","zh":"我很好。","wrong":["我饿了。","我累了。","我不舒服。"],"scene":"别人问你好不好","replyTo":"How are you?"}',1,2),(102,10,'ph0003','How are you?','{"kind":"phrase","zh":"你好吗？","wrong":["再见。","早上好。","谢谢。"],"scene":"想问问朋友好不好"}',1,3),(200,20,'b','b','{}',1,1)`,
		`INSERT INTO questions VALUES(1000,100,'listen_zh','choice','听一听，选出中文意思','[{"kpId":100,"label":"早上好。"},{"label":"下午好。"},{"label":"晚上好。"},{"label":"晚安。"}]','{"index":0}','{}','{"text":"Good morning.","lang":"en-US"}',1),(1001,100,'listen_en','choice','听一听，点出这句英语','[{"kpId":100,"label":"Good morning."},{"label":"Good afternoon."},{"label":"Hello!"},{"label":"Good night."}]','{"index":0}','{}','{"text":"Good morning.","lang":"en-US"}',1),(1002,100,'scene','choice','这种时候该说哪一句？','[{"kpId":100,"label":"Good morning."},{"label":"Good afternoon."},{"label":"Hello!"},{"label":"Good night."}]','{"index":0}','{"kind":"scene","text":"早上见到老师"}','{"text":"Good morning.","lang":"en-US"}',1),(1003,101,'reply','choice','对方说了这句话，你怎么答？','[{"kpId":101,"label":"I''m fine."},{"label":"Hello!"},{"label":"Thank you."},{"label":"Good morning."}]','{"index":0}','{"kind":"prompt","text":"How are you?"}','{"text":"How are you?","lang":"en-US"}',1),(1004,101,'listen_zh','choice','听一听，选出中文意思','[{"kpId":101,"label":"我很好。"},{"label":"我饿了。"},{"label":"我累了。"},{"label":"我不舒服。"}]','{"index":0}','{}','{"text":"I''m fine.","lang":"en-US"}',1),(2000,200,'listen','choice','听一听','[{"label":"b"}]','{"index":0}','{}','{"text":"b"}',1)`,
	}
}
