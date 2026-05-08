package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/lizarusi/fundpulse/internal/config"
	"github.com/lizarusi/fundpulse/internal/launchd"
	"github.com/lizarusi/fundpulse/internal/scraper"
	"github.com/lizarusi/fundpulse/internal/storage"
	"github.com/lizarusi/fundpulse/internal/wizard"
)

func cmdConfig(args []string) error {
	if len(args) > 0 && args[0] == "show" {
		return cmdConfigShow()
	}
	return cmdConfigEdit()
}

func cmdConfigShow() error {
	cfgPath := config.DefaultPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config (have you run `fundpulse init`?): %w", err)
	}

	var names map[string]string
	if store, err := storage.Open(config.DefaultDBPath()); err == nil {
		_ = store.Migrate()
		names = fundNamesFromDB(store)
		store.Close()
	}

	renderConfigSummary(os.Stdout, cfgPath, cfg, names)
	return nil
}

func cmdConfigEdit() error {
	cfgPath := config.DefaultPath()
	oldCfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config (have you run `fundpulse init`?): %w", err)
	}

	newCfg, err := wizard.Run(os.Stdin, os.Stdout, oldCfg, wizard.DefaultValidator())
	if err != nil {
		return fmt.Errorf("wizard: %w", err)
	}
	if err := config.Validate(newCfg); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	if err := config.Save(cfgPath, newCfg); err != nil {
		return err
	}
	fmt.Printf("→ Saved config to %s\n", cfgPath)

	store, err := storage.Open(config.DefaultDBPath())
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.Migrate(); err != nil {
		return err
	}

	added := newFunds(oldCfg.Funds, newCfg.Funds)
	for _, f := range added {
		fmt.Printf("  → New fund %s — fetching current snapshot... ", f.FundID)
		snap, err := scraper.FetchSnapshot(f.URL)
		if err != nil {
			fmt.Printf("failed: %v\n", err)
			continue
		}
		_ = store.UpsertFund(storage.Fund{
			FundID:        f.FundID,
			URL:           f.URL,
			Name:          snap.Name,
			Currency:      snap.Currency,
			PurchaseDate:  f.PurchaseDate,
			PurchaseUnits: f.PurchaseUnits,
			PurchasePrice: f.PurchasePrice,
		})
		_ = store.UpsertPrice(storage.Price{
			FundID: f.FundID,
			Date:   snap.NAVDate,
			NAV:    snap.NAV,
			Source: "config",
		})
		fmt.Printf("ok (%s, %.2f %s)\n", snap.Name, snap.NAV, snap.Currency)
	}
	if len(added) > 0 {
		fmt.Printf("Tip: run `fundpulse backfill` to fill historical prices for the new fund(s).\n")
	}

	if oldCfg.ScheduleTime != newCfg.ScheduleTime {
		hour, minute, err := launchd.ParseScheduleTime(newCfg.ScheduleTime)
		if err != nil {
			return err
		}
		bin, err := exec.LookPath(os.Args[0])
		if err != nil {
			bin = os.Args[0]
		}
		if err := launchd.Install(launchd.Config{
			Label:     appLabel,
			Binary:    bin,
			Args:      []string{"run"},
			Hour:      hour,
			Minute:    minute,
			LogPath:   config.DefaultLogPath(),
			RunAtLoad: false,
		}); err != nil {
			return fmt.Errorf("reinstall launchd: %w", err)
		}
		fmt.Printf("→ Schedule changed: now runs daily at %02d:%02d\n", hour, minute)
	}

	return nil
}

func newFunds(oldF, newF []config.FundEntry) []config.FundEntry {
	have := make(map[string]struct{}, len(oldF))
	for _, f := range oldF {
		have[f.FundID] = struct{}{}
	}
	var out []config.FundEntry
	for _, f := range newF {
		if _, seen := have[f.FundID]; !seen {
			out = append(out, f)
		}
	}
	return out
}
