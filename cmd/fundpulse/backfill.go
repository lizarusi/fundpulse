package main

import (
	"fmt"
	"time"

	"github.com/lizarusi/fundpulse/internal/config"
	"github.com/lizarusi/fundpulse/internal/scraper"
	"github.com/lizarusi/fundpulse/internal/storage"
)

func cmdBackfill(args []string) error {
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	store, err := storage.Open(config.DefaultDBPath())
	if err != nil {
		return err
	}
	defer store.Close()

	fmt.Println("Backfilling historical data...")
	for _, f := range cfg.Funds {
		fmt.Printf("→ %s: fetching since %s... ", f.FundID, f.PurchaseDate.Format("2006-01-02"))
		
		history, err := scraper.FetchHistory(f.FundID, f.PurchaseDate, time.Now())
		if err != nil {
			fmt.Printf("error: %v\n", err)
			continue
		}

		count := 0
		for _, p := range history {
			_ = store.UpsertPrice(storage.Price{
				FundID: f.FundID,
				Date:   p.Date,
				NAV:    p.Value,
				Source: "backfill",
			})
			count++
		}
		fmt.Printf("done (%d prices)\n", count)
	}

	fmt.Println("Backfill complete.")
	return nil
}
