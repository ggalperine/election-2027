import { getSimulate } from "../lib/api";
import { useAsync } from "../lib/useAsync";
import { pct, frDate, safeColor } from "../lib/format";
import { AsyncState } from "./ui";
import { ChartFrame } from "./ChartFrame";

// P(x) formatted as a French percentage. Below 0.5% we say "<0,5 %" rather
// than "0 %" — a small but non-zero chance should not read as impossible.
// The probability cells use white-space:nowrap (CSS) so ">99,5 %" never wraps.
function prob(p: number): string {
  if (p <= 0) return "0 %";
  if (p < 0.005) return "<0,5 %";
  if (p > 0.995) return ">99,5 %";
  return `${(p * 100).toFixed(0).replace(".", ",")} %`;
}

export function Probabilities({ cycle, window }: { cycle: string; window: number }) {
  const { data, error, loading } = useAsync(
    () => getSimulate(cycle, window),
    [cycle, window]
  );

  const rows = data?.candidates ?? [];
  // Show the contenders; hide the long tail of ~0% also-rans (keep top 8).
  const shown = rows.slice(0, 8);
  const maxQ = Math.max(0.01, ...shown.map((r) => r.prob_qualify));

  const subtitle = data
    ? `Au ${frDate(data.as_of)} · scrutin le ${frDate(data.election_date)} · ` +
      `${data.days_left} j restants · ${data.n_sims.toLocaleString("fr-FR")} simulations`
    : "—";

  return (
    <div className="card">
      <ChartFrame
        title="Probabilités — simulation Monte Carlo"
        subtitle={subtitle}
        source="Modèle probabiliste : 20 000 tirages de la distribution prédictive (erreur d'échantillonnage ⊕ dispersion inter-instituts ⊕ dérive temporelle en marche aléatoire, lois de Student). Qualification = arrivée dans les deux premiers ; victoire conditionnée aux sondages de 2nd tour."
      >
        <AsyncState
          loading={loading}
          error={error}
          empty={!loading && rows.length === 0}
          emptyText={`Pas de simulation pour ce cycle (${cycle}).`}
        />

        {shown.length > 0 && (
          <table className="prob-table tnum">
            <thead>
              <tr>
                <th className="prob-cand">Candidat</th>
                <th>Score médian</th>
                <th>Intervalle 90 %</th>
                <th>P(2nd tour)</th>
                <th>P(en tête)</th>
                <th>P(victoire)</th>
              </tr>
            </thead>
            <tbody>
              {shown.map((r) => {
                const c = safeColor(r.color);
                return (
                  <tr key={r.candidate}>
                    <td className="prob-cand">
                      <span className="prob-dot" style={{ background: c }} />
                      <span className="prob-name">{r.candidate}</span>
                      <span className="prob-party">{r.party}</span>
                    </td>
                    <td>{pct(r.p50)} %</td>
                    <td className="muted">
                      {pct(r.p05)} – {pct(r.p95)}
                    </td>
                    <td>
                      <div className="prob-bar-wrap">
                        <div
                          className="prob-bar"
                          style={{
                            width: `${(r.prob_qualify / maxQ) * 100}%`,
                            background: c,
                          }}
                        />
                        <span className="prob-val">{prob(r.prob_qualify)}</span>
                      </div>
                    </td>
                    <td>{prob(r.prob_lead)}</td>
                    <td className="prob-win">{prob(r.prob_win)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </ChartFrame>
    </div>
  );
}
