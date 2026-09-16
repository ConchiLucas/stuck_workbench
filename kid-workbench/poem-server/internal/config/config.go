package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/conchi/study-learning/mastery"
)

type Database struct {
	Driver   string
	DSN      string
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	Options  string
}

func (d Database) ConnectionString() string {
	if d.DSN != "" {
		return d.DSN
	}
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s %s",
		d.Host, d.User, d.Password, d.Name, d.Port, d.Options)
}

type Config struct {
	Addr            string
	DB              Database
	ContentAdminURL string
	Mastery         mastery.Config
}

func Load() Config {
	return Config{
		Addr: env("APP_ADDR", ":19161"),
		DB: Database{
			Driver:   env("APP_DB_DRIVER", "postgres"),
			DSN:      os.Getenv("APP_DSN"),
			Host:     env("APP_DB_HOST", "127.0.0.1"),
			Port:     env("APP_DB_PORT", "15432"),
			Name:     env("APP_DB_NAME", "study_workbench"),
			User:     env("APP_DB_USER", "conchi"),
			Password: env("APP_DB_PASSWORD", "conchi123456"),
			Options:  env("APP_DB_CONFIG", "sslmode=disable TimeZone=Asia/Shanghai"),
		},
		ContentAdminURL: strings.TrimRight(env("APP_CONTENT_ADMIN_URL", "http://127.0.0.1:19091"), "/"),
		Mastery: mastery.Config{
			BaseMasterStreak: envInt("MASTERY_BASE_STREAK", 2), MinAccuracy: 0.8,
			ShakyMinAttempts: 3, ShakyAccuracy: 0.6, EaseMin: 1.3, EaseMax: 2.8,
			EaseUp: 0.1, EaseDown: 0.2, MaxIntervalDays: 60,
		},
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if value, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return value
	}
	return fallback
}
