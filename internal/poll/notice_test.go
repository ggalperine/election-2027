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

func TestParseSample(t *testing.T) {
	raw, err := os.ReadFile("testdata_notice.txt")
	if err != nil {
		t.Skipf("fixture missing: %v", err)
	}
	if n := parseSample(string(raw)); n < 1000 || n > 3000 {
		t.Errorf("sample size out of expected range: %d", n)
	}
}
