import type { MomentumPoint } from "../lib/api";
import { getMomentum } from "../lib/api";
import { useAsync } from "../lib/useAsync";
import { pct, safeColor } from "../lib/format";
import { AsyncState } from "./ui";
import { ChartFrame } from "./ChartFrame";

/** Signed percentage-point value with French comma and explicit + sign. */
function signedPts(v: number, digits = 1): string {
  const sign = v > 0 ? "+" : v < 0 ? "−" : "";
  return `${sign}${pct(Math.abs(v), digits)}`;
}

function MomentumCard({ m }: { m: MomentumPoint }) {
  const c = safeColor(m.color);
  const dir = m.direction;
  const arrow = dir === "up" ? "▲" : dir === "down" ? "▼" : "—";
  const rate =
    dir === "flat"
      ? "stable"
      : `${signedPts(m.per_week)}/sem`;

  return (
    <div className="mom-card">
      <div className="mom-cand">
        <span className="dot" style={{ background: c }} />
        <span className="mom-name">{m.candidate}</span>
      </div>
      <div className="mom-current tnum">
        {pct(m.current)}
        <span className="unit">&nbsp;%</span>
      </div>
      <div className={`mom-chip ${dir}`}>
        <span className="mom-arrow" aria-hidden="true">
          {arrow}
        </span>
        <span className="mom-delta tnum">
          {dir === "flat" ? "—" : `${signedPts(m.delta)} pts`}
        </span>
      </div>
      <div className="mom-rate tnum">
        {dir === "flat" ? "sur 30 j" : rate}
      </div>
    </div>
  );
}

export function Momentum({ cycle, round }: { cycle: string; round: number }) {
  const { data, error, loading } = useAsync(
    () => getMomentum(cycle, round, 30),
    [cycle, round]
  );

  const rows = data ?? [];
  // Biggest movers first (backend already sorts by |delta| desc, but be safe).
  const sorted = [...rows].sort((a, b) => Math.abs(b.delta) - Math.abs(a.delta));
  const up = sorted
    .filter((m) => m.direction === "up")
    .sort((a, b) => b.delta - a.delta);
  const down = sorted
    .filter((m) => m.direction === "down")
    .sort((a, b) => a.delta - b.delta);
  const flat = sorted.filter((m) => m.direction === "flat");

  const empty = !loading && rows.length === 0;
  const notEnough = !loading && !error && rows.length > 0 && rows.length < 2;

  return (
    <div className="card">
      <ChartFrame
        title="Momentum"
        subtitle="Progression des candidats sur les 30 derniers jours (points de %)."
        source="Source : nsppolls ; Wikipédia"
      >
        <AsyncState
          loading={loading}
          error={error}
          empty={empty}
          emptyText={`Pas de momentum pour ce tour (${cycle}, tour ${round}).`}
        />

        {notEnough && (
          <div className="state">
            Pas assez d'historique pour calculer le momentum.
          </div>
        )}

        {!notEnough && rows.length >= 2 && (
          <div className="mom-columns">
            <div className="mom-col">
              <div className="mom-col-head mom-col-head--up">En hausse</div>
              {up.length > 0 ? (
                <div className="mom-grid">
                  {up.map((m) => (
                    <MomentumCard key={m.candidate} m={m} />
                  ))}
                </div>
              ) : (
                <p className="mom-col-empty">Aucun candidat en hausse.</p>
              )}
            </div>

            <div className="mom-col">
              <div className="mom-col-head mom-col-head--down">En baisse</div>
              {down.length > 0 ? (
                <div className="mom-grid">
                  {down.map((m) => (
                    <MomentumCard key={m.candidate} m={m} />
                  ))}
                </div>
              ) : (
                <p className="mom-col-empty">Aucun candidat en baisse.</p>
              )}
            </div>

            {flat.length > 0 && (
              <div className="mom-col mom-col--flat">
                <div className="mom-col-head mom-col-head--flat">Stables</div>
                <div className="mom-grid">
                  {flat.map((m) => (
                    <MomentumCard key={m.candidate} m={m} />
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </ChartFrame>
    </div>
  );
}
