package wizard

import (
	"bytes"
	"strings"
	"testing"
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
	in := strings.NewReader("my-token\n-100123\nn\n")
	var out bytes.Buffer
	cfg, err := Run(in, &out, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if cfg.Telegram.BotToken != "my-token" {
		t.Errorf("BotToken = %q, want my-token", cfg.Telegram.BotToken)
	}
	if cfg.Telegram.ChannelID != "-100123" {
		t.Errorf("ChannelID = %q, want -100123", cfg.Telegram.ChannelID)
	}
	if len(cfg.Funds) != 0 {
		t.Errorf("Funds = %v, want empty (user declined)", cfg.Funds)
	}
}

func TestRunCollectsFund(t *testing.T) {
	input := strings.Join([]string{
		"my-token",
		"-100123",
		"y",
		"https://www.analizy.pl/fundusze-zagraniczne/FIL133_A_USD/x",
		"2025-09-15",
		"100",
		"14.20",
		"n",
	}, "\n") + "\n"
	in := strings.NewReader(input)
	var out bytes.Buffer
	cfg, err := Run(in, &out, nil)
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
