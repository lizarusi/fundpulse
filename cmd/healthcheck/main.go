package main

import (
	"flag"
	"fmt"
	"os"
)

const appLabel = "com.user.investments-healthcheck"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "init":
		err = cmdInit(args)
	case "run":
		err = cmdRun(args)
	case "show":
		err = cmdShow(args)
	case "uninstall":
		err = cmdUninstall(args)
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`healthcheck — daily fund portfolio update for analizy.pl

Usage:
  healthcheck init             First-time setup (Telegram + funds + launchd)
  healthcheck run [--dry-run]  Run the daily job (scrape, store, send)
  healthcheck show             Print the current report without sending Telegram
  healthcheck uninstall        Remove launchd schedule (data and config preserved)

Files:
  ~/.config/investments-healthcheck/config.yaml          (editable)
  ~/Library/Application Support/investments-healthcheck/data.db
  ~/Library/Logs/investments-healthcheck/run.log`)
}

func parseFlags(args []string) (*flag.FlagSet, *bool, error) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	dry := fs.Bool("dry-run", false, "do not send Telegram message")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	return fs, dry, nil
}
