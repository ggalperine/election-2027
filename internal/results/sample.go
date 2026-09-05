package results

import (
	"context"

	"github.com/ggalperine/election2027/internal/models"
)

// SampleSource emits demo department-level results so the map renders without
// the real gov export. Covers a handful of INSEE department codes.
type SampleSource struct{ Election string }

func (s SampleSource) Name() string { return "sample" }

type deptSeed struct {
	code, name string
	registered int
	winner     string
	winnerPct  float64
}

var deptSeeds = []deptSeed{
	{"75", "Paris", 1300000, "Gabriel Attal", 34.2},
	{"13", "Bouches-du-Rhône", 1200000, "Jordan Bardella", 38.1},
	{"59", "Nord", 1700000, "Jordan Bardella", 33.5},
	{"69", "Rhône", 1100000, "Gabriel Attal", 31.0},
	{"33", "Gironde", 1000000, "Raphaël Glucksmann", 27.4},
	{"31", "Haute-Garonne", 900000, "Jean-Luc Mélenchon", 29.8},
	{"06", "Alpes-Maritimes", 800000, "Jordan Bardella", 41.0},
	{"44", "Loire-Atlantique", 950000, "Gabriel Attal", 30.2},
}

func (s SampleSource) Fetch(ctx context.Context) ([]models.GeoResult, error) {
	el := s.Election
	if el == "" {
		el = "presidentielle-2022-t1"
	}
	var out []models.GeoResult
	for _, d := range deptSeeds {
		votes := int(float64(d.registered) * 0.7)
		out = append(out, models.GeoResult{
			Election:   el,
			GeoLevel:   "departement",
			GeoCode:    d.code,
			GeoName:    d.name,
			Registered: d.registered,
			VotesCast:  votes,
			Candidate:  d.winner,
			Votes:      int(float64(votes) * d.winnerPct / 100),
			Pct:        d.winnerPct,
		})
		// runner-up so the map has >1 row per dept
		out = append(out, models.GeoResult{
			Election:   el,
			GeoLevel:   "departement",
			GeoCode:    d.code,
			GeoName:    d.name,
			Registered: d.registered,
			VotesCast:  votes,
			Candidate:  "Jean-Luc Mélenchon",
			Votes:      int(float64(votes) * (d.winnerPct - 8) / 100),
			Pct:        d.winnerPct - 8,
		})
	}
	return out, nil
}
