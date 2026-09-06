// Blackout légal — loi n°77-808 du 19 juillet 1977, art. 11.
// La diffusion de sondages électoraux est interdite la veille et le jour du
// scrutin, jusqu'à la fermeture du dernier bureau de vote en métropole (20h00,
// heure de Paris). Pendant cette fenêtre le site masque toute donnée issue de
// sondages.
//
// Fenêtres exprimées en UTC (indépendant du fuseau du visiteur).
// Avril 2027 = CEST (UTC+2) : samedi 00h00 Paris = vendredi 22h00 UTC,
//                             dimanche 20h00 Paris = dimanche 18h00 UTC.
//
// ⚠️ Dates du scrutin 2027 À CONFIRMER par décret de convocation des électeurs
// (non encore publié). Mettre à jour dès parution au JO.
export type BlackoutWindow = {
  label: string;
  startUtc: string; // inclus
  endUtc: string; // exclus
};

export const BLACKOUT_WINDOWS: BlackoutWindow[] = [
  {
    label: "1er tour de l'élection présidentielle 2027",
    startUtc: "2027-04-09T22:00:00Z", // samedi 10 avril 00h00 Paris
    endUtc: "2027-04-11T18:00:00Z", // dimanche 11 avril 20h00 Paris
  },
  {
    label: "2nd tour de l'élection présidentielle 2027",
    startUtc: "2027-04-23T22:00:00Z", // samedi 24 avril 00h00 Paris
    endUtc: "2027-04-25T18:00:00Z", // dimanche 25 avril 20h00 Paris
  },
];

export function activeBlackout(now: Date = new Date()): BlackoutWindow | null {
  const t = now.getTime();
  for (const w of BLACKOUT_WINDOWS) {
    if (t >= Date.parse(w.startUtc) && t < Date.parse(w.endUtc)) return w;
  }
  return null;
}

// Fin de blackout formatée en heure de Paris, pour le message utilisateur.
export function formatParis(iso: string): string {
  return new Intl.DateTimeFormat("fr-FR", {
    timeZone: "Europe/Paris",
    dateStyle: "full",
    timeStyle: "short",
  }).format(Date.parse(iso));
}
