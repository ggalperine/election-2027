package poll

import (
	"context"
	"net/http"
	"strings"

	"golang.org/x/net/html"

	"github.com/ggalperine/election2027/internal/models"
)

// Commission scrapes the Commission des sondages — the legal registry where
// every published French poll must be filed (commission-des-sondages.fr). It is
// the authoritative INDEX of which polls exist (institut, sponsor, date, notice
// PDF) for a cycle, across every institut, and stays current (unlike the dead
// nsppolls dataset). It does NOT carry voting-intention percentages — those live
// in each notice PDF — so it is the base layer of the hybrid pipeline: the
// per-institut services (or a PDF enricher) fill the numbers.
//
// PHASE 2 TODO: the public listing is an old framed site; locate the notices
// listing endpoint (year-filtered) and map each <li> to a NoticeRef. Until the
// parser is complete Fetch returns index metadata with empty Results, which the
// ingester skips (nothing is published, nothing regresses).
type Commission struct {
	Cycle   string
	ListURL string
	Client  *http.Client
}

const commissionDefaultList = "https://www.commission-des-sondages.fr/notices/"

func (c Commission) Name() string { return "commission:" + c.cycle() }

func (c Commission) cycle() string {
	if c.Cycle != "" {
		return c.Cycle
	}
	return "2027"
}

// NoticeRef is one filed poll from the registry index (metadata only).
type NoticeRef struct {
	Institut  string
	Sponsor   string
	Date      string
	NoticeURL string
}

func (c Commission) Fetch(ctx context.Context) ([]models.RawPoll, error) {
	url := c.ListURL
	if url == "" {
		url = commissionDefaultList
	}
	root, err := fetchHTML(ctx, url, c.Client)
	if err != nil {
		return nil, err
	}
	refs := parseCommissionIndex(root)
	out := make([]models.RawPoll, 0, len(refs))
	for _, r := range refs {
		// Metadata-only for now: no Results → ingester skips publishing until a
		// percentage enricher (per-institut scraper / PDF parser) fills them.
		out = append(out, models.RawPoll{
			Cycle:     c.cycle(),
			Pollster:  r.Institut,
			Sponsor:   r.Sponsor,
			SourceURL: r.NoticeURL,
			Results:   map[string]float64{},
		})
	}
	return out, nil
}

// parseCommissionIndex extracts presidential-poll notice links from the registry
// index. Best-effort against the current markup; refined in phase 2.
func parseCommissionIndex(root *html.Node) []NoticeRef {
	var out []NoticeRef
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := attr(n, "href")
			if strings.HasSuffix(strings.ToLower(href), ".pdf") {
				txt := strings.TrimSpace(textOf(n))
				if strings.Contains(strings.ToLower(txt), "pres") {
					out = append(out, NoticeRef{NoticeURL: href})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return out
}
