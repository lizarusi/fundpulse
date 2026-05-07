package analyzer

import (
	"fmt"

	"github.com/lizarusi/investments-healthcheck/internal/portfolio"
)

type Level int

const (
	Stable Level = iota
	Good
	VeryGood
	Warning
	Alert
)

func (l Level) String() string {
	switch l {
	case VeryGood:
		return "VERY GOOD"
	case Good:
		return "GOOD"
	case Stable:
		return "STABLE"
	case Warning:
		return "WARNING"
	case Alert:
		return "ALERT"
	default:
		return "?"
	}
}

func (l Level) Emoji() string {
	switch l {
	case VeryGood:
		return "🟢🟢"
	case Good:
		return "🟢"
	case Stable:
		return "⚪"
	case Warning:
		return "🟡"
	case Alert:
		return "🔴"
	default:
		return "?"
	}
}

type Thresholds struct {
	AlertSingleDayPct       float64
	Alert5dCumulativePct    float64
	AlertPortfolio5dPct     float64
	Warning5dCumulativePct  float64
	Good5dCumulativePct     float64
	VeryGood5dCumulativePct float64
}

type Verdict struct {
	Level   Level
	Reasons []string
}

func PerFund(f portfolio.FundMetrics, t Thresholds) Verdict {
	v := Verdict{Level: Stable}
	if f.Change1DPct <= -t.AlertSingleDayPct {
		v.Level = Alert
		v.Reasons = append(v.Reasons, fmt.Sprintf("1d drop %.2f%% ≥ %.1f%%", -f.Change1DPct, t.AlertSingleDayPct))
	}
	if f.HasFiveDays {
		if f.Change5DPct <= -t.Alert5dCumulativePct {
			v.Level = Alert
			v.Reasons = append(v.Reasons, fmt.Sprintf("5d drop %.2f%% ≥ %.1f%%", -f.Change5DPct, t.Alert5dCumulativePct))
		} else if v.Level != Alert && f.Change5DPct <= -t.Warning5dCumulativePct {
			v.Level = Warning
			v.Reasons = append(v.Reasons, fmt.Sprintf("5d drop %.2f%% ≥ %.1f%%", -f.Change5DPct, t.Warning5dCumulativePct))
		} else if v.Level == Stable && f.Change5DPct >= t.VeryGood5dCumulativePct {
			v.Level = VeryGood
			v.Reasons = append(v.Reasons, fmt.Sprintf("5d gain %.2f%%", f.Change5DPct))
		} else if v.Level == Stable && f.Change5DPct >= t.Good5dCumulativePct {
			v.Level = Good
			v.Reasons = append(v.Reasons, fmt.Sprintf("5d gain %.2f%%", f.Change5DPct))
		}
	}
	return v
}

func Overall(funds []portfolio.FundMetrics, port portfolio.PortfolioMetrics, t Thresholds) Verdict {
	v := Verdict{Level: Stable}

	for _, f := range funds {
		fv := PerFund(f, t)
		if fv.Level == Alert || fv.Level == Warning {
			if fv.Level > v.Level {
				v = fv
			}
		}
	}

	if port.Change5DPct <= -t.AlertPortfolio5dPct {
		v.Level = Alert
		v.Reasons = append(v.Reasons, fmt.Sprintf("portfolio 5d drop %.2f%% ≥ %.1f%%", -port.Change5DPct, t.AlertPortfolio5dPct))
		return v
	}

	if v.Level == Stable {
		if port.Change5DPct >= t.VeryGood5dCumulativePct {
			v.Level = VeryGood
			v.Reasons = append(v.Reasons, fmt.Sprintf("portfolio 5d gain %.2f%%", port.Change5DPct))
		} else if port.Change5DPct >= t.Good5dCumulativePct {
			v.Level = Good
			v.Reasons = append(v.Reasons, fmt.Sprintf("portfolio 5d gain %.2f%%", port.Change5DPct))
		}
	}

	return v
}
