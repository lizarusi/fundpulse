package portfolio

import (
	"math"
	"testing"
	"time"
)

func almost(a, b float64) bool { return math.Abs(a-b) < 0.001 }

func mkPrices(start time.Time, navs ...float64) []float64Day {
	out := make([]float64Day, len(navs))
	for i, nav := range navs {
		out[i] = float64Day{Date: start.AddDate(0, 0, i), NAV: nav}
	}
	return out
}

type float64Day struct {
	Date time.Time
	NAV  float64
}

func TestComputeFundMetrics(t *testing.T) {
	purchase := PurchaseInfo{
		Units: 100,
		Price: 10.0,
		Date:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	prices := []Price{
		{Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), NAV: 11.0},
		{Date: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC), NAV: 11.5},
		{Date: time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC), NAV: 11.2},
		{Date: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC), NAV: 11.8},
		{Date: time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC), NAV: 12.0},
		{Date: time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC), NAV: 12.1},
	}

	m := ComputeFund(purchase, prices)

	if !almost(m.CurrentValue, 1210.0) {
		t.Errorf("CurrentValue = %v, want 1210.0", m.CurrentValue)
	}
	if !almost(m.ProfitLoss, 210.0) {
		t.Errorf("ProfitLoss = %v, want 210.0", m.ProfitLoss)
	}
	if !almost(m.ProfitLossPct, 21.0) {
		t.Errorf("ProfitLossPct = %v, want 21.0", m.ProfitLossPct)
	}
	if !almost(m.Change1DPct, (12.1-12.0)/12.0*100) {
		t.Errorf("Change1DPct = %v, want ~0.833", m.Change1DPct)
	}
	if !almost(m.Change5DPct, (12.1-11.0)/11.0*100) {
		t.Errorf("Change5DPct = %v, want ~10.0", m.Change5DPct)
	}
	if !m.HasFiveDays {
		t.Error("HasFiveDays should be true")
	}
}

func TestComputeFundColdStart(t *testing.T) {
	purchase := PurchaseInfo{Units: 100, Price: 10.0, Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)}
	prices := []Price{
		{Date: time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC), NAV: 11.0},
		{Date: time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC), NAV: 11.5},
	}
	m := ComputeFund(purchase, prices)
	if m.HasFiveDays {
		t.Error("HasFiveDays should be false with only 2 days of history")
	}
	if !almost(m.ProfitLossPct, 15.0) {
		t.Errorf("ProfitLossPct = %v, want 15.0 (vs purchase price)", m.ProfitLossPct)
	}
	if !almost(m.Change1DPct, (11.5-11.0)/11.0*100) {
		t.Errorf("Change1DPct = %v", m.Change1DPct)
	}
}

func TestComputePortfolioTotal(t *testing.T) {
	funds := []FundMetrics{
		{Currency: "USD", CurrentValue: 1000, ProfitLoss: 100, Change5DPct: 2.0},
		{Currency: "USD", CurrentValue: 2000, ProfitLoss: 200, Change5DPct: -1.0},
	}
	p := ComputePortfolio(funds, "USD")
	if !almost(p.TotalValue, 3000) {
		t.Errorf("TotalValue = %v, want 3000", p.TotalValue)
	}
	if !almost(p.TotalProfitLoss, 300) {
		t.Errorf("TotalProfitLoss = %v, want 300", p.TotalProfitLoss)
	}
	expectedPct := (1000.0/3000.0)*2.0 + (2000.0/3000.0)*-1.0
	if !almost(p.Change5DPct, expectedPct) {
		t.Errorf("Change5DPct = %v, want %v", p.Change5DPct, expectedPct)
	}
}
