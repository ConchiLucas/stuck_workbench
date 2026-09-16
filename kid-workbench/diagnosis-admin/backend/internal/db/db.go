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

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
