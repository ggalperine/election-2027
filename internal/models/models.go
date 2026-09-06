package models

import "time"

// RawPoll is the message emitted by the poll-ingester and consumed by the aggregator.
type RawPoll struct {
	ExternalID string             `json:"external_id"`
	Cycle      string             `json:"cycle"` // election cycle, e.g. "2027" or "2022"
	Pollster   string             `json:"pollster"`
	Sponsor    string             `json:"sponsor,omitempty"`
	FieldStart *time.Time         `json:"field_start,omitempty"`
	FieldEnd   time.Time          `json:"field_end"`
	SampleSize int                `json:"sample_size"`
	Round      int                `json:"round"`
	SourceURL  string             `json:"source_url"`
	Results    map[string]float64 `json:"results"`          // candidate name -> pct
	Parties    map[string]string  `json:"parties,omitempty"` // candidate name -> party
}

// GeoResult is the message emitted by the results-ingester (official gov data).
type GeoResult struct {
	Election   string  `json:"election"`
	GeoLevel   string  `json:"geo_level"`
	GeoCode    string  `json:"geo_code"`
	GeoName    string  `json:"geo_name"`
	Registered int     `json:"registered"`
	VotesCast  int     `json:"votes_cast"`
	Candidate  string  `json:"candidate"`
	Votes      int     `json:"votes"`
	Pct        float64 `json:"pct"`
}

// Poll is one individual survey with its source, for the "sources" table (API response).
type Poll struct {
	ExternalID string             `json:"external_id"`
	Pollster   string             `json:"pollster"`
	Sponsor    string             `json:"sponsor,omitempty"`
	FieldEnd   string             `json:"field_end"`
	SampleSize int                `json:"sample_size"`
	Round      int                `json:"round"`
	SourceURL  string             `json:"source_url"`
	Results    map[string]float64 `json:"results"`
}

// DuelObs is one second-round head-to-head observation (a poll testing exactly
// two candidates), used to calibrate the run-off model.
type DuelObs struct {
	Date       time.Time
	Pollster   string
	A          string
	PctA       float64
	B          string
	PctB       float64
	SampleSize int
}

// RawResult is a single (poll date, candidate, pct, sample) tuple fed to the stats engine.
type RawResult struct {
	Date       time.Time
	Pollster   string
	Candidate  string
	Party      string
	Color      string
	Pct        float64
	SampleSize int
}

// PollsterStat summarizes one institut's activity for a cycle+round.
type PollsterStat struct {
	Pollster  string  `json:"pollster"`
	Website   string  `json:"website"`
	NPolls    int     `json:"n_polls"`
	AvgSample int     `json:"avg_sample"`
	FirstPoll string  `json:"first_poll"`
	LastPoll  string  `json:"last_poll"`
}

// HouseEffect is an institut's average signed deviation from the consensus for a candidate.
type HouseEffect struct {
	Pollster  string  `json:"pollster"`
	Candidate string  `json:"candidate"`
	Color     string  `json:"color"`
	Delta     float64 `json:"delta"` // institut mean − consensus mean (percentage points)
	NPolls    int     `json:"n_polls"`
}

// MomentumPoint captures a candidate's short-term trend: current weighted
// average vs the value `lookback` days earlier.
type MomentumPoint struct {
	Candidate string  `json:"candidate"`
	Party     string  `json:"party"`
	Color     string  `json:"color"`
	Current   float64 `json:"current"`
	Previous  float64 `json:"previous"`
	Delta     float64 `json:"delta"`     // current − previous (percentage points)
	PerWeek   float64 `json:"per_week"`  // delta normalized to pts/week
	Direction string  `json:"direction"` // "up" | "down" | "flat"
	NPolls    int     `json:"n_polls"`
}

// Summary is the top-of-page overview for a cycle+round.
type Summary struct {
	Cycle       string  `json:"cycle"`
	Round       int     `json:"round"`
	NPolls      int     `json:"n_polls"`
	NPollsters  int     `json:"n_pollsters"`
	FirstPoll   string  `json:"first_poll"`
	LastPoll    string  `json:"last_poll"`
	Leader        string  `json:"leader"`
	LeaderPct     float64 `json:"leader_pct"`
	LeaderColor   string  `json:"leader_color"`
	Margin        float64 `json:"margin"` // leader − runner-up
	LastUpdated   string  `json:"last_updated"`    // when data was last ingested (RFC3339)
	LatestPoll    string  `json:"latest_poll"`     // field_end of the most recent poll
	LatestPollster string `json:"latest_pollster"` // institut of the most recent poll
}

// AggregatePoint is one candidate's weighted average with a 95% confidence interval.
type AggregatePoint struct {
	AsOf      string  `json:"as_of"`
	Candidate string  `json:"candidate"`
	Party     string  `json:"party"`
	Color     string  `json:"color"`
	AvgPct    float64 `json:"avg_pct"`
	Lo        float64 `json:"lo"`      // 95% CI lower
	Hi        float64 `json:"hi"`      // 95% CI upper
	NPolls    int     `json:"n_polls"`
	NEff      float64 `json:"n_eff"`   // Kish effective sample size
}

// ForecastPoint is a projected value with a widening uncertainty band.
type ForecastPoint struct {
	Method    string  `json:"method"` // "holt" | "linreg"
	Date      string  `json:"date"`
	Candidate string  `json:"candidate"`
	Color     string  `json:"color"`
	AvgPct    float64 `json:"avg_pct"`
	Lo        float64 `json:"lo"`
	Hi        float64 `json:"hi"`
	Projected bool    `json:"projected"` // false = fitted history, true = future
}

// ForecastProb is one candidate's probabilistic forecast from the Monte Carlo
// simulation: predictive interval plus qualification / lead / win probabilities.
type ForecastProb struct {
	Candidate   string  `json:"candidate"`
	Party       string  `json:"party"`
	Color       string  `json:"color"`
	Mean        float64 `json:"mean"`         // weighted mean share (%)
	P05         float64 `json:"p05"`          // 5th percentile of predictive distribution
	P50         float64 `json:"p50"`          // median
	P95         float64 `json:"p95"`          // 95th percentile
	ProbLead    float64 `json:"prob_lead"`    // P(finishes 1st in round 1)
	ProbQualify float64 `json:"prob_qualify"` // P(reaches the run-off, top 2)
	ProbWin     float64 `json:"prob_win"`     // P(wins the election)
	NPolls      int     `json:"n_polls"`
}

// Forecast is the full Monte Carlo forecast for a cycle's first round.
type Forecast struct {
	AsOf         string         `json:"as_of"`
	ElectionDate string         `json:"election_date"`
	DaysLeft     int            `json:"days_left"`
	NSims        int            `json:"n_sims"`
	Candidates   []ForecastProb `json:"candidates"`
}

// DuelSummary is one candidate's average run-off score against a given opponent.
type DuelSummary struct {
	Opponent string  `json:"opponent"`
	Color    string  `json:"color"`
	Share    float64 `json:"share"`   // this candidate's mean second-round %
	Wins     bool    `json:"wins"`    // share > 50
	NPolls   int     `json:"n_polls"`
}

// Analysis is the on-demand deep-dive report for a single candidate (premium
// product). It compiles the whole engine's view of one candidate.
type Analysis struct {
	Candidate   string         `json:"candidate"`
	Party       string         `json:"party"`
	Color       string         `json:"color"`
	Cycle       string         `json:"cycle"`
	Rank        int            `json:"rank"`
	AvgPct      float64        `json:"avg_pct"`
	Lo          float64        `json:"lo"`
	Hi          float64        `json:"hi"`
	NPolls      int            `json:"n_polls"`
	MomentumDelta float64      `json:"momentum_delta"` // pts over 30 days
	MomentumDir   string       `json:"momentum_dir"`
	ProbLead    float64        `json:"prob_lead"`
	ProbQualify float64        `json:"prob_qualify"`
	ProbWin     float64        `json:"prob_win"`
	P05         float64        `json:"p05"`
	P50         float64        `json:"p50"`
	P95         float64        `json:"p95"`
	HouseEffects []HouseEffect `json:"house_effects"` // this candidate only
	Duels       []DuelSummary  `json:"duels"`          // sorted best → worst
	GeneratedAt string         `json:"generated_at"`
	Narrative   string         `json:"narrative,omitempty"` // AI synthesis (premium)
}

// ActualResult is the final official result of a past election (reference line).
type ActualResult struct {
	Candidate string  `json:"candidate"`
	Color     string  `json:"color"`
	Pct       float64 `json:"pct"`
	Won       bool    `json:"won"`
}
