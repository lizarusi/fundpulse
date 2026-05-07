package main

import (
	"fmt"
	"time"

	"github.com/lizarusi/investments-healthcheck/internal/analyzer"
	"github.com/lizarusi/investments-healthcheck/internal/config"
	"github.com/lizarusi/investments-healthcheck/internal/portfolio"
	"github.com/lizarusi/investments-healthcheck/internal/scraper"
	"github.com/lizarusi/investments-healthcheck/internal/storage"
	"github.com/lizarusi/investments-healthcheck/internal/telegram"
)

func cmdRun(args []string) error {
	_, dry, err := parseFlags(args)
	if err != nil {
		return err
	}
	return runOnce(*dry, false)
}

func cmdShow(args []string) error {
	return runOnce(true, true)
}

func runOnce(dryRun, skipDBWrite bool) error {
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return fmt.Errorf("load config (have you run `healthcheck init`?): %w", err)
	}
	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("config invalid: %w", err)
	}

	store, err := storage.Open(config.DefaultDBPath())
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.Migrate(); err != nil {
		return err
	}

	thresholds := analyzer.Thresholds(cfg.Thresholds)

	var fundLines []telegram.FundLine
	var fundMetrics []portfolio.FundMetrics
	var partial bool
	var firstScrapeErr error

	for _, entry := range cfg.Funds {
		snap, err := scraper.FetchSnapshot(entry.URL)
		if err != nil {
			partial = true
			if firstScrapeErr == nil {
				firstScrapeErr = err
			}
			fmt.Fprintf(stderr(), "scrape %s: %v\n", entry.FundID, err)
			continue
		}

		if !skipDBWrite {
			if err := store.UpsertFund(storage.Fund{
				FundID:        entry.FundID,
				URL:           entry.URL,
				Name:          snap.Name,
				Currency:      snap.Currency,
				PurchaseDate:  entry.PurchaseDate,
				PurchaseUnits: entry.PurchaseUnits,
				PurchasePrice: entry.PurchasePrice,
			}); err != nil {
				return fmt.Errorf("upsert fund %s: %w", entry.FundID, err)
			}
			if err := store.UpsertPrice(storage.Price{
				FundID: entry.FundID,
				Date:   snap.NAVDate,
				NAV:    snap.NAV,
				Source: "daily",
			}); err != nil {
				return fmt.Errorf("upsert price %s: %w", entry.FundID, err)
			}
		}

		since := snap.NAVDate.AddDate(0, 0, -60)
		recent, err := store.RecentPrices(entry.FundID, since)
		if err != nil {
			return fmt.Errorf("recent prices %s: %w", entry.FundID, err)
		}
		if skipDBWrite {
			has := false
			for _, p := range recent {
				if p.Date.Equal(snap.NAVDate) {
					has = true
					break
				}
			}
			if !has {
				recent = append(recent, storage.Price{
					FundID: entry.FundID, Date: snap.NAVDate, NAV: snap.NAV, Source: "show",
				})
			}
		}

		prices := make([]portfolio.Price, len(recent))
		for i, p := range recent {
			prices[i] = portfolio.Price{Date: p.Date, NAV: p.NAV}
		}

		fm := portfolio.ComputeFund(portfolio.PurchaseInfo{
			Units: entry.PurchaseUnits,
			Price: entry.PurchasePrice,
			Date:  entry.PurchaseDate,
		}, prices)
		fm.FundID = entry.FundID
		fm.Name = snap.Name
		fm.Currency = snap.Currency
		fm.RiskLabel = snap.RiskLabel
		fm.RiskLevel = snap.RiskLevel
		fundMetrics = append(fundMetrics, fm)

		fv := analyzer.PerFund(fm, thresholds)
		fundLines = append(fundLines, telegram.FundLine{
			Name:          snap.Name,
			Currency:      snap.Currency,
			Verdict:       fv,
			Change1DPct:   fm.Change1DPct,
			Change30DPct:  fm.Change30DPct,
			HasThirtyDays: fm.HasThirtyDays,
			ProfitLoss:    fm.ProfitLoss,
			RiskLabel:     snap.RiskLabel,
		})
	}

	if len(fundMetrics) == 0 {
		_ = store.RecordRun("error", fmt.Sprintf("no funds scraped successfully: %v", firstScrapeErr))
		return fmt.Errorf("no funds scraped successfully: %w", firstScrapeErr)
	}

	port := portfolio.ComputePortfolio(fundMetrics, cfg.BaseCurrency)
	overall := analyzer.Overall(fundMetrics, port, thresholds)

	report := telegram.Report{
		Date:       time.Now(),
		Currency:   cfg.BaseCurrency,
		TotalValue: port.TotalValue,
		TotalPL:    port.TotalProfitLoss,
		TotalPLPct: port.TotalPLPct,
		Verdict:    overall,
		Funds:      fundLines,
	}
	msg := telegram.Render(report)

	if dryRun {
		fmt.Println(msg)
	} else {
		client := telegram.NewClient(cfg.Telegram.BotToken, cfg.Telegram.ChannelID)
		if err := client.Send(msg); err != nil {
			_ = store.RecordRun("error", fmt.Sprintf("telegram send: %v", err))
			return fmt.Errorf("telegram send: %w", err)
		}
	}

	status := "ok"
	if partial {
		status = "partial"
	}
	if !skipDBWrite {
		_ = store.RecordRun(status, fmt.Sprintf("%d funds, verdict=%s", len(fundMetrics), overall.Level))
	}
	return nil
}
