package poll

import (
	"os"
	"testing"
)

// OpinionWay Presitrack V2 (Commission notice 10259, Les Echos, 10 sept 2026) is
// a 3-hypothesis notice whose Hypothesis-1 table files Le Pen at 4% and Zemmour
// at 34% — the reverse of the consistent Hyp-2/Hyp-3 (Le Pen 34, Zemmour 4).
// Because that swapped table still sums to ~100 it clears the validation gate, so
// we must repair it from the peer hypotheses rather than publish Le Pen at 4%.
func TestPresitrackBaseHypothesisReconciled(t *testing.T) {
	b, err := os.ReadFile("testdata_presitrack.txt")
	if err != nil {
		t.Fatal(err)
	}
	base, ok := parseFirstHypothesis(string(b))
	if !ok {
		t.Fatal("expected a valid first-round hypothesis")
	}
	// Stable candidates repaired to the peer median.
	if got := base["Le Pen"]; got != 34 {
		t.Errorf("Le Pen = %.0f, want 34 (swapped source value 4 not repaired)", got)
	}
	if got := base["Zemmour"]; got != 4 {
		t.Errorf("Zemmour = %.0f, want 4 (swapped source value 34 not repaired)", got)
	}
	// Bloc candidates (present in <3 hypotheses / small deltas) must be untouched.
	if got := base["Attal"]; got != 8 {
		t.Errorf("Attal = %.0f, want 8 (bloc candidate wrongly overwritten)", got)
	}
	if got := base["Philippe"]; got != 15 {
		t.Errorf("Philippe = %.0f, want 15 (bloc candidate wrongly overwritten)", got)
	}
}

// Verian (Commission notice 10225, L'Hémicycle, 10 juillet 2025) lays its first
// round out TRANSPOSED — candidate names across a header line, one "Total" %
// row beneath — so the per-line candidate→% scans find nothing. The transposed
// fallback must read it by column.
func TestVerianTransposedFirstRound(t *testing.T) {
	b, err := os.ReadFile("testdata_verian.txt")
	if err != nil {
		t.Fatal(err)
	}
	res, ok := parseFirstHypothesis(string(b))
	if !ok {
		t.Fatal("expected the transposed table to parse")
	}
	want := map[string]float64{
		"Roussel": 2, "Mélenchon": 15, "Glucksmann": 11, "Philippe": 17,
		"Attal": 8, "Retailleau": 7, "Le Pen": 37, "Zemmour": 3,
	}
	if len(res) != len(want) {
		t.Fatalf("got %d candidates, want %d: %v", len(res), len(want), res)
	}
	for c, w := range want {
		if res[c] != w {
			t.Errorf("%s = %.0f, want %.0f", c, res[c], w)
		}
	}
}
