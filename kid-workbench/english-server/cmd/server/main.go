package main

import (
	"log"

	"github.com/conchi/english-server/internal/asset"
	"github.com/conchi/english-server/internal/catalog"
	"github.com/conchi/english-server/internal/config"
	"github.com/conchi/english-server/internal/db"
	"github.com/conchi/english-server/internal/home"
	httpapi "github.com/conchi/english-server/internal/http"
	"github.com/conchi/english-server/internal/plan"
	"github.com/conchi/english-server/internal/practice"
	"github.com/conchi/english-server/internal/progress"
	"github.com/conchi/english-server/internal/quiz"
	"github.com/conchi/study-learning/englishcontent"
	"github.com/conchi/study-learning/mastery"
)

func main() {
	cfg := config.Load()
	database, err := db.Open(cfg.DB)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	if err := englishcontent.MigrateMedia(database); err != nil {
		log.Fatalf("migrate media: %v", err)
	}
	store, err := asset.NewStore(cfg.Assets)
	if err != nil {
		log.Fatalf("configure object storage: %v", err)
	}
	if err := httpapi.NewRouter(httpapi.Deps{
		Readiness: httpapi.ReadinessGroup{db.Checker{DB: database}, store},
		Catalog:   catalog.NewRepository(database), Assets: asset.NewService(store),
		Content:  cfg.ContentAdminURL,
		Progress: progress.NewService(database), Home: home.NewService(database),
		Plans:    plan.NewService(database, cfg.ContentAdminURL),
		Practice: practice.NewService(database, mastery.DefaultConfig()),
		Quiz:     quiz.NewService(database),
	}).Run(cfg.Addr); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
