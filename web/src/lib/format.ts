// Small shared formatting & color helpers.

/** 33 → "33,0" (French decimal comma). */
export function pct(v: number, digits = 1): string {
  return v.toLocaleString("fr-FR", {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  });
}

/** "2026-09-01" → "1 sept. 2026" */
export function frDate(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  return d.toLocaleDateString("fr-FR", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

/** "2026-09-01" → "sept. 26" (compact, for axis ticks) */
export function frMonth(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  return d.toLocaleDateString("fr-FR", { month: "short", year: "2-digit" });
}

/** Group integer thousands with French spacing: 5281.6 → "5 282" */
export function intFr(v: number): string {
  return Math.round(v).toLocaleString("fr-FR");
}

const NEUTRAL = "#94a3b8";
/** Backend sends #888888 for some candidates; substitute a cleaner neutral. */
export function safeColor(c: string): string {
  return !c || c.toLowerCase() === "#888888" ? NEUTRAL : c;
}

/** Convert hex to rgba for translucent CI bands. */
export function withAlpha(hex: string, alpha: number): string {
  const h = safeColor(hex).replace("#", "");
  const full =
    h.length === 3
      ? h.split("").map((x) => x + x).join("")
      : h.padEnd(6, "0").slice(0, 6);
  const r = parseInt(full.slice(0, 2), 16);
  const g = parseInt(full.slice(2, 4), 16);
  const b = parseInt(full.slice(4, 6), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}
