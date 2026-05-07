package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	want := Config{
		Telegram: Telegram{BotToken: "abc:123", ChannelID: "-100123"},
		BaseCurrency: "USD",
		ScheduleTime: "18:00",
		Thresholds: Thresholds{
			AlertSingleDayPct:        3.0,
			Alert5dCumulativePct:     7.0,
			AlertPortfolio5dPct:      5.0,
			Warning5dCumulativePct:   3.0,
			Good5dCumulativePct:      1.0,
			VeryGood5dCumulativePct:  5.0,
		},
		Funds: []FundEntry{{
			FundID:        "FIL133_A_USD",
			URL:           "https://www.analizy.pl/fundusze-zagraniczne/FIL133_A_USD/x",
			PurchaseDate:  time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			PurchaseUnits: 100.0,
			PurchasePrice: 14.20,
		}},
	}
	if err := Save(p, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Telegram != want.Telegram {
		t.Errorf("Telegram = %+v, want %+v", got.Telegram, want.Telegram)
	}
	if got.Thresholds != want.Thresholds {
		t.Errorf("Thresholds = %+v, want %+v", got.Thresholds, want.Thresholds)
	}
	if len(got.Funds) != 1 || got.Funds[0].FundID != "FIL133_A_USD" {
		t.Errorf("Funds = %+v, want one FIL133_A_USD", got.Funds)
	}
	if !got.Funds[0].PurchaseDate.Equal(want.Funds[0].PurchaseDate) {
		t.Errorf("PurchaseDate = %v, want %v", got.Funds[0].PurchaseDate, want.Funds[0].PurchaseDate)
	}
}

func TestDefaultsApplied(t *testing.T) {
	c := WithDefaults(Config{})
	if c.BaseCurrency != "USD" {
		t.Errorf("BaseCurrency = %q, want USD", c.BaseCurrency)
	}
	if c.ScheduleTime != "18:00" {
		t.Errorf("ScheduleTime = %q, want 18:00", c.ScheduleTime)
	}
	if c.Thresholds.AlertSingleDayPct != 3.0 {
		t.Errorf("AlertSingleDayPct = %v, want 3.0", c.Thresholds.AlertSingleDayPct)
	}
	if c.Thresholds.AlertPortfolio5dPct != 5.0 {
		t.Errorf("AlertPortfolio5dPct = %v, want 5.0", c.Thresholds.AlertPortfolio5dPct)
	}
}

func TestValidateRejectsMissingTelegram(t *testing.T) {
	c := WithDefaults(Config{
		Funds: []FundEntry{{FundID: "X", URL: "https://x"}},
	})
	if err := Validate(c); err == nil {
		t.Fatal("expected error for missing Telegram, got nil")
	}
}

func TestValidateRejectsEmptyFunds(t *testing.T) {
	c := WithDefaults(Config{
		Telegram: Telegram{BotToken: "t", ChannelID: "c"},
	})
	if err := Validate(c); err == nil {
		t.Fatal("expected error for empty funds, got nil")
	}
}
