package scraper

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFetchSnapshot(t *testing.T) {
	fixture, err := os.ReadFile("testdata/fil133.html")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	snap, err := FetchSnapshot(srv.URL)
	if err != nil {
		t.Fatalf("FetchSnapshot: %v", err)
	}
	if snap.NAV != 14.80 {
		t.Errorf("NAV = %v, want 14.80", snap.NAV)
	}
	if snap.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", snap.Currency)
	}
}

func TestFetchSnapshotHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := FetchSnapshot(srv.URL)
	if err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}
