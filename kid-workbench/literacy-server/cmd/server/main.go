package main

import (
	"log"

	"github.com/conchi/literacy-server/internal/asset"
	"github.com/conchi/literacy-server/internal/catalog"
	"github.com/conchi/literacy-server/internal/config"
	"github.com/conchi/literacy-server/internal/db"
	"github.com/conchi/literacy-server/internal/home"
	httpapi "github.com/conchi/literacy-server/internal/http"
	"github.com/conchi/literacy-server/internal/plan"
	"github.com/conchi/literacy-server/internal/practice"
	"github.com/conchi/literacy-server/internal/progress"
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
		Catalog:  catalog.NewRepository(database),
		Assets:   asset.NewService(store),
		Progress: progress.NewService(database),
		Plans:    plan.NewService(database),
		Practice: practiceService,
		Home:     home.NewService(database),
	}).Run(cfg.Addr); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
