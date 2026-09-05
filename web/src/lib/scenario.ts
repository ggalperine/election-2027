// Pure, React-free scenario recomputation.
// Models a candidate WITHDRAWING and transferring a share of their votes to an
// ally; the rest of the vote pool is redistributed proportionally across the
// remaining field, which is then re-ranked to see who qualifies for round two.

export interface ScenarioRule {
  withdraw: string;
  beneficiary: string;
  rate: number; // 0..1 — share of the withdrawer's votes transferred to the ally
}

export interface Cand {
  candidate: string;
  avg_pct: number;
  color: string;
  party?: string;
}

export interface ScenarioResult extends Cand {
  qualified: boolean;
  delta: number; // new pct − original base pct (0 if not in base)
}

/** Case-insensitive / whitespace-insensitive key for matching candidate names. */
const norm = (s: string): string => s.trim().toLowerCase();

/**
 * Apply withdrawal + transfer rules to a base field and re-rank.
 *
 * For each rule the withdrawer leaves the field; `rate` of their votes go to the
 * named beneficiary, and the remaining `(1-rate)` share is pooled. The pool is
 * then distributed across ALL remaining candidates in proportion to their
 * post-transfer share (so the field renormalises to ~100 %).
 */
export function applyScenario(base: Cand[], rules: ScenarioRule[]): ScenarioResult[] {
  const basePct = new Map<string, number>();
  for (const c of base) basePct.set(norm(c.candidate), c.avg_pct);

  // Working copy of the field, keyed for lookup.
  const field = new Map<string, Cand>();
  for (const c of base) field.set(norm(c.candidate), { ...c });

  // Only keep rules whose withdrawer is actually present in the field.
  const active = rules.filter((r) => {
    const w = norm(r.withdraw);
    return w && field.has(w);
  });

  let pool = 0;

  for (const rule of active) {
    const wKey = norm(rule.withdraw);
    const withdrawer = field.get(wKey);
    if (!withdrawer) continue; // already removed by a prior rule
    field.delete(wKey);

    const rate = Math.max(0, Math.min(1, rule.rate));
    const transferred = rate * withdrawer.avg_pct;
    const leftover = (1 - rate) * withdrawer.avg_pct;

    const bKey = norm(rule.beneficiary);
    const beneficiary = field.get(bKey);
    if (beneficiary) {
      beneficiary.avg_pct += transferred;
      pool += leftover;
    } else {
      // No valid beneficiary in the remaining field: nothing is directly
      // transferred, so the whole withdrawer share becomes redistributable.
      pool += withdrawer.avg_pct;
    }
  }

  const remaining = [...field.values()];

  // Distribute the pool proportionally to each remaining candidate's
  // post-transfer share.
  const total = remaining.reduce((s, c) => s + c.avg_pct, 0);
  if (pool > 0 && total > 0) {
    for (const c of remaining) {
      c.avg_pct += pool * (c.avg_pct / total);
    }
  }

  const results: ScenarioResult[] = remaining
    .map((c) => ({
      ...c,
      qualified: false,
      delta: c.avg_pct - (basePct.get(norm(c.candidate)) ?? c.avg_pct),
    }))
    .sort((a, b) => b.avg_pct - a.avg_pct);

  for (let i = 0; i < results.length && i < 2; i++) {
    results[i].qualified = true;
  }

  return results;
}
