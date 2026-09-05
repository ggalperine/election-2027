// Typed API client for the Go gateway.
// In dev, Vite proxies /api → http://localhost:8080. In prod, same-origin.
const BASE = import.meta.env.VITE_API_BASE ?? "";

/* ------------------------------------------------------------------ */
/* Types                                                               */
/* ------------------------------------------------------------------ */

export interface Cycle {
  cycle: string;
  has_actual: boolean;
}

/** A weighted poll-of-polls point (used for both /aggregates and /timeseries). */
export interface AggregatePoint {
  as_of: string;
  candidate: string;
  party: string;
  color: string;
  avg_pct: number;
  lo: number; // 95% CI lower bound
  hi: number; // 95% CI upper bound
  n_polls: number;
  n_eff: number; // effective sample size
}

export type ForecastMethod = "holt" | "linreg";

export interface ForecastPoint {
  method: ForecastMethod;
  date: string;
  candidate: string;
  color: string;
  avg_pct: number;
  lo: number;
  hi: number;
  projected: boolean; // false = fitted history, true = future projection
}

export interface Poll {
  external_id: string;
  pollster: string;
  sponsor?: string;
  field_end: string;
  sample_size: number;
  round: number;
  source_url: string;
  results: Record<string, number>; // candidate → pct
}

export interface ActualResult {
  candidate: string;
  color: string;
  pct: number;
  won: boolean;
}

export interface GeoWinner {
  geo_code: string;
  geo_name: string;
  winner: string;
  color: string;
  pct: number;
}

/** Top-of-page overview for a cycle+round. */
export interface Summary {
  cycle: string;
  round: number;
  n_polls: number;
  n_pollsters: number;
  first_poll: string;
  last_poll: string;
  leader: string;
  leader_pct: number;
  leader_color: string;
  margin: number; // leader − runner-up (percentage points)
}

/** One institut's activity for a cycle+round. */
export interface PollsterStat {
  pollster: string;
  website: string;
  n_polls: number;
  avg_sample: number;
  first_poll: string;
  last_poll: string;
}

/** An institut's signed deviation from consensus for a candidate. */
export interface HouseEffect {
  pollster: string;
  candidate: string;
  color: string;
  delta: number; // institut mean − consensus mean (percentage points)
  n_polls: number;
}

/* ------------------------------------------------------------------ */
/* Fetch helper                                                        */
/* ------------------------------------------------------------------ */

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`);
  if (!res.ok) throw new Error(`API ${path}: ${res.status}`);
  return res.json() as Promise<T>;
}

/* ------------------------------------------------------------------ */
/* Endpoints                                                           */
/* ------------------------------------------------------------------ */

export const getCycles = () => get<Cycle[]>("/api/cycles");

/** window = days back from the latest poll; 0 = since the beginning; undefined = server default. */
const win = (w?: number) => (w === undefined ? "" : `&window=${w}`);

export const getAggregates = (cycle: string, round = 1, window?: number) =>
  get<AggregatePoint[]>(`/api/aggregates?cycle=${cycle}&round=${round}${win(window)}`);

export const getTimeSeries = (cycle: string, round = 1, window?: number) =>
  get<AggregatePoint[]>(`/api/timeseries?cycle=${cycle}&round=${round}${win(window)}`);

export const getForecast = (cycle: string, round = 1, window?: number) =>
  get<ForecastPoint[]>(`/api/forecast?cycle=${cycle}&round=${round}${win(window)}`);

export const getSummary = (cycle: string, round = 1, window?: number) =>
  get<Summary>(`/api/summary?cycle=${cycle}&round=${round}${win(window)}`);

export const getPollsters = (cycle: string, round = 1) =>
  get<PollsterStat[]>(`/api/pollsters?cycle=${cycle}&round=${round}`);

export const getHouseEffects = (cycle: string, round = 1) =>
  get<HouseEffect[]>(`/api/houseeffects?cycle=${cycle}&round=${round}`);

export const getPolls = (cycle: string, round = 1, limit = 100) =>
  get<Poll[]>(`/api/polls?cycle=${cycle}&round=${round}&limit=${limit}`);

export const getActual = (cycle: string, round = 1) =>
  get<ActualResult[]>(`/api/actual?cycle=${cycle}&round=${round}`);

export const getMap = (
  election = "presidentielle-2022-t1",
  level = "departement"
) => get<GeoWinner[]>(`/api/map?election=${election}&level=${level}`);
