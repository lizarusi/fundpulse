package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/lizarusi/fundpulse/internal/config"
	"github.com/lizarusi/fundpulse/internal/launchd"
	"github.com/lizarusi/fundpulse/internal/scraper"
	"github.com/lizarusi/fundpulse/internal/storage"
	"github.com/lizarusi/fundpulse/internal/wizard"
)

func cmdInit(args []string) error {
	cfgPath := config.DefaultPath()
	var initialCfg config.Config
	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Fprintf(os.Stderr, "config already exists at %s — running wizard will allow you to edit it\n", cfgPath)
		initialCfg, err = config.Load(cfgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not load existing config: %v\n", err)
			initialCfg = config.WithDefaults(config.Config{})
		}
	} else {
		initialCfg = config.WithDefaults(config.Config{})
	}

	cfg, err := wizard.Run(os.Stdin, os.Stdout, initialCfg, wizard.DefaultValidator())
	if err != nil {
		return fmt.Errorf("wizard: %w", err)
	}

	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("validation failed (run `init` again to retry): %w", err)
	}

	if err := config.Save(cfgPath, cfg); err != nil {
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

	for _, f := range cfg.Funds {
		snap, err := scraper.FetchSnapshot(f.URL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ! could not scrape %s now (%v); will retry on next run\n", f.FundID, err)
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

		fmt.Printf("  ✓ %s — %s\n", f.FundID, snap.Name)
		fmt.Printf("    → Fetching history since %s... ", f.PurchaseDate.Format("2006-01-02"))
		history, err := scraper.FetchHistory(f.FundID, f.PurchaseDate, time.Now())
		if err != nil {
			fmt.Printf("failed: %v\n", err)
		} else {
			count := 0
			for _, p := range history {
				_ = store.UpsertPrice(storage.Price{
					FundID: f.FundID,
					Date:   p.Date,
					NAV:    p.Value,
					Source: "init",
				})
				count++
			}
			// Also ensure today's snapshot is in (it might be newer than history series)
			_ = store.UpsertPrice(storage.Price{
				FundID: f.FundID,
				Date:   snap.NAVDate,
				NAV:    snap.NAV,
				Source: "daily",
			})
			fmt.Printf("done (%d prices)\n", count)
		}
	}
	fmt.Printf("→ Initialised database at %s\n", config.DefaultDBPath())

	hour, minute, err := launchd.ParseScheduleTime(cfg.ScheduleTime)
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
		return fmt.Errorf("install launchd: %w", err)
	}
	fmt.Printf("→ Scheduled daily at %02d:%02d via launchd (label %s)\n", hour, minute, appLabel)
	fmt.Println("Done. Run `fundpulse show` to preview the report, or `fundpulse run` to send the first Telegram message now.")
	return nil
}

func cmdUninstall(args []string) error {
	if err := launchd.Uninstall(appLabel); err != nil {
		return fmt.Errorf("uninstall launchd: %w", err)
	}
	fmt.Printf("→ Removed launchd schedule (%s).\n", appLabel)
	fmt.Println("Config and database preserved. Delete them manually if desired:")
	fmt.Printf("  rm %s\n", config.DefaultPath())
	fmt.Printf("  rm %s\n", config.DefaultDBPath())
	return nil
}
