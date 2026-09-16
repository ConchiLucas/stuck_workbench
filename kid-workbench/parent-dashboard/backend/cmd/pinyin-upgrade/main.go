package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/conchi/study-workbench/internal/config"
	"github.com/conchi/study-workbench/internal/db"
	"github.com/conchi/study-workbench/internal/service"
	"os"
)

func main() {
	apply := flag.Bool("apply", false, "apply upgrade with audit")
	dry := flag.Bool("dry-run", false, "preview upgrade in a rolled-back transaction")
	flag.Parse()
	if *apply && *dry {
		fmt.Fprintln(os.Stderr, "choose --apply or --dry-run")
		os.Exit(2)
	}
	cfg := config.Load()
	d, e := db.Open(cfg.Driver, cfg.DSN)
	if e != nil {
		fmt.Fprintln(os.Stderr, "cannot connect to database")
		os.Exit(1)
	}
	result, e := service.UpgradePinyin(d, *apply)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	json.NewEncoder(os.Stdout).Encode(result)
}
