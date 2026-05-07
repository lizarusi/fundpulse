package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lizarusi/investments-healthcheck/internal/analyzer"
)

type Report struct {
	Date       time.Time
	Currency   string
	TotalValue float64
	TotalPL    float64
	TotalPLPct float64
	Verdict    analyzer.Verdict
	Funds      []FundLine
}

type FundLine struct {
	Name          string
	Currency      string
	Verdict       analyzer.Verdict
	Change1DPct   float64
	Change30DPct  float64
	HasThirtyDays bool
	ProfitLoss    float64
	RiskLabel     string
}

func Render(r Report) string {
	var b strings.Builder

	fmt.Fprintf(&b, "📊 Daily portfolio update — %s\n", r.Date.Format("2006-01-02"))
	fmt.Fprintf(&b, "💰 Total: %s   P/L: %s (%s)\n",
		formatMoney(r.Currency, r.TotalValue),
		formatSignedMoney(r.Currency, r.TotalPL),
		formatSignedPct(r.TotalPLPct),
	)
	fmt.Fprintf(&b, "Verdict: %s %s\n\n", r.Verdict.Level.Emoji(), r.Verdict.Level)

	b.WriteString("Funds:\n")
	for _, f := range r.Funds {
		fmt.Fprintf(&b, "%s %s %s (1d)", f.Verdict.Level.Emoji(), f.Name, formatSignedPct(f.Change1DPct))
		if f.HasThirtyDays {
			fmt.Fprintf(&b, " / %s (30d)", formatSignedPct(f.Change30DPct))
		}
		fmt.Fprintf(&b, " / P/L %s", formatSignedMoney(f.Currency, f.ProfitLoss))
		if f.RiskLabel != "" {
			fmt.Fprintf(&b, " [%s]", f.RiskLabel)
		}
		if f.Verdict.Level == analyzer.Alert || f.Verdict.Level == analyzer.Warning {
			reason := ""
			if len(f.Verdict.Reasons) > 0 {
				reason = ": " + f.Verdict.Reasons[0]
			}
			fmt.Fprintf(&b, "  ⚠️ %s%s", f.Verdict.Level, reason)
		}
		b.WriteString("\n")
	}

	if len(r.Verdict.Reasons) > 0 && r.Verdict.Level != analyzer.Stable {
		fmt.Fprintf(&b, "Why: %s.\n", strings.Join(r.Verdict.Reasons, "; "))
	}

	return b.String()
}

func formatMoney(currency string, v float64) string {
	sym := currencySymbol(currency)
	return fmt.Sprintf("%s%s", sym, formatThousands(v))
}

func formatSignedMoney(currency string, v float64) string {
	sym := currencySymbol(currency)
	if v >= 0 {
		return fmt.Sprintf("+%s%s", sym, formatThousands(v))
	}
	return fmt.Sprintf("-%s%s", sym, formatThousands(-v))
}

func formatSignedPct(p float64) string {
	if p >= 0 {
		return fmt.Sprintf("+%.1f%%", p)
	}
	return fmt.Sprintf("%.1f%%", p)
}

func currencySymbol(c string) string {
	switch c {
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "PLN":
		return "zł "
	default:
		return c + " "
	}
}

func formatThousands(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	dot := strings.Index(s, ".")
	intPart, fracPart := s[:dot], s[dot:]
	n := len(intPart)
	if n <= 3 {
		return intPart + fracPart
	}
	var out strings.Builder
	first := n % 3
	if first > 0 {
		out.WriteString(intPart[:first])
		if n > first {
			out.WriteString(",")
		}
	}
	for i := first; i < n; i += 3 {
		out.WriteString(intPart[i : i+3])
		if i+3 < n {
			out.WriteString(",")
		}
	}
	out.WriteString(fracPart)
	return out.String()
}

type Client struct {
	token   string
	chatID  string
	baseURL string
	http    *http.Client
}

func NewClient(token, chatID string) *Client {
	return &Client{
		token:   token,
		chatID:  chatID,
		baseURL: "https://api.telegram.org",
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Send(text string) error {
	payload := map[string]string{
		"chat_id":                  c.chatID,
		"text":                     text,
		"disable_web_page_preview": "true",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	url := fmt.Sprintf("%s/bot%s/sendMessage", c.baseURL, c.token)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
