import { useMemo, useState } from "react";
import { getAggregates } from "../lib/api";
import { useAsync } from "../lib/useAsync";
import { pct, safeColor } from "../lib/format";
import { applyScenario, type Cand, type ScenarioRule, type ScenarioResult } from "../lib/scenario";
import { AsyncState } from "./ui";
import { ChartFrame } from "./ChartFrame";

/** Find a candidate in the field by case-insensitive / trimmed name. */
function has(field: Cand[], name: string): boolean {
  const n = name.trim().toLowerCase();
  return field.some((c) => c.candidate.trim().toLowerCase() === n);
}

/** Signed, French-formatted delta chip text: +1,2 / −0,8 / — */
function deltaText(d: number): string {
  if (Math.abs(d) < 0.05) return "—";
  const sign = d > 0 ? "+" : "−";
  return `${sign}${pct(Math.abs(d))}`;
}

function DeltaChip({ d }: { d: number }) {
  const cls = Math.abs(d) < 0.05 ? "flat" : d > 0 ? "pos" : "neg";
  return <span className={`sc-delta ${cls} tnum`}>{deltaText(d)}</span>;
}

/** A compact recomputed leaderboard: bars over a faint ghost base bar. */
function ScenarioBars({
  results,
  base,
  limit,
}: {
  results: ScenarioResult[];
  base: Cand[];
  limit?: number;
}) {
  const baseByKey = new Map(base.map((c) => [c.candidate.trim().toLowerCase(), c.avg_pct]));
  const shown = limit ? results.slice(0, limit) : results;
  const max = Math.max(1, ...results.map((r) => r.avg_pct), ...base.map((c) => c.avg_pct));

  return (
    <div className="sc-bars">
      {shown.map((r) => {
        const c = safeColor(r.color);
        const ghost = baseByKey.get(r.candidate.trim().toLowerCase()) ?? 0;
        return (
          <div className="sc-row" key={r.candidate}>
            <div className="sc-name">
              <span className="dot" style={{ background: c }} />
              <span className="sc-cand">{r.candidate}</span>
              {r.qualified && (
                <span className="sc-badge" title="Qualifié pour le 2nd tour">
                  ✓ Qualifié 2nd tour
                </span>
              )}
            </div>
            <div className="sc-track">
              {ghost > 0 && (
                <span
                  className="sc-ghost"
                  style={{ width: `${(ghost / max) * 100}%`, background: c }}
                />
              )}
              <span
                className="sc-fill"
                style={{ width: `${(r.avg_pct / max) * 100}%`, background: c }}
              />
            </div>
            <span className="sc-val tnum">{pct(r.avg_pct)}%</span>
            <DeltaChip d={r.delta} />
          </div>
        );
      })}
    </div>
  );
}

export function ScenarioLab({
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

  const [rate, setRate] = useState(65); // percent; shared across all presets/builder

  // Custom builder state.
  const [w1, setW1] = useState("");
  const [b1, setB1] = useState("");
  const [w2, setW2] = useState("");
  const [b2, setB2] = useState("");

  const field: Cand[] = useMemo(
    () =>
      (data ?? []).map((r) => ({
        candidate: r.candidate,
        avg_pct: r.avg_pct,
        color: r.color,
        party: r.party,
      })),
    [data]
  );

  const r01 = rate / 100;

  // Preset comparison for the central bloc.
  const attalRun = useMemo(
    () => applyScenario(field, [{ withdraw: "Philippe", beneficiary: "Attal", rate: r01 }]),
    [field, r01]
  );
  const philippeRun = useMemo(
    () => applyScenario(field, [{ withdraw: "Attal", beneficiary: "Philippe", rate: r01 }]),
    [field, r01]
  );

  // Custom builder rules.
  const customRules: ScenarioRule[] = useMemo(() => {
    const rules: ScenarioRule[] = [];
    if (w1 && b1 && w1 !== b1) rules.push({ withdraw: w1, beneficiary: b1, rate: r01 });
    if (w2 && b2 && w2 !== b2 && w2 !== w1) rules.push({ withdraw: w2, beneficiary: b2, rate: r01 });
    return rules;
  }, [w1, b1, w2, b2, r01]);

  const customRun = useMemo(
    () => applyScenario(field, customRules),
    [field, customRules]
  );

  if (round === 2) {
    return (
      <div className="card">
        <p className="state">Les scénarios de report s'appliquent au 1er tour.</p>
      </div>
    );
  }

  const centralBloc = has(field, "Attal") || has(field, "Philippe");
  const withdrawn = new Set([w1, w2].filter(Boolean));

  const rateSlider = (
    <div className="sc-slider">
      <label htmlFor="sc-rate">
        Report des voix vers l'allié{" "}
        <strong className="tnum">{rate} %</strong>
      </label>
      <input
        id="sc-rate"
        type="range"
        min={0}
        max={100}
        step={1}
        value={rate}
        onChange={(e) => setRate(Number(e.target.value))}
      />
    </div>
  );

  return (
    <div className="card">
      <ChartFrame
        title="Report des voix — qui se qualifie ?"
        subtitle="Retirez un candidat du champ et transférez ses voix à un allié, puis observez le nouveau classement du 1er tour."
        source="Source : nsppolls ; Wikipédia — Sondages agrégés, moyenne pondérée ; hypothèses de report"
      >

      <AsyncState
        loading={loading}
        error={error}
        empty={!loading && field.length === 0}
        emptyText={`Pas d'agrégat pour ce tour (${cycle}, tour ${round}).`}
      />

      {field.length > 0 && (
        <>
          {rateSlider}

          {centralBloc ? (
            <div className="sc-presets">
              <div className="sc-preset">
                <div className="sc-preset-head">Bloc central : Attal candidat</div>
                <div className="sc-preset-sub muted">Philippe se retire → Attal</div>
                <ScenarioBars results={attalRun} base={field} limit={5} />
              </div>
              <div className="sc-preset">
                <div className="sc-preset-head">Bloc central : Philippe candidat</div>
                <div className="sc-preset-sub muted">Attal se retire → Philippe</div>
                <ScenarioBars results={philippeRun} base={field} limit={5} />
              </div>
            </div>
          ) : (
            <p className="state">
              Scénario du bloc central indisponible pour ce cycle.
            </p>
          )}

          <div className="sc-builder">
            <div className="house-head" style={{ marginTop: 28 }}>
              <h4>Scénario personnalisé</h4>
              <p className="muted">
                Choisissez jusqu'à deux retraits et le bénéficiaire de chaque report.
              </p>
            </div>

            <div className="sc-rules">
              <ScenarioRuleRow
                idx={1}
                field={field}
                withdrawn={withdrawn}
                withdraw={w1}
                beneficiary={b1}
                onWithdraw={setW1}
                onBeneficiary={setB1}
              />
              <ScenarioRuleRow
                idx={2}
                field={field}
                withdrawn={withdrawn}
                withdraw={w2}
                beneficiary={b2}
                onWithdraw={setW2}
                onBeneficiary={setB2}
              />
            </div>

            <ScenarioBars results={customRun} base={field} />
          </div>

          <p className="muted" style={{ fontSize: 12.5, marginTop: 18 }}>
            Hypothèses de report — les transferts réels de voix sont incertains.
          </p>
        </>
      )}
      </ChartFrame>
    </div>
  );
}

function ScenarioRuleRow({
  idx,
  field,
  withdrawn,
  withdraw,
  beneficiary,
  onWithdraw,
  onBeneficiary,
}: {
  idx: number;
  field: Cand[];
  withdrawn: Set<string>;
  withdraw: string;
  beneficiary: string;
  onWithdraw: (v: string) => void;
  onBeneficiary: (v: string) => void;
}) {
  const beneficiaries = field.filter(
    (c) => c.candidate !== withdraw && !withdrawn.has(c.candidate)
  );
  return (
    <div className="sc-rule">
      <label>
        <span className="sc-rule-lbl">Candidat qui se retire {idx}</span>
        <select value={withdraw} onChange={(e) => onWithdraw(e.target.value)}>
          <option value="">— aucun —</option>
          {field.map((c) => (
            <option key={c.candidate} value={c.candidate}>
              {c.candidate}
            </option>
          ))}
        </select>
      </label>
      <label>
        <span className="sc-rule-lbl">Bénéficiaire</span>
        <select
          value={beneficiary}
          onChange={(e) => onBeneficiary(e.target.value)}
          disabled={!withdraw}
        >
          <option value="">— aucun —</option>
          {beneficiaries.map((c) => (
            <option key={c.candidate} value={c.candidate}>
              {c.candidate}
            </option>
          ))}
        </select>
      </label>
    </div>
  );
}
