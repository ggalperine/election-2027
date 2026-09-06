package poll

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/ggalperine/election2027/internal/models"
)

// Commission scrapes the Commission des sondages — the legal registry where
// every published French poll must be filed (commission-des-sondages.fr). It is
// the authoritative INDEX of every poll (institut, sponsor, date, notice PDF)
// across all instituts, and stays current (unlike the dead nsppolls dataset).
//
// The index carries no percentages, so for each presidential notice we fetch the
// official PDF, extract its text with `pdftotext -layout`, and parse the first
// round's base hypothesis (see notice.go — layout-agnostic, candidate-anchored,
// with a validation gate that quarantines anything it cannot read cleanly).
type Commission struct {
	Cycle        string
	ListURL      string
	Client       *http.Client
	MaxNotices   int // cap PDFs fetched per run (0 = default)
	PdftotextBin string
}

const commissionDefaultList = "https://www.commission-des-sondages.fr/notices/"

func (c Commission) Name() string { return "commission:" + c.cycle() }

func (c Commission) cycle() string {
	if c.Cycle != "" {
		return c.Cycle
	}
	return "2027"
}

// NoticeRef is one filed poll from the registry index.
type NoticeRef struct {
	ID        string
	Institut  string
	Sponsor   string
	Date      time.Time
	NoticeURL string
}

func (c Commission) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (c Commission) Fetch(ctx context.Context) ([]models.RawPoll, error) {
	url := c.ListURL
	if url == "" {
		url = commissionDefaultList
	}
	root, err := fetchHTML(ctx, url, c.client())
	if err != nil {
		return nil, err
	}
	refs := parseCommissionIndex(root)
	max := c.MaxNotices
	if max <= 0 {
		max = 60
	}
	bin := c.PdftotextBin
	if bin == "" {
		bin = "pdftotext"
	}

	var out []models.RawPoll
	for _, r := range refs {
		if len(out) >= max {
			break
		}
		text, err := c.noticeText(ctx, bin, r.NoticeURL)
		if err != nil {
			// Loud, not silent: a fetch/extract failure is logged upstream by
			// skipping; we simply move on.
			continue
		}
		results, ok := parseFirstHypothesis(text)
		if !ok {
			continue // validation gate rejected — quarantined, not ingested
		}
		out = append(out, models.RawPoll{
			ExternalID: "cds-" + r.ID + "-t1",
			Cycle:      c.cycle(),
			Pollster:   r.Institut,
			Sponsor:    r.Sponsor,
			FieldEnd:   r.Date,
			SampleSize: parseSample(text),
			Round:      1,
			SourceURL:  r.NoticeURL,
			Results:    results,
		})
	}
	return out, nil
}

// noticeText downloads a notice PDF and returns its text via `pdftotext -layout`.
func (c Commission) noticeText(ctx context.Context, bin, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "elyseometre.fr poll aggregator (+https://elyseometre.fr; contact@elyseometre.fr)")
	resp, err := c.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("notice %s: status %d", url, resp.StatusCode)
	}
	pdf, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20)) // 20 MB cap
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, bin, "-layout", "-", "-")
	cmd.Stdin = bytes.NewReader(pdf)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext: %w", err)
	}
	return stdout.String(), nil
}

var (
	cdsMonths = map[string]time.Month{
		"janvier": time.January, "février": time.February, "fevrier": time.February,
		"mars": time.March, "avril": time.April, "mai": time.May, "juin": time.June,
		"juillet": time.July, "août": time.August, "aout": time.August,
		"septembre": time.September, "octobre": time.October, "novembre": time.November,
		"décembre": time.December, "decembre": time.December,
	}
	monthTitleRe = regexp.MustCompile(`(?i)\b(janvier|février|fevrier|mars|avril|mai|juin|juillet|août|aout|septembre|octobre|novembre|décembre|decembre)\s+(\d{4})`)
	dayMonthRe   = regexp.MustCompile(`(?i)(\d{1,2})(?:er)?\s+(janvier|février|fevrier|mars|avril|mai|juin|juillet|août|aout|septembre|octobre|novembre|décembre|decembre)`)
	idRe         = regexp.MustCompile(`^\s*(\d{4,6})\b`)
	// canonical institut name -> its uppercase surface forms in the index line
	institutForms = []struct {
		canon string
		forms []string
	}{
		{"Cluster17", []string{"CLUSTER 17", "CLUSTER17"}},
		{"Ipsos", []string{"IPSOS BVA", "IPSOS"}},
		{"Ifop", []string{"IFOP"}},
		{"Elabe", []string{"ELABE"}},
		{"Odoxa", []string{"ODOXA"}},
		{"OpinionWay", []string{"OPINIONWAY", "OPINION WAY"}},
		{"Harris Interactive", []string{"HARRIS INTERACTIVE", "HARRIS"}},
		{"Verian", []string{"VERIAN", "KANTAR"}},
		{"YouGov", []string{"YOUGOV"}},
		{"Toluna Harris Interactive", []string{"TOLUNA"}},
	}
)

// parseCommissionIndex walks the registry index in document order, tracking the
// current month/year header, and returns one NoticeRef per presidential notice.
func parseCommissionIndex(root *html.Node) []NoticeRef {
	var out []NoticeRef
	year := 0
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "h2" && strings.Contains(attrClass(n), "notices-mois-titre") {
				if m := monthTitleRe.FindStringSubmatch(textOf(n)); m != nil {
					fmt.Sscanf(m[2], "%d", &year)
				}
			}
			if n.Data == "a" && strings.Contains(attrClass(n), "pdf_download") {
				if ref, ok := parseNoticeLine(textOf(n), attr(n, "href"), year); ok {
					out = append(out, ref)
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

// parseNoticeLine turns "10251 Pres IV CLUSTER 17 Le Point 5 septembre" + year
// into a NoticeRef. Only presidential ("Pres") lines are kept.
func parseNoticeLine(text, href string, year int) (NoticeRef, bool) {
	text = strings.Join(strings.Fields(text), " ")
	if !strings.Contains(strings.ToLower(text), "pres") {
		return NoticeRef{}, false
	}
	ref := NoticeRef{NoticeURL: absURL(href)}
	if m := idRe.FindStringSubmatch(text); m != nil {
		ref.ID = m[1]
	}
	upper := strings.ToUpper(text)
	for _, it := range institutForms {
		for _, f := range it.forms {
			if strings.Contains(upper, f) {
				ref.Institut = it.canon
				break
			}
		}
		if ref.Institut != "" {
			break
		}
	}
	if ref.Institut == "" {
		return NoticeRef{}, false // unknown institut → skip loudly (nothing published)
	}
	if m := dayMonthRe.FindStringSubmatch(text); m != nil && year > 0 {
		var day int
		fmt.Sscanf(m[1], "%d", &day)
		if mon, ok := cdsMonths[strings.ToLower(m[2])]; ok {
			ref.Date = time.Date(year, mon, day, 0, 0, 0, 0, time.UTC)
		}
	}
	return ref, ref.ID != ""
}

func absURL(href string) string {
	if strings.HasPrefix(href, "http") {
		return href
	}
	return "https://www.commission-des-sondages.fr" + href
}

func attrClass(n *html.Node) string { return attr(n, "class") }
