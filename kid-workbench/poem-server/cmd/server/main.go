package main

import (
	"log"

	"github.com/conchi/poem-server/internal/config"
	"github.com/conchi/poem-server/internal/db"
	httpapi "github.com/conchi/poem-server/internal/http"
	"github.com/conchi/poem-server/internal/plan"
	"github.com/conchi/poem-server/internal/practice"
	"github.com/conchi/poem-server/internal/studyplan"
	"github.com/conchi/study-learning/poemcontent"
)

func main() {
	cfg := config.Load()
	database, err := db.Open(cfg.DB)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	if err := poemcontent.EnsureMaterials(database); err != nil {
		log.Fatalf("ensure poem materials: %v", err)
	}
	if err := httpapi.NewRouterWithDeps(httpapi.Deps{
		Pavilions: plan.NewStore(),
		Plans:     studyplan.NewService(database, cfg.ContentAdminURL),
		Practice:  practice.NewService(database, cfg.Mastery),
		Database:  database,
	}).Run(cfg.Addr); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
