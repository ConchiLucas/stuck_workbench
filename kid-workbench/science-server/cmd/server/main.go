package main

import (
	"log"

	"github.com/conchi/study-science/internal/asset"
	"github.com/conchi/study-science/internal/catalog"
	"github.com/conchi/study-science/internal/config"
	"github.com/conchi/study-science/internal/db"
	"github.com/conchi/study-science/internal/home"
	httpapi "github.com/conchi/study-science/internal/http"
	"github.com/conchi/study-science/internal/plan"
	"github.com/conchi/study-science/internal/practice"
	"github.com/conchi/study-science/internal/progress"
	"github.com/conchi/study-science/internal/quiz"
	"github.com/conchi/study-learning/sciencecontent"
)

func main() {
	cfg := config.Load()
	database, err := db.Open(cfg.DB)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	store, err := asset.NewStore(cfg.Assets)
	if err != nil {
		log.Fatalf("configure object storage: %v", err)
	}
	if err := sciencecontent.EnsureMaterials(database); err != nil {
		log.Fatalf("ensure science materials: %v", err)
	}
	practiceService := practice.NewService(database, cfg.Mastery)
	catalogRepo := catalog.NewRepository(database)
	if err := httpapi.NewRouter(httpapi.Deps{
		Catalog:          catalogRepo,
		Assets:           asset.NewService(store, catalogRepo),
		Progress:         progress.NewService(database),
		Plans:            plan.NewService(database, cfg.ContentAdminURL),
		Practice:         practiceService,
		Home:             home.NewService(database),
		Quiz:             quiz.NewService(database),
		Database:         database,
		AssetsConfigured: store != nil,
	}).Run(cfg.Addr); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
