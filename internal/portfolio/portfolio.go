package portfolio

import "time"

type PurchaseInfo struct {
	Units float64
	Price float64
	Date  time.Time
}

type Price struct {
	Date time.Time
	NAV  float64
}

type FundMetrics struct {
	FundID        string
	Name          string
	Currency      string
	CurrentNAV    float64
	NAVDate       time.Time
	CurrentValue  float64
	ProfitLoss    float64
	ProfitLossPct float64
	Change1DPct   float64
	Change5DPct   float64
	Change30DPct  float64
	HasFiveDays   bool
	HasThirtyDays bool
	RiskLabel     string
	RiskLevel     int
}

type PortfolioMetrics struct {
	Currency        string
	TotalValue      float64
	TotalProfitLoss float64
	TotalPLPct      float64
	Change5DPct     float64
}

func ComputeFund(purchase PurchaseInfo, prices []Price) FundMetrics {
	var m FundMetrics
	if len(prices) == 0 {
		return m
	}
	last := prices[len(prices)-1]
	m.CurrentNAV = last.NAV
	m.NAVDate = last.Date
	m.CurrentValue = last.NAV * purchase.Units
	m.ProfitLoss = m.CurrentValue - (purchase.Price * purchase.Units)
	if purchase.Price > 0 {
		m.ProfitLossPct = (last.NAV - purchase.Price) / purchase.Price * 100
	}
	if len(prices) >= 2 {
		prev := prices[len(prices)-2].NAV
		if prev > 0 {
			m.Change1DPct = (last.NAV - prev) / prev * 100
		}
	}
	if len(prices) >= 6 {
		ref := prices[len(prices)-6].NAV
		if ref > 0 {
			m.Change5DPct = (last.NAV - ref) / ref * 100
			m.HasFiveDays = true
		}
	}
	if len(prices) >= 31 {
		ref := prices[len(prices)-31].NAV
		if ref > 0 {
			m.Change30DPct = (last.NAV - ref) / ref * 100
			m.HasThirtyDays = true
		}
	}
	return m
}

func ComputePortfolio(funds []FundMetrics, baseCurrency string) PortfolioMetrics {
	p := PortfolioMetrics{Currency: baseCurrency}
	for _, f := range funds {
		p.TotalValue += f.CurrentValue
		p.TotalProfitLoss += f.ProfitLoss
	}
	cost := p.TotalValue - p.TotalProfitLoss
	if cost > 0 {
		p.TotalPLPct = p.TotalProfitLoss / cost * 100
	}
	if p.TotalValue > 0 {
		for _, f := range funds {
			weight := f.CurrentValue / p.TotalValue
			p.Change5DPct += weight * f.Change5DPct
		}
	}
	return p
}
