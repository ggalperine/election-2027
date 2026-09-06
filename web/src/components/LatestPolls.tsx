import { useMemo } from "react";
import { getPolls, getAggregates, type Poll } from "../lib/api";
import { useAsync } from "../lib/useAsync";
import { pct, intFr, frDate, safeColor } from "../lib/format";
import { ChartFrame } from "./ChartFrame";
import { AsyncState } from "./ui";

const NEUTRAL = "#94a3b8";
const LATEST_COUNT = 6;
const TOP_N = 3;

/** Top-N candidates (name, pct) inside a poll's results map, sorted desc. */
function topN(p: Poll, n: number): { cand: string; val: number }[] {
  return Object.entries(p.results)
    .map(([cand, val]) => ({ cand, val }))
    .sort((a, b) => b.val - a.val)
    .slice(0, n);
}

export function LatestPolls({ cycle, round }: { cycle: string; round: number }) {
  const { data, error, loading } = useAsync(
    () => getPolls(cycle, round, LATEST_COUNT),
    [cycle, round]
  );
  // Aggregate carries candidate → color; use it to tint the dots.
  const { data: agg } = useAsync(
    () => getAggregates(cycle, round),
    [cycle, round]
  );

  const colorOf = useMemo(() => {
    const map = new Map<string, string>();
    for (const a of agg ?? []) map.set(a.candidate, safeColor(a.color));
    return (cand: string) => map.get(cand) ?? NEUTRAL;
  }, [agg]);

  const polls = (data ?? []).slice(0, LATEST_COUNT);

  return (
    <ChartFrame
      title="Derniers sondages"
      subtitle="Les enquêtes les plus récentes, du plus récent au plus ancien."
      source="Source : Commission des sondages"
    >
      <AsyncState
        loading={loading}
        error={error}
        empty={!loading && polls.length === 0}
      />

      {polls.length > 0 && (
        <div className="latest-list">
          {polls.map((p) => (
            <article className="latest-row" key={p.external_id}>
              <div className="latest-meta">
                <div className="latest-pollster">
                  {p.pollster}
                  {p.sponsor && (
                    <span className="latest-sponsor"> · {p.sponsor}</span>
                  )}
                </div>
                <div className="latest-sub">
                  <span className="tnum">{frDate(p.field_end)}</span>
                  <span className="latest-dot-sep">·</span>
                  <span className="tnum">éch. {intFr(p.sample_size)}</span>
                </div>
              </div>

              <div className="latest-cands">
                {topN(p, TOP_N).map((c) => (
                  <span className="latest-chip" key={c.cand}>
                    <span
                      className="dot"
                      style={{ background: colorOf(c.cand) }}
                    />
                    <span className="latest-cand-name">{c.cand}</span>
                    <span className="latest-cand-val tnum">{pct(c.val)}</span>
                  </span>
                ))}
              </div>

              <div className="latest-src">
                {p.source_url ? (
                  <a href={p.source_url} target="_blank" rel="noopener">
                    source ↗
                  </a>
                ) : (
                  <span className="muted">—</span>
                )}
              </div>
            </article>
          ))}
        </div>
      )}
    </ChartFrame>
  );
}
