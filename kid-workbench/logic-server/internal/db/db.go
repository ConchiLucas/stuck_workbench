package db

import (
	"context"
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/conchi/logic-server/internal/config"
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

type Checker struct{ DB *gorm.DB }

func (c Checker) Check(ctx context.Context) error {
	if c.DB == nil {
		return fmt.Errorf("database is not configured")
	}
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
