package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Fund struct {
	FundID        string
	URL           string
	Name          string
	Currency      string
	PurchaseDate  time.Time
	PurchaseUnits float64
	PurchasePrice float64
}

type Price struct {
	FundID string
	Date   time.Time
	NAV    float64
	Source string
}

type Run struct {
	ID      int64
	RanAt   time.Time
	Status  string
	Message string
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

const schema = `
CREATE TABLE IF NOT EXISTS funds (
  fund_id        TEXT PRIMARY KEY,
  url            TEXT NOT NULL,
  name           TEXT NOT NULL,
  currency       TEXT NOT NULL,
  purchase_date  TEXT,
  purchase_units REAL,
  purchase_price REAL,
  added_at       TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS prices (
  fund_id TEXT NOT NULL REFERENCES funds(fund_id) ON DELETE CASCADE,
  date    TEXT NOT NULL,
  nav     REAL NOT NULL,
  source  TEXT NOT NULL,
  PRIMARY KEY (fund_id, date)
);

CREATE TABLE IF NOT EXISTS runs (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  ran_at  TEXT NOT NULL DEFAULT (datetime('now')),
  status  TEXT NOT NULL,
  message TEXT
);
`

func (s *Store) Migrate() error {
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

const dateFmt = "2006-01-02"

func (s *Store) UpsertFund(f Fund) error {
	_, err := s.db.Exec(`
INSERT INTO funds (fund_id, url, name, currency, purchase_date, purchase_units, purchase_price)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(fund_id) DO UPDATE SET
  url=excluded.url,
  name=excluded.name,
  currency=excluded.currency,
  purchase_date=excluded.purchase_date,
  purchase_units=excluded.purchase_units,
  purchase_price=excluded.purchase_price`,
		f.FundID, f.URL, f.Name, f.Currency,
		nullableDate(f.PurchaseDate), nullableFloat(f.PurchaseUnits), nullableFloat(f.PurchasePrice),
	)
	if err != nil {
		return fmt.Errorf("upsert fund: %w", err)
	}
	return nil
}

func (s *Store) GetFund(id string) (Fund, error) {
	row := s.db.QueryRow(`SELECT fund_id, url, name, currency, purchase_date, purchase_units, purchase_price FROM funds WHERE fund_id = ?`, id)
	return scanFund(row)
}

func (s *Store) ListFunds() ([]Fund, error) {
	rows, err := s.db.Query(`SELECT fund_id, url, name, currency, purchase_date, purchase_units, purchase_price FROM funds ORDER BY fund_id`)
	if err != nil {
		return nil, fmt.Errorf("list funds: %w", err)
	}
	defer rows.Close()
	var out []Fund
	for rows.Next() {
		f, err := scanFund(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanFund(r rowScanner) (Fund, error) {
	var f Fund
	var dateStr sql.NullString
	var units, price sql.NullFloat64
	if err := r.Scan(&f.FundID, &f.URL, &f.Name, &f.Currency, &dateStr, &units, &price); err != nil {
		return Fund{}, err
	}
	if dateStr.Valid {
		t, err := time.Parse(dateFmt, dateStr.String)
		if err != nil {
			return Fund{}, fmt.Errorf("parse purchase_date %q: %w", dateStr.String, err)
		}
		f.PurchaseDate = t
	}
	if units.Valid {
		f.PurchaseUnits = units.Float64
	}
	if price.Valid {
		f.PurchasePrice = price.Float64
	}
	return f, nil
}

func (s *Store) UpsertPrice(p Price) error {
	_, err := s.db.Exec(`
INSERT INTO prices (fund_id, date, nav, source)
VALUES (?, ?, ?, ?)
ON CONFLICT(fund_id, date) DO UPDATE SET
  nav=excluded.nav,
  source=excluded.source`,
		p.FundID, p.Date.Format(dateFmt), p.NAV, p.Source,
	)
	if err != nil {
		return fmt.Errorf("upsert price: %w", err)
	}
	return nil
}

func (s *Store) RecentPrices(fundID string, since time.Time) ([]Price, error) {
	rows, err := s.db.Query(
		`SELECT fund_id, date, nav, source FROM prices WHERE fund_id = ? AND date >= ? ORDER BY date ASC`,
		fundID, since.Format(dateFmt),
	)
	if err != nil {
		return nil, fmt.Errorf("recent prices: %w", err)
	}
	defer rows.Close()
	var out []Price
	for rows.Next() {
		var p Price
		var dateStr string
		if err := rows.Scan(&p.FundID, &dateStr, &p.NAV, &p.Source); err != nil {
			return nil, err
		}
		t, err := time.Parse(dateFmt, dateStr)
		if err != nil {
			return nil, fmt.Errorf("parse date %q: %w", dateStr, err)
		}
		p.Date = t
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) RecordRun(status, message string) error {
	_, err := s.db.Exec(`INSERT INTO runs (status, message) VALUES (?, ?)`, status, message)
	if err != nil {
		return fmt.Errorf("record run: %w", err)
	}
	return nil
}

func (s *Store) LastRuns(limit int) ([]Run, error) {
	rows, err := s.db.Query(`SELECT id, ran_at, status, COALESCE(message, '') FROM runs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("last runs: %w", err)
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		var r Run
		var ranAt string
		if err := rows.Scan(&r.ID, &ranAt, &r.Status, &r.Message); err != nil {
			return nil, err
		}
		t, err := time.Parse("2006-01-02 15:04:05", ranAt)
		if err != nil {
			return nil, fmt.Errorf("parse ran_at %q: %w", ranAt, err)
		}
		r.RanAt = t
		out = append(out, r)
	}
	return out, rows.Err()
}

func nullableDate(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Format(dateFmt)
}

func nullableFloat(v float64) any {
	if v == 0 {
		return nil
	}
	return v
}
