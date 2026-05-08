package main

import (
	"testing"

	"github.com/lizarusi/fundpulse/internal/config"
)

func TestNewFundsDetectsAdditions(t *testing.T) {
	old := []config.FundEntry{{FundID: "A"}, {FundID: "B"}}
	updated := []config.FundEntry{{FundID: "A"}, {FundID: "B"}, {FundID: "C"}, {FundID: "D"}}
	got := newFunds(old, updated)
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	gotIDs := map[string]bool{got[0].FundID: true, got[1].FundID: true}
	if !gotIDs["C"] || !gotIDs["D"] {
		t.Errorf("got %+v, want [C, D]", got)
	}
}

func TestNewFundsIgnoresRemovals(t *testing.T) {
	old := []config.FundEntry{{FundID: "A"}, {FundID: "B"}}
	updated := []config.FundEntry{{FundID: "A"}}
	got := newFunds(old, updated)
	if len(got) != 0 {
		t.Errorf("len=%d, want 0 (removals are not 'new')", len(got))
	}
}

func TestNewFundsNoChange(t *testing.T) {
	old := []config.FundEntry{{FundID: "A"}, {FundID: "B"}}
	updated := []config.FundEntry{{FundID: "A"}, {FundID: "B"}}
	got := newFunds(old, updated)
	if len(got) != 0 {
		t.Errorf("len=%d, want 0", len(got))
	}
}
