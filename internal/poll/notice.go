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

// firstRoundMarker matches any "1er tour" / "premier tour" heading. We anchor on
// this (far more stable across instituts than table layout) then vet the heading
// context, because the presidential wording is often wrapped across lines.
var firstRoundMarker = regexp.MustCompile(`(?i)(1er|premier)\s+tour`)

// headExclude rejects a heading that introduces a NON-2027-intention table:
// a past-vote reconstitution/recall, or a legislative/European election.
var headExclude = regexp.MustCompile(`(?i)reconstitu|souvenir|législ|legisl|europ|municipal|sénatorial`)

// segCut marks where a first-round presidential segment must STOP, so it never
// bleeds into an adjacent foreign table (another election, a party list, the
// run-off, or the next hypothesis) that follows within the same marker gap.
var segCut = regexp.MustCompile(`(?i)reconstitu|souvenir|législ|legisl|europ|municipal|sénatorial|\bliste\b|(2(nd|e|ème)?|second|deuxième)\s+tour`)

// parseFirstHypothesis extracts the first-round BASE-hypothesis intentions.
// For each "1er tour" marker it vets the surrounding heading (must concern the
// présidentielle, must not be a reconstitution/other-election table), then
// extracts candidate→pct from the segment up to the next marker. The first
// segment passing the validation gate wins — so methodology prose, the
// questionnaire (no %), the 2022 recall and run-off tables are all skipped.
func parseFirstHypothesis(text string) (map[string]float64, bool) {
	locs := firstRoundMarker.FindAllStringIndex(text, -1)
	for i, loc := range locs {
		// Exclusion is judged on a TIGHT window (a reconstitution/other-election
		// keyword right next to the marker), so a distant one doesn't disqualify.
		excl := strings.ToLower(text[clamp(loc[0]-80, 0, len(text)):clamp(loc[1]+40, 0, len(text))])
		if headExclude.MatchString(excl) {
			continue
		}
		// "présidentiel" is looked for on a WIDER window, because a data-table
		// marker (e.g. "ITV 1er tour") can sit a few lines below the question
		// that names the présidentielle.
		ctx := strings.ToLower(text[clamp(loc[0]-260, 0, len(text)):clamp(loc[1]+160, 0, len(text))])
		if !strings.Contains(ctx, "présidentiel") && !strings.Contains(ctx, "presidentiel") {
			continue
		}
		start := loc[1]
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		seg := text[start:end]
		if c := segCut.FindStringIndex(seg); c != nil {
			seg = seg[:c[0]] // stop before any foreign table in the gap
		}
		if res, ok := extractCandidatePcts(seg); ok {
			return res, true
		}
	}
	// Fallback: some instituts print the full-field table far from its question.
	// A block of ≥10 KNOWN 2027 candidates summing ~100 is almost certainly the
	// first-round intention wherever it sits (a 2022 recall has <10 in-map names;
	// favourability tables don't sum to 100; Elabe's 6-candidate configs are too
	// small) — so we accept it even without a nearby marker.
	return globalFullFieldScan(text)
}

// blockBreak ends a candidate block: a foreign section or a non-vote table.
var blockBreak = regexp.MustCompile(`(?i)reconstitu|souvenir|législ|legisl|europ|municipal|sénatorial|\bliste\b|conduite par|image|confiance|favorable|popularité|(2nd|2ème|2eme|2e|second|deuxième)\s+tour`)

func globalFullFieldScan(text string) (map[string]float64, bool) {
	lines := strings.Split(text, "\n")
	block := map[string]float64{}
	gap := 0
	closeBlock := func() (map[string]float64, bool) {
		if len(block) >= 10 {
			var sum float64
			for _, v := range block {
				sum += v
			}
			if sum >= 90 && sum <= 110 {
				return block, true
			}
		}
		block = map[string]float64{}
		return nil, false
	}
	for _, line := range lines {
		if blockBreak.MatchString(line) {
			if r, ok := closeBlock(); ok {
				return r, true
			}
			continue
		}
		matched := false
		ll := strings.ToLower(line)
		for _, form := range candidateForms {
			if !strings.Contains(ll, form) {
				continue
			}
			matched = true
			canon := candidate2027[form]
			if _, seen := block[canon]; !seen {
				if ms := pctRe.FindAllStringSubmatch(line, -1); len(ms) > 0 {
					if f, err := strconv.ParseFloat(strings.Replace(ms[len(ms)-1][1], ",", ".", 1), 64); err == nil {
						block[canon] = f
					}
				}
			}
			break
		}
		if matched {
			gap = 0
		} else if gap++; gap > 3 {
			if r, ok := closeBlock(); ok {
				return r, true
			}
		}
	}
	return closeBlock()
}

// secondRoundMarker matches a run-off heading ("2nd tour" / "second tour").
var secondRoundMarker = regexp.MustCompile(`(?i)(2nd|2ème|2eme|2e|second|deuxième)\s+tour`)

// anyRoundMarker bounds segments on either round's heading.
var anyRoundMarker = regexp.MustCompile(`(?i)(1er|premier|2nd|2ème|2eme|2e|second|deuxième)\s+tour`)

// round2Cut stops a run-off segment before a foreign table or a first-round one.
var round2Cut = regexp.MustCompile(`(?i)reconstitu|souvenir|législ|legisl|europ|municipal|sénatorial|\bliste\b|(1er|premier)\s+tour`)

// parseDuels extracts every second-round head-to-head (two candidates summing to
// ~100) from a notice. Each duel becomes a round-2 poll. Deduped by pair.
func parseDuels(text string) []map[string]float64 {
	markers := anyRoundMarker.FindAllStringIndex(text, -1)
	var duels []map[string]float64
	seen := map[string]bool{}
	for i, loc := range markers {
		if !secondRoundMarker.MatchString(text[loc[0]:loc[1]]) {
			continue
		}
		excl := strings.ToLower(text[clamp(loc[0]-80, 0, len(text)):clamp(loc[1]+40, 0, len(text))])
		if headExclude.MatchString(excl) {
			continue
		}
		ctx := strings.ToLower(text[clamp(loc[0]-260, 0, len(text)):clamp(loc[1]+160, 0, len(text))])
		if !strings.Contains(ctx, "présidentiel") && !strings.Contains(ctx, "presidentiel") {
			continue
		}
		end := len(text)
		if i+1 < len(markers) {
			end = markers[i+1][0]
		}
		seg := text[loc[1]:end]
		if c := round2Cut.FindStringIndex(seg); c != nil {
			seg = seg[:c[0]]
		}
		if pair, ok := extractDuel(seg); ok {
			key := duelPairKey(pair)
			if !seen[key] {
				seen[key] = true
				duels = append(duels, pair)
			}
		}
	}
	return duels
}

// extractDuel reads EXACTLY two candidates (summing ~100) from a run-off block.
func extractDuel(seg string) (map[string]float64, bool) {
	res := map[string]float64{}
	for _, line := range strings.Split(seg, "\n") {
		ll := strings.ToLower(line)
		for _, form := range candidateForms {
			if !strings.Contains(ll, form) {
				continue
			}
			canon := candidate2027[form]
			if _, seen := res[canon]; seen {
				break
			}
			if ms := pctRe.FindAllStringSubmatch(line, -1); len(ms) > 0 {
				if f, err := strconv.ParseFloat(strings.Replace(ms[len(ms)-1][1], ",", ".", 1), 64); err == nil {
					res[canon] = f
				}
			}
			break
		}
	}
	if len(res) != 2 {
		return nil, false
	}
	var sum float64
	for _, v := range res {
		sum += v
	}
	if sum < 90 || sum > 110 {
		return nil, false
	}
	return res, true
}

func duelPairKey(m map[string]float64) string {
	names := make([]string, 0, 2)
	for k := range m {
		names = append(names, k)
	}
	if len(names) == 2 && names[0] > names[1] {
		names[0], names[1] = names[1], names[0]
	}
	return strings.Join(names, "|")
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

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
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
