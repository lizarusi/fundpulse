package telegram

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lizarusi/fundpulse/internal/analyzer"
)

func sampleReport() Report {
	return Report{
		Date:       time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC),
		Currency:   "USD",
		TotalValue: 24560.40,
		TotalPL:    1234.20,
		TotalPLPct: 5.3,
		Verdict:    analyzer.Verdict{Level: analyzer.Good, Reasons: []string{"portfolio 5d gain 1.2%"}},
		Funds: []FundLine{
			{
				Name: "Fidelity Global Dividend", Currency: "USD",
				Verdict:       analyzer.Verdict{Level: analyzer.Good},
				Change1DPct:   1.2, Change30DPct: 4.5, HasThirtyDays: true,
				ProfitLoss: 520.0,
				RiskLabel:  "Podwyższone",
			},
			{
				Name: "Some Bond Fund", Currency: "USD",
				Verdict:       analyzer.Verdict{Level: analyzer.Warning, Reasons: []string{"5d drop 4.0% ≥ 3.0%"}},
				Change1DPct:   -0.4, Change30DPct: -1.1, HasThirtyDays: true,
				ProfitLoss: -48.0,
				RiskLabel:  "Podwyższone",
			},
			{
				Name: "Crashy Fund", Currency: "USD",
				Verdict:       analyzer.Verdict{Level: analyzer.Alert, Reasons: []string{"1d drop 3.4% ≥ 3.0%"}},
				Change1DPct:   -3.4, Change30DPct: -8.2, HasThirtyDays: true,
				ProfitLoss: -310.0,
				RiskLabel:  "Wysokie",
			},
		},
	}
}

func TestRenderIncludesHeaderAndTotals(t *testing.T) {
	out := Render(sampleReport())
	mustContain := []string{
		"2026-05-07",
		"$24,560.40",
		"+$1,234.20",
		"+5.3%",
		"GOOD",
		"Fidelity Global Dividend",
		"Some Bond Fund",
		"Crashy Fund",
		"ALERT",
		"Elevated",
		"High",
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q\n--\n%s", s, out)
		}
	}
}

func TestRenderColdStartFundOmits30D(t *testing.T) {
	r := sampleReport()
	r.Funds[0].HasThirtyDays = false
	out := Render(r)
	if strings.Contains(out, "+4.5% (30d)") {
		t.Errorf("expected 30d to be hidden when HasThirtyDays=false:\n%s", out)
	}
}

func TestSendPostsToBotAPI(t *testing.T) {
	var gotBody []byte
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = b
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))
	}))
	defer srv.Close()

	c := NewClient("test-token", "-100123")
	c.baseURL = srv.URL
	if err := c.Send("hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if !strings.Contains(gotPath, "/bottest-token/sendMessage") {
		t.Errorf("path = %q, want contains /bottest-token/sendMessage", gotPath)
	}
	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, gotBody)
	}
	if payload["chat_id"] != "-100123" {
		t.Errorf("chat_id = %v, want -100123", payload["chat_id"])
	}
	if payload["text"] != "hello" {
		t.Errorf("text = %v, want hello", payload["text"])
	}
}

func TestSendReturnsErrorOnAPIFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"description":"bad request"}`))
	}))
	defer srv.Close()

	c := NewClient("token", "chat")
	c.baseURL = srv.URL
	if err := c.Send("hi"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
