package scraper

import (
	"fmt"
	"net/http"
	"time"
)

const userAgent = "investments-healthcheck/0.1 (+https://github.com/lizarusi/investments-healthcheck)"

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
