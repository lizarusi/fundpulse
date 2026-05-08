package wizard

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lizarusi/fundpulse/internal/config"
)

func TestExtractFundIDFromURL(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://www.analizy.pl/fundusze-zagraniczne/FIL133_A_USD/fidelity-funds-x", "FIL133_A_USD"},
		{"https://www.analizy.pl/fundusze-inwestycyjne-otwarte/PKO123/foo", "PKO123"},
		{"https://www.analizy.pl/fundusze-zagraniczne/ABC_XYZ/", "ABC_XYZ"},
	}
	for _, tc := range cases {
		got, err := ExtractFundID(tc.url)
		if err != nil {
			t.Errorf("ExtractFundID(%q) error: %v", tc.url, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ExtractFundID(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}

func TestExtractFundIDInvalid(t *testing.T) {
	for _, u := range []string{"", "https://example.com", "not a url"} {
		if _, err := ExtractFundID(u); err == nil {
			t.Errorf("ExtractFundID(%q) should error", u)
		}
	}
}

func TestRunCapturesTokenAndChannel(t *testing.T) {
	in := strings.NewReader("my-token\n-100123\n12:00\nEUR\nn\n")
	var out bytes.Buffer
	cfg, err := Run(in, &out, config.Config{}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if cfg.Telegram.BotToken != "my-token" {
		t.Errorf("BotToken = %q, want my-token", cfg.Telegram.BotToken)
	}
	if cfg.Telegram.ChannelID != "-100123" {
		t.Errorf("ChannelID = %q, want -100123", cfg.Telegram.ChannelID)
	}
	if cfg.ScheduleTime != "12:00" {
		t.Errorf("ScheduleTime = %q, want 12:00", cfg.ScheduleTime)
	}
	if cfg.BaseCurrency != "EUR" {
		t.Errorf("BaseCurrency = %q, want EUR", cfg.BaseCurrency)
	}
	if len(cfg.Funds) != 0 {
		t.Errorf("Funds = %v, want empty (user declined)", cfg.Funds)
	}
}

func TestRunCollectsFund(t *testing.T) {
	input := strings.Join([]string{
		"my-token",
		"-100123",
		"18:00",
		"USD",
		"y",
		"https://www.analizy.pl/fundusze-zagraniczne/FIL133_A_USD/x",
		"2025-09-15",
		"100",
		"14.20",
		"n",
	}, "\n") + "\n"
	in := strings.NewReader(input)
	var out bytes.Buffer
	cfg, err := Run(in, &out, config.Config{}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(cfg.Funds) != 1 {
		t.Fatalf("Funds count = %d, want 1", len(cfg.Funds))
	}
	f := cfg.Funds[0]
	if f.FundID != "FIL133_A_USD" {
		t.Errorf("FundID = %q, want FIL133_A_USD", f.FundID)
	}
	if f.PurchaseUnits != 100 {
		t.Errorf("PurchaseUnits = %v, want 100", f.PurchaseUnits)
	}
	if f.PurchasePrice != 14.20 {
		t.Errorf("PurchasePrice = %v, want 14.20", f.PurchasePrice)
	}
}

func TestRunEditsExistingConfig(t *testing.T) {
	initial := config.Config{
		Telegram: config.Telegram{
			BotToken:  "old-token",
			ChannelID: "old-channel",
		},
		ScheduleTime: "09:00",
		BaseCurrency: "GBP",
	}
	// User presses enter (empty line) for all prompts, accepting defaults.
	// Then declines adding a fund.
	input := "\n\n\n\nn\n"
	in := strings.NewReader(input)
	var out bytes.Buffer
	cfg, err := Run(in, &out, initial, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if cfg.Telegram.BotToken != "old-token" {
		t.Errorf("BotToken = %q, want old-token", cfg.Telegram.BotToken)
	}
	if cfg.Telegram.ChannelID != "old-channel" {
		t.Errorf("ChannelID = %q, want old-channel", cfg.Telegram.ChannelID)
	}
	if cfg.ScheduleTime != "09:00" {
		t.Errorf("ScheduleTime = %q, want 09:00", cfg.ScheduleTime)
	}
	if cfg.BaseCurrency != "GBP" {
		t.Errorf("BaseCurrency = %q, want GBP", cfg.BaseCurrency)
	}
}

func TestRunGranularFundEditing(t *testing.T) {
	initial := config.Config{
		Funds: []config.FundEntry{
			{FundID: "KEEP", URL: "https://www.analizy.pl/fundusze-zagraniczne/KEEP/x"},
			{FundID: "DELETE", URL: "https://www.analizy.pl/fundusze-zagraniczne/DELETE/x"},
			{FundID: "EDIT", URL: "https://www.analizy.pl/fundusze-zagraniczne/EDIT/x", PurchaseUnits: 10},
		},
	}
	input := strings.Join([]string{
		"token", "channel", "18:00", "USD",
		"k",                             // keep first
		"d",                             // delete second
		"e",                             // edit third
		"",                              // keep URL
		"2025-01-01", "20", "15.0",      // new data for EDIT
		"n",                             // do not add more
	}, "\n") + "\n"

	in := strings.NewReader(input)
	var out bytes.Buffer
	cfg, err := Run(in, &out, initial, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(cfg.Funds) != 2 {
		t.Fatalf("Funds count = %d, want 2", len(cfg.Funds))
	}
	if cfg.Funds[0].FundID != "KEEP" {
		t.Errorf("First fund = %s, want KEEP", cfg.Funds[0].FundID)
	}
	if cfg.Funds[1].FundID != "EDIT" {
		t.Errorf("Second fund = %s, want EDIT", cfg.Funds[1].FundID)
	}
	if cfg.Funds[1].PurchaseUnits != 20 {
		t.Errorf("EDIT fund units = %v, want 20", cfg.Funds[1].PurchaseUnits)
	}
}
