import { getAggregates } from "../lib/api";
import { useAsync } from "../lib/useAsync";
import { pct, safeColor } from "../lib/format";
import { AsyncState } from "./ui";
import { ChartFrame } from "./ChartFrame";

// Round 3 = per-candidate best-config view: each candidate's highest score
// across all first-round hypotheses. Complementary to the poll-of-polls; the
// total exceeds 100 % because mutually-exclusive scenarios are superposed.
export function BestConfig({ cycle, window }: { cycle: string; window: number }) {
  const { data, error, loading } = useAsync(
    () => getAggregates(cycle, 3, window),
    [cycle, window]
  );

  const rows = (data ?? []).slice().sort((a, b) => b.avg_pct - a.avg_pct);
  const max = Math.max(1, ...rows.map((r) => r.avg_pct));

  return (
    <div className="card">
      <ChartFrame
        title="Meilleur score par candidat — sa configuration"
        subtitle="Score le plus élevé mesuré pour chaque candidat, dans l'hypothèse construite autour de lui (ex. Attal dans le scénario Attal, Philippe dans le scénario Philippe)."
        source="Vue complémentaire — la somme dépasse 100 % : les scénarios sont mutuellement exclusifs et superposés. N'entre PAS dans la moyenne pondérée."
      >
        <AsyncState
          loading={loading}
          error={error}
          empty={!loading && rows.length === 0}
          emptyText="Pas de données de configuration pour ce cycle."
        />
        {rows.length > 0 && (
          <div className="cfg-list tnum">
            {rows.map((r) => {
              const c = safeColor(r.color);
              return (
                <div className="cfg-row" key={r.candidate}>
                  <span className="cfg-name">{r.candidate}</span>
                  <div className="cfg-bar-wrap">
                    <div
                      className="cfg-bar"
                      style={{ width: `${(r.avg_pct / max) * 100}%`, background: c }}
                    />
                  </div>
                  <span className="cfg-val">
                    {pct(r.avg_pct)} <span className="unit">%</span>
                  </span>
                </div>
              );
            })}
          </div>
        )}
      </ChartFrame>
    </div>
  );
}
