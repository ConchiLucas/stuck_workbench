package main

import (
	"log"

	"github.com/conchi/math-server/internal/asset"
	"github.com/conchi/math-server/internal/catalog"
	"github.com/conchi/math-server/internal/config"
	"github.com/conchi/math-server/internal/content"
	"github.com/conchi/math-server/internal/db"
	"github.com/conchi/math-server/internal/home"
	httpapi "github.com/conchi/math-server/internal/http"
	"github.com/conchi/math-server/internal/plan"
	"github.com/conchi/math-server/internal/practice"
	"github.com/conchi/math-server/internal/progress"
	"github.com/conchi/math-server/internal/quiz"
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
	practiceService := practice.NewService(database, cfg.Mastery)
	if err := httpapi.NewRouter(httpapi.Deps{
		Content: content.NewService(cfg.ContentURL),
		Catalog: catalog.NewRepository(database), Progress: progress.NewService(database), Plans: plan.NewService(database),
		Assets: asset.NewService(store), Database: database, Practice: practiceService, Home: home.NewService(database), Readiness: store, Quiz: quiz.NewService(database),
	}).Run(cfg.Addr); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
