package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Telegram     Telegram    `yaml:"telegram"`
	BaseCurrency string      `yaml:"base_currency"`
	ScheduleTime string      `yaml:"schedule_time"`
	Thresholds   Thresholds  `yaml:"thresholds"`
	Funds        []FundEntry `yaml:"funds"`
}

type Telegram struct {
	BotToken  string `yaml:"bot_token"`
	ChannelID string `yaml:"channel_id"`
}

type Thresholds struct {
	AlertSingleDayPct       float64 `yaml:"alert_single_day_pct"`
	Alert5dCumulativePct    float64 `yaml:"alert_5d_cumulative_pct"`
	AlertPortfolio5dPct     float64 `yaml:"alert_portfolio_5d_pct"`
	Warning5dCumulativePct  float64 `yaml:"warning_5d_cumulative_pct"`
	Good5dCumulativePct     float64 `yaml:"good_5d_cumulative_pct"`
	VeryGood5dCumulativePct float64 `yaml:"very_good_5d_cumulative_pct"`
}

type FundEntry struct {
	FundID        string    `yaml:"fund_id"`
	URL           string    `yaml:"url"`
	PurchaseDate  time.Time `yaml:"purchase_date"`
	PurchaseUnits float64   `yaml:"purchase_units"`
	PurchasePrice float64   `yaml:"purchase_price"`
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("parse yaml: %w", err)
	}
	return WithDefaults(c), nil
}

func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func WithDefaults(c Config) Config {
	if c.BaseCurrency == "" {
		c.BaseCurrency = "USD"
	}
	if c.ScheduleTime == "" {
		c.ScheduleTime = "18:00"
	}
	if c.Thresholds.AlertSingleDayPct == 0 {
		c.Thresholds.AlertSingleDayPct = 3.0
	}
	if c.Thresholds.Alert5dCumulativePct == 0 {
		c.Thresholds.Alert5dCumulativePct = 7.0
	}
	if c.Thresholds.AlertPortfolio5dPct == 0 {
		c.Thresholds.AlertPortfolio5dPct = 5.0
	}
	if c.Thresholds.Warning5dCumulativePct == 0 {
		c.Thresholds.Warning5dCumulativePct = 3.0
	}
	if c.Thresholds.Good5dCumulativePct == 0 {
		c.Thresholds.Good5dCumulativePct = 1.0
	}
	if c.Thresholds.VeryGood5dCumulativePct == 0 {
		c.Thresholds.VeryGood5dCumulativePct = 5.0
	}
	return c
}

func Validate(c Config) error {
	if c.Telegram.BotToken == "" {
		return fmt.Errorf("telegram.bot_token is required")
	}
	if c.Telegram.ChannelID == "" {
		return fmt.Errorf("telegram.channel_id is required")
	}
	if len(c.Funds) == 0 {
		return fmt.Errorf("at least one fund must be configured")
	}
	for i, f := range c.Funds {
		if f.FundID == "" {
			return fmt.Errorf("funds[%d]: fund_id is required", i)
		}
		if f.URL == "" {
			return fmt.Errorf("funds[%d] (%s): url is required", i, f.FundID)
		}
	}
	return nil
}

func DefaultPath() string {
	if p := os.Getenv("HEALTHCHECK_CONFIG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "investments-healthcheck", "config.yaml")
}

func DefaultDBPath() string {
	if p := os.Getenv("HEALTHCHECK_DB"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "investments-healthcheck", "data.db")
}

func DefaultLogPath() string {
	if p := os.Getenv("HEALTHCHECK_LOG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Logs", "investments-healthcheck", "run.log")
}
