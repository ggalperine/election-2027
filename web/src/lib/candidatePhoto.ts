// Maps a candidate's canonical short name (as returned by the API — the same
// canon used server-side in internal/poll/notice.go) to a local portrait in
// web/public/candidates/. Portraits are Wikimedia Commons thumbnails (CC-BY /
// public domain), fetched and re-encoded to small webp avatars.
const SLUGS: Record<string, string> = {
  "Le Pen": "le-pen",
  Mélenchon: "melenchon",
  Philippe: "philippe",
  Attal: "attal",
  Glucksmann: "glucksmann",
  Retailleau: "retailleau",
  Zemmour: "zemmour",
  Tondelier: "tondelier",
  Roussel: "roussel",
  Villepin: "villepin",
  "Dupont-Aignan": "dupont-aignan",
  Arthaud: "arthaud",
  Poutou: "poutou",
  Bardella: "bardella",
  Lisnard: "lisnard",
  Ruffin: "ruffin",
  Hollande: "hollande",
};

export function candidatePhoto(name: string): string | null {
  const slug = SLUGS[name];
  return slug ? `/candidates/${slug}.webp` : null;
}

// initials builds a 1–2 letter fallback (e.g. "Le Pen" → "LP") for candidates
// without a portrait.
export function initials(name: string): string {
  return name
    .split(/[\s-]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0])
    .join("")
    .toUpperCase();
}
