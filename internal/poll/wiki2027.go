package poll

import (
	"context"
	"fmt"
	"hash/fnv"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/ggalperine/election2027/internal/models"
)

// Wiki2027 scrapes the FR Wikipedia page that consolidates every published 2027
// presidential poll (all instituts). It is the only aggregated open source for
// the 2027 cycle. Parsing is best-effort: malformed rows are skipped.
type Wiki2027 struct {
	Page   string
	Client *http.Client
	// Round2Only restricts output to second-round duel tables. First-round
	// intentions now come from the Commission des sondages notices (legal
	// source); Wikipedia is kept only for the run-off duels until the notice
	// parser covers the second round too.
	Round2Only bool
}

const wikiAPITemplate = "https://fr.wikipedia.org/w/api.php?action=parse&page=%s&prop=text&format=json&formatversion=2"
const wiki2027Page = "Liste_de_sondages_sur_l%27%C3%A9lection_pr%C3%A9sidentielle_fran%C3%A7aise_de_2027"

func (w Wiki2027) Name() string { return "wikipedia:2027" }

// candidateHeader matches "Mélenchon(LFI)" style headers -> candidate, party.
var candidateHeader = regexp.MustCompile(`^(.+?)\s*(?:\[[a-z]\])?\((.+?)\)$`)
var refBracket = regexp.MustCompile(`\[[^\]]*\]`)
var leadingNum = regexp.MustCompile(`([0-9]+(?:[.,][0-9]+)?)`)

func (w Wiki2027) Fetch(ctx context.Context) ([]models.RawPoll, error) {
	page := w.Page
	if page == "" {
		page = wiki2027Page
	}
	cli := w.Client
	if cli == nil {
		cli = &http.Client{Timeout: 60 * time.Second}
	}
	url := fmt.Sprintf(wikiAPITemplate, page)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	// Wikipedia rejects requests without a descriptive User-Agent (HTTP 403).
	req.Header.Set("User-Agent", "election2027-aggregator/1.0 (https://github.com/ggalperine/election2027)")
	req.Header.Set("Accept", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wiki fetch: status %d", resp.StatusCode)
	}
	var payload struct {
		Parse struct {
			Text string `json:"text"`
		} `json:"parse"`
	}
	if err := decodeJSON(resp, &payload); err != nil {
		return nil, err
	}

	doc, err := html.Parse(strings.NewReader(payload.Parse.Text))
	if err != nil {
		return nil, err
	}
	// Walk in document order, tracking the year from the most recent heading
	// (date cells inside tables omit the year). Only the FIRST wikitable — the
	// main, uniformly-structured "1er tour" voting-intention table — is parsed;
	// the page's many alternative-hypothesis tables have heterogeneous layouts
	// that misalign, so they are intentionally skipped.
	var polls []models.RawPoll
	year := 0
	roundOneTaken := false // only the first (main) 1st-round table is trusted
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h2", "h3", "h4":
				if y := yearInText(textOf(n)); y > 0 {
					year = y
				}
			case "table":
				if strings.Contains(attr(n, "class"), "wikitable") && year > 0 {
					switch countCandidates(n) {
					case 2: // 2nd-round duel table
						polls = append(polls, parseTable(n, year, 2)...)
					default:
						if !roundOneTaken && !w.Round2Only {
							got := parseTable(n, year, 1)
							if len(got) > 0 {
								polls = append(polls, got...)
								roundOneTaken = true
							}
						}
					}
					return // don't descend into a parsed table
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return polls, nil
}

// countCandidates returns the max number of "Name(Party)" header cells in any
// row of the table (i.e. the size of the candidate field).
func countCandidates(tbl *html.Node) int {
	max := 0
	for _, r := range findRows(tbl) {
		n := 0
		for _, h := range cellTexts(r, "th", "td") {
			clean := strings.TrimSpace(refBracket.ReplaceAllString(h, ""))
			if m := candidateHeader.FindStringSubmatch(clean); m != nil && !isNonCandidate(cleanName(m[1])) {
				n++
			}
		}
		if n > max {
			max = n
		}
	}
	return max
}

var yearRe = regexp.MustCompile(`20\d\d`)

func yearInText(s string) int {
	if m := yearRe.FindString(s); m != "" {
		y, _ := strconv.Atoi(m)
		return y
	}
	return 0
}

type cand struct{ name, party string }

// minCandidates filters out 2nd-round duel tables (2 candidates) so only
// first-round voting-intention tables are ingested as round 1.
const minCandidates = 5

// partyByAbbrev maps the party abbreviations used in the Wikipedia headers to
// the full names in the parties colour table.
var partyByAbbrev = map[string]string{
	"RN": "Rassemblement national", "REC": "Reconquête", "LR": "Les Républicains",
	"RE": "Renaissance", "HOR": "Horizons", "MODEM": "MoDem", "MDM": "MoDem",
	"PS": "Parti socialiste", "PP": "Place publique", "LE": "Europe Écologie Les Verts",
	"EELV": "Europe Écologie Les Verts", "LFI": "La France insoumise",
	"PCF": "Parti communiste français", "DLF": "Debout la France",
	"LO": "Lutte ouvrière", "NPA": "Nouveau Parti anticapitaliste", "NPA-A": "Nouveau Parti anticapitaliste",
}

func fullParty(abbrev string) string {
	if full, ok := partyByAbbrev[strings.ToUpper(strings.TrimSpace(abbrev))]; ok {
		return full
	}
	return ""
}

// normalizePollster collapses institut spelling variants and returns "" for
// non-institut cells (section artifacts like "Résultats").
func normalizePollster(name string) string {
	l := strings.ToLower(strings.TrimSpace(name))
	l = strings.ReplaceAll(l, "-", " ")
	l = strings.Join(strings.Fields(l), " ")
	switch {
	case strings.HasPrefix(l, "harris"):
		return "Harris Interactive"
	case strings.HasPrefix(l, "cluster"):
		return "Cluster17"
	case strings.HasPrefix(l, "opinion"):
		return "OpinionWay"
	case strings.HasPrefix(l, "ifop"):
		return "Ifop"
	case strings.HasPrefix(l, "ipsos"):
		return "Ipsos"
	case strings.HasPrefix(l, "elabe"):
		return "Elabe"
	case strings.HasPrefix(l, "odoxa"):
		return "Odoxa"
	case strings.HasPrefix(l, "bva"):
		return "BVA"
	case strings.HasPrefix(l, "csa"):
		return "CSA"
	case strings.HasPrefix(l, "kantar"):
		return "Kantar"
	case strings.HasPrefix(l, "verian"):
		return "Verian"
	case strings.HasPrefix(l, "toluna"):
		return "Toluna Harris"
	}
	return "" // unknown -> not a real institut, skip
}

// parseTable extracts one poll per "primary" data row (institut + date + sample +
// one value per candidate). Candidate columns come from the header row with the
// most "Name(Party)" cells. Alternative-hypothesis sub-rows are skipped.
func parseTable(tbl *html.Node, year, round int) []models.RawPoll {
	rows := findRows(tbl)
	if len(rows) < 2 {
		return nil
	}
	// Minimum candidates required: 5 for a 1st-round table, 2 for a duel.
	minCand := minCandidates
	if round == 2 {
		minCand = 2
	}
	// Candidate header row = the row with the most candidate-shaped cells.
	var cands []cand
	for _, r := range rows {
		var got []cand
		for _, h := range cellTexts(r, "th", "td") {
			// Strip footnote refs (e.g. "Glucksmann[c](PP)[c]") BEFORE matching,
			// otherwise a trailing [c] breaks the "Name(Party)" pattern and the
			// candidate is dropped — which shifts every later column.
			clean := strings.TrimSpace(refBracket.ReplaceAllString(h, ""))
			if m := candidateHeader.FindStringSubmatch(clean); m != nil {
				name := cleanName(m[1])
				if !isNonCandidate(name) {
					got = append(got, cand{name: name, party: m[2]})
				}
			}
		}
		if len(got) > len(cands) {
			cands = got
		}
	}
	if len(cands) < minCand {
		return nil
	}
	// Excluded candidates (e.g. Bardella — the RN ticket is Le Pen). Skip any
	// table featuring one so alternative-RN-candidate hypotheses aren't ingested.
	for _, c := range cands {
		if isExcludedCandidate(c.name) {
			return nil
		}
	}
	// For duels take exactly the 2 candidates; for round 1 take the full field.
	if round == 2 && len(cands) != 2 {
		return nil
	}
	nCand := len(cands)

	var out []models.RawPoll
	for _, r := range rows {
		cells := cellTexts(r, "th", "td")
		if len(cells) < nCand+3 {
			continue // sub-row (alt hypothesis) or note; skip
		}
		pollster := normalizePollster(cleanName(cells[0]))
		if pollster == "" {
			continue // not a recognised institut
		}
		fin, ok := parseFrenchDate(cells[1], year)
		if !ok {
			continue
		}
		values := cells[3 : 3+nCand] // institut, date, sample, then one per candidate
		results := map[string]float64{}
		parties := map[string]string{}
		for i, c := range cands {
			if v, ok := parsePct(values[i]); ok {
				results[c.name] = v
				if full := fullParty(c.party); full != "" {
					parties[c.name] = full
				}
			}
		}
		if len(results) < minCand {
			continue
		}
		// Sanity: intentions should sum to roughly 100%. Rows that don't
		// (column misalignment, partial parses) are dropped.
		var sum float64
		for _, v := range results {
			sum += v
		}
		if sum < 88 || sum > 118 {
			continue
		}
		// hash the candidate set so distinct hypotheses/matchups on the same day don't collide.
		id := fmt.Sprintf("wiki-t%d-%s-%s-%s", round, slug(pollster), fin.Format("20060102"), resultsHash(results))
		out = append(out, models.RawPoll{
			ExternalID: id,
			Cycle:      "2027",
			Pollster:   pollster,
			FieldEnd:   fin,
			SampleSize: parseInt(cells[2]),
			Round:      round,
			SourceURL:  "https://fr.wikipedia.org/wiki/" + wiki2027Page,
			Results:    results,
			Parties:    parties,
		})
	}
	return out
}

// excludedCandidates are dropped from ingestion (not running / duplicate ticket).
var excludedCandidates = map[string]bool{"bardella": true}

func isExcludedCandidate(name string) bool {
	return excludedCandidates[strings.ToLower(strings.TrimSpace(name))]
}

func isNonCandidate(s string) bool {
	l := strings.ToLower(strings.TrimSpace(s))
	switch l {
	case "", "autre", "autres", "divers", "nspp", "abstention", "sondeur", "date", "dates", "échantillon", "echantillon":
		return true
	}
	return false
}

func cleanName(s string) string {
	s = refBracket.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, " ", " ")
	// drop "pour <sponsor>" suffix on pollster cells
	if i := strings.Index(strings.ToLower(s), " pour "); i > 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func parsePct(s string) (float64, bool) {
	s = strings.ReplaceAll(s, " ", "")
	m := leadingNum.FindString(s)
	if m == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.Replace(m, ",", ".", 1), 64)
	if err != nil || v <= 0 || v > 100 {
		return 0, false
	}
	return v, true
}

func parseInt(s string) int {
	s = refBracket.ReplaceAllString(s, "")
	s = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
	n, _ := strconv.Atoi(s)
	return n
}

var frMonths = map[string]time.Month{
	"janv": 1, "févr": 2, "fevr": 2, "mars": 3, "avr": 4, "mai": 5, "juin": 6,
	"juil": 7, "août": 8, "aout": 8, "sept": 9, "oct": 10, "nov": 11, "déc": 12, "dec": 12,
}

// parseFrenchDate handles "10-12 sept.", "5 sept. 2025", "1er sept." etc.
// The year in the string wins; otherwise fallbackYear (from the section heading)
// is used. Returns the last (end) day of the range.
func parseFrenchDate(s string, fallbackYear int) (time.Time, bool) {
	s = refBracket.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, " ", " ")
	s = strings.ToLower(strings.TrimSpace(s))
	tokens := strings.Fields(strings.NewReplacer(".", " ", "-", " ", "–", " ", "er", "").Replace(s))
	var days []int
	var mon time.Month
	var year int
	for _, t := range tokens {
		if n, err := strconv.Atoi(t); err == nil {
			if n > 1900 {
				year = n
			} else if n >= 1 && n <= 31 {
				days = append(days, n)
			}
			continue
		}
		for prefix, m := range frMonths {
			if strings.HasPrefix(t, prefix) {
				mon = m
				break
			}
		}
	}
	if year == 0 {
		year = fallbackYear
	}
	if mon == 0 || year == 0 || len(days) == 0 {
		return time.Time{}, false
	}
	day := days[len(days)-1]
	return time.Date(year, mon, day, 0, 0, 0, 0, time.UTC), true
}

// resultsHash is a short stable digest of a candidate->pct map, used to keep
// distinct same-day hypotheses from colliding on external_id.
func resultsHash(m map[string]float64) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s%.1f;", k, m[k])
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(b.String()))
	return fmt.Sprintf("%08x", h.Sum32())
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToLower(s), "-"), "-")
}
