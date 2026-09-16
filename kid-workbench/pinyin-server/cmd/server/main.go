package main

import (
	"log"

	"github.com/conchi/pinyin-server/internal/asset"
	"github.com/conchi/pinyin-server/internal/catalog"
	"github.com/conchi/pinyin-server/internal/config"
	"github.com/conchi/pinyin-server/internal/db"
	"github.com/conchi/pinyin-server/internal/home"
	httpapi "github.com/conchi/pinyin-server/internal/http"
	"github.com/conchi/pinyin-server/internal/plan"
	"github.com/conchi/pinyin-server/internal/practice"
	"github.com/conchi/pinyin-server/internal/progress"
	"github.com/conchi/pinyin-server/internal/quiz"
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
	quizService := quiz.NewService(database, cfg.Mastery)
	practiceService := practice.NewService(database, cfg.Mastery)
	if err := httpapi.NewRouter(httpapi.Deps{
		Catalog:    catalog.NewRepository(database),
		Assets:     asset.NewService(store),
		Progress:   progress.NewService(database),
		Plans:      plan.NewService(database),
		Practice:   practiceService,
		Home:       home.NewService(database),
		Quiz:       quizService,
		FormalQuiz: quizService,
	}).Run(cfg.Addr); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
