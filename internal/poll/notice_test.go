package poll

import (
	"os"
	"testing"
)

// Real Cluster17 notice (Commission des sondages, 5 sept. 2026) extracted with
// `pdftotext -layout`. Guards the layout-agnostic, candidate-anchored parser.
func TestParseFirstHypothesis_Cluster17(t *testing.T) {
	raw, err := os.ReadFile("testdata_notice.txt")
	if err != nil {
		t.Skipf("fixture missing: %v", err)
	}
	res, ok := parseFirstHypothesis(string(raw))
	if !ok {
		t.Fatalf("validation gate rejected a valid notice; parsed=%v", res)
	}
	// Hypothèse 1 redressé figures from the notice.
	want := map[string]float64{
		"Le Pen":        30.0,
		"Mélenchon":     17.9,
		"Philippe":      18.2,
		"Glucksmann":    11.6,
		"Retailleau":    7.5,
		"Zemmour":       3.5,
	}
	for cand, exp := range want {
		if got := res[cand]; got != exp {
			t.Errorf("%s: got %.1f, want %.1f", cand, got, exp)
		}
	}
	if len(res) < 10 {
		t.Errorf("expected ~12 candidates, got %d: %v", len(res), res)
	}
}

// Ifop notice: bare numbers (no %), NSPP reported separately, and a 2022
// reconstitution table that must NOT be mistaken for the 2027 intentions.
func TestParseFirstHypothesis_Ifop(t *testing.T) {
	raw, err := os.ReadFile("testdata_ifop.txt")
	if err != nil {
		t.Skipf("fixture missing: %v", err)
	}
	res, ok := parseFirstHypothesis(string(raw))
	if !ok {
		t.Fatalf("validation gate rejected a valid Ifop notice; parsed=%v", res)
	}
	want := map[string]float64{
		"Le Pen":     33,
		"Mélenchon":  16,
		"Philippe":   14.5,
		"Attal":      8,
		"Glucksmann": 11,
		"Retailleau": 6,
	}
	for cand, exp := range want {
		if got := res[cand]; got != exp {
			t.Errorf("%s: got %.1f, want %.1f (2027 intentions, not 2022 reconstitution)", cand, got, exp)
		}
	}
	// Macron must NOT appear — that would mean we parsed the 2022 recall table.
	if _, bad := res["Macron"]; bad {
		t.Errorf("parsed the 2022 reconstitution table (Macron present)")
	}
}

func TestParseSample(t *testing.T) {
	raw, err := os.ReadFile("testdata_notice.txt")
	if err != nil {
		t.Skipf("fixture missing: %v", err)
	}
	if n := parseSample(string(raw)); n < 1000 || n > 3000 {
		t.Errorf("sample size out of expected range: %d", n)
	}
}
