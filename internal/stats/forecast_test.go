package stats

import (
	"testing"
	"time"
)

// A candidate who just entered the field shows a one-off level jump (Hollande:
// 2.4% → 7% then flat). The damped-trend Holt forecast must NOT extrapolate that
// jump into a runaway projection (the undamped version reached ~45%).
func TestHoltDampedNoRunaway(t *testing.T) {
	base := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	pts := []seriesPt{
		{t: base, v: 2.4, col: "#f0426d"},
		{t: base.AddDate(0, 0, 59), v: 7, col: "#f0426d"},
		{t: base.AddDate(0, 0, 62), v: 7, col: "#f0426d"},
		{t: base.AddDate(0, 0, 63), v: 7, col: "#f0426d"},
	}
	out := holtForecast("Hollande", pts, 90, 7)

	var maxProj float64
	for _, p := range out {
		if p.Projected && p.AvgPct > maxProj {
			maxProj = p.AvgPct
		}
	}
	// Last observed level is 7%; a damped forecast should stay in its neighborhood,
	// well under 20% — never the ~45% the undamped model produced.
	if maxProj > 20 {
		t.Errorf("Holt projection runs away: max %.1f%%, want <= 20%%", maxProj)
	}
	if maxProj < 7 {
		t.Errorf("Holt projection collapsed below last level: max %.1f%%", maxProj)
	}
}
