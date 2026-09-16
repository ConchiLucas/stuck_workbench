package main

import (
	"log"

	"github.com/conchi/chengyu-server/internal/config"
	"github.com/conchi/chengyu-server/internal/db"
	httpapi "github.com/conchi/chengyu-server/internal/http"
	"github.com/conchi/chengyu-server/internal/plan"
	"github.com/conchi/chengyu-server/internal/practice"
	"github.com/conchi/study-learning/chengyucontent"
	"github.com/conchi/study-learning/mastery"
)

func main() {
	cfg := config.Load()
	database, err := db.Open(cfg.DB)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	if err := chengyucontent.MigrateMedia(database); err != nil {
		log.Fatalf("migrate chengyu media: %v", err)
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
