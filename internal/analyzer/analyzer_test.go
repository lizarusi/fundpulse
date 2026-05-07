package analyzer

import (
	"testing"

	"github.com/lizarusi/investments-healthcheck/internal/portfolio"
)

func defaultThresholds() Thresholds {
	return Thresholds{
		AlertSingleDayPct:       3.0,
		Alert5dCumulativePct:    7.0,
		AlertPortfolio5dPct:     5.0,
		Warning5dCumulativePct:  3.0,
		Good5dCumulativePct:     1.0,
		VeryGood5dCumulativePct: 5.0,
	}
}

func TestPerFundAlertOnSingleDayDrop(t *testing.T) {
	f := portfolio.FundMetrics{Change1DPct: -3.5, Change5DPct: -1.0, HasFiveDays: true}
	v := PerFund(f, defaultThresholds())
	if v.Level != Alert {
		t.Errorf("Level = %v, want Alert", v.Level)
	}
	if len(v.Reasons) == 0 {
		t.Error("Reasons should not be empty")
	}
}

func TestPerFundAlertOn5DayCumulativeDrop(t *testing.T) {
	f := portfolio.FundMetrics{Change1DPct: -1.0, Change5DPct: -8.0, HasFiveDays: true}
	v := PerFund(f, defaultThresholds())
	if v.Level != Alert {
		t.Errorf("Level = %v, want Alert", v.Level)
	}
}

func TestPerFundWarningOn5DayMildDrop(t *testing.T) {
	f := portfolio.FundMetrics{Change1DPct: -0.5, Change5DPct: -4.0, HasFiveDays: true}
	v := PerFund(f, defaultThresholds())
	if v.Level != Warning {
		t.Errorf("Level = %v, want Warning", v.Level)
	}
}

func TestPerFundStableOnTinyMoves(t *testing.T) {
	f := portfolio.FundMetrics{Change1DPct: 0.2, Change5DPct: 0.5, HasFiveDays: true}
	v := PerFund(f, defaultThresholds())
	if v.Level != Stable {
		t.Errorf("Level = %v, want Stable", v.Level)
	}
}

func TestPerFundColdStartSkips5dRules(t *testing.T) {
	f := portfolio.FundMetrics{Change1DPct: -1.0, Change5DPct: -10.0, HasFiveDays: false}
	v := PerFund(f, defaultThresholds())
	if v.Level == Alert || v.Level == Warning {
		t.Errorf("Level = %v, want non-alert (cold start should skip 5d rules)", v.Level)
	}
}

func TestPerFundColdStartStillAlertsOnSingleDayDrop(t *testing.T) {
	f := portfolio.FundMetrics{Change1DPct: -5.0, Change5DPct: 0, HasFiveDays: false}
	v := PerFund(f, defaultThresholds())
	if v.Level != Alert {
		t.Errorf("Level = %v, want Alert (1d rule applies even cold)", v.Level)
	}
}

func TestOverallAlertOnPortfolioDrop(t *testing.T) {
	funds := []portfolio.FundMetrics{
		{Change1DPct: -0.5, Change5DPct: -2.0, HasFiveDays: true},
	}
	port := portfolio.PortfolioMetrics{Change5DPct: -6.0}
	v := Overall(funds, port, defaultThresholds())
	if v.Level != Alert {
		t.Errorf("Level = %v, want Alert", v.Level)
	}
}

func TestOverallVeryGoodOnPortfolioGain(t *testing.T) {
	funds := []portfolio.FundMetrics{
		{Change1DPct: 0.5, Change5DPct: 5.5, HasFiveDays: true},
	}
	port := portfolio.PortfolioMetrics{Change5DPct: 5.5}
	v := Overall(funds, port, defaultThresholds())
	if v.Level != VeryGood {
		t.Errorf("Level = %v, want VeryGood", v.Level)
	}
}

func TestOverallWorstFundWins(t *testing.T) {
	funds := []portfolio.FundMetrics{
		{Change1DPct: 0.5, Change5DPct: 5.5, HasFiveDays: true},
		{Change1DPct: -3.5, Change5DPct: -1.0, HasFiveDays: true},
	}
	port := portfolio.PortfolioMetrics{Change5DPct: 2.0}
	v := Overall(funds, port, defaultThresholds())
	if v.Level != Alert {
		t.Errorf("Level = %v, want Alert (worst fund wins)", v.Level)
	}
}
