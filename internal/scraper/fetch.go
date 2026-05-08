package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const userAgent = "fundpulse/0.1 (+https://github.com/lizarusi/fundpulse)"

var defaultClient = &http.Client{Timeout: 30 * time.Second}

func FetchSnapshot(url string) (FundSnapshot, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return FundSnapshot{}, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", "pl,en;q=0.8")

	resp, err := defaultClient.Do(req)
	if err != nil {
		return FundSnapshot{}, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return FundSnapshot{}, fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}

	return Parse(resp.Body)
}

type HistoryPoint struct {
	Date  time.Time
	Value float64
}

type historyResponse struct {
	Series []struct {
		ID    string `json:"id"`
		Price []struct {
			Date  any `json:"date"`
			Value any `json:"value"`
		} `json:"price"`
	} `json:"series"`
}

func FetchHistory(fundID string, from, to time.Time) ([]HistoryPoint, error) {
	url := fmt.Sprintf("https://www.analizy.pl/api/quotation/chart/%s?dateFrom=%s&dateTo=%s",
		fundID, from.Format("2006-01-02"), to.Format("2006-01-02"))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var hr historyResponse
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		return nil, err
	}

	var points []HistoryPoint
	targetID := "fund_" + fundID
	for _, s := range hr.Series {
		if s.ID != targetID {
			continue
		}
		points = make([]HistoryPoint, 0, len(s.Price))
		for _, p := range s.Price {
			var t time.Time
			switch v := p.Date.(type) {
			case float64:
				t = time.Unix(int64(v)/1000, (int64(v)%1000)*1000000).UTC()
			case string:
				// Try Unix timestamp string
				if ms, err := strconv.ParseInt(v, 10, 64); err == nil {
					t = time.Unix(ms/1000, (ms%1000)*1000000).UTC()
				} else if parsed, err := time.Parse("2006-01-02", v); err == nil {
					t = parsed.UTC()
				} else {
					continue
				}
			default:
				continue
			}

			var val float64
			switch v := p.Value.(type) {
			case float64:
				val = v
			case string:
				if parsed, err := strconv.ParseFloat(v, 64); err == nil {
					val = parsed
				} else {
					continue
				}
			default:
				continue
			}

			points = append(points, HistoryPoint{
				Date:  t,
				Value: val,
			})
		}
		break
	}
	return points, nil
}
