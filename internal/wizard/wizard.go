package wizard

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lizarusi/fundpulse/internal/config"
	"github.com/lizarusi/fundpulse/internal/scraper"
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

func Run(in io.Reader, out io.Writer, initialCfg config.Config, v Validator) (config.Config, error) {
	r := bufio.NewReader(in)
	cfg := initialCfg

	fmt.Fprintln(out, "Investments Healthcheck — setup")
	fmt.Fprintln(out, strings.Repeat("-", 50))

	token, err := prompt(r, out, "Telegram bot token", cfg.Telegram.BotToken)
	if err != nil {
		return cfg, err
	}
	cfg.Telegram.BotToken = token

	channel, err := prompt(r, out, "Telegram channel ID (e.g. -100123...)", cfg.Telegram.ChannelID)
	if err != nil {
		return cfg, err
	}
	cfg.Telegram.ChannelID = channel

	schedule, err := prompt(r, out, "Schedule time (HH:MM)", cfg.ScheduleTime)
	if err != nil {
		return cfg, err
	}
	cfg.ScheduleTime = schedule

	currency, err := prompt(r, out, "Base currency", cfg.BaseCurrency)
	if err != nil {
		return cfg, err
	}
	cfg.BaseCurrency = currency

	if len(cfg.Funds) > 0 {
		fmt.Fprintln(out, "\nExisting funds:")
		var newFunds []config.FundEntry
		for i, f := range cfg.Funds {
			fmt.Fprintf(out, "  %d. %s (%s)\n", i+1, f.FundID, f.URL)
			action, err := prompt(r, out, "     (k)eep / (e)dit / (d)elete", "k")
			if err != nil {
				return cfg, err
			}
			switch strings.ToLower(action) {
			case "e":
				updated, err := promptFund(r, out, v, f)
				if err != nil {
					fmt.Fprintf(out, "     ! %v — keeping original\n", err)
					newFunds = append(newFunds, f)
				} else {
					newFunds = append(newFunds, updated)
				}
			case "d":
				fmt.Fprintf(out, "     - deleted %s\n", f.FundID)
			default: // "k" or anything else
				newFunds = append(newFunds, f)
			}
		}
		cfg.Funds = newFunds
	}

	for {
		msg := "Add a fund?"
		if len(cfg.Funds) == 0 {
			msg = "Add your first fund?"
		}
		more, err := prompt(r, out, msg+" (y/n)", "y")
		if err != nil {
			return cfg, err
		}
		if strings.ToLower(more) != "y" {
			break
		}
		f, err := promptFund(r, out, v, config.FundEntry{})
		if err != nil {
			fmt.Fprintf(out, "  ! %v — skipping this fund\n", err)
			continue
		}
		cfg.Funds = append(cfg.Funds, f)
	}

	return cfg, nil
}

func promptFund(r *bufio.Reader, out io.Writer, v Validator, initial config.FundEntry) (config.FundEntry, error) {
	urlStr, err := prompt(r, out, "  Fund URL on analizy.pl", initial.URL)
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

	defaultDate := ""
	if !initial.PurchaseDate.IsZero() {
		defaultDate = initial.PurchaseDate.Format("2006-01-02")
	}
	dateStr, err := prompt(r, out, "  Purchase date (YYYY-MM-DD)", defaultDate)
	if err != nil {
		return config.FundEntry{}, err
	}
	d, err := time.Parse("2006-01-02", strings.TrimSpace(dateStr))
	if err != nil {
		return config.FundEntry{}, fmt.Errorf("parse date: %w", err)
	}

	defaultUnits := ""
	if initial.PurchaseUnits != 0 {
		defaultUnits = strconv.FormatFloat(initial.PurchaseUnits, 'f', -1, 64)
	}
	unitsStr, err := prompt(r, out, "  Units purchased", defaultUnits)
	if err != nil {
		return config.FundEntry{}, err
	}
	units, err := strconv.ParseFloat(strings.TrimSpace(unitsStr), 64)
	if err != nil {
		return config.FundEntry{}, fmt.Errorf("parse units: %w", err)
	}

	defaultPrice := ""
	if initial.PurchasePrice != 0 {
		defaultPrice = strconv.FormatFloat(initial.PurchasePrice, 'f', -1, 64)
	}
	priceStr, err := prompt(r, out, "  Purchase NAV price", defaultPrice)
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

func prompt(r *bufio.Reader, out io.Writer, msg string, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Fprintf(out, "%s [%s]: ", msg, defaultValue)
	} else {
		fmt.Fprintf(out, "%s: ", msg)
	}
	line, err := r.ReadString('\n')
	if err != nil && (err != io.EOF || line == "") {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" && defaultValue != "" {
		return defaultValue, nil
	}
	return line, nil
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
