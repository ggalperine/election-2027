import { useMemo, useState } from "react";
import {
  Area,
  CartesianGrid,
  ComposedChart,
  Line,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { getTimeSeries, getActual, type ActualResult } from "../lib/api";
import { useAsync } from "../lib/useAsync";
import { pct, frDate, frMonth, safeColor, withAlpha } from "../lib/format";
import { AsyncState, Segmented } from "./ui";
import { ChartFrame } from "./ChartFrame";
import {
  economistGrid,
  economistXAxis,
  economistYAxis,
  tooltipCursor,
  BASELINE_STROKE,
} from "../lib/chartTheme";

interface Series {
  candidate: string;
  color: string;
}

/** Pivot flat rows into { as_of, "<cand>": pct, "<cand>__band": [lo,hi] } */
function pivot(
  points: { as_of: string; candidate: string; color: string; avg_pct: number; lo: number; hi: number }[]
) {
  const byDate = new Map<string, Record<string, number | string | [number, number]>>();
  const seriesMap = new Map<string, Series>();
  let latestPct: Record<string, number> = {};

  for (const p of points) {
    if (!seriesMap.has(p.candidate))
      seriesMap.set(p.candidate, { candidate: p.candidate, color: safeColor(p.color) });
    const row = byDate.get(p.as_of) ?? { as_of: p.as_of };
    row[p.candidate] = p.avg_pct;
    row[`${p.candidate}__band`] = [p.lo, p.hi];
    byDate.set(p.as_of, row);
  }

  const rows = [...byDate.values()].sort((a, b) =>
    String(a.as_of).localeCompare(String(b.as_of))
  );

  // rank series by their latest value for legend / z-order
  const last = rows[rows.length - 1] ?? {};
  for (const [cand] of seriesMap) {
    const v = last[cand];
    if (typeof v === "number") latestPct[cand] = v;
  }
  const series = [...seriesMap.values()].sort(
    (a, b) => (latestPct[b.candidate] ?? 0) - (latestPct[a.candidate] ?? 0)
  );

  return { rows, series };
}

function TrendTooltip({
  active,
  payload,
  label,
  actual,
}: any) {
  if (!active || !payload?.length) return null;
  const lines = payload
    .filter((p: any) => !p.dataKey?.endsWith("__band") && typeof p.value === "number")
    .sort((a: any, b: any) => b.value - a.value);
  return (
    <div className="rc-tooltip">
      <div className="tt-title">{frDate(String(label))}</div>
      {lines.map((p: any) => (
        <div className="tt-row" key={p.dataKey}>
          <span className="k">
            <span className="dot" style={{ background: p.color }} />
            {p.dataKey}
          </span>
          <span className="v tnum">{pct(p.value)} %</span>
        </div>
      ))}
      {actual && actual.length > 0 && (
        <div className="tt-ci" style={{ marginTop: 6 }}>
          Ligne pointillée = résultat réel
        </div>
      )}
    </div>
  );
}

export function TrendTracker({
  cycle,
  round,
  hasActual,
  window,
}: {
  cycle: string;
  round: number;
  hasActual: boolean;
  window: number;
}) {
  const ts = useAsync(
    () => getTimeSeries(cycle, round, window),
    [cycle, round, window]
  );
  const act = useAsync<ActualResult[]>(
    () => (hasActual ? getActual(cycle, round) : Promise.resolve([])),
    [cycle, round, hasActual]
  );

  const [showBands, setShowBands] = useState(false);
  const [hidden, setHidden] = useState<Set<string>>(new Set());

  const { rows, series } = useMemo(() => pivot(ts.data ?? []), [ts.data]);

  const actualByLastName = useMemo(() => {
    const m = new Map<string, ActualResult>();
    for (const a of act.data ?? []) {
      const last = a.candidate.split(" ").slice(-1)[0];
      m.set(last, a);
      m.set(a.candidate, a);
    }
    return m;
  }, [act.data]);

  const toggle = (c: string) =>
    setHidden((prev) => {
      const n = new Set(prev);
      n.has(c) ? n.delete(c) : n.add(c);
      return n;
    });

  const visible = series.filter((s) => !hidden.has(s.candidate));

  return (
    <div className="card">
      <ChartFrame
        title="Évolution des intentions de vote"
        subtitle="Moyenne glissante pondérée par candidat · bande = intervalle de confiance à 95 %"
        source="Source : nsppolls ; Wikipédia — Sondages agrégés, moyenne pondérée"
        actions={
          <Segmented<"line" | "band">
            value={showBands ? "band" : "line"}
            onChange={(v) => setShowBands(v === "band")}
            options={[
              { value: "line", label: "Lignes" },
              { value: "band", label: "+ IC 95 %" },
            ]}
          />
        }
      >

      <AsyncState
        loading={ts.loading}
        error={ts.error}
        empty={!ts.loading && rows.length === 0}
        emptyText={`Pas d'historique pour ce tour (${cycle}, tour ${round}).`}
      />

      {rows.length > 0 && (
        <>
          <ResponsiveContainer width="100%" height={430}>
            <ComposedChart data={rows} margin={{ left: 4, right: 8, top: 8 }}>
              <CartesianGrid {...economistGrid} />
              <XAxis
                {...economistXAxis}
                dataKey="as_of"
                tickFormatter={(v) => frMonth(String(v))}
              />
              <YAxis {...economistYAxis} unit=" %" />
              <ReferenceLine y={0} stroke={BASELINE_STROKE} strokeWidth={1} />
              <Tooltip
                content={<TrendTooltip actual={act.data} />}
                cursor={tooltipCursor}
              />

              {showBands &&
                visible.map((s) => (
                  <Area
                    key={`${s.candidate}__band`}
                    dataKey={`${s.candidate}__band`}
                    stroke="none"
                    fill={withAlpha(s.color, 0.14)}
                    isAnimationActive={false}
                    activeDot={false}
                    connectNulls
                  />
                ))}

              {visible.map((s) => (
                <Line
                  key={s.candidate}
                  dataKey={s.candidate}
                  type="monotone"
                  stroke={s.color}
                  strokeWidth={2.25}
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  dot={false}
                  activeDot={{ r: 4, strokeWidth: 0 }}
                  isAnimationActive={false}
                  connectNulls
                />
              ))}

              {/* actual final result reference lines */}
              {visible.map((s) => {
                const a =
                  actualByLastName.get(s.candidate) ??
                  actualByLastName.get(s.candidate.split(" ").slice(-1)[0]);
                if (!a) return null;
                return (
                  <ReferenceLine
                    key={`ref-${s.candidate}`}
                    y={a.pct}
                    stroke={s.color}
                    strokeDasharray="4 4"
                    strokeWidth={1.25}
                    ifOverflow="extendDomain"
                  />
                );
              })}
            </ComposedChart>
          </ResponsiveContainer>

          <div className="legend">
            {series.map((s) => (
              <span
                key={s.candidate}
                className={`item toggle ${hidden.has(s.candidate) ? "off" : ""}`}
                onClick={() => toggle(s.candidate)}
                title="Afficher / masquer"
              >
                <i className="swatch" style={{ background: s.color }} />
                {s.candidate}
              </span>
            ))}
          </div>
          {hasActual && (act.data?.length ?? 0) > 0 && (
            <p className="muted" style={{ fontSize: 12.5, marginTop: 10 }}>
              Les lignes pointillées horizontales indiquent le résultat officiel
              final de chaque candidat.
            </p>
          )}
        </>
      )}
      </ChartFrame>
    </div>
  );
}
