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
		`CREATE TABLE attempts(id INTEGER PRIMARY KEY AUTOINCREMENT,child_id INTEGER,kp_id INTEGER,question_id INTEGER,plan_item_id INTEGER,is_correct BOOLEAN,cost_ms INTEGER,source TEXT,client_id TEXT,selected TEXT DEFAULT '',skill_code TEXT DEFAULT '',created_at DATETIME)`,
		`CREATE UNIQUE INDEX idx_attempts_idem ON attempts(child_id,client_id)`,
		`CREATE TABLE mastery_states(child_id INTEGER,kp_id INTEGER,status TEXT,attempts INTEGER,correct INTEGER,streak INTEGER,best_streak INTEGER,ease REAL,interval_days INTEGER,due_at DATETIME,first_seen_at DATETIME,mastered_at DATETIME,updated_at DATETIME,PRIMARY KEY(child_id,kp_id))`,
		`CREATE TABLE mastery_skills(child_id INTEGER,kp_id INTEGER,skill_code TEXT,status TEXT,attempts INTEGER,correct INTEGER,streak INTEGER,best_streak INTEGER,ease REAL,interval_days INTEGER,due_at DATETIME,first_seen_at DATETIME,mastered_at DATETIME,updated_at DATETIME,PRIMARY KEY(child_id,kp_id,skill_code))`,
		`CREATE TABLE daily_stats(child_id INTEGER,stat_date TEXT,practice_sec INTEGER DEFAULT 0,attempts INTEGER DEFAULT 0,correct INTEGER DEFAULT 0,newly_mastered INTEGER DEFAULT 0,review_done INTEGER DEFAULT 0,checked_in BOOLEAN DEFAULT FALSE,PRIMARY KEY(child_id,stat_date))`,
		`CREATE TABLE flower_ledger(id INTEGER PRIMARY KEY AUTOINCREMENT,child_id INTEGER,delta INTEGER,reason TEXT,ref_type TEXT,ref_id INTEGER,created_at DATETIME)`,
		`INSERT INTO children(id,name,flowers,created_at) VALUES(1,'安安',0,CURRENT_TIMESTAMP)`,
		`INSERT INTO subjects VALUES(1,'chengyu'),(2,'phrase')`,
		`INSERT INTO modules VALUES(10,1,'daily',1),(20,2,'greet',1)`,
		`INSERT INTO knowledge_points VALUES
		(100,10,'cy001','一心一意','{"kind":"chengyu","pinyin":"yì xīn yì yì","meaning":"集中精神，做事专心","example":"做作业要一心一意。","wrong":["三心二意","慢慢来","随便玩玩"]}',1,1),
		(101,10,'cy002','二话不说','{"kind":"chengyu","pinyin":"èr huà bù shuō","meaning":"不说别的话，立刻行动","example":"听到口令，他二话不说就跑了起来。","wrong":["说很多话","慢慢商量","再想一想"]}',1,2),
		(102,10,'cy003','三心二意','{"kind":"chengyu","pinyin":"sān xīn èr yì","meaning":"心思不专一，不坚定","example":"学习不能三心二意。","wrong":["一心一意","非常专心","坚持到底"]}',1,3),
		(103,10,'cy004','五颜六色','{"kind":"chengyu","pinyin":"wǔ yán liù sè","meaning":"形容色彩繁多","example":"花园里开着五颜六色的花。","wrong":["只有一种颜色","黑漆漆的","灰蒙蒙的"]}',1,4),
		(200,20,'ph0001','Good morning.','{"kind":"phrase","zh":"早上好。"}',1,1)`,
		`INSERT INTO questions VALUES
		(1000,100,'meaning','choice','这个成语是什么意思？','[{"label":"集中精神，做事专心"},{"label":"三心二意"},{"label":"慢慢来"},{"label":"随便玩玩"}]','{"index":0}','{"kind":"char","text":"一心一意"}','{"text":"一心一意","lang":"zh-CN"}',1),
		(1001,100,'pick','choice','听一听，点出这个成语','[{"label":"一心一意"},{"label":"二话不说"},{"label":"三心二意"},{"label":"五颜六色"}]','{"index":0}','{"kind":"sound"}','{"text":"一心一意","lang":"zh-CN"}',1),
		(1002,101,'meaning','choice','这个成语是什么意思？','[{"label":"不说别的话，立刻行动"},{"label":"说很多话"},{"label":"慢慢商量"},{"label":"再想一想"}]','{"index":0}','{"kind":"char","text":"二话不说"}','{"text":"二话不说","lang":"zh-CN"}',1),
		(1003,101,'pick','choice','听一听，点出这个成语','[{"label":"二话不说"},{"label":"一心一意"},{"label":"三心二意"},{"label":"五颜六色"}]','{"index":0}','{"kind":"sound"}','{"text":"二话不说","lang":"zh-CN"}',1),
		(1004,102,'meaning','choice','这个成语是什么意思？','[{"label":"心思不专一，不坚定"},{"label":"一心一意"},{"label":"非常专心"},{"label":"坚持到底"}]','{"index":0}','{"kind":"char","text":"三心二意"}','{"text":"三心二意","lang":"zh-CN"}',1),
		(1005,103,'meaning','choice','这个成语是什么意思？','[{"label":"形容色彩繁多"},{"label":"只有一种颜色"},{"label":"黑漆漆的"},{"label":"灰蒙蒙的"}]','{"index":0}','{"kind":"char","text":"五颜六色"}','{"text":"五颜六色","lang":"zh-CN"}',1),
		(2000,200,'listen_zh','choice','听一听','[{"label":"早上好。"}]','{"index":0}','{}','{"text":"Good morning."}',1)`,
	}
}
