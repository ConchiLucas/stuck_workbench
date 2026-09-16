package main

import (
	"log"

	"github.com/conchi/phrase-server/internal/config"
	"github.com/conchi/phrase-server/internal/db"
	httpapi "github.com/conchi/phrase-server/internal/http"
	"github.com/conchi/phrase-server/internal/plan"
	"github.com/conchi/phrase-server/internal/practice"
	"github.com/conchi/study-learning/mastery"
	"github.com/conchi/study-learning/phrasecontent"
)

func main() {
	cfg := config.Load()
	database, err := db.Open(cfg.DB)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	if err := phrasecontent.MigrateMedia(database); err != nil {
		log.Fatalf("migrate phrase media: %v", err)
	}
	plans := plan.NewService(database, cfg.ContentAdminURL)
	if err := httpapi.NewRouter(httpapi.Deps{
		Readiness: db.Checker{DB: database},
		Plans:     plans,
		Practice:  practice.NewService(database, mastery.DefaultConfig()),
		Media:     plans,
	}).Run(cfg.Addr); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
