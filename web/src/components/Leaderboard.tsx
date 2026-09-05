import { getAggregates } from "../lib/api";
import { useAsync } from "../lib/useAsync";
import { pct, intFr, frDate, safeColor } from "../lib/format";
import { AsyncState } from "./ui";
import { ChartFrame } from "./ChartFrame";

export function Leaderboard({
  cycle,
  round,
  window,
}: {
  cycle: string;
  round: number;
  window: number;
}) {
  const { data, error, loading } = useAsync(
    () => getAggregates(cycle, round, window),
    [cycle, round, window]
  );

  const rows = data ?? [];
  const asOf = rows[0]?.as_of;
  const totalPolls = rows.reduce((m, r) => Math.max(m, r.n_polls), 0);
  const nEff = rows[0]?.n_eff ?? 0;

  const subtitle =
    (asOf ? `Au ${frDate(asOf)}` : "—") +
    (rows.length > 0
      ? ` · ${totalPolls} sondages agrégés · échantillon effectif ≈ ${intFr(nEff)}`
      : "");

  return (
    <div className="card">
      <ChartFrame
        title="Intentions de vote — moyenne pondérée"
        subtitle={subtitle}
        source="Source : nsppolls ; Wikipédia — Sondages agrégés, moyenne pondérée ; IC à 95 %"
      >

      <AsyncState
        loading={loading}
        error={error}
        empty={!loading && rows.length === 0}
        emptyText={`Pas d'agrégat pour ce tour (${cycle}, tour ${round}).`}
      />

      {rows.length > 0 && (
        <div className="stat-grid">
          {rows.map((r, i) => {
            const c = safeColor(r.color);
            const margin = ((r.hi - r.lo) / 2).toFixed(1).replace(".", ",");
            return (
              <div className="stat-card" key={r.candidate}>
                <span className="accent-bar" style={{ background: c }} />
                <span className="rank">#{i + 1}</span>
                <div className="cand">{r.candidate}</div>
                <div className="party">{r.party}</div>
                <div className="big tnum">
                  {pct(r.avg_pct)}
                  <span className="unit">%</span>
                </div>
                <div className="ci tnum">
                  IC 95 % {pct(r.lo)} – {pct(r.hi)} <span className="muted">(± {margin})</span>
                </div>
                <div className="foot tnum">
                  <span>{r.n_polls} sond.</span>
                  <span>n≈ {intFr(r.n_eff)}</span>
                </div>
              </div>
            );
          })}
        </div>
      )}
      </ChartFrame>
    </div>
  );
}
