package poll

import (
	"context"
	"fmt"
	"time"

	"github.com/ggalperine/election2027/internal/models"
)

// SampleSource generates deterministic-ish demo polls so the whole pipeline
// runs end-to-end without external data. Swap for CSVSource in production.
type SampleSource struct{ Seed int }

func (s SampleSource) Name() string { return "sample" }

// Ordering reflects public 1st-round polling trends (Philippe ahead of Attal).
// These are illustrative demo figures, not real survey results.
var sampleCandidates = map[string]float64{
	"Jordan Bardella":    33,
	"Édouard Philippe":   21,
	"Jean-Luc Mélenchon": 14,
	"Gabriel Attal":      13,
	"Raphaël Glucksmann": 10,
}

type pollster struct {
	name   string
	source string
}

// Real pollster homepages so the source links resolve. Point at the actual
// study URL when wiring a real feed.
var samplePollsters = []pollster{
	{"Ifop", "https://www.ifop.com/sondages/"},
	{"Ipsos", "https://www.ipsos.com/fr-fr"},
	{"OpinionWay", "https://www.opinion-way.com/fr/sondages-publies.html"},
	{"Elabe", "https://elabe.fr/nos-sondages/"},
	{"Odoxa", "https://www.odoxa.fr/sondages/"},
}

func (s SampleSource) Fetch(ctx context.Context) ([]models.RawPoll, error) {
	now := time.Now().UTC()
	var out []models.RawPoll
	for i, ps := range samplePollsters {
		end := now.AddDate(0, 0, -i)
		res := map[string]float64{}
		// small deterministic wobble per pollster so charts have spread
		wob := float64((i*7+s.Seed)%5) - 2
		for name, base := range sampleCandidates {
			res[name] = base + wob*0.3
		}
		out = append(out, models.RawPoll{
			ExternalID: fmt.Sprintf("sample-%s-%s", ps.name, end.Format("2006-01-02")),
			Pollster:   ps.name,
			FieldEnd:   end,
			SampleSize: 1000 + i*100,
			Round:      1,
			SourceURL:  ps.source,
			Results:    res,
		})
	}
	return out, nil
}
