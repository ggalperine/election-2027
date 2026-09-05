// Package results ingests official election results published by the Ministry
// of the Interior (elections.interieur.gouv.fr). That site publishes open-data
// exports (data.gouv.fr / CSV/XML) per election; wire the URL for the target
// election. A sample generator is provided for local dev.
package results

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ggalperine/election2027/internal/models"
)

type Source interface {
	Name() string
	Fetch(ctx context.Context) ([]models.GeoResult, error)
}

// CSVSource parses an open-data results export by department.
// Columns expected (adapt to the real file):
// election,geo_level,geo_code,geo_name,registered,votes_cast,candidate,votes,pct
type CSVSource struct {
	URL    string
	Client *http.Client
}

func (c CSVSource) Name() string { return "gov-csv:" + c.URL }

func (c CSVSource) Fetch(ctx context.Context) ([]models.GeoResult, error) {
	cli := c.Client
	if cli == nil {
		cli = &http.Client{Timeout: 60 * time.Second}
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, nil)
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gov fetch: status %d", resp.StatusCode)
	}
	return parseCSV(resp.Body)
}

func parseCSV(r io.Reader) ([]models.GeoResult, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	head, err := cr.Read()
	if err != nil {
		return nil, err
	}
	idx := map[string]int{}
	for i, h := range head {
		idx[strings.TrimSpace(strings.ToLower(h))] = i
	}
	get := func(rec []string, k string) string {
		if i, ok := idx[k]; ok && i < len(rec) {
			return rec[i]
		}
		return ""
	}
	var out []models.GeoResult
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		pct, _ := strconv.ParseFloat(strings.Replace(get(rec, "pct"), ",", ".", 1), 64)
		out = append(out, models.GeoResult{
			Election:   get(rec, "election"),
			GeoLevel:   get(rec, "geo_level"),
			GeoCode:    get(rec, "geo_code"),
			GeoName:    get(rec, "geo_name"),
			Registered: atoi(get(rec, "registered")),
			VotesCast:  atoi(get(rec, "votes_cast")),
			Candidate:  get(rec, "candidate"),
			Votes:      atoi(get(rec, "votes")),
			Pct:        pct,
		})
	}
	return out, nil
}

func atoi(s string) int { n, _ := strconv.Atoi(strings.TrimSpace(s)); return n }
