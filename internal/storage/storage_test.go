package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func openTest(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return s
}

func TestFundRoundTrip(t *testing.T) {
	s := openTest(t)
	want := Fund{
		FundID:        "FIL133_A_USD",
		URL:           "https://www.analizy.pl/fundusze-zagraniczne/FIL133_A_USD/x",
		Name:          "Fidelity Funds Global Dividend Plus Fund A (Acc) (USD)",
		Currency:      "USD",
		PurchaseDate:  time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
		PurchaseUnits: 100.0,
		PurchasePrice: 14.20,
	}
	if err := s.UpsertFund(want); err != nil {
		t.Fatalf("UpsertFund: %v", err)
	}
	got, err := s.GetFund("FIL133_A_USD")
	if err != nil {
		t.Fatalf("GetFund: %v", err)
	}
	if got.Name != want.Name || got.Currency != want.Currency || got.PurchaseUnits != want.PurchaseUnits {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if !got.PurchaseDate.Equal(want.PurchaseDate) {
		t.Errorf("PurchaseDate = %v, want %v", got.PurchaseDate, want.PurchaseDate)
	}
}

func TestListFunds(t *testing.T) {
	s := openTest(t)
	for _, id := range []string{"A", "B", "C"} {
		if err := s.UpsertFund(Fund{FundID: id, URL: "u", Name: id, Currency: "USD"}); err != nil {
			t.Fatal(err)
		}
	}
	funds, err := s.ListFunds()
	if err != nil {
		t.Fatal(err)
	}
	if len(funds) != 3 {
		t.Errorf("len = %d, want 3", len(funds))
	}
}

func TestPriceUpsertDeduplicates(t *testing.T) {
	s := openTest(t)
	if err := s.UpsertFund(Fund{FundID: "X", URL: "u", Name: "X", Currency: "USD"}); err != nil {
		t.Fatal(err)
	}
	d := time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	if err := s.UpsertPrice(Price{FundID: "X", Date: d, NAV: 10.0, Source: "daily"}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertPrice(Price{FundID: "X", Date: d, NAV: 11.0, Source: "daily"}); err != nil {
		t.Fatal(err)
	}
	prices, err := s.RecentPrices("X", d.AddDate(0, 0, -1))
	if err != nil {
		t.Fatal(err)
	}
	if len(prices) != 1 {
		t.Fatalf("len = %d, want 1", len(prices))
	}
	if prices[0].NAV != 11.0 {
		t.Errorf("NAV = %v, want 11.0 (latest upsert wins)", prices[0].NAV)
	}
}

func TestRecentPricesOrdered(t *testing.T) {
	s := openTest(t)
	if err := s.UpsertFund(Fund{FundID: "X", URL: "u", Name: "X", Currency: "USD"}); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	for i, nav := range []float64{10.0, 10.5, 11.0, 10.8} {
		if err := s.UpsertPrice(Price{
			FundID: "X",
			Date:   base.AddDate(0, 0, -i),
			NAV:    nav,
			Source: "daily",
		}); err != nil {
			t.Fatal(err)
		}
	}
	since := base.AddDate(0, 0, -10)
	prices, err := s.RecentPrices("X", since)
	if err != nil {
		t.Fatal(err)
	}
	if len(prices) != 4 {
		t.Fatalf("len = %d, want 4", len(prices))
	}
	for i := 1; i < len(prices); i++ {
		if !prices[i-1].Date.Before(prices[i].Date) {
			t.Errorf("not sorted ascending: %v then %v", prices[i-1].Date, prices[i].Date)
		}
	}
}

func TestRecordRun(t *testing.T) {
	s := openTest(t)
	if err := s.RecordRun("ok", "all good"); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordRun("error", "boom"); err != nil {
		t.Fatal(err)
	}
	runs, err := s.LastRuns(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 {
		t.Fatalf("len = %d, want 2", len(runs))
	}
	if runs[0].Status != "error" {
		t.Errorf("most recent status = %q, want error (DESC ordered)", runs[0].Status)
	}
}
