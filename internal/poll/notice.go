package poll

import (
	"regexp"
	"strconv"
	"strings"
)

// Parsing the Commission notice PDFs is deliberately anchored on KNOWN CANDIDATE
// NAMES rather than on any table layout, because each institut formats its
// notice differently and may change its template. We scan the extracted text,
// find each candidate, and read the percentage next to it — layout-agnostic.
// A validation gate (enough candidates + plausible total) quarantines anything
// we cannot parse cleanly instead of ingesting garbage.

// candidate2027 maps every surface form we expect in a notice to the canonical
// short name used in the database (so "Marine Le Pen" and "Le Pen" both map to
// "Le Pen" and don't create duplicate candidates).
var candidate2027 = map[string]string{
	"marine le pen":         "Le Pen",
	"le pen":                "Le Pen",
	"jean-luc mélenchon":    "Mélenchon",
	"mélenchon":             "Mélenchon",
	"édouard philippe":      "Philippe",
	"edouard philippe":      "Philippe",
	"philippe":              "Philippe",
	"gabriel attal":         "Attal",
	"attal":                 "Attal",
	"raphaël glucksmann":    "Glucksmann",
	"glucksmann":            "Glucksmann",
	"bruno retailleau":      "Retailleau",
	"retailleau":            "Retailleau",
	"éric zemmour":          "Zemmour",
	"eric zemmour":          "Zemmour",
	"zemmour":               "Zemmour",
	"marine tondelier":      "Tondelier",
	"tondelier":             "Tondelier",
	"fabien roussel":        "Roussel",
	"roussel":               "Roussel",
	"dominique de villepin": "Villepin",
	"de villepin":           "Villepin",
	"villepin":              "Villepin",
	"nicolas dupont-aignan": "Dupont-Aignan",
	"dupont-aignan":         "Dupont-Aignan",
	"nathalie arthaud":      "Arthaud",
	"arthaud":               "Arthaud",
	"philippe poutou":       "Poutou",
	"poutou":                "Poutou",
	"jordan bardella":       "Bardella",
	"bardella":              "Bardella",
	"david lisnard":         "Lisnard",
	"lisnard":               "Lisnard",
	"françois ruffin":       "Ruffin",
	"ruffin":                "Ruffin",
	"françois hollande":     "Hollande",
	"hollande":              "Hollande",
}

// Order longest-first so "de villepin" wins over "villepin", etc.
var candidateForms = sortedKeysByLenDesc(candidate2027)

// A percentage cell: 1–2 digits, optional decimal, optional % sign. The % is
// optional because some instituts (e.g. Ifop) print bare numbers in Brut/
// Redressé columns rather than "%"-suffixed values. We always take the LAST such
// cell on a candidate line (the redressé / final column).
var pctRe = regexp.MustCompile(`(\d{1,2}(?:[.,]\d)?)\s*%?`)

// presidentialFirstRoundQ matches the legally-standardised first-round vote
// question that heads each intentions table ("Si le 1er tour de l'élection
// présidentielle avait lieu…"). We anchor on this wording — which is far more
// stable across instituts than any table layout — to isolate the presidential
// first round from run-off, European or legislative tables in the same notice.
var presidentialFirstRoundQ = regexp.MustCompile(`(?i)(1er|premier)\s+tour\s+de\s+l.{0,3}élection\s+présidentielle`)

// otherQuestion marks the start of any other question block (2nd round, or the
// next hypothesis) so a first-round segment stops before it.
var otherQuestion = regexp.MustCompile(`(?i)(2(nd|e|ème)?|second|deuxième)\s+tour\s+de\s+l.{0,3}élection|hypothèse\s+[2-9]|élections?\s+(européennes|législatives)|liste`)

// parseFirstHypothesis extracts the first-round BASE-hypothesis intentions.
// It splits the notice into segments starting at each presidential first-round
// question and returns the first segment that passes the validation gate — so
// the questionnaire (no %) and unrelated tables (European/legislative/run-off)
// are skipped, and Hypothèse 1 is picked over 2, 3, …
func parseFirstHypothesis(text string) (map[string]float64, bool) {
	locs := presidentialFirstRoundQ.FindAllStringIndex(text, -1)
	for i, loc := range locs {
		start := loc[1]
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		seg := text[start:end]
		// Cut the segment at the next non-first-round question if it appears.
		if m := otherQuestion.FindStringIndex(seg); m != nil {
			seg = seg[:m[0]]
		}
		if res, ok := extractCandidatePcts(seg); ok {
			return res, true
		}
	}
	return nil, false
}

// extractCandidatePcts reads candidate→pct (last % on the line = "redressé")
// from one table segment and applies the validation gate.
func extractCandidatePcts(seg string) (map[string]float64, bool) {
	results := map[string]float64{}
	for _, line := range strings.Split(seg, "\n") {
		ll := strings.ToLower(line)
		for _, form := range candidateForms {
			if !strings.Contains(ll, form) {
				continue
			}
			canon := candidate2027[form]
			if _, seen := results[canon]; seen {
				break
			}
			if ms := pctRe.FindAllStringSubmatch(line, -1); len(ms) > 0 {
				v := ms[len(ms)-1][1]
				if f, err := strconv.ParseFloat(strings.Replace(v, ",", ".", 1), 64); err == nil {
					results[canon] = f
				}
			}
			break // one candidate per line
		}
	}
	var sum float64
	for _, v := range results {
		sum += v
	}
	if len(results) < 8 || sum < 85 || sum > 115 {
		return results, false
	}
	return results, true
}

// parseSample pulls the sample size ("Échantillon : 1 628") from notice text.
func parseSample(text string) int {
	re := regexp.MustCompile(`(?i)échantillon\s*(?:utile)?\s*[:=]?\s*([0-9][0-9 \.]{2,})`)
	m := re.FindStringSubmatch(text)
	if m == nil {
		return 0
	}
	clean := strings.NewReplacer(" ", "", ".", "", " ", "").Replace(m[1])
	n, _ := strconv.Atoi(clean)
	return n
}

func indexOfFirst(s string, subs ...string) int {
	best := -1
	for _, sub := range subs {
		if i := strings.Index(s, sub); i >= 0 && (best < 0 || i < best) {
			best = i
		}
	}
	return best
}

func sortedKeysByLenDesc(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if len(keys[j]) > len(keys[i]) {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
