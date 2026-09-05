package poll

import "strings"

// pollsterSites maps institut names (as they appear in feeds) to their official
// website. Used to enrich the sources table in the UI. All these instituts are
// already consolidated inside the nsppolls and Wikipedia feeds.
var pollsterSites = map[string]string{
	"ifop":                        "https://www.ifop.com",
	"ipsos":                       "https://www.ipsos.com/fr-fr",
	"opinionway":                  "https://www.opinion-way.com",
	"opinion way":                 "https://www.opinion-way.com",
	"harris interactive":          "https://harris-interactive.fr",
	"harris":                      "https://harris-interactive.fr",
	"kantar":                      "https://www.kantar.com",
	"kantar public":               "https://www.kantarpublic.com",
	"csa":                         "https://www.csa.eu",
	"bva":                         "https://www.bva-group.com",
	"bva xsight":                  "https://www.bva-group.com",
	"elabe":                       "https://elabe.fr",
	"odoxa":                       "https://www.odoxa.fr",
	"cluster17":                   "https://cluster17.com",
	"cluster 17":                  "https://cluster17.com",
	"mediametrie":                 "https://www.mediametrie.fr",
	"médiamétrie":                 "https://www.mediametrie.fr",
	"yougov":                      "https://fr.yougov.com",
	"verian":                      "https://www.veriangroup.com",
	"toluna harris":               "https://harris-interactive.fr",
	"redfield & wilton strategies": "https://redfieldandwiltonstrategies.com",
	"atlasintel":                  "https://atlasintel.org",
}

// PollsterSite returns the official website for an institut, or "" if unknown.
func PollsterSite(name string) string {
	return pollsterSites[strings.ToLower(strings.TrimSpace(name))]
}
