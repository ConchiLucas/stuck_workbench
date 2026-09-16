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
	if err := gdb.Exec(`CREATE TABLE IF NOT EXISTS literacy_writing_templates (version VARCHAR(64) NOT NULL, kp_id BIGINT NOT NULL, content TEXT NOT NULL, created_at TIMESTAMP NOT NULL, PRIMARY KEY (version, kp_id))`).Error; err != nil {
		return err
	}
	if err := gdb.Exec(`CREATE TABLE IF NOT EXISTS material_revisions (revision_id VARCHAR(64) PRIMARY KEY, subject_code VARCHAR(30) NOT NULL, kp_id BIGINT NOT NULL, content TEXT NOT NULL, media TEXT NOT NULL, source_revision VARCHAR(64) NOT NULL DEFAULT '', created_at TIMESTAMP NOT NULL)`).Error; err != nil {
		return err
	}
	sql := `
CREATE TABLE IF NOT EXISTS literacy_assets (
  kp_id BIGINT PRIMARY KEY,
  char_text VARCHAR(8) NOT NULL,
  module_code VARCHAR(50) NOT NULL DEFAULT '',
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  module_order INT NOT NULL DEFAULT 0,
  kp_order INT NOT NULL DEFAULT 0,
  needs_sense_image BOOLEAN NOT NULL,
  needs_sense_image_override BOOLEAN,
  glyph_image_url VARCHAR(512) NOT NULL DEFAULT '',
  sense_image_url VARCHAR(512) NOT NULL DEFAULT '',
  speech_audio_url VARCHAR(512) NOT NULL DEFAULT '',
  synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`
	if gdb.Dialector.Name() == "sqlite" {
		sql = `
CREATE TABLE IF NOT EXISTS literacy_assets (
  kp_id INTEGER PRIMARY KEY,
  char_text TEXT NOT NULL,
  module_code TEXT NOT NULL DEFAULT '',
  module_name TEXT NOT NULL DEFAULT '',
  module_order INTEGER NOT NULL DEFAULT 0,
  kp_order INTEGER NOT NULL DEFAULT 0,
  needs_sense_image INTEGER NOT NULL,
  needs_sense_image_override INTEGER,
  glyph_image_url TEXT NOT NULL DEFAULT '',
  sense_image_url TEXT NOT NULL DEFAULT '',
  speech_audio_url TEXT NOT NULL DEFAULT '',
  synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	}
	if err := gdb.Exec(sql).Error; err != nil {
		return err
	}
	// Additive upgrades; preserve existing assets and revisions.
	for _, column := range []struct{ table, name, ddl string }{
		{"literacy_assets", "writing_template_version", "TEXT NOT NULL DEFAULT ''"},
		{"literacy_assets", "material_epoch", "TEXT NOT NULL DEFAULT ''"},
		{"literacy_assets", "material_pending", "BOOLEAN NOT NULL DEFAULT FALSE"},
		{"material_revisions", "source_revision", "VARCHAR(64) NOT NULL DEFAULT ''"},
	} {
		if !gdb.Migrator().HasColumn(column.table, column.name) {
			if err := gdb.Exec("ALTER TABLE " + column.table + " ADD COLUMN " + column.name + " " + column.ddl).Error; err != nil {
				return err
			}
		}
	}
	// Existing DBs: add speech_audio_url if missing.
	if gdb.Dialector.Name() == "sqlite" {
		_ = gdb.Exec(`ALTER TABLE literacy_assets ADD COLUMN speech_audio_url TEXT NOT NULL DEFAULT ''`).Error
	} else {
		_ = gdb.Exec(`ALTER TABLE literacy_assets ADD COLUMN IF NOT EXISTS speech_audio_url VARCHAR(512) NOT NULL DEFAULT ''`).Error
	}

	pinyinSQL := `
CREATE TABLE IF NOT EXISTS pinyin_assets (
  kp_id BIGINT PRIMARY KEY,
  letter VARCHAR(16) NOT NULL DEFAULT '',
  module_code VARCHAR(50) NOT NULL DEFAULT '',
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  module_order INT NOT NULL DEFAULT 0,
  kp_order INT NOT NULL DEFAULT 0,
  solo_text VARCHAR(16) NOT NULL DEFAULT '',
  word_text VARCHAR(16) NOT NULL DEFAULT '',
  word_examples TEXT NOT NULL DEFAULT '',
  word_example_speech_urls TEXT NOT NULL DEFAULT '',
  solo_speech_url VARCHAR(512) NOT NULL DEFAULT '',
  word_speech_url VARCHAR(512) NOT NULL DEFAULT '',
  glyph_image_url VARCHAR(512) NOT NULL DEFAULT '',
  synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`
	if gdb.Dialector.Name() == "sqlite" {
		pinyinSQL = `
CREATE TABLE IF NOT EXISTS pinyin_assets (
  kp_id INTEGER PRIMARY KEY,
  letter TEXT NOT NULL DEFAULT '',
  module_code TEXT NOT NULL DEFAULT '',
  module_name TEXT NOT NULL DEFAULT '',
  module_order INTEGER NOT NULL DEFAULT 0,
  kp_order INTEGER NOT NULL DEFAULT 0,
  solo_text TEXT NOT NULL DEFAULT '',
  word_text TEXT NOT NULL DEFAULT '',
  word_examples TEXT NOT NULL DEFAULT '',
  word_example_speech_urls TEXT NOT NULL DEFAULT '',
  solo_speech_url TEXT NOT NULL DEFAULT '',
  word_speech_url TEXT NOT NULL DEFAULT '',
  glyph_image_url TEXT NOT NULL DEFAULT '',
  synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	}
	if err := gdb.Exec(pinyinSQL).Error; err != nil {
		return err
	}
	// Existing DBs: add glyph_image_url if missing.
	if gdb.Dialector.Name() == "sqlite" {
		_ = gdb.Exec(`ALTER TABLE pinyin_assets ADD COLUMN glyph_image_url TEXT NOT NULL DEFAULT ''`).Error
		_ = gdb.Exec(`ALTER TABLE pinyin_assets ADD COLUMN word_examples TEXT NOT NULL DEFAULT ''`).Error
		_ = gdb.Exec(`ALTER TABLE pinyin_assets ADD COLUMN word_example_speech_urls TEXT NOT NULL DEFAULT ''`).Error
	} else {
		_ = gdb.Exec(`ALTER TABLE pinyin_assets ADD COLUMN IF NOT EXISTS glyph_image_url VARCHAR(512) NOT NULL DEFAULT ''`).Error
		_ = gdb.Exec(`ALTER TABLE pinyin_assets ADD COLUMN IF NOT EXISTS word_examples TEXT NOT NULL DEFAULT ''`).Error
		_ = gdb.Exec(`ALTER TABLE pinyin_assets ADD COLUMN IF NOT EXISTS word_example_speech_urls TEXT NOT NULL DEFAULT ''`).Error
	}
	if err := migratePinyinSyllables(gdb); err != nil {
		return err
	}

	mathSQL := `
CREATE TABLE IF NOT EXISTS math_assets (
  kp_id BIGINT PRIMARY KEY,
  title VARCHAR(80) NOT NULL DEFAULT '',
  kind VARCHAR(24) NOT NULL DEFAULT '',
  payload TEXT NOT NULL DEFAULT '{}',
  difficulty INT NOT NULL DEFAULT 1,
  module_code VARCHAR(50) NOT NULL DEFAULT '',
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  module_order INT NOT NULL DEFAULT 0,
  kp_order INT NOT NULL DEFAULT 0,
  glyph_image_url VARCHAR(512) NOT NULL DEFAULT '',
  speech_audio_url VARCHAR(512) NOT NULL DEFAULT '',
  speech_text VARCHAR(120) NOT NULL DEFAULT '',
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`
	if gdb.Dialector.Name() == "sqlite" {
		mathSQL = `
CREATE TABLE IF NOT EXISTS math_assets (
  kp_id INTEGER PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL DEFAULT '',
  payload TEXT NOT NULL DEFAULT '{}',
  difficulty INTEGER NOT NULL DEFAULT 1,
  module_code TEXT NOT NULL DEFAULT '',
  module_name TEXT NOT NULL DEFAULT '',
  module_order INTEGER NOT NULL DEFAULT 0,
  kp_order INTEGER NOT NULL DEFAULT 0,
  glyph_image_url TEXT NOT NULL DEFAULT '',
  speech_audio_url TEXT NOT NULL DEFAULT '',
  speech_text TEXT NOT NULL DEFAULT '',
  enabled INTEGER NOT NULL DEFAULT 1,
  synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	}
	if err := gdb.Exec(mathSQL).Error; err != nil {
		return err
	}
	if gdb.Dialector.Name() == "sqlite" {
		_ = gdb.Exec(`ALTER TABLE math_assets ADD COLUMN glyph_image_url TEXT NOT NULL DEFAULT ''`).Error
		_ = gdb.Exec(`ALTER TABLE math_assets ADD COLUMN speech_audio_url TEXT NOT NULL DEFAULT ''`).Error
		_ = gdb.Exec(`ALTER TABLE math_assets ADD COLUMN speech_text TEXT NOT NULL DEFAULT ''`).Error
		_ = gdb.Exec(`ALTER TABLE math_assets ADD COLUMN enabled INTEGER NOT NULL DEFAULT 1`).Error
	} else {
		_ = gdb.Exec(`ALTER TABLE math_assets ADD COLUMN IF NOT EXISTS glyph_image_url VARCHAR(512) NOT NULL DEFAULT ''`).Error
		_ = gdb.Exec(`ALTER TABLE math_assets ADD COLUMN IF NOT EXISTS speech_audio_url VARCHAR(512) NOT NULL DEFAULT ''`).Error
		_ = gdb.Exec(`ALTER TABLE math_assets ADD COLUMN IF NOT EXISTS speech_text VARCHAR(120) NOT NULL DEFAULT ''`).Error
		_ = gdb.Exec(`ALTER TABLE math_assets ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT TRUE`).Error
	}

	englishSQL := `
CREATE TABLE IF NOT EXISTS english_assets (
  kp_id BIGINT PRIMARY KEY,
  word_text VARCHAR(32) NOT NULL DEFAULT '',
  module_code VARCHAR(50) NOT NULL DEFAULT '',
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  module_order INT NOT NULL DEFAULT 0,
  kp_order INT NOT NULL DEFAULT 0,
  needs_sense_image BOOLEAN NOT NULL,
  needs_sense_image_override BOOLEAN,
  glyph_image_url VARCHAR(512) NOT NULL DEFAULT '',
  sense_image_url VARCHAR(512) NOT NULL DEFAULT '',
  speech_audio_url VARCHAR(512) NOT NULL DEFAULT '',
  synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`
	if gdb.Dialector.Name() == "sqlite" {
		englishSQL = `
CREATE TABLE IF NOT EXISTS english_assets (
  kp_id INTEGER PRIMARY KEY,
  word_text TEXT NOT NULL DEFAULT '',
  module_code TEXT NOT NULL DEFAULT '',
  module_name TEXT NOT NULL DEFAULT '',
  module_order INTEGER NOT NULL DEFAULT 0,
  kp_order INTEGER NOT NULL DEFAULT 0,
  needs_sense_image INTEGER NOT NULL,
  needs_sense_image_override INTEGER,
  glyph_image_url TEXT NOT NULL DEFAULT '',
  sense_image_url TEXT NOT NULL DEFAULT '',
  speech_audio_url TEXT NOT NULL DEFAULT '',
  synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	}
	if err := gdb.Exec(englishSQL).Error; err != nil {
		return err
	}
	if gdb.Dialector.Name() == "sqlite" {
		if err := gdb.Exec(`
CREATE TABLE IF NOT EXISTS english_sentences (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  text TEXT NOT NULL,
  tokens_json TEXT NOT NULL,
  target_kp_id INTEGER NOT NULL,
  speech_audio_url TEXT NOT NULL DEFAULT '',
  content_hash TEXT NOT NULL DEFAULT '',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`).Error; err != nil {
			return err
		}
		if err := gdb.Exec(`
CREATE TABLE IF NOT EXISTS english_passages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  passage TEXT NOT NULL,
  prompt TEXT NOT NULL,
  answer_kp_id INTEGER NOT NULL,
  option_kp_ids_json TEXT NOT NULL,
  content_hash TEXT NOT NULL DEFAULT '',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`).Error; err != nil {
			return err
		}
	} else {
		if err := gdb.Exec(`
CREATE TABLE IF NOT EXISTS english_sentences (
  id BIGSERIAL PRIMARY KEY,
  code VARCHAR(64) NOT NULL UNIQUE,
  text VARCHAR(200) NOT NULL,
  tokens_json TEXT NOT NULL,
  target_kp_id BIGINT NOT NULL,
  speech_audio_url VARCHAR(512) NOT NULL DEFAULT '',
  content_hash VARCHAR(64) NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`).Error; err != nil {
			return err
		}
		if err := gdb.Exec(`
CREATE TABLE IF NOT EXISTS english_passages (
  id BIGSERIAL PRIMARY KEY,
  code VARCHAR(64) NOT NULL UNIQUE,
  passage TEXT NOT NULL,
  prompt VARCHAR(200) NOT NULL,
  answer_kp_id BIGINT NOT NULL,
  option_kp_ids_json TEXT NOT NULL,
  content_hash VARCHAR(64) NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`).Error; err != nil {
			return err
		}
	}

	scienceSQL := `
CREATE TABLE IF NOT EXISTS science_assets (
  kp_id BIGINT PRIMARY KEY,
  title VARCHAR(64) NOT NULL DEFAULT '',
  module_code VARCHAR(50) NOT NULL DEFAULT '',
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  module_order INT NOT NULL DEFAULT 0,
  kp_order INT NOT NULL DEFAULT 0,
  needs_sense_image BOOLEAN NOT NULL,
  needs_sense_image_override BOOLEAN,
  glyph_image_url VARCHAR(512) NOT NULL DEFAULT '',
  sense_image_url VARCHAR(512) NOT NULL DEFAULT '',
  speech_audio_url VARCHAR(512) NOT NULL DEFAULT '',
  synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`
	if gdb.Dialector.Name() == "sqlite" {
		scienceSQL = `
CREATE TABLE IF NOT EXISTS science_assets (
  kp_id INTEGER PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  module_code TEXT NOT NULL DEFAULT '',
  module_name TEXT NOT NULL DEFAULT '',
  module_order INTEGER NOT NULL DEFAULT 0,
  kp_order INTEGER NOT NULL DEFAULT 0,
  needs_sense_image INTEGER NOT NULL,
  needs_sense_image_override INTEGER,
  glyph_image_url TEXT NOT NULL DEFAULT '',
  sense_image_url TEXT NOT NULL DEFAULT '',
  speech_audio_url TEXT NOT NULL DEFAULT '',
  synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	}
	if err := gdb.Exec(scienceSQL).Error; err != nil {
		return err
	}
	if gdb.Dialector.Name() == "sqlite" {
		columns := []struct {
			name string
			ddl  string
		}{
			{name: "summary", ddl: `ALTER TABLE science_assets ADD COLUMN summary TEXT NOT NULL DEFAULT ''`},
			{name: "explanation", ddl: `ALTER TABLE science_assets ADD COLUMN explanation TEXT NOT NULL DEFAULT ''`},
			{name: "fun_fact", ddl: `ALTER TABLE science_assets ADD COLUMN fun_fact TEXT NOT NULL DEFAULT ''`},
			{name: "review_status", ddl: `ALTER TABLE science_assets ADD COLUMN review_status TEXT NOT NULL DEFAULT 'published'`},
			{name: "reviewed_at", ddl: `ALTER TABLE science_assets ADD COLUMN reviewed_at DATETIME`},
			{name: "content_version", ddl: `ALTER TABLE science_assets ADD COLUMN content_version INTEGER NOT NULL DEFAULT 1`},
			{name: "glyph_object_key", ddl: `ALTER TABLE science_assets ADD COLUMN glyph_object_key TEXT NOT NULL DEFAULT ''`},
			{name: "sense_object_key", ddl: `ALTER TABLE science_assets ADD COLUMN sense_object_key TEXT NOT NULL DEFAULT ''`},
			{name: "speech_object_key", ddl: `ALTER TABLE science_assets ADD COLUMN speech_object_key TEXT NOT NULL DEFAULT ''`},
		}
		for _, column := range columns {
			has, err := sqliteHasColumn(gdb, "science_assets", column.name)
			if err != nil {
				return err
			}
			if !has {
				if err := gdb.Exec(column.ddl).Error; err != nil {
					return err
				}
			}
		}
	} else {
		if err := gdb.Exec(`
ALTER TABLE science_assets ADD COLUMN IF NOT EXISTS summary TEXT NOT NULL DEFAULT '';
ALTER TABLE science_assets ADD COLUMN IF NOT EXISTS explanation TEXT NOT NULL DEFAULT '';
ALTER TABLE science_assets ADD COLUMN IF NOT EXISTS fun_fact TEXT NOT NULL DEFAULT '';
ALTER TABLE science_assets ADD COLUMN IF NOT EXISTS review_status VARCHAR(16) NOT NULL DEFAULT 'published';
ALTER TABLE science_assets ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ;
ALTER TABLE science_assets ADD COLUMN IF NOT EXISTS content_version INT NOT NULL DEFAULT 1;
ALTER TABLE science_assets ADD COLUMN IF NOT EXISTS glyph_object_key TEXT NOT NULL DEFAULT '';
ALTER TABLE science_assets ADD COLUMN IF NOT EXISTS sense_object_key TEXT NOT NULL DEFAULT '';
ALTER TABLE science_assets ADD COLUMN IF NOT EXISTS speech_object_key TEXT NOT NULL DEFAULT '';
`).Error; err != nil {
			return err
		}
	}

	if err := migratePoemAssets(gdb); err != nil {
		return err
	}
	if err := migratePhraseSpeech(gdb); err != nil {
		return err
	}
	if err := migrateChengyuSpeech(gdb); err != nil {
		return err
	}

	return nil
}

func migrateChengyuSpeech(gdb *gorm.DB) error {
	sql := `
CREATE TABLE IF NOT EXISTS chengyu_item_speech (
  kp_id BIGINT NOT NULL,
  kind VARCHAR(16) NOT NULL,
  text TEXT NOT NULL DEFAULT '',
  sha256 VARCHAR(64) NOT NULL DEFAULT '',
  data BYTEA NOT NULL,
  PRIMARY KEY (kp_id, kind)
)`
	if gdb.Dialector.Name() == "sqlite" {
		sql = `
CREATE TABLE IF NOT EXISTS chengyu_item_speech (
  kp_id INTEGER NOT NULL,
  kind TEXT NOT NULL,
  text TEXT NOT NULL DEFAULT '',
  sha256 TEXT NOT NULL DEFAULT '',
  data BLOB NOT NULL,
  PRIMARY KEY (kp_id, kind)
)`
	}
	return gdb.Exec(sql).Error
}

func migratePhraseSpeech(gdb *gorm.DB) error {
	sql := `
CREATE TABLE IF NOT EXISTS phrase_item_speech (
  kp_id BIGINT PRIMARY KEY,
  text TEXT NOT NULL DEFAULT '',
  sha256 VARCHAR(64) NOT NULL DEFAULT '',
  data BYTEA NOT NULL
)`
	if gdb.Dialector.Name() == "sqlite" {
		sql = `
CREATE TABLE IF NOT EXISTS phrase_item_speech (
  kp_id INTEGER PRIMARY KEY,
  text TEXT NOT NULL DEFAULT '',
  sha256 TEXT NOT NULL DEFAULT '',
  data BLOB NOT NULL
)`
	}
	return gdb.Exec(sql).Error
}

func migratePoemAssets(gdb *gorm.DB) error {
	sql := `
CREATE TABLE IF NOT EXISTS poem_assets (
  kp_id BIGINT PRIMARY KEY,
  title VARCHAR(80) NOT NULL DEFAULT '',
  author VARCHAR(40) NOT NULL DEFAULT '',
  line1 VARCHAR(80) NOT NULL DEFAULT '',
  line2 VARCHAR(80) NOT NULL DEFAULT '',
  lines_json TEXT NOT NULL DEFAULT '[]',
  difficulty INT NOT NULL DEFAULT 1,
  module_code VARCHAR(50) NOT NULL DEFAULT '',
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  module_order INT NOT NULL DEFAULT 0,
  kp_order INT NOT NULL DEFAULT 0,
  synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`
	if gdb.Dialector.Name() == "sqlite" {
		sql = `
CREATE TABLE IF NOT EXISTS poem_assets (
  kp_id INTEGER PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  author TEXT NOT NULL DEFAULT '',
  line1 TEXT NOT NULL DEFAULT '',
  line2 TEXT NOT NULL DEFAULT '',
  lines_json TEXT NOT NULL DEFAULT '[]',
  difficulty INTEGER NOT NULL DEFAULT 1,
  module_code TEXT NOT NULL DEFAULT '',
  module_name TEXT NOT NULL DEFAULT '',
  module_order INTEGER NOT NULL DEFAULT 0,
  kp_order INTEGER NOT NULL DEFAULT 0,
  synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	}
	return gdb.Exec(sql).Error
}

func migratePinyinSyllables(gdb *gorm.DB) error {
	sql := `
CREATE TABLE IF NOT EXISTS pinyin_syllable_assets (
  id BIGSERIAL PRIMARY KEY,
  initial_text VARCHAR(16) NOT NULL,
  final_text VARCHAR(16) NOT NULL,
  tone SMALLINT NOT NULL,
  syllable_text VARCHAR(32) NOT NULL,
  speech_text VARCHAR(32) NOT NULL,
  speech_url VARCHAR(512) NOT NULL DEFAULT '',
  difficulty SMALLINT NOT NULL DEFAULT 1,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(initial_text, final_text, tone)
)`
	if gdb.Dialector.Name() == "sqlite" {
		sql = `
CREATE TABLE IF NOT EXISTS pinyin_syllable_assets (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  initial_text TEXT NOT NULL,
  final_text TEXT NOT NULL,
  tone INTEGER NOT NULL,
  syllable_text TEXT NOT NULL,
  speech_text TEXT NOT NULL,
  speech_url TEXT NOT NULL DEFAULT '',
  difficulty INTEGER NOT NULL DEFAULT 1,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(initial_text, final_text, tone)
)`
	}
	if err := gdb.Exec(sql).Error; err != nil {
		return err
	}

	seed := [][5]any{
		{"b", "ā", 1, "bā", "八"}, {"p", "ā", 1, "pā", "趴"},
		{"m", "ā", 1, "mā", "妈"}, {"f", "ā", 1, "fā", "发"},
		{"d", "à", 4, "dà", "大"}, {"t", "à", 4, "tà", "踏"},
		{"n", "à", 4, "nà", "那"}, {"l", "à", 4, "là", "辣"},
		{"j", "ī", 1, "jī", "鸡"}, {"q", "ī", 1, "qī", "七"},
		{"x", "ī", 1, "xī", "西"}, {"y", "ī", 1, "yī", "一"},
		{"zh", "ū", 1, "zhū", "猪"}, {"ch", "ū", 1, "chū", "出"},
		{"sh", "ū", 1, "shū", "书"}, {"r", "ú", 2, "rú", "如"},
	}
	for _, row := range seed {
		if err := gdb.Exec(`
INSERT INTO pinyin_syllable_assets
  (initial_text, final_text, tone, syllable_text, speech_text, enabled)
VALUES (?, ?, ?, ?, ?, TRUE)
ON CONFLICT(initial_text, final_text, tone) DO NOTHING`, row[:]...).Error; err != nil {
			return err
		}
	}
	return nil
}

func sqliteHasColumn(gdb *gorm.DB, table, column string) (bool, error) {
	var rows []struct {
		Name string `gorm:"column:name"`
	}
	if err := gdb.Raw("PRAGMA table_info(" + table + ")").Scan(&rows).Error; err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.Name == column {
			return true, nil
		}
	}
	return false, nil
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
