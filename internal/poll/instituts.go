package poll

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/net/html"

	"github.com/ggalperine/election2027/internal/models"
)

// Institut is a single polling house scraped directly from its own website
// (no Wikipedia, no third-party aggregator). Each runs as its own microservice
// (cmd/institut-ingester with INSTITUT=<key>), so a layout change on one site
// only breaks that one service.
//
// parse turns the fetched HTML tree of ListURL into RawPolls. It is nil until a
// site-specific parser is written; the framework (fetch, schedule, publish) is
// shared. Name MUST match the canonical pollster name in pollsters.go so house
// effects and dedup line up with any other source.
type Institut struct {
	Key      string
	Name     string
	Homepage string
	ListURL  string
	Cycle    string
	parse    func(root *html.Node, inst Institut) ([]models.RawPoll, error)
	Client   *http.Client
}

// Name satisfies the Source interface.
func (i Institut) Name() string { return "institut:" + i.Key }

// Fetch implements Source: pull the institut's poll-listing page and parse it.
func (i Institut) Fetch(ctx context.Context) ([]models.RawPoll, error) {
	if i.parse == nil {
		return nil, fmt.Errorf("institut %q: parser not implemented yet (ListURL=%s)", i.Key, i.ListURL)
	}
	root, err := fetchHTML(ctx, i.ListURL, i.Client)
	if err != nil {
		return nil, err
	}
	return i.parse(root, i)
}

// Instituts is the registry: one entry per polling house. ListURL points at the
// page on the institut's own site that lists its 2027 presidential polls.
var Instituts = map[string]Institut{
	"ifop": {
		Key: "ifop", Name: "Ifop", Cycle: "2027",
		Homepage: "https://www.ifop.com",
		ListURL:  "https://www.ifop.com/sondages/", // TODO confirm exact 2027 listing path
		parse:    nil,
	},
	"ipsos": {
		Key: "ipsos", Name: "Ipsos", Cycle: "2027",
		Homepage: "https://www.ipsos.com/fr-fr",
		ListURL:  "https://www.ipsos.com/fr-fr/nos-sondages-et-enquetes",
		parse:    nil,
	},
	"elabe": {
		Key: "elabe", Name: "Elabe", Cycle: "2027",
		Homepage: "https://elabe.fr",
		ListURL:  "https://elabe.fr/category/sondages/",
		parse:    nil,
	},
	"opinionway": {
		Key: "opinionway", Name: "OpinionWay", Cycle: "2027",
		Homepage: "https://www.opinion-way.com",
		ListURL:  "https://www.opinion-way.com/fr/sondage-d-opinion.html",
		parse:    nil,
	},
	"harris": {
		Key: "harris", Name: "Harris Interactive", Cycle: "2027",
		Homepage: "https://harris-interactive.fr",
		ListURL:  "https://harris-interactive.fr/opinion_polls/",
		parse:    nil,
	},
	"odoxa": {
		Key: "odoxa", Name: "Odoxa", Cycle: "2027",
		Homepage: "https://www.odoxa.fr",
		ListURL:  "https://www.odoxa.fr/sondages/",
		parse:    nil,
	},
	"cluster17": {
		Key: "cluster17", Name: "Cluster17", Cycle: "2027",
		Homepage: "https://cluster17.com",
		ListURL:  "https://cluster17.com/sondages/",
		parse:    nil,
	},
	"verian": {
		Key: "verian", Name: "Verian", Cycle: "2027",
		Homepage: "https://www.veriangroup.com",
		ListURL:  "https://www.veriangroup.com/news",
		parse:    nil,
	},
}

// SourceByKey resolves an INSTITUT env value to its Source. "commission" is the
// legal base index (all instituts); the rest are per-institut site scrapers.
func SourceByKey(key string) (Source, bool) {
	if key == "commission" {
		return Commission{Cycle: "2027"}, true
	}
	inst, ok := Instituts[key]
	return inst, ok
}

// fetchHTML GETs a URL with a descriptive User-Agent (several sites 403 without
// one) and parses the body into an HTML node tree.
func fetchHTML(ctx context.Context, url string, cli *http.Client) (*html.Node, error) {
	if cli == nil {
		cli = &http.Client{Timeout: 45 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "elyseometre.fr poll aggregator (+https://elyseometre.fr; contact@elyseometre.fr)")
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}
	return html.Parse(resp.Body)
}
