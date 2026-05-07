package scraper

import (
	"os"
	"testing"
	"time"
)

func TestParseFidelityFund(t *testing.T) {
	f, err := os.Open("testdata/fil133.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	got, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	want := FundSnapshot{
		Name:        "Fidelity Funds Global Dividend Plus Fund A (Acc) (USD)",
		Currency:    "USD",
		NAV:         14.80,
		NAVDate:     time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC),
		Change1DAbs: 0.35,
		Change1DPct: 2.42,
		RiskLabel:   "Podwyższone",
		RiskLevel:   4,
	}

	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.Currency != want.Currency {
		t.Errorf("Currency = %q, want %q", got.Currency, want.Currency)
	}
	if got.NAV != want.NAV {
		t.Errorf("NAV = %v, want %v", got.NAV, want.NAV)
	}
	if !got.NAVDate.Equal(want.NAVDate) {
		t.Errorf("NAVDate = %v, want %v", got.NAVDate, want.NAVDate)
	}
	if got.Change1DAbs != want.Change1DAbs {
		t.Errorf("Change1DAbs = %v, want %v", got.Change1DAbs, want.Change1DAbs)
	}
	if got.Change1DPct != want.Change1DPct {
		t.Errorf("Change1DPct = %v, want %v", got.Change1DPct, want.Change1DPct)
	}
	if got.RiskLabel != want.RiskLabel {
		t.Errorf("RiskLabel = %q, want %q", got.RiskLabel, want.RiskLabel)
	}
	if got.RiskLevel != want.RiskLevel {
		t.Errorf("RiskLevel = %v, want %v", got.RiskLevel, want.RiskLevel)
	}
}
