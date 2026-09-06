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

var pctRe = regexp.MustCompile(`(\d{1,2}(?:[.,]\d)?)\s*%`)

// parseFirstHypothesis extracts the first-round base-hypothesis intentions from
// notice text. Returns candidate→pct (redressé where two figures are present).
// ok is false when the result fails the validation gate.
func parseFirstHypothesis(text string) (results map[string]float64, ok bool) {
	// Scope to the first-round section to avoid picking up run-off duel numbers.
	lower := strings.ToLower(text)
	start := indexOfFirst(lower, "hypothèse 1", "1er tour", "premier tour")
	if start < 0 {
		start = 0
	}
	// Stop before any explicit second-round section.
	end := len(text)
	if i := indexOfFirst(lower[start:], "2nd tour", "second tour", "2ème tour", "deuxième tour"); i >= 0 {
		end = start + i
	}
	scope := text[start:end]

	results = map[string]float64{}
	for _, line := range strings.Split(scope, "\n") {
		ll := strings.ToLower(line)
		for _, form := range candidateForms {
			if !strings.Contains(ll, form) {
				continue
			}
			canon := candidate2027[form]
			if _, seen := results[canon]; seen {
				continue
			}
			// take the LAST percentage on the line = "redressé" column
			if ms := pctRe.FindAllStringSubmatch(line, -1); len(ms) > 0 {
				v := ms[len(ms)-1][1]
				if f, err := strconv.ParseFloat(strings.Replace(v, ",", ".", 1), 64); err == nil {
					results[canon] = f
				}
			}
			break // one candidate per line
		}
	}

	// Validation gate: enough candidates + total near 100 %.
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
