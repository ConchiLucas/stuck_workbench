package db

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/conchi/pinyin-server/internal/config"
)

func Open(cfg config.Database) (*gorm.DB, error) {
	switch cfg.Driver {
	case "postgres":
		return gorm.Open(postgres.Open(cfg.ConnectionString()), &gorm.Config{TranslateError: true})
	case "sqlite":
		return gorm.Open(sqlite.Open(cfg.ConnectionString()), &gorm.Config{TranslateError: true})
	default:
		return nil, fmt.Errorf("unsupported database driver %q", cfg.Driver)
	}
}
