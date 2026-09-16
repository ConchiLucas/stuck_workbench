package db

import (
	"fmt"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func DSNFromEnv() string {
	if v := os.Getenv("APP_DSN"); v != "" {
		return v
	}
	host := env("APP_DB_HOST", "127.0.0.1")
	port := env("APP_DB_PORT", "15432")
	user := env("APP_DB_USER", "conchi")
	pass := env("APP_DB_PASSWORD", "conchi123456")
	name := env("APP_DB_NAME", "study_workbench")
	cfg := env("APP_DB_CONFIG", "sslmode=disable")
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s %s",
		host, user, pass, name, port, cfg)
}

func OpenPostgres(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
}

func OpenSQLite(dsn string) (*gorm.DB, error) {
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		return nil, err
	}
	if err := gdb.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, err
	}
	return gdb, nil
}

func Migrate(gdb *gorm.DB) error {
	sql := `
CREATE TABLE IF NOT EXISTS question_tasks (
  id BIGSERIAL PRIMARY KEY,
  subject_code VARCHAR(30) NOT NULL,
  title VARCHAR(80) NOT NULL,
  module_code VARCHAR(50) NOT NULL,
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  target_count INT NOT NULL DEFAULT 10,
  status VARCHAR(20) NOT NULL DEFAULT 'draft',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_question_tasks_subject_status ON question_tasks(subject_code, status);
CREATE INDEX IF NOT EXISTS idx_question_tasks_module ON question_tasks(module_code);

CREATE TABLE IF NOT EXISTS question_task_items (
  id BIGSERIAL PRIMARY KEY,
  task_id BIGINT NOT NULL REFERENCES question_tasks(id) ON DELETE CASCADE,
  seq INT NOT NULL,
  kp_id BIGINT NOT NULL,
  question_id BIGINT NOT NULL,
  UNIQUE(task_id, seq),
  UNIQUE(task_id, question_id)
);`
	if gdb.Dialector.Name() == "sqlite" {
		sql = `
CREATE TABLE IF NOT EXISTS question_tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  subject_code TEXT NOT NULL,
  title TEXT NOT NULL,
  module_code TEXT NOT NULL,
  module_name TEXT NOT NULL DEFAULT '',
  target_count INTEGER NOT NULL DEFAULT 10,
  status TEXT NOT NULL DEFAULT 'draft',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_question_tasks_subject_status ON question_tasks(subject_code, status);
CREATE INDEX IF NOT EXISTS idx_question_tasks_module ON question_tasks(module_code);

CREATE TABLE IF NOT EXISTS question_task_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id INTEGER NOT NULL REFERENCES question_tasks(id) ON DELETE CASCADE,
  seq INTEGER NOT NULL,
  kp_id INTEGER NOT NULL,
  question_id INTEGER NOT NULL,
  UNIQUE(task_id, seq),
  UNIQUE(task_id, question_id)
);`
	}
	return gdb.Exec(sql).Error
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
