package stats

// Probabilistic forecast via Monte Carlo simulation.
//
// This is the inferential layer on top of the descriptive poll-of-polls. It
// answers the questions a point estimate cannot: "what is the probability that
// candidate X qualifies for the run-off?" and "who is favoured to win?".
//
// Method (in the lineage of the FiveThirtyEight and Economist election models):
//
//  1. For each candidate we take the recency/sample-weighted mean share μ_i and
//     its standard error σ_i (sampling error ⊕ between-poll dispersion — the
//     same quadrature used for the 95% CI in Snapshot).
//
//  2. We inflate σ_i with a *time-to-election drift* term. Vote intentions are a
//     random walk: the variance of where opinion lands on election day grows
//     linearly with the number of days remaining. σ_drift = σ_day·√(daysLeft).
//     Far from the vote the forecast is deliberately humble; on the eve it
//     collapses to the polling error alone.
//
//  3. We draw the latent share from a Student-t (ν df) rather than a Gaussian.
//     Fat tails encode the empirical fact that polling can miss *systematically*
//     (2016, 2022), not just by sampling noise.
//
//  4. Draws are clamped to ≥0 and renormalised to sum to 100 within each
//     simulation. Renormalisation induces the negative correlation between
//     candidates that a share space requires (if one is over-estimated the
//     others must give ground).
//
//  5. Over N simulations we count how often each candidate finishes 1st (leads
//     the first round) and top-2 (qualifies for the run-off), and record the
//     full predictive distribution (P05/P50/P95).
//
//  6. Run-off: conditional on the two qualifiers of a simulation, the winner is
//     drawn from a duel model calibrated on the actual second-round polling
//     (each candidate's mean run-off score). P(win) = P(qualify ∧ win duel).

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/ggalperine/election2027/internal/models"
)

const (
	// σ_day: daily innovation of the latent vote share (points). Calibrated so
	// that ~200 days out the drift sd is ≈ 1.1 pt, growing the band sensibly.
	DefaultDriftPerDay = 0.08
	// Student-t degrees of freedom for the polling-error tails.
	DefaultDoF = 5.0
	// Simulations per forecast. 20k keeps the Monte Carlo error on a probability
	// below ~0.3 points while staying well under a second.
	DefaultNSims = 20000
	// Deterministic seed: identical inputs → identical probabilities (no jitter
	// between refreshes, which would look like noise to users).
	simSeed = 20270411
)

// candStat is the weighted mean and SE for one candidate (pre-simulation).
type candStat struct {
	cand   string
	party  string
	color  string
	mu     float64 // weighted mean share (%)
	se     float64 // standard error of the mean (points)
	nPolls int
}

// weightedStats reproduces Snapshot's μ and SE per candidate but returns the
// raw (unrounded, unclamped) values the simulator needs.
func weightedStats(rows []models.RawResult, asOf time.Time, windowDays int, tau float64) []candStat {
	if tau <= 0 {
		tau = DefaultTau
	}
	type acc struct {
		sumW, sumWP, sumWPP, sumResp float64
		n                            int
		meta                         candidateMeta
	}
	m := map[string]*acc{}
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
		w := resp * decay
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
	out := make([]candStat, 0, len(m))
	for cand, a := range m {
		if a.sumW == 0 {
			continue
		}
		avg := a.sumWP / a.sumW
		p := avg / 100
		samplingSE := math.Sqrt(p*(1-p)/a.sumResp) * 100
		betweenSE := 0.0
		if a.n > 1 {
			variance := a.sumWPP/a.sumW - avg*avg
			if variance < 0 {
				variance = 0
			}
			betweenSE = math.Sqrt(variance) / math.Sqrt(float64(a.n))
		}
		se := math.Sqrt(samplingSE*samplingSE + betweenSE*betweenSE)
		out = append(out, candStat{
			cand: cand, party: a.meta.party, color: a.meta.color,
			mu: avg, se: se, nPolls: a.n,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].mu > out[j].mu })
	return out
}

// duelKey is an unordered candidate pair.
func duelKey(a, b string) string {
	if a <= b {
		return a + "\x00" + b
	}
	return b + "\x00" + a
}

// duelModel holds, per candidate pair, the recency-weighted mean vote share of
// the alphabetically-first candidate — the empirical run-off split.
type duelModel struct {
	shareFirst map[string]float64 // key -> mean share of the lexicographically smaller name
}

// buildDuelModel aggregates the head-to-head observations into a pair matrix,
// recency-weighting each duel by sample size × exp(-age/tau).
func buildDuelModel(duels []models.DuelObs, asOf time.Time, tau float64) duelModel {
	if tau <= 0 {
		tau = DefaultTau
	}
	type acc struct{ sumW, sumWShareFirst float64 }
	m := map[string]*acc{}
	for _, d := range duels {
		age := asOf.Sub(d.Date).Hours() / 24
		if age < 0 {
			continue
		}
		resp := float64(d.SampleSize)
		if resp <= 0 {
			resp = 1000
		}
		w := resp * math.Exp(-age/tau)
		// share of the lexicographically-smaller name in this duel
		first, share := d.A, d.PctA
		if d.B < d.A {
			first, share = d.B, d.PctB
		}
		_ = first
		key := duelKey(d.A, d.B)
		a := m[key]
		if a == nil {
			a = &acc{}
			m[key] = a
		}
		a.sumW += w
		a.sumWShareFirst += w * share
	}
	out := duelModel{shareFirst: map[string]float64{}}
	for k, a := range m {
		if a.sumW > 0 {
			out.shareFirst[k] = a.sumWShareFirst / a.sumW
		}
	}
	return out
}

// probFirstWins returns the modelled share of the lexicographically-smaller
// name in an a-vs-b duel, and whether the pair was actually polled.
func (dm duelModel) shareOfFirst(a, b string) (float64, bool) {
	s, ok := dm.shareFirst[duelKey(a, b)]
	return s, ok
}

// Simulate runs the Monte Carlo forecast. round1 rows drive qualification;
// duels (may be nil) calibrate the run-off via a candidate-pair matrix.
func Simulate(round1 []models.RawResult, duels []models.DuelObs, asOf time.Time, windowDays int, tau float64,
	electionDate time.Time, nSims int, driftPerDay, dof float64) models.Forecast {

	if nSims <= 0 {
		nSims = DefaultNSims
	}
	if driftPerDay <= 0 {
		driftPerDay = DefaultDriftPerDay
	}
	if dof <= 1 {
		dof = DefaultDoF
	}
	cands := weightedStats(round1, asOf, windowDays, tau)
	daysLeft := electionDate.Sub(asOf).Hours() / 24
	if daysLeft < 0 {
		daysLeft = 0
	}
	driftVar := driftPerDay * driftPerDay * daysLeft // random-walk variance ∝ time

	k := len(cands)
	dm := buildDuelModel(duels, asOf, tau)

	// Total sd per candidate = polling SE ⊕ drift.
	sigma := make([]float64, k)
	for i, c := range cands {
		sigma[i] = math.Sqrt(c.se*c.se + driftVar)
	}

	rng := rand.New(rand.NewSource(simSeed))
	leadCount := make([]int, k)
	qualifyCount := make([]int, k)
	winCount := make([]int, k)
	samples := make([][]float64, k)
	for i := range samples {
		samples[i] = make([]float64, nSims)
	}
	draw := make([]float64, k)

	for s := 0; s < nSims; s++ {
		// 1. draw + clamp
		var total float64
		for i := range cands {
			v := cands[i].mu + sigma[i]*studentT(rng, dof)
			if v < 0 {
				v = 0
			}
			draw[i] = v
			total += v
		}
		// 2. renormalise to 100
		if total == 0 {
			total = 1
		}
		for i := range draw {
			draw[i] = draw[i] / total * 100
			samples[i][s] = draw[i]
		}
		// 3. rank: find 1st and 2nd
		first, second := -1, -1
		for i := range draw {
			if first == -1 || draw[i] > draw[first] {
				second = first
				first = i
			} else if second == -1 || draw[i] > draw[second] {
				second = i
			}
		}
		leadCount[first]++
		qualifyCount[first]++
		if second >= 0 {
			qualifyCount[second]++
			// 4. run-off between first and second
			winner := duelWinner(rng, cands[first].cand, cands[second].cand, draw[first], draw[second], dm)
			if winner == 0 {
				winCount[first]++
			} else {
				winCount[second]++
			}
		}
	}

	pts := make([]models.ForecastProb, 0, k)
	inv := 1.0 / float64(nSims)
	for i, c := range cands {
		col := samples[i]
		sort.Float64s(col)
		pts = append(pts, models.ForecastProb{
			Candidate:    c.cand,
			Party:        c.party,
			Color:        c.color,
			Mean:         round1val(c.mu),
			P05:          round1val(pctile(col, 0.05)),
			P50:          round1val(pctile(col, 0.50)),
			P95:          round1val(pctile(col, 0.95)),
			ProbLead:     round3(float64(leadCount[i]) * inv),
			ProbQualify:  round3(float64(qualifyCount[i]) * inv),
			ProbWin:      round3(float64(winCount[i]) * inv),
			NPolls:       c.nPolls,
		})
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].ProbQualify > pts[j].ProbQualify })

	return models.Forecast{
		AsOf:         asOf.Format("2006-01-02"),
		ElectionDate: electionDate.Format("2006-01-02"),
		DaysLeft:     int(math.Round(daysLeft)),
		NSims:        nSims,
		Candidates:   pts,
	}
}

// duelWinner returns 0 if candidate a wins the run-off, 1 if b wins.
// The split comes from the actual head-to-head polling for this exact pair
// (duel matrix); when that pair was never polled we fall back to the relative
// first-round shares. A per-simulation Gaussian shock (~run-off polling error)
// is applied to propagate duel uncertainty.
func duelWinner(rng *rand.Rand, a, b string, firstA, firstB float64, dm duelModel) int {
	var shareA float64
	if sf, ok := dm.shareOfFirst(a, b); ok {
		// sf is the polled share of the lexicographically-smaller name.
		if a <= b {
			shareA = sf
		} else {
			shareA = 100 - sf
		}
	} else {
		den := firstA + firstB
		if den == 0 {
			den = 1
		}
		shareA = firstA / den * 100 // fallback: relative first-round strength
	}
	// duel uncertainty ≈ 2.5 pt sd (typical run-off polling error)
	shareA += rng.NormFloat64() * 2.5
	if shareA >= 50 {
		return 0
	}
	return 1
}

// studentT draws from a Student-t with ν degrees of freedom: t = z / √(g/ν),
// g ~ χ²_ν (sum of ν squared standard normals).
func studentT(rng *rand.Rand, nu float64) float64 {
	z := rng.NormFloat64()
	n := int(nu)
	if n < 1 {
		n = 1
	}
	var g float64
	for i := 0; i < n; i++ {
		x := rng.NormFloat64()
		g += x * x
	}
	return z / math.Sqrt(g/float64(n))
}

// pctile returns the q-quantile of a pre-sorted slice.
func pctile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(q * float64(len(sorted)-1))
	return sorted[idx]
}

func round1val(v float64) float64 { return math.Round(v*10) / 10 }
func round3(v float64) float64    { return math.Round(v*1000) / 1000 }
