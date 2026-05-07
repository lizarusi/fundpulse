package wizard

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lizarusi/investments-healthcheck/internal/config"
	"github.com/lizarusi/investments-healthcheck/internal/scraper"
)

type Validator interface {
	Validate(fundURL string) (string, error)
}

type liveValidator struct{}

func (liveValidator) Validate(fundURL string) (string, error) {
	snap, err := scraper.FetchSnapshot(fundURL)
	if err != nil {
		return "", err
	}
	return snap.Name, nil
}

func DefaultValidator() Validator { return liveValidator{} }

func Run(in io.Reader, out io.Writer, v Validator) (config.Config, error) {
	r := bufio.NewReader(in)
	cfg := config.WithDefaults(config.Config{})

	fmt.Fprintln(out, "Investments Healthcheck — first-time setup")
	fmt.Fprintln(out, strings.Repeat("-", 50))

	token, err := prompt(r, out, "Telegram bot token: ")
	if err != nil {
		return cfg, err
	}
	cfg.Telegram.BotToken = token

	channel, err := prompt(r, out, "Telegram channel ID (e.g. -100123...): ")
	if err != nil {
		return cfg, err
	}
	cfg.Telegram.ChannelID = channel

	for {
		more, err := prompt(r, out, "Add a fund? (y/n): ")
		if err != nil {
			return cfg, err
		}
		if strings.ToLower(strings.TrimSpace(more)) != "y" {
			break
		}
		f, err := promptFund(r, out, v)
		if err != nil {
			fmt.Fprintf(out, "  ! %v — skipping this fund\n", err)
			continue
		}
		cfg.Funds = append(cfg.Funds, f)
	}

	return cfg, nil
}

func promptFund(r *bufio.Reader, out io.Writer, v Validator) (config.FundEntry, error) {
	urlStr, err := prompt(r, out, "  Fund URL on analizy.pl: ")
	if err != nil {
		return config.FundEntry{}, err
	}
	fundID, err := ExtractFundID(urlStr)
	if err != nil {
		return config.FundEntry{}, fmt.Errorf("extract fund id: %w", err)
	}
	if v != nil {
		if name, err := v.Validate(urlStr); err == nil {
			fmt.Fprintf(out, "    -> %s (%s)\n", name, fundID)
		} else {
			fmt.Fprintf(out, "    ! could not verify (%v); proceeding anyway\n", err)
		}
	}

	dateStr, err := prompt(r, out, "  Purchase date (YYYY-MM-DD): ")
	if err != nil {
		return config.FundEntry{}, err
	}
	d, err := time.Parse("2006-01-02", strings.TrimSpace(dateStr))
	if err != nil {
		return config.FundEntry{}, fmt.Errorf("parse date: %w", err)
	}

	unitsStr, err := prompt(r, out, "  Units purchased: ")
	if err != nil {
		return config.FundEntry{}, err
	}
	units, err := strconv.ParseFloat(strings.TrimSpace(unitsStr), 64)
	if err != nil {
		return config.FundEntry{}, fmt.Errorf("parse units: %w", err)
	}

	priceStr, err := prompt(r, out, "  Purchase NAV price: ")
	if err != nil {
		return config.FundEntry{}, err
	}
	price, err := strconv.ParseFloat(strings.TrimSpace(priceStr), 64)
	if err != nil {
		return config.FundEntry{}, fmt.Errorf("parse price: %w", err)
	}

	return config.FundEntry{
		FundID:        fundID,
		URL:           urlStr,
		PurchaseDate:  d,
		PurchaseUnits: units,
		PurchasePrice: price,
	}, nil
}

func prompt(r *bufio.Reader, out io.Writer, msg string) (string, error) {
	fmt.Fprint(out, msg)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func ExtractFundID(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	if !strings.Contains(u.Host, "analizy.pl") {
		return "", fmt.Errorf("not an analizy.pl URL: %s", rawURL)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("URL path too short: %s", u.Path)
	}
	id := parts[1]
	if id == "" {
		return "", fmt.Errorf("empty fund id in URL: %s", rawURL)
	}
	return id, nil
}
