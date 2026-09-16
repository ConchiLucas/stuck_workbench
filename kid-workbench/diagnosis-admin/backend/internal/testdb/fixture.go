package testdb

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/study-diagnosis-admin/internal/db"
)

func Open(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(SQL).Error)
	return gdb
}

const SQL = `
CREATE TABLE children (
  id INTEGER PRIMARY KEY, name TEXT, grade TEXT, flowers INTEGER DEFAULT 0
);
CREATE TABLE subjects (
  id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INTEGER
);
CREATE TABLE modules (
  id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER
);
CREATE TABLE knowledge_points (
  id INTEGER PRIMARY KEY, module_id INTEGER, code TEXT, title TEXT, payload TEXT DEFAULT '{}',
  difficulty INTEGER DEFAULT 1, order_no INTEGER
);
CREATE TABLE questions (
  id INTEGER PRIMARY KEY, kp_id INTEGER, code TEXT, type TEXT, stem TEXT,
  options TEXT, answer TEXT, difficulty INTEGER
);
CREATE TABLE attempts (
  id INTEGER PRIMARY KEY, child_id INTEGER, kp_id INTEGER, question_id INTEGER,
  is_correct INTEGER, cost_ms INTEGER, source TEXT, client_id TEXT, created_at DATETIME
);
CREATE TABLE mastery_states (
  child_id INTEGER, kp_id INTEGER, status TEXT, attempts INTEGER, correct INTEGER,
  streak INTEGER, best_streak INTEGER, ease REAL, interval_days INTEGER,
  due_at DATETIME, first_seen_at DATETIME, mastered_at DATETIME, updated_at DATETIME,
  PRIMARY KEY (child_id, kp_id)
);
CREATE TABLE mastery_skills (
  child_id INTEGER, kp_id INTEGER, skill_code TEXT, status TEXT, attempts INTEGER, correct INTEGER,
  streak INTEGER, best_streak INTEGER, ease REAL, interval_days INTEGER,
  due_at DATETIME, first_seen_at DATETIME, mastered_at DATETIME, updated_at DATETIME,
  PRIMARY KEY (child_id, kp_id, skill_code)
);
CREATE TABLE study_plans (
  id INTEGER PRIMARY KEY, child_id INTEGER, plan_date TEXT, seq_no INTEGER, subject_code TEXT,
  status TEXT, target_count INTEGER, done_count INTEGER, correct_count INTEGER
);
CREATE TABLE plan_items (
  id INTEGER PRIMARY KEY, plan_id INTEGER, seq INTEGER, kp_id INTEGER, question_id INTEGER,
  status TEXT, tries INTEGER, cost_ms INTEGER, picks TEXT, question_answer TEXT, answered_at DATETIME,
  question_snapshot TEXT DEFAULT ''
);

INSERT INTO children(id, name, grade) VALUES (1, '卢沁一', '大班');
INSERT INTO subjects(id, code, name, icon, order_no) VALUES
 (1, 'literacy', '识字', '📖', 1),
 (2, 'english', '英语', '🌍', 2),
 (3, 'game', '游戏', '🎮', 99);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES
 (1, 1, 'g1', '第1组', 1),
 (2, 2, 'greet', '问候', 1),
 (3, 3, 'levels', '关卡', 1);
INSERT INTO knowledge_points(id, module_id, code, title, order_no) VALUES
 (10, 1, 'yi', '一', 1),
 (20, 2, 'hello', 'Hello!', 1),
 (30, 3, 'lv1', '第1关', 1);
INSERT INTO questions(id, kp_id, code, type, stem, options, answer, difficulty) VALUES
 (101, 10, 'glyph_sense', 'choice', '看字', '[]', '{"index":0}', 1),
 (102, 10, 'write_char', 'write', '手写', '[]', '{}', 1),
 (201, 20, 'listen', 'choice', '听单词', '[{"label":"Hi"},{"label":"Hello"}]', '{"index":1}', 1);

INSERT INTO mastery_states(child_id, kp_id, status, attempts, correct, streak, best_streak, ease, interval_days, updated_at)
VALUES
 (1, 10, 'learning', 6, 3, 0, 1, 2.5, 0, '2026-09-01'),
 (1, 20, 'shaky', 8, 2, 0, 1, 2.5, 0, '2026-09-01'),
 (1, 30, 'shaky', 4, 0, 0, 0, 2.5, 0, '2026-09-01');

INSERT INTO mastery_skills(child_id, kp_id, skill_code, status, attempts, correct, streak, best_streak, ease, interval_days, updated_at)
VALUES
 (1, 10, 'glyph_sense', 'mastered', 3, 3, 2, 2, 2.5, 1, '2026-09-01'),
 (1, 10, 'write_char', 'shaky', 3, 0, 0, 0, 2.5, 0, '2026-09-01'),
 (1, 20, 'listen', 'shaky', 8, 2, 0, 1, 2.5, 0, '2026-09-01');

INSERT INTO attempts(id, child_id, kp_id, question_id, is_correct, cost_ms, source, client_id, created_at) VALUES
 (1, 1, 20, 201, 0, 18000, 'quiz', 'a1', '2026-09-07 10:00:00'),
 (2, 1, 20, 201, 0, 4000, 'quiz', 'a2', '2026-09-07 10:05:00'),
 (3, 1, 10, 102, 0, 2000, 'quiz', 'a3', '2026-09-07 10:10:00');

INSERT INTO study_plans(id, child_id, plan_date, seq_no, subject_code, status, target_count, done_count, correct_count)
VALUES (9, 1, '2026-09-07', 1, 'english', 'done', 1, 1, 0);
INSERT INTO plan_items(id, plan_id, seq, kp_id, question_id, status, tries, cost_ms, picks, question_answer, answered_at)
VALUES (90, 9, 1, 20, 201, 'wrong', 2, 18000, '0,0', '{"index":1}', '2026-09-07 10:05:00');
`
