package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/lizarusi/investments-healthcheck/internal/config"
	"github.com/lizarusi/investments-healthcheck/internal/launchd"
	"github.com/lizarusi/investments-healthcheck/internal/scraper"
	"github.com/lizarusi/investments-healthcheck/internal/storage"
	"github.com/lizarusi/investments-healthcheck/internal/wizard"
)

func cmdInit(args []string) error {
	cfgPath := config.DefaultPath()
	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Fprintf(os.Stderr, "config already exists at %s — running wizard will overwrite it\n", cfgPath)
	}

	cfg, err := wizard.Run(os.Stdin, os.Stdout, wizard.DefaultValidator())
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
		_ = store.UpsertPrice(storage.Price{
			FundID: f.FundID,
			Date:   snap.NAVDate,
			NAV:    snap.NAV,
			Source: "daily",
		})
		fmt.Printf("  ✓ %s — %s @ %.2f %s on %s\n", f.FundID, snap.Name, snap.NAV, snap.Currency, snap.NAVDate.Format("2006-01-02"))
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
	fmt.Println("Done. Run `healthcheck show` to preview the report, or `healthcheck run` to send the first Telegram message now.")
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
