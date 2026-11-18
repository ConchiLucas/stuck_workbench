package config

import (
	"fmt"
	"os"
)

type Database struct {
	Driver, DSN, Host, Port, Name, User, Password, Options string
}

func (d Database) ConnectionString() string {
	if d.DSN != "" {
		return d.DSN
	}
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s %s", d.Host, d.User, d.Password, d.Name, d.Port, d.Options)
}

type Config struct {
	Addr string
	DB   Database
}

func Load() Config {
	return Config{
		Addr: env("APP_ADDR", ":19191"),
		DB: Database{
			Driver: env("APP_DB_DRIVER", "postgres"), DSN: os.Getenv("APP_DSN"),
			Host: env("APP_DB_HOST", "100.120.144.76"), Port: env("APP_DB_PORT", "5432"),
			Name: env("APP_DB_NAME", "study_workbench"), User: env("APP_DB_USER", "sw_kid_runtime"),
			Password: env("APP_DB_PASSWORD", ""), Options: env("APP_DB_CONFIG", "sslmode=disable TimeZone=Asia/Shanghai"),
		},
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
