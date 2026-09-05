// Package stats is the scientific aggregation engine: sample-size + recency
// weighted poll-of-polls, 95% confidence intervals via the Kish effective
// sample size, and a Holt linear-trend forecast with a widening uncertainty band.
package stats

import (
	"math"
	"sort"
	"time"

	"github.com/ggalperine/election2027/internal/models"
)

const (
	Z95         = 1.959964 // 95% normal quantile
	DefaultTau  = 21.0     // recency half-life-ish decay (days)
	DefaultWin  = 45       // snapshot trailing window (days)
)

// weight = sampleSize * exp(-ageDays / tau). Bigger, fresher polls count more.
func weight(sampleSize int, ageDays, tau float64) float64 {
	s := float64(sampleSize)
	if s <= 0 {
		s = 1000 // sensible default when a feed omits sample size
	}
	return s * math.Exp(-ageDays/tau)
}

// clampPct keeps a percentage in [0,100].
func clampPct(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// candidateMeta remembers party/colour seen for a candidate.
type candidateMeta struct{ party, color string }

// Snapshot computes the current weighted average + 95% CI per candidate using
// polls whose date is within `windowDays` of `asOf`.
func Snapshot(rows []models.RawResult, asOf time.Time, windowDays int, tau float64) []models.AggregatePoint {
	if tau <= 0 {
		tau = DefaultTau
	}
	type acc struct {
		sumW, sumWP, sumWPP float64 // Σw, Σw·p, Σw·p²  (p in %)
		sumResp             float64 // Σ(decayed respondents) — effective N
		n                   int
		meta                candidateMeta
	}
	m := map[string]*acc{}
	// windowDays <= 0 means "since the beginning" (no lower bound).
	useCutoff := windowDays > 0
	cutoff := asOf.AddDate(0, 0, -windowDays)
	for _, r := range rows {
		if useCutoff && r.Date.Before(cutoff) {
			continue
		}
		if r.Date.After(asOf) {
			continue
		}
		age := asOf.Sub(r.Date).Hours() / 24
		decay := math.Exp(-age / tau)
		resp := float64(r.SampleSize)
		if resp <= 0 {
			resp = 1000
		}
		w := resp * decay // weight ∝ sample size × recency
		a := m[r.Candidate]
		if a == nil {
			a = &acc{meta: candidateMeta{party: r.Party, color: r.Color}}
			m[r.Candidate] = a
		}
		a.sumW += w
		a.sumWP += w * r.Pct
		a.sumWPP += w * r.Pct * r.Pct
		a.sumResp += resp * decay
		a.n++
	}
	out := make([]models.AggregatePoint, 0, len(m))
	for cand, a := range m {
		if a.sumW == 0 {
			continue
		}
		avg := a.sumWP / a.sumW
		p := avg / 100

		// Two independent error sources, added in quadrature:
		//  1. sampling error over the total effective respondent pool
		samplingSE := math.Sqrt(p*(1-p)/a.sumResp) * 100
		//  2. between-poll dispersion (house effects, real movement)
		betweenSE := 0.0
		if a.n > 1 {
			variance := a.sumWPP/a.sumW - avg*avg // weighted variance of pct
			if variance < 0 {
				variance = 0
			}
			betweenSE = math.Sqrt(variance) / math.Sqrt(float64(a.n))
		}
		se := math.Sqrt(samplingSE*samplingSE + betweenSE*betweenSE)

		out = append(out, models.AggregatePoint{
			AsOf:      asOf.Format("2006-01-02"),
			Candidate: cand,
			Party:     a.meta.party,
			Color:     a.meta.color,
			AvgPct:    round1(avg),
			Lo:        round1(clampPct(avg - Z95*se)),
			Hi:        round1(clampPct(avg + Z95*se)),
			NPolls:    a.n,
			NEff:      round1(a.sumResp),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AvgPct > out[j].AvgPct })
	return out
}

// Series computes a trailing-window weighted average + CI at each poll date
// (per candidate) so the frontend can draw trend lines with confidence bands.
func Series(rows []models.RawResult, windowDays int, tau float64) []models.AggregatePoint {
	// Unique sorted dates.
	dateSet := map[string]time.Time{}
	for _, r := range rows {
		dateSet[r.Date.Format("2006-01-02")] = r.Date
	}
	dates := make([]time.Time, 0, len(dateSet))
	for _, d := range dateSet {
		dates = append(dates, d)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

	var out []models.AggregatePoint
	for _, d := range dates {
		out = append(out, Snapshot(rows, d, windowDays, tau)...)
	}
	return out
}

type seriesPt struct {
	t   time.Time
	v   float64
	col string
}

func seriesByCandidate(rows []models.RawResult, windowDays int, tau float64) map[string][]seriesPt {
	series := Series(rows, windowDays, tau)
	byCand := map[string][]seriesPt{}
	for _, s := range series {
		t, _ := time.Parse("2006-01-02", s.AsOf)
		byCand[s.Candidate] = append(byCand[s.Candidate], seriesPt{t: t, v: s.AvgPct, col: s.Color})
	}
	for k := range byCand {
		pts := byCand[k]
		sort.Slice(pts, func(i, j int) bool { return pts[i].t.Before(pts[j].t) })
		byCand[k] = pts
	}
	return byCand
}

// ForecastAll runs BOTH forecasting methods (Holt exponential smoothing and
// weighted linear regression) so the frontend can overlay and compare them.
func ForecastAll(rows []models.RawResult, windowDays int, tau float64, horizonDays, stepDays int) []models.ForecastPoint {
	byCand := seriesByCandidate(rows, windowDays, tau)
	var out []models.ForecastPoint
	for cand, pts := range byCand {
		if len(pts) < 3 {
			continue
		}
		out = append(out, holtForecast(cand, pts, horizonDays, stepDays)...)
		out = append(out, linregForecast(cand, pts, horizonDays, stepDays)...)
	}
	return out
}

// holtForecast: double exponential smoothing (level + trend).
func holtForecast(cand string, pts []seriesPt, horizonDays, stepDays int) []models.ForecastPoint {
	col := pts[len(pts)-1].col
	const alpha, beta = 0.5, 0.3
	level := pts[0].v
	trend := pts[1].v - pts[0].v
	var resid []float64
	for i := 1; i < len(pts); i++ {
		resid = append(resid, pts[i].v-(level+trend))
		newLevel := alpha*pts[i].v + (1-alpha)*(level+trend)
		trend = beta*(newLevel-level) + (1-beta)*trend
		level = newLevel
	}
	sigma := std(resid)
	last := pts[len(pts)-1].t
	out := historyPoints("holt", cand, col, pts)
	for k := 1; k <= horizonDays/stepDays; k++ {
		f := level + float64(k)*trend
		band := Z95 * sigma * math.Sqrt(float64(k))
		out = append(out, projPoint("holt", cand, col, last.AddDate(0, 0, k*stepDays), f, band))
	}
	return out
}

// linregForecast: recency-weighted ordinary least squares on time (days).
func linregForecast(cand string, pts []seriesPt, horizonDays, stepDays int) []models.ForecastPoint {
	t0 := pts[0].t
	const tau = DefaultTau
	last := pts[len(pts)-1].t
	var sw, swx, swy, swxx, swxy float64
	for _, p := range pts {
		x := p.t.Sub(t0).Hours() / 24
		age := last.Sub(p.t).Hours() / 24
		w := math.Exp(-age / tau)
		sw += w
		swx += w * x
		swy += w * p.v
		swxx += w * x * x
		swxy += w * x * p.v
	}
	denom := sw*swxx - swx*swx
	if denom == 0 {
		return nil
	}
	slope := (sw*swxy - swx*swy) / denom
	intercept := (swy - slope*swx) / sw
	// residual sigma
	var resid []float64
	for _, p := range pts {
		x := p.t.Sub(t0).Hours() / 24
		resid = append(resid, p.v-(intercept+slope*x))
	}
	sigma := std(resid)
	out := historyPoints("linreg", cand, pts[len(pts)-1].col, pts)
	lastX := last.Sub(t0).Hours() / 24
	for k := 1; k <= horizonDays/stepDays; k++ {
		x := lastX + float64(k*stepDays)
		f := intercept + slope*x
		band := Z95 * sigma * math.Sqrt(float64(k))
		out = append(out, projPoint("linreg", cand, pts[len(pts)-1].col, last.AddDate(0, 0, k*stepDays), f, band))
	}
	return out
}

func historyPoints(method, cand, col string, pts []seriesPt) []models.ForecastPoint {
	out := make([]models.ForecastPoint, 0, len(pts))
	for _, p := range pts {
		out = append(out, models.ForecastPoint{
			Method: method, Date: p.t.Format("2006-01-02"), Candidate: cand, Color: col,
			AvgPct: round1(p.v), Lo: round1(p.v), Hi: round1(p.v), Projected: false,
		})
	}
	return out
}

func projPoint(method, cand, col string, d time.Time, f, band float64) models.ForecastPoint {
	return models.ForecastPoint{
		Method: method, Date: d.Format("2006-01-02"), Candidate: cand, Color: col,
		AvgPct: round1(clampPct(f)), Lo: round1(clampPct(f - band)), Hi: round1(clampPct(f + band)),
		Projected: true,
	}
}

// Momentum compares the current weighted snapshot with the snapshot `lookback`
// days earlier, per candidate, to flag who is rising or falling.
func Momentum(rows []models.RawResult, asOf time.Time, lookbackDays, windowDays int, tau float64) []models.MomentumPoint {
	now := Snapshot(rows, asOf, windowDays, tau)
	prev := Snapshot(rows, asOf.AddDate(0, 0, -lookbackDays), windowDays, tau)
	prevBy := map[string]float64{}
	for _, p := range prev {
		prevBy[p.Candidate] = p.AvgPct
	}
	weeks := float64(lookbackDays) / 7
	out := make([]models.MomentumPoint, 0, len(now))
	for _, n := range now {
		pv, ok := prevBy[n.Candidate]
		if !ok {
			continue // no comparable earlier value
		}
		delta := round1(n.AvgPct - pv)
		dir := "flat"
		if delta >= 0.5 {
			dir = "up"
		} else if delta <= -0.5 {
			dir = "down"
		}
		perWeek := 0.0
		if weeks > 0 {
			perWeek = round1(delta / weeks)
		}
		out = append(out, models.MomentumPoint{
			Candidate: n.Candidate, Party: n.Party, Color: n.Color,
			Current: n.AvgPct, Previous: round1(pv), Delta: delta,
			PerWeek: perWeek, Direction: dir, NPolls: n.NPolls,
		})
	}
	// Biggest movers first (by absolute delta).
	sort.Slice(out, func(i, j int) bool { return math.Abs(out[i].Delta) > math.Abs(out[j].Delta) })
	return out
}

// HouseEffects computes each institut's average signed deviation from the
// consensus (all-pollster mean) per candidate. Positive delta = the institut
// tends to score that candidate higher than the field. Only institut/candidate
// pairs with at least minPolls polls are returned.
func HouseEffects(rows []models.RawResult, minPolls int) []models.HouseEffect {
	// consensus mean + colour per candidate
	type stat struct {
		sum   float64
		n     int
		color string
	}
	consensus := map[string]*stat{}
	for _, r := range rows {
		c := consensus[r.Candidate]
		if c == nil {
			c = &stat{color: r.Color}
			consensus[r.Candidate] = c
		}
		c.sum += r.Pct
		c.n++
	}
	// per (pollster, candidate) mean
	type key struct{ pollster, candidate string }
	byPair := map[key]*stat{}
	for _, r := range rows {
		k := key{r.Pollster, r.Candidate}
		p := byPair[k]
		if p == nil {
			p = &stat{color: r.Color}
			byPair[k] = p
		}
		p.sum += r.Pct
		p.n++
	}
	var out []models.HouseEffect
	for k, p := range byPair {
		if p.n < minPolls {
			continue
		}
		c := consensus[k.candidate]
		if c == nil || c.n == 0 {
			continue
		}
		delta := p.sum/float64(p.n) - c.sum/float64(c.n)
		out = append(out, models.HouseEffect{
			Pollster:  k.pollster,
			Candidate: k.candidate,
			Color:     p.color,
			Delta:     round1(delta),
			NPolls:    p.n,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pollster != out[j].Pollster {
			return out[i].Pollster < out[j].Pollster
		}
		return math.Abs(out[i].Delta) > math.Abs(out[j].Delta)
	})
	return out
}

func std(xs []float64) float64 {
	if len(xs) < 2 {
		return 1.0
	}
	var mean float64
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	var ss float64
	for _, x := range xs {
		ss += (x - mean) * (x - mean)
	}
	return math.Sqrt(ss / float64(len(xs)-1))
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
