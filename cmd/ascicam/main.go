package main

import (
	"fmt"
	"os"

	"ascicam/internal/app"
)

func main() {
	var (
		cfg app.Config
		err error
	)

	if len(os.Args) == 1 {
		cfg = app.DefaultConfig()
	} else {
		cfg, err = app.ParseConfig(os.Args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "ascicam: %v\n", err)
			os.Exit(1)
		}
	}

	var outcome app.RunOutcome
	if cfg.UseTUI {
		err = app.RunTUI(cfg)
	} else {
		outcome, err = app.Run(cfg)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ascicam: %v\n", err)
		os.Exit(1)
	}
	if !cfg.UseTUI && outcome.Notice != "" {
		fmt.Fprintln(os.Stdout, outcome.Notice)
	}
}
