package poll

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ggalperine/election2027/internal/models"
)

// NSPPolls consumes the community-maintained nsppolls dataset (data.gouv.fr /
// github.com/nsppolls/nsppolls), which consolidates every French institut
// (Ifop, Ipsos, OpinionWay, Harris, Kantar, Elabe, BVA, Cluster17, Odoxa, ...)
// for a presidential cycle. Default URL is the 2022 cycle.
type NSPPolls struct {
	URL    string
	Cycle  string
	Client *http.Client
}

const nsppollsDefaultURL = "https://raw.githubusercontent.com/nsppolls/nsppolls/master/presidentielle.json"

func (n NSPPolls) Name() string { return "nsppolls:" + n.cycleOr() }

func (n NSPPolls) cycleOr() string {
	if n.Cycle != "" {
		return n.Cycle
	}
	return "2022"
}

// nsppolls JSON schema (subset we use).
type nspPoll struct {
	ID           string `json:"id"`
	Institut     string `json:"nom_institut"`
	Debut        string `json:"debut_enquete"`
	Fin          string `json:"fin_enquete"`
	Commanditaire string `json:"commanditaire"`
	Lien         string `json:"lien"`
	Echantillon  int    `json:"echantillon"`
	Tours        []struct {
		Tour       string `json:"tour"`
		Hypotheses []struct {
			Candidats []struct {
				Candidat   string   `json:"candidat"`
				Parti      []string `json:"parti"`
				Intentions float64  `json:"intentions"`
			} `json:"candidats"`
		} `json:"hypotheses"`
	} `json:"tours"`
}

func (n NSPPolls) Fetch(ctx context.Context) ([]models.RawPoll, error) {
	url := n.URL
	if url == "" {
		url = nsppollsDefaultURL
	}
	cli := n.Client
	if cli == nil {
		cli = &http.Client{Timeout: 60 * time.Second}
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nsppolls fetch: status %d", resp.StatusCode)
	}
	var raw []nspPoll
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	cycle := n.cycleOr()
	var out []models.RawPoll
	for _, p := range raw {
		fin, err := time.Parse("2006-01-02", p.Fin)
		if err != nil {
			continue
		}
		var start *time.Time
		if d, err := time.Parse("2006-01-02", p.Debut); err == nil {
			start = &d
		}
		for _, t := range p.Tours {
			round := roundNum(t.Tour)
			if len(t.Hypotheses) == 0 {
				continue
			}
			h := t.Hypotheses[0] // base hypothesis
			results := map[string]float64{}
			parties := map[string]string{}
			for _, c := range h.Candidats {
				results[c.Candidat] = c.Intentions
				if len(c.Parti) > 0 {
					parties[c.Candidat] = c.Parti[0]
				}
			}
			if len(results) == 0 {
				continue
			}
			out = append(out, models.RawPoll{
				ExternalID: fmt.Sprintf("nsp-%s-t%d", p.ID, round),
				Cycle:      cycle,
				Pollster:   p.Institut,
				Sponsor:    p.Commanditaire,
				FieldStart: start,
				FieldEnd:   fin,
				SampleSize: p.Echantillon,
				Round:      round,
				SourceURL:  p.Lien,
				Results:    results,
				Parties:    parties,
			})
		}
	}
	return out, nil
}

func roundNum(tour string) int {
	switch tour {
	case "Premier tour":
		return 1
	case "Deuxième tour", "Second tour":
		return 2
	default:
		return 1
	}
}
