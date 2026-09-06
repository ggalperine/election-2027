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
import { getForecast, type ForecastMethod, type ForecastPoint } from "../lib/api";
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

const METHOD_LABEL: Record<ForecastMethod, string> = {
  holt: "Lissage exponentiel (Holt)",
  linreg: "Régression linéaire pondérée",
};

function pivotForecast(points: ForecastPoint[], method: ForecastMethod) {
  const byDate = new Map<string, Record<string, number | string | [number, number]>>();
  const colors = new Map<string, string>();
  let splitDate: string | undefined;

  for (const p of points) {
    if (p.method !== method) continue;
    colors.set(p.candidate, safeColor(p.color));
    const row = byDate.get(p.date) ?? { date: p.date };
    row[p.candidate] = p.avg_pct;
    if (p.projected) {
      row[`${p.candidate}__band`] = [p.lo, p.hi];
      if (!splitDate || p.date < splitDate) splitDate = p.date;
    }
    byDate.set(p.date, row);
  }

  const rows = [...byDate.values()].sort((a, b) =>
    String(a.date).localeCompare(String(b.date))
  );
  const series = [...colors.entries()]
    .map(([candidate, color]) => ({ candidate, color }))
    .sort((a, b) => {
      const la = rows[rows.length - 1]?.[a.candidate];
      const lb = rows[rows.length - 1]?.[b.candidate];
      return (typeof lb === "number" ? lb : 0) - (typeof la === "number" ? la : 0);
    });

  return { rows, series, splitDate };
}

function FcTooltip({ active, payload, label, splitDate }: any) {
  if (!active || !payload?.length) return null;
  const lines = payload
    .filter((p: any) => !p.dataKey?.endsWith("__band") && typeof p.value === "number")
    .sort((a: any, b: any) => b.value - a.value);
  const projected = splitDate && String(label) >= splitDate;
  return (
    <div className="rc-tooltip">
      <div className="tt-title">
        {frDate(String(label))}
        {projected && (
          <span className="muted" style={{ fontWeight: 500 }}> · projection</span>
        )}
      </div>
      {lines.map((p: any) => (
        <div className="tt-row" key={p.dataKey}>
          <span className="k">
            <span className="dot" style={{ background: p.color }} />
            {p.dataKey}
          </span>
          <span className="v tnum">{pct(p.value)} %</span>
        </div>
      ))}
    </div>
  );
}

export function Forecast({
  cycle,
  round,
  window,
}: {
  cycle: string;
  round: number;
  window: number;
}) {
  const { data, error, loading } = useAsync(
    () => getForecast(cycle, round, window),
    [cycle, round, window]
  );
  const [method, setMethod] = useState<ForecastMethod>("holt");
  const [hidden, setHidden] = useState<Set<string>>(new Set());

  const { rows, series, splitDate } = useMemo(
    () => pivotForecast(data ?? [], method),
    [data, method]
  );

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
        title={`Projection à l'horizon du scrutin — ${METHOD_LABEL[method]}`}
        subtitle="Zone ombrée = incertitude croissante · à droite de la ligne verticale = futur estimé"
        source="Source : Commission des sondages — Sondages agrégés, moyenne pondérée"
        actions={
          <Segmented<ForecastMethod>
            value={method}
            onChange={setMethod}
            options={[
              { value: "holt", label: "Holt" },
              { value: "linreg", label: "Régression" },
            ]}
          />
        }
      >

      <AsyncState
        loading={loading}
        error={error}
        empty={!loading && rows.length === 0}
        emptyText={`Pas de projection pour ce tour (${cycle}, tour ${round}).`}
      />

      {rows.length > 0 && (
        <>
          <ResponsiveContainer width="100%" height={430}>
            <ComposedChart data={rows} margin={{ left: 4, right: 8, top: 8 }}>
              <CartesianGrid {...economistGrid} />
              <XAxis
                {...economistXAxis}
                dataKey="date"
                tickFormatter={(v) => frMonth(String(v))}
              />
              <YAxis {...economistYAxis} unit=" %" />
              <ReferenceLine y={0} stroke={BASELINE_STROKE} strokeWidth={1} />
              <Tooltip
                content={<FcTooltip splitDate={splitDate} />}
                cursor={tooltipCursor}
              />

              {visible.map((s) => (
                <Area
                  key={`${s.candidate}__band`}
                  dataKey={`${s.candidate}__band`}
                  stroke="none"
                  fill={withAlpha(s.color, 0.16)}
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

              {splitDate && (
                <ReferenceLine
                  x={splitDate}
                  stroke="var(--border-strong)"
                  strokeDasharray="5 4"
                  label={{
                    value: "aujourd'hui",
                    position: "insideTopRight",
                    fontSize: 11,
                    fill: "var(--muted)",
                  }}
                />
              )}
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
        </>
      )}
      </ChartFrame>
    </div>
  );
}
