// gateway is the public REST API consumed by the web frontend. Aggregation,
// confidence intervals and forecasts are computed on read by the stats engine.
package main

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/log"

	"github.com/ggalperine/election2027/internal/config"
	"github.com/ggalperine/election2027/internal/httpx"
	"github.com/ggalperine/election2027/internal/models"
	"github.com/ggalperine/election2027/internal/poll"
	"github.com/ggalperine/election2027/internal/stats"
	"github.com/ggalperine/election2027/internal/store"
)

const (
	snapshotWindow = stats.DefaultWin
	seriesWindow   = 30
	forecastDays   = 90
	forecastStep   = 7
)

// firstRoundDate is the date of the first round of each cycle, used as the
// horizon of the probabilistic forecast. 2027 dates ARE PROVISIONAL — confirm
// against the décret de convocation des électeurs once published.
var firstRoundDate = map[string]string{
	"2027": "2027-04-11",
	"2022": "2022-04-10",
}

func electionDate(cycle string, fallback time.Time) time.Time {
	if s, ok := firstRoundDate[cycle]; ok {
		if d, err := time.Parse("2006-01-02", s); err == nil {
			return d
		}
	}
	return fallback
}

func main() {
	cfg := config.Load()
	ctx := context.Background()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("store")
	}
	defer st.Close()

	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{AllowedOrigins: []string{"*"}, AllowedMethods: []string{"GET"}}))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, 200, map[string]string{"status": "ok"})
	})

	r.Get("/api/cycles", func(w http.ResponseWriter, req *http.Request) {
		data, err := st.Cycles(req.Context())
		respond(w, data, err)
	})

	// Weighted snapshot with 95% CI (leaderboard / bar chart).
	// window query param: days back from the latest poll; 0 = since the beginning.
	r.Get("/api/aggregates", func(w http.ResponseWriter, req *http.Request) {
		cycle, round := cycleRound(req)
		rows, err := st.RawResults(req.Context(), cycle, round)
		if err != nil {
			respond(w, nil, err)
			return
		}
		httpx.JSON(w, 200, stats.Snapshot(rows, latestDate(rows), queryInt(req, "window", snapshotWindow), stats.DefaultTau))
	})

	// Rolling weighted average + CI over time (trend chart with bands).
	r.Get("/api/timeseries", func(w http.ResponseWriter, req *http.Request) {
		cycle, round := cycleRound(req)
		rows, err := st.RawResults(req.Context(), cycle, round)
		if err != nil {
			respond(w, nil, err)
			return
		}
		httpx.JSON(w, 200, stats.Series(rows, queryInt(req, "window", seriesWindow), stats.DefaultTau))
	})

	// Per-institut activity (poll counts, avg sample, date range, website).
	r.Get("/api/pollsters", func(w http.ResponseWriter, req *http.Request) {
		cycle, round := cycleRound(req)
		data, err := st.PollsterStats(req.Context(), cycle, round)
		if err != nil {
			respond(w, nil, err)
			return
		}
		for i := range data {
			data[i].Website = poll.PollsterSite(data[i].Pollster)
		}
		httpx.JSON(w, 200, data)
	})

	// Momentum: who's rising/falling vs `lookback` days ago (default 30).
	r.Get("/api/momentum", func(w http.ResponseWriter, req *http.Request) {
		cycle, round := cycleRound(req)
		rows, err := st.RawResults(req.Context(), cycle, round)
		if err != nil {
			respond(w, nil, err)
			return
		}
		lookback := queryInt(req, "lookback", 30)
		window := queryInt(req, "window", snapshotWindow)
		httpx.JSON(w, 200, stats.Momentum(rows, latestDate(rows), lookback, window, stats.DefaultTau))
	})

	// House effects: each institut's signed deviation from consensus per candidate.
	r.Get("/api/houseeffects", func(w http.ResponseWriter, req *http.Request) {
		cycle, round := cycleRound(req)
		rows, err := st.RawResults(req.Context(), cycle, round)
		if err != nil {
			respond(w, nil, err)
			return
		}
		httpx.JSON(w, 200, stats.HouseEffects(rows, 2))
	})

	// Top-of-page overview for a cycle+round.
	r.Get("/api/summary", func(w http.ResponseWriter, req *http.Request) {
		cycle, round := cycleRound(req)
		rows, err := st.RawResults(req.Context(), cycle, round)
		if err != nil {
			respond(w, nil, err)
			return
		}
		sum := buildSummary(cycle, round, rows, queryInt(req, "window", snapshotWindow))
		if lu, fe, ps, err := st.LatestPoll(req.Context(), cycle, round); err == nil {
			sum.LastUpdated = lu.UTC().Format(time.RFC3339)
			sum.LatestPoll = fe.Format("2006-01-02")
			sum.LatestPollster = ps
		}
		httpx.JSON(w, 200, sum)
	})

	// Both forecast methods (Holt + weighted linear regression) for comparison.
	r.Get("/api/forecast", func(w http.ResponseWriter, req *http.Request) {
		cycle, round := cycleRound(req)
		rows, err := st.RawResults(req.Context(), cycle, round)
		if err != nil {
			respond(w, nil, err)
			return
		}
		httpx.JSON(w, 200, stats.ForecastAll(rows, queryInt(req, "window", seriesWindow), stats.DefaultTau, forecastDays, forecastStep))
	})

	// Probabilistic forecast: Monte Carlo simulation of the first round giving
	// P(lead), P(qualify for run-off) and P(win) per candidate, plus predictive
	// intervals. round1 rows drive qualification; round2 rows calibrate the duel.
	r.Get("/api/simulate", func(w http.ResponseWriter, req *http.Request) {
		cycle, _ := cycleRound(req)
		r1, err := st.RawResults(req.Context(), cycle, 1)
		if err != nil {
			respond(w, nil, err)
			return
		}
		duels, err := st.Duels(req.Context(), cycle)
		if err != nil {
			respond(w, nil, err)
			return
		}
		asOf := latestDate(r1)
		ed := electionDate(cycle, asOf)
		nsims := queryInt(req, "sims", stats.DefaultNSims)
		httpx.JSON(w, 200, stats.Simulate(r1, duels, asOf, queryInt(req, "window", snapshotWindow),
			stats.DefaultTau, ed, nsims, stats.DefaultDriftPerDay, stats.DefaultDoF))
	})

	// On-demand deep-dive analysis for one candidate (premium report data).
	r.Get("/api/analysis", func(w http.ResponseWriter, req *http.Request) {
		cycle, _ := cycleRound(req)
		cand := req.URL.Query().Get("candidate")
		if cand == "" {
			httpx.JSON(w, 400, map[string]string{"error": "candidate is required"})
			return
		}
		r1, err := st.RawResults(req.Context(), cycle, 1)
		if err != nil {
			respond(w, nil, err)
			return
		}
		duels, err := st.Duels(req.Context(), cycle)
		if err != nil {
			respond(w, nil, err)
			return
		}
		a, ok := buildAnalysis(cycle, cand, r1, duels)
		if !ok {
			httpx.JSON(w, 404, map[string]string{"error": "unknown candidate for this cycle"})
			return
		}
		httpx.JSON(w, 200, a)
	})

	// Individual polls with pollster, sponsor and source link (sources table).
	r.Get("/api/polls", func(w http.ResponseWriter, req *http.Request) {
		cycle, round := cycleRound(req)
		limit := queryInt(req, "limit", 60)
		data, err := st.RecentPolls(req.Context(), cycle, round, limit)
		respond(w, data, err)
	})

	// Official final result of a past election (poll-vs-outcome comparison).
	r.Get("/api/actual", func(w http.ResponseWriter, req *http.Request) {
		cycle, round := cycleRound(req)
		data, err := st.ActualResults(req.Context(), cycle, round)
		respond(w, data, err)
	})

	// Winner per department for the France map (choropleth).
	r.Get("/api/map", func(w http.ResponseWriter, req *http.Request) {
		election := def(req.URL.Query().Get("election"), "presidentielle-2022-t1")
		level := def(req.URL.Query().Get("level"), "departement")
		data, err := st.GeoWinners(req.Context(), election, level)
		respond(w, data, err)
	})

	log.Info().Str("addr", cfg.HTTPAddr).Msg("gateway listening")
	if err := http.ListenAndServe(cfg.HTTPAddr, r); err != nil {
		log.Fatal().Err(err).Msg("serve")
	}
}

// buildSummary derives the top-of-page overview from raw rows + pollster stats.
func buildSummary(cycle string, round int, rows []models.RawResult, window int) models.Summary {
	s := models.Summary{Cycle: cycle, Round: round}
	asOf := latestDate(rows)
	// Coverage + counts reflect the SELECTED window (window<=0 = since the
	// beginning) so the summary changes with the period control.
	var first, last time.Time
	pollsters := map[string]struct{}{}
	polls := map[string]struct{}{} // distinct (pollster, date) ≈ distinct polls
	for _, r := range rows {
		if window > 0 && r.Date.Before(asOf.AddDate(0, 0, -window)) {
			continue
		}
		if r.Date.After(asOf) {
			continue
		}
		if first.IsZero() || r.Date.Before(first) {
			first = r.Date
		}
		if last.IsZero() || r.Date.After(last) {
			last = r.Date
		}
		pollsters[r.Pollster] = struct{}{}
		polls[r.Pollster+"|"+r.Date.Format("2006-01-02")] = struct{}{}
	}
	s.NPolls = len(polls)
	s.NPollsters = len(pollsters)
	if !first.IsZero() {
		s.FirstPoll = first.Format("2006-01-02")
	}
	if !last.IsZero() {
		s.LastPoll = last.Format("2006-01-02")
	}
	snap := stats.Snapshot(rows, asOf, window, stats.DefaultTau)
	if len(snap) > 0 {
		s.Leader = snap[0].Candidate
		s.LeaderPct = snap[0].AvgPct
		s.LeaderColor = snap[0].Color
		if len(snap) > 1 {
			s.Margin = round1(snap[0].AvgPct - snap[1].AvgPct)
		}
	}
	return s
}

func round1(v float64) float64 { return float64(int(v*10+0.5)) / 10 }

// buildAnalysis compiles the full engine view of one candidate for the premium
// on-demand report: standing, momentum, house effects, forecast and run-off duels.
func buildAnalysis(cycle, cand string, r1 []models.RawResult, duels []models.DuelObs) (models.Analysis, bool) {
	asOf := latestDate(r1)
	snap := stats.Snapshot(r1, asOf, snapshotWindow, stats.DefaultTau)
	a := models.Analysis{Candidate: cand, Cycle: cycle}
	found := false
	for i, p := range snap {
		if p.Candidate == cand {
			a.Rank = i + 1
			a.Party, a.Color = p.Party, p.Color
			a.AvgPct, a.Lo, a.Hi, a.NPolls = p.AvgPct, p.Lo, p.Hi, p.NPolls
			found = true
			break
		}
	}
	if !found {
		return a, false
	}

	for _, m := range stats.Momentum(r1, asOf, 30, snapshotWindow, stats.DefaultTau) {
		if m.Candidate == cand {
			a.MomentumDelta, a.MomentumDir = m.Delta, m.Direction
			break
		}
	}

	fc := stats.Simulate(r1, duels, asOf, snapshotWindow, stats.DefaultTau,
		electionDate(cycle, asOf), stats.DefaultNSims, stats.DefaultDriftPerDay, stats.DefaultDoF)
	for _, p := range fc.Candidates {
		if p.Candidate == cand {
			a.ProbLead, a.ProbQualify, a.ProbWin = p.ProbLead, p.ProbQualify, p.ProbWin
			a.P05, a.P50, a.P95 = p.P05, p.P50, p.P95
			break
		}
	}

	for _, he := range stats.HouseEffects(r1, 2) {
		if he.Candidate == cand {
			a.HouseEffects = append(a.HouseEffects, he)
		}
	}

	a.Duels = candidateDuels(cand, duels)
	a.GeneratedAt = asOf.Format("2006-01-02")
	return a, true
}

// candidateDuels averages the candidate's second-round score against each
// opponent, sorted strongest → weakest.
func candidateDuels(cand string, duels []models.DuelObs) []models.DuelSummary {
	type acc struct {
		sum   float64
		n     int
		color string
	}
	by := map[string]*acc{}
	for _, d := range duels {
		var mine, share float64
		var opp string
		if d.A == cand {
			share, opp = d.PctA, d.B
		} else if d.B == cand {
			share, opp = d.PctB, d.A
		} else {
			continue
		}
		mine = share
		x := by[opp]
		if x == nil {
			x = &acc{}
			by[opp] = x
		}
		x.sum += mine
		x.n++
	}
	out := make([]models.DuelSummary, 0, len(by))
	for opp, x := range by {
		share := round1(x.sum / float64(x.n))
		out = append(out, models.DuelSummary{
			Opponent: opp, Share: share, Wins: share > 50, NPolls: x.n,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Share > out[j].Share })
	return out
}

func respond(w http.ResponseWriter, data any, err error) {
	if err != nil {
		httpx.JSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	httpx.JSON(w, 200, data)
}

func cycleRound(r *http.Request) (string, int) {
	return def(r.URL.Query().Get("cycle"), "2027"), queryInt(r, "round", 1)
}

// latestDate returns the most recent poll date, or now if there are none.
func latestDate(rows []models.RawResult) time.Time {
	var latest time.Time
	for _, r := range rows {
		if r.Date.After(latest) {
			latest = r.Date
		}
	}
	if latest.IsZero() {
		return time.Now().UTC()
	}
	return latest
}

func queryInt(r *http.Request, key string, def int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func def(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
