import { useMemo } from "react";
import { getPollsters, getHouseEffects, type HouseEffect } from "../lib/api";
import { useAsync } from "../lib/useAsync";
import { intFr, frDate, safeColor } from "../lib/format";
import { AsyncState } from "./ui";
import { ChartFrame } from "./ChartFrame";

/** French signed number: 1.2 → "+1,2", -0.8 → "−0,8" */
function signed(v: number, digits = 1): string {
  const sign = v > 0 ? "+" : v < 0 ? "−" : "";
  const abs = Math.abs(v).toLocaleString("fr-FR", {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  });
  return `${sign}${abs}`;
}

/** House effects grouped by institut, largest |delta| first within each. */
function groupByPollster(effects: HouseEffect[]) {
  const byPollster = new Map<string, HouseEffect[]>();
  let maxAbs = 0;
  for (const e of effects) {
    maxAbs = Math.max(maxAbs, Math.abs(e.delta));
    const list = byPollster.get(e.pollster) ?? [];
    list.push(e);
    byPollster.set(e.pollster, list);
  }
  const groups = [...byPollster.entries()].map(([pollster, items]) => ({
    pollster,
    items: [...items].sort((a, b) => Math.abs(b.delta) - Math.abs(a.delta)),
  }));
  // stable order: by pollster name
  groups.sort((a, b) => a.pollster.localeCompare(b.pollster));
  return { groups, maxAbs: maxAbs || 1 };
}

export function Instituts({ cycle, round }: { cycle: string; round: number }) {
  const ps = useAsync(() => getPollsters(cycle, round), [cycle, round]);
  const he = useAsync(() => getHouseEffects(cycle, round), [cycle, round]);

  const pollsters = ps.data ?? [];
  const { groups, maxAbs } = useMemo(
    () => groupByPollster(he.data ?? []),
    [he.data]
  );

  return (
    <div className="card">
      <ChartFrame
        title="Détails par institut"
        subtitle={
          pollsters.length > 0
            ? `${pollsters.length} instituts · activité et biais maison`
            : "Activité et biais des instituts"
        }
        source="Source : nsppolls ; Wikipédia — Biais maison = écart moyen à la moyenne de tous les sondages"
      >

      {/* (a) Activity table --------------------------------------------- */}
      <AsyncState
        loading={ps.loading}
        error={ps.error}
        empty={!ps.loading && pollsters.length === 0}
        emptyText={`Aucun institut pour ce tour (${cycle}, tour ${round}).`}
      />

      {pollsters.length > 0 && (
        <div className="table-scroll">
          <table className="polls">
            <thead>
              <tr>
                <th>Institut</th>
                <th className="num">Nb sondages</th>
                <th className="num">Échantillon moyen</th>
                <th>Période</th>
              </tr>
            </thead>
            <tbody>
              {pollsters.map((p) => (
                <tr key={p.pollster}>
                  <td className="pollster">
                    {p.website ? (
                      <a href={p.website} target="_blank" rel="noreferrer">
                        {p.pollster} ↗
                      </a>
                    ) : (
                      p.pollster
                    )}
                  </td>
                  <td className="num tnum">{p.n_polls}</td>
                  <td className="num tnum">
                    {p.avg_sample > 0 ? intFr(p.avg_sample) : "—"}
                  </td>
                  <td className="tnum">
                    {frDate(p.first_poll)} → {frDate(p.last_poll)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* (b) House effects --------------------------------------------- */}
      <div className="house-head">
        <h4>Biais maison</h4>
        <p className="muted">
          Écart moyen de chaque institut par rapport à la moyenne de tous les
          sondages (points).
        </p>
      </div>

      <AsyncState
        loading={he.loading}
        error={he.error}
        empty={!he.loading && groups.length === 0}
        emptyText="Pas assez de données pour estimer les biais maison."
      />

      {groups.length > 0 && (
        <div className="house-effects">
          {groups.map((g) => (
            <div className="he-group" key={g.pollster}>
              <div className="he-pollster">{g.pollster}</div>
              <div className="he-bars">
                {g.items.map((e) => {
                  const c = safeColor(e.color);
                  const frac = Math.min(Math.abs(e.delta) / maxAbs, 1);
                  const positive = e.delta >= 0;
                  return (
                    <div
                      className="he-row"
                      key={`${g.pollster}-${e.candidate}`}
                      title={`${e.candidate} : ${signed(e.delta)} pts (${e.n_polls} sond.)`}
                    >
                      <span className="he-cand">
                        <span className="dot" style={{ background: c }} />
                        {e.candidate}
                      </span>
                      <span className="he-track">
                        <span className="he-center" />
                        <span
                          className={`he-fill ${positive ? "pos" : "neg"}`}
                          style={{
                            width: `${frac * 50}%`,
                            background: c,
                          }}
                        />
                      </span>
                      <span
                        className={`he-val tnum ${positive ? "pos" : "neg"}`}
                      >
                        {signed(e.delta)}
                      </span>
                    </div>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      )}
      </ChartFrame>
    </div>
  );
}
