// Package poll defines pollster data sources. Real French poll data has no
// single free API, so sources are pluggable: a CSV/JSON loader, a scraper, or
// the built-in sample generator for local dev.
package poll

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

// Source fetches the latest polls it can find.
type Source interface {
	Name() string
	Fetch(ctx context.Context) ([]models.RawPoll, error)
}

// CSVSource pulls a CSV feed. Expected columns:
// external_id,pollster,field_end,sample_size,round,source_url,<candidate cols...>
// Community-maintained CSVs exist (e.g. nsppolls dataset). Point URL at one.
type CSVSource struct {
	URL        string
	Client     *http.Client
	Candidates []string // header names that are candidate percentage columns
}

func (c CSVSource) Name() string { return "csv:" + c.URL }

func (c CSVSource) Fetch(ctx context.Context) ([]models.RawPoll, error) {
	cli := c.Client
	if cli == nil {
		cli = &http.Client{Timeout: 30 * time.Second}
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, nil)
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("csv fetch: status %d", resp.StatusCode)
	}
	return parseCSV(resp.Body, c.Candidates)
}

func parseCSV(r io.Reader, candidateCols []string) ([]models.RawPoll, error) {
	cr := csv.NewReader(r)
	head, err := cr.Read()
	if err != nil {
		return nil, err
	}
	idx := map[string]int{}
	for i, h := range head {
		idx[strings.TrimSpace(h)] = i
	}
	var polls []models.RawPoll
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		fe, _ := time.Parse("2006-01-02", rec[idx["field_end"]])
		p := models.RawPoll{
			ExternalID: rec[idx["external_id"]],
			Pollster:   rec[idx["pollster"]],
			FieldEnd:   fe,
			SampleSize: atoi(rec[idx["sample_size"]]),
			Round:      atoiDefault(rec[idx["round"]], 1),
			SourceURL:  rec[idx["source_url"]],
			Results:    map[string]float64{},
		}
		for _, cand := range candidateCols {
			if i, ok := idx[cand]; ok && i < len(rec) && rec[i] != "" {
				if v, err := strconv.ParseFloat(rec[i], 64); err == nil {
					p.Results[cand] = v
				}
			}
		}
		polls = append(polls, p)
	}
	return polls, nil
}

func atoi(s string) int { n, _ := strconv.Atoi(strings.TrimSpace(s)); return n }
func atoiDefault(s string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return n
	}
	return def
}
