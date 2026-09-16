package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

type ObjectStorage struct {
	Endpoint, AccessKey, SecretKey, Bucket, BasePath string
	UseTLS                                           bool
}

type Config struct {
	Addr             string
	ContentAdminURL string
	DB               Database
	Assets           ObjectStorage
}

func Load() Config {
	return Config{
		Addr:             env("APP_ADDR", ":19131"),
		ContentAdminURL: strings.TrimRight(env("APP_CONTENT_ADMIN_URL", "http://localhost:19091"), "/"),
		DB: Database{
			Driver: env("APP_DB_DRIVER", "postgres"), DSN: os.Getenv("APP_DSN"),
			Host: env("APP_DB_HOST", "127.0.0.1"), Port: env("APP_DB_PORT", "15432"),
			Name: env("APP_DB_NAME", "study_workbench"), User: env("APP_DB_USER", "conchi"),
			Password: env("APP_DB_PASSWORD", "conchi123456"), Options: env("APP_DB_CONFIG", "sslmode=disable TimeZone=Asia/Shanghai"),
		},
		Assets: ObjectStorage{
			Endpoint: env("APP_MINIO_ENDPOINT", "minio:9000"), AccessKey: env("APP_MINIO_ACCESS_KEY", "conchi"),
			SecretKey: env("APP_MINIO_SECRET_KEY", "conchi123456"), Bucket: env("APP_MINIO_BUCKET", "english-material"),
			BasePath: env("APP_MINIO_BASE_PATH", "image-story"), UseTLS: envBool("APP_MINIO_USE_SSL", false),
		},
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}
