package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/lizarusi/fundpulse/internal/config"
)

func TestMaskToken(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"short", "*****"},
		{"12345678", "********"},
		{"123456789", "1234**789"},
		{"***REMOVED***", "8674*******************IoS"},
	}
	for _, tc := range cases {
		got := maskToken(tc.in)
		if got != tc.want {
			t.Errorf("maskToken(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRenderConfigSummaryIncludesEverything(t *testing.T) {
	cfg := config.WithDefaults(config.Config{
		Telegram: config.Telegram{
			BotToken:  "***REMOVED***",
			ChannelID: "-1003900879151",
		},
		Funds: []config.FundEntry{
			{
				FundID:        "FIL133_A_USD",
				URL:           "https://www.analizy.pl/x",
				PurchaseDate:  time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
				PurchaseUnits: 11059,
				PurchasePrice: 13.30,
			},
		},
	})
	var buf bytes.Buffer
	renderConfigSummary(&buf, "/tmp/config.yaml", cfg, map[string]string{
		"FIL133_A_USD": "Fidelity Funds Global Dividend",
	})

	out := buf.String()
	mustContain := []string{
		"/tmp/config.yaml",
		"8674*******************IoS",
		"-1003900879151",
		"18:00",
		"USD",
		"3.0",
		"Funds (1):",
		"FIL133_A_USD",
		"Fidelity Funds Global Dividend",
		"2025-09-15",
		"11059",
		"13.30",
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q\n--\n%s", s, out)
		}
	}
}

func TestRenderConfigSummaryDoesNotLeakBotToken(t *testing.T) {
	token := "***REMOVED***"
	cfg := config.WithDefaults(config.Config{
		Telegram: config.Telegram{BotToken: token, ChannelID: "x"},
	})
	var buf bytes.Buffer
	renderConfigSummary(&buf, "p", cfg, nil)
	if strings.Contains(buf.String(), token) {
		t.Errorf("output leaks bot token in plaintext")
	}
}
