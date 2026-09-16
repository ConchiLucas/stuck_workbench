package main

import (
	"context"
	"log"
	"os"

	"github.com/conchi/study-learning/logiccontent"
	"github.com/conchi/study-learning/poemcontent"
	"github.com/conchi/study-learning/sciencecontent"
	"github.com/conchi/study-task-admin/internal/chengyutask"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/conchi/study-task-admin/internal/englishtask"
	httpapi "github.com/conchi/study-task-admin/internal/http"
	"github.com/conchi/study-task-admin/internal/logictask"
	"github.com/conchi/study-task-admin/internal/mathtask"
	"github.com/conchi/study-task-admin/internal/phrasetask"
	"github.com/conchi/study-task-admin/internal/pinyintask"
	"github.com/conchi/study-task-admin/internal/poemtask"
	"github.com/conchi/study-task-admin/internal/qtask"
	"github.com/conchi/study-task-admin/internal/reviewsuggestion"
	"github.com/conchi/study-task-admin/internal/sciencetask"
	"github.com/conchi/study-task-admin/internal/taskgen"
)

func main() {
	addr := env("APP_ADDR", ":19201")
	dsn := db.DSNFromEnv()
	gdb, err := db.OpenPostgres(dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(gdb); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := taskgen.Migrate(gdb); err != nil {
		log.Fatalf("migrate generation: %v", err)
	}
	if err := mathtask.Migrate(gdb); err != nil {
		log.Fatalf("migrate arithmetic: %v", err)
	}
	if err := reviewsuggestion.Migrate(gdb); err != nil {
		log.Fatalf("migrate review suggestions: %v", err)
	}
	if err := pinyintask.Migrate(gdb); err != nil {
		log.Fatalf("migrate pinyin tasks: %v", err)
	}
	if err := englishtask.Migrate(gdb); err != nil {
		log.Fatalf("migrate english tasks: %v", err)
	}
	if err := phrasetask.Migrate(gdb); err != nil {
		log.Fatalf("migrate phrase tasks: %v", err)
	}
	if err := chengyutask.Migrate(gdb); err != nil {
		log.Fatalf("migrate chengyu tasks: %v", err)
	}
	if err := sciencetask.Migrate(gdb); err != nil {
		log.Fatalf("migrate science tasks: %v", err)
	}
	if err := poemtask.Migrate(gdb); err != nil {
		log.Fatalf("migrate poem tasks: %v", err)
	}
	if err := logictask.Migrate(gdb); err != nil {
		log.Fatalf("migrate logic tasks: %v", err)
	}
	if err := sciencecontent.EnsureMaterials(gdb); err != nil {
		log.Printf("ensure science materials: %v", err)
	}
	if err := poemcontent.EnsureMaterials(gdb); err != nil {
		log.Printf("ensure poem materials: %v", err)
	}
	if err := logiccontent.EnsureMaterials(gdb); err != nil {
		log.Printf("ensure logic materials: %v", err)
	}
	content := env("APP_CONTENT_ADMIN_URL", "http://127.0.0.1:19091")
	pinyinTasks := pinyintask.New(gdb, content)
	mathTasks := mathtask.New(gdb, content)
	englishTasks := englishtask.New(gdb, content)
	phraseTasks := phrasetask.New(gdb, content)
	chengyuTasks := chengyutask.New(gdb, content)
	scienceTasks := sciencetask.New(gdb, content)
	poemTasks := poemtask.New(gdb, content)
	logicTasks := logictask.New(gdb, content)
	gen := taskgen.New(gdb, taskgen.NewMaterialClient(content))
	gen.Support = taskgen.NewRuntimeSupport(env("APP_LITERACY_SERVER_URL", "http://127.0.0.1:19151"), env("APP_LITERACY_APP_URL", "http://127.0.0.1:19152"))
	gen.Support.WritingEnabled = env("APP_WRITING_ENABLED", "true") != "false"
	if err := gen.RecoverRuns(); err != nil {
		log.Fatalf("recover generation: %v", err)
	}
	go taskgen.NewWorker(gen, env("APP_AUTO_REVIEW_ENABLED", "false") == "true").Run(context.Background())
	suggestions := reviewsuggestion.New(gdb, gen, env("APP_REVIEW_SUGGESTIONS_ENABLED", "false") == "true")
	go suggestions.Run(context.Background())
	router := httpapi.NewRouter(httpapi.Deps{ReviewSuggestions: suggestions, QTask: qtask.NewService(gdb), Generation: gen, Math: mathTasks, Pinyin: pinyinTasks, English: englishTasks, Phrase: phraseTasks, Chengyu: chengyuTasks, Science: scienceTasks, Poem: poemTasks, Logic: logicTasks})
	log.Printf("study-task-admin listening on %s (db=%s)", addr, env("APP_DB_NAME", "study_workbench"))
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
