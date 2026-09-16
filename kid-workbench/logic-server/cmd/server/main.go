package main

import (
	"log"

	"github.com/conchi/logic-server/internal/config"
	"github.com/conchi/logic-server/internal/db"
	httpapi "github.com/conchi/logic-server/internal/http"
	"github.com/conchi/logic-server/internal/plan"
	"github.com/conchi/logic-server/internal/practice"
	"github.com/conchi/logic-server/internal/quiz"
	"github.com/conchi/study-learning/logiccontent"
	"github.com/conchi/study-learning/mastery"
)

func main() {
	cfg := config.Load()
	database, err := db.Open(cfg.DB)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	if err := logiccontent.EnsureMaterials(database); err != nil {
		log.Fatalf("ensure logic materials: %v", err)
	}
	if err := httpapi.NewRouter(httpapi.Deps{
		Quiz:      quiz.NewService(database),
		Plans:     plan.NewService(database),
		Practice:  practice.NewService(database, mastery.Config{BaseMasterStreak: 2, MinAccuracy: 0.8, ShakyMinAttempts: 3, ShakyAccuracy: 0.6, EaseMin: 1.3, EaseMax: 2.8, EaseUp: 0.1, EaseDown: 0.2, MaxIntervalDays: 60}),
		Database:  database,
		Readiness: db.Checker{DB: database},
	}).Run(cfg.Addr); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
