import { useMemo } from "react";
import { getAggregates, getActual, type ActualResult, type AggregatePoint } from "../lib/api";
import { useAsync } from "../lib/useAsync";
import { pct, safeColor } from "../lib/format";
import { AsyncState } from "./ui";

/** Normalise a name to its surname-ish key for matching poll ↔ actual. */
function key(name: string): string {
  const parts = name.trim().split(/\s+/);
  // keep last two tokens to catch "Le Pen", "Dupont-Aignan"
  const tail = parts.slice(-2).join(" ").toLowerCase();
  const last = parts.slice(-1)[0].toLowerCase();
  return `${tail}|${last}`;
}

interface CmpRow {
  name: string;
  color: string;
  poll?: number;
  actual: number;
  won: boolean;
  diff?: number;
}

function build(agg: AggregatePoint[], actual: ActualResult[]): CmpRow[] {
  const pollByKey = new Map<string, AggregatePoint>();
  for (const a of agg) {
    const [tail, last] = key(a.candidate).split("|");
    pollByKey.set(tail, a);
    pollByKey.set(last, a);
  }

  return actual
    .map((a) => {
      const [tail, last] = key(a.candidate).split("|");
      const p = pollByKey.get(tail) ?? pollByKey.get(last);
      return {
        name: a.candidate,
        color: safeColor(a.color !== "#888888" ? a.color : p?.color ?? a.color),
        poll: p?.avg_pct,
        actual: a.pct,
        won: a.won,
        diff: p ? a.pct - p.avg_pct : undefined,
      };
    })
    .sort((x, y) => y.actual - x.actual);
}

export function PollVsOutcome({ cycle, round }: { cycle: string; round: number }) {
  const agg = useAsync(() => getAggregates(cycle, round), [cycle, round]);
  const act = useAsync(() => getActual(cycle, round), [cycle, round]);

  const rows = useMemo(
    () => build(agg.data ?? [], act.data ?? []),
    [agg.data, act.data]
  );

  const max = Math.max(1, ...rows.flatMap((r) => [r.actual, r.poll ?? 0]));
  const loading = agg.loading || act.loading;
  const error = agg.error || act.error;

  // biggest polling miss for the caption
  const biggestMiss = useMemo(() => {
    let best: CmpRow | undefined;
    for (const r of rows)
      if (r.diff != null && (!best || Math.abs(r.diff) > Math.abs(best.diff!)))
        best = r;
    return best;
  }, [rows]);

  return (
    <div className="card">
      <div className="card-head">
        <div>
          <h3>Sondages vs résultat réel</h3>
          <div className="meta">
            Moyenne finale des sondages comparée au résultat officiel du scrutin
          </div>
        </div>
      </div>

      <AsyncState
        loading={loading}
        error={error}
        empty={!loading && rows.length === 0}
      />

      {rows.length > 0 && (
        <>
          <div className="cmp-legend">
            <span className="k">
              <span className="box poll" /> Sondages (moyenne)
            </span>
            <span className="k">
              <span className="box" /> Résultat officiel
            </span>
          </div>

          <div className="cmp-list">
            {rows.map((r) => (
              <div className="cmp-row" key={r.name}>
                <div className="cmp-name">
                  <span className="dot" style={{ background: r.color }} />
                  {r.name}
                  {r.won && <span className="crown" title="Vainqueur">👑</span>}
                </div>
                <div className="cmp-bars">
                  <div className="cmp-bar">
                    <span className="tag">Sondages</span>
                    <span className="track">
                      <span
                        className="fill poll"
                        style={{
                          width: `${((r.poll ?? 0) / max) * 100}%`,
                          background: r.color,
                        }}
                      />
                    </span>
                    <span className="val tnum">
                      {r.poll != null ? `${pct(r.poll)}%` : "—"}
                    </span>
                  </div>
                  <div className="cmp-bar">
                    <span className="tag">Réel</span>
                    <span className="track">
                      <span
                        className="fill"
                        style={{
                          width: `${(r.actual / max) * 100}%`,
                          background: r.color,
                        }}
                      />
                    </span>
                    <span className="val tnum">{pct(r.actual)}%</span>
                  </div>
                </div>
              </div>
            ))}
          </div>

          {biggestMiss?.diff != null && (
            <p className="muted" style={{ fontSize: 12.5, marginTop: 16 }}>
              Écart le plus marqué : <strong>{biggestMiss.name}</strong>,{" "}
              {biggestMiss.diff > 0 ? "sous-estimé" : "surestimé"} de{" "}
              {pct(Math.abs(biggestMiss.diff))} point
              {Math.abs(biggestMiss.diff) >= 2 ? "s" : ""} par les sondages.
            </p>
          )}
        </>
      )}
    </div>
  );
}
