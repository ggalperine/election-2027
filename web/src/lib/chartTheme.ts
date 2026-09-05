// Shared Economist-style chart theme constants for recharts.
// Restrained, editorial: horizontal-only grid, right-side axis, thin rules.

/** Economist brand red — the single accent used across the dashboard. */
export const ECONOMIST_RED = "#E3120B";

/** Near-black ink and muted gray, matching the CSS tokens. */
export const INK = "#121212";
export const MUTED = "#595959";
export const AXIS_TICK = "#8a8a8a";

/** Horizontal gridline stroke (soft ecru-gray). */
export const GRID_STROKE = "#E4E4DE";

/** Zero baseline, slightly darker than the grid. */
export const BASELINE_STROKE = "#c7c7bf";

/**
 * Economist categorical palette — for NON-semantic series only
 * (e.g. the two forecast methods, scenario deltas). Candidate/party
 * colors stay semantic and must NOT be overridden by this.
 */
export const ECONOMIST_PALETTE = [
  "#006BA2",
  "#3EBCD2",
  "#EBB434",
  "#B4405F",
  "#E3120B",
  "#9A607F",
  "#F97A1F",
  "#6794A7",
  "#A2B627",
  "#01A2AC",
] as const;

/** Common axis tick style object (gray, small). */
export const axisTick = { fontSize: 11, fill: AXIS_TICK } as const;

/** Grid props: horizontal only, thin, no vertical lines. */
export const economistGrid = {
  stroke: GRID_STROKE,
  strokeWidth: 1,
  vertical: false,
} as const;

/**
 * Common X-axis props: subtle/absent axis line, no tick line, gray ticks.
 */
export const economistXAxis = {
  tick: axisTick,
  tickLine: false as const,
  axisLine: { stroke: GRID_STROKE } as const,
  minTickGap: 28,
};

/**
 * Common Y-axis props: RIGHT-oriented, no axis/tick line, gray ticks.
 */
export const economistYAxis = {
  orientation: "right" as const,
  tick: axisTick,
  tickLine: false as const,
  axisLine: false as const,
  width: 44,
};

/** Tooltip cursor style: thin dashed gray line. */
export const tooltipCursor = {
  stroke: "#c7c7bf",
  strokeDasharray: "3 3",
} as const;
