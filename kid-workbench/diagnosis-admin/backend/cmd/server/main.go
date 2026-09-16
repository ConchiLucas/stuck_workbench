package main

import (
	"log"
	"os"

	"github.com/conchi/study-diagnosis-admin/internal/db"
	"github.com/conchi/study-diagnosis-admin/internal/diagnosis"
	httpapi "github.com/conchi/study-diagnosis-admin/internal/http"
	"github.com/conchi/study-diagnosis-admin/internal/knowledge"
	"github.com/conchi/study-diagnosis-admin/internal/reviewclient"
)

func main() {
	addr := env("APP_ADDR", ":19211")
	dsn := db.DSNFromEnv()
	gdb, err := db.OpenPostgres(dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	router := httpapi.NewRouter(httpapi.Deps{Diagnosis: diagnosis.NewService(gdb), Knowledge: knowledge.New(gdb), Reviews: reviewclient.New(env("APP_TASK_ADMIN_URL", "http://127.0.0.1:19201"), env("APP_REVIEW_SUGGESTIONS_ENABLED", "false") == "true")})
	log.Printf("study-diagnosis-admin listening on %s (read-only db=%s)", addr, env("APP_DB_NAME", "study_workbench"))
	if err := router.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
