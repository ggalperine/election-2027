import { useMemo, useState } from "react";
import { getPolls, type Poll } from "../lib/api";
import { useAsync } from "../lib/useAsync";
import { pct, intFr, frDate, safeColor } from "../lib/format";
import { AsyncState } from "./ui";

type SortKey = "pollster" | "sponsor" | "field_end" | "sample_size" | "lead";
type Dir = "asc" | "desc";

/** Winner (candidate, pct) inside a single poll's results map. */
function topOf(p: Poll): { cand: string; val: number } | null {
  let best: { cand: string; val: number } | null = null;
  for (const [cand, val] of Object.entries(p.results))
    if (!best || val > best.val) best = { cand, val };
  return best;
}

export function SourcesTable({ cycle, round }: { cycle: string; round: number }) {
  const { data, error, loading } = useAsync(
    () => getPolls(cycle, round, 120),
    [cycle, round]
  );
  const [sort, setSort] = useState<SortKey>("field_end");
  const [dir, setDir] = useState<Dir>("desc");

  const polls = data ?? [];

  // stable candidate → color hint from all polls not available here; derive palette by lead
  const rows = useMemo(() => {
    const withTop = polls.map((p) => ({ p, top: topOf(p) }));
    const sorted = [...withTop].sort((a, b) => {
      let cmp = 0;
      switch (sort) {
        case "pollster":
          cmp = a.p.pollster.localeCompare(b.p.pollster);
          break;
        case "sponsor":
          cmp = (a.p.sponsor ?? "").localeCompare(b.p.sponsor ?? "");
          break;
        case "field_end":
          cmp = a.p.field_end.localeCompare(b.p.field_end);
          break;
        case "sample_size":
          cmp = a.p.sample_size - b.p.sample_size;
          break;
        case "lead":
          cmp = (a.top?.val ?? 0) - (b.top?.val ?? 0);
          break;
      }
      return dir === "asc" ? cmp : -cmp;
    });
    return sorted;
  }, [polls, sort, dir]);

  const setSortKey = (k: SortKey) => {
    if (k === sort) setDir((d) => (d === "asc" ? "desc" : "asc"));
    else {
      setSort(k);
      setDir(k === "field_end" || k === "sample_size" || k === "lead" ? "desc" : "asc");
    }
  };

  const arrow = (k: SortKey) =>
    sort === k ? <span className="arrow">{dir === "asc" ? "↑" : "↓"}</span> : null;

  const instituts = new Set(polls.map((p) => p.pollster)).size;

  return (
    <div className="card">
      <div className="card-head">
        <div>
          <h3>Toutes les sources</h3>
          <div className="meta">
            {polls.length > 0
              ? `${polls.length} sondages · ${instituts} instituts · lien vers la source officielle`
              : "Sondages individuels agrégés"}
          </div>
        </div>
      </div>

      <AsyncState
        loading={loading}
        error={error}
        empty={!loading && polls.length === 0}
      />

      {polls.length > 0 && (
        <div className="table-scroll">
          <table className="polls">
            <thead>
              <tr>
                <th onClick={() => setSortKey("pollster")}>Institut {arrow("pollster")}</th>
                <th onClick={() => setSortKey("sponsor")}>Commanditaire {arrow("sponsor")}</th>
                <th onClick={() => setSortKey("field_end")}>Fin de terrain {arrow("field_end")}</th>
                <th className="num" onClick={() => setSortKey("sample_size")}>
                  Échantillon {arrow("sample_size")}
                </th>
                <th onClick={() => setSortKey("lead")}>En tête {arrow("lead")}</th>
                <th>Source</th>
              </tr>
            </thead>
            <tbody>
              {rows.map(({ p, top }) => (
                <tr key={p.external_id}>
                  <td className="pollster">{p.pollster}</td>
                  <td>{p.sponsor ?? <span className="muted">—</span>}</td>
                  <td className="tnum">{frDate(p.field_end)}</td>
                  <td className="num tnum">{intFr(p.sample_size)}</td>
                  <td>
                    {top ? (
                      <span className="lead-tag">
                        <span
                          className="dot"
                          style={{ background: leadColor(top.cand) }}
                        />
                        {top.cand}{" "}
                        <span className="muted tnum">{pct(top.val)}%</span>
                      </span>
                    ) : (
                      <span className="muted">—</span>
                    )}
                  </td>
                  <td>
                    {p.source_url ? (
                      <a href={p.source_url} target="_blank" rel="noreferrer">
                        source ↗
                      </a>
                    ) : (
                      <span className="muted">—</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

/* Lightweight color hints for the "en tête" dot (results maps carry no color). */
const LEAD_COLORS: Record<string, string> = {
  "Le Pen": "#0d378a",
  "Marine Le Pen": "#0d378a",
  Bardella: "#0d378a",
  Mélenchon: "#cc2443",
  Philippe: "#00b0f0",
  Attal: "#ffb600",
  Macron: "#ffb600",
  Glucksmann: "#f0426d",
  Zemmour: "#5a3e8e",
  Retailleau: "#0066cc",
};
function leadColor(cand: string): string {
  return safeColor(LEAD_COLORS[cand] ?? "#94a3b8");
}
