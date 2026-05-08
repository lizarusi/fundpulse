package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lizarusi/fundpulse/internal/config"
)

const appLabel = "com.lizarusi.fundpulse"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	if cmd != "-h" && cmd != "--help" && cmd != "help" {
		config.MigrateLegacyPaths(os.Stderr)
	}

	var err error
	switch cmd {
	case "init":
		err = cmdInit(args)
	case "config":
		err = cmdConfig(args)
	case "backfill":
		err = cmdBackfill(args)
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
	fmt.Println(`fundpulse — daily fund portfolio update for analizy.pl

Usage:
  fundpulse init             First-time setup (Telegram + funds + launchd)
  fundpulse config           Edit existing config interactively (Telegram, schedule, funds)
  fundpulse config show      Print current config (bot token masked)
  fundpulse backfill         Fetch all missing historical data for configured funds
  fundpulse run [--dry-run]  Run the daily job (scrape, store, send)
  fundpulse show             Print the current report without sending Telegram
  fundpulse uninstall        Remove launchd schedule (data and config preserved)

Files:
  ~/.config/fundpulse/config.yaml          (editable)
  ~/Library/Application Support/fundpulse/data.db
  ~/Library/Logs/fundpulse/run.log`)
}

func parseFlags(args []string) (*flag.FlagSet, *bool, error) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	dry := fs.Bool("dry-run", false, "do not send Telegram message")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	return fs, dry, nil
}
