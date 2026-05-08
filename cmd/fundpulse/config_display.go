package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/lizarusi/fundpulse/internal/config"
	"github.com/lizarusi/fundpulse/internal/storage"
)

func maskToken(s string) string {
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-7) + s[len(s)-3:]
}

func renderConfigSummary(w io.Writer, path string, cfg config.Config, fundNames map[string]string) {
	fmt.Fprintf(w, "Config: %s\n\n", path)

	fmt.Fprintln(w, "Telegram:")
	fmt.Fprintf(w, "  bot_token:    %s\n", maskToken(cfg.Telegram.BotToken))
	fmt.Fprintf(w, "  channel_id:   %s\n", cfg.Telegram.ChannelID)

	fmt.Fprintf(w, "\nSchedule:       %s daily\n", cfg.ScheduleTime)
	fmt.Fprintf(w, "Base currency:  %s\n", cfg.BaseCurrency)

	fmt.Fprintln(w, "\nThresholds:")
	fmt.Fprintf(w, "  alert_single_day_pct:        %.1f\n", cfg.Thresholds.AlertSingleDayPct)
	fmt.Fprintf(w, "  alert_5d_cumulative_pct:     %.1f\n", cfg.Thresholds.Alert5dCumulativePct)
	fmt.Fprintf(w, "  alert_portfolio_5d_pct:      %.1f\n", cfg.Thresholds.AlertPortfolio5dPct)
	fmt.Fprintf(w, "  warning_5d_cumulative_pct:   %.1f\n", cfg.Thresholds.Warning5dCumulativePct)
	fmt.Fprintf(w, "  good_5d_cumulative_pct:      %.1f\n", cfg.Thresholds.Good5dCumulativePct)
	fmt.Fprintf(w, "  very_good_5d_cumulative_pct: %.1f\n", cfg.Thresholds.VeryGood5dCumulativePct)

	fmt.Fprintf(w, "\nFunds (%d):\n", len(cfg.Funds))
	for i, f := range cfg.Funds {
		fmt.Fprintf(w, "  %d. %s", i+1, f.FundID)
		if name, ok := fundNames[f.FundID]; ok && name != "" {
			fmt.Fprintf(w, " — %s", name)
		}
		fmt.Fprintln(w)
		fmt.Fprintf(w, "       URL: %s\n", f.URL)
		fmt.Fprintf(w, "       Bought: %s   Units: %s   Price: %.2f\n",
			f.PurchaseDate.Format("2006-01-02"),
			formatFloat(f.PurchaseUnits),
			f.PurchasePrice,
		)
	}
}

func formatFloat(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%g", v)
}

func fundNamesFromDB(store *storage.Store) map[string]string {
	out := map[string]string{}
	if store == nil {
		return out
	}
	funds, err := store.ListFunds()
	if err != nil {
		return out
	}
	for _, f := range funds {
		out[f.FundID] = f.Name
	}
	return out
}
