// Shared Economist-style chart theme constants for recharts.
// Restrained, editorial: horizontal-only grid, right-side axis, thin rules.

/** Economist brand red — the single accent used across the dashboard. */
export const ECONOMIST_RED = "#E3120B";

/**
 * Ink / muted / grid colors reference the CSS custom properties so recharts
 * SVG elements auto-switch between light and dark via the
 * `prefers-color-scheme` media query in index.css. SVG stroke/fill accept
 * CSS var() values, so no JS toggle or re-render is needed.
 */
export const INK = "var(--ink)";
export const MUTED = "var(--muted)";
export const AXIS_TICK = "var(--muted)";

/** Horizontal gridline stroke (soft ecru-gray in light, faint gray in dark). */
export const GRID_STROKE = "var(--grid)";

/** Zero baseline, slightly stronger than the grid. */
export const BASELINE_STROKE = "var(--border-strong)";

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
  axisLine: { stroke: "var(--border)" } as const,
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
  stroke: "var(--border-strong)",
  strokeDasharray: "3 3",
} as const;

/**
 * Tooltip contentStyle for any recharts <Tooltip> that does NOT render a
 * custom `content` element. Uses tokens so it adapts to dark mode.
 * (The dashboard's charts mostly use custom `.rc-tooltip` content, which is
 * themed in index.css, but this is exported for completeness/reuse.)
 */
export const tooltipContentStyle = {
  background: "var(--surface)",
  border: "1px solid var(--border)",
  borderRadius: 4,
  color: "var(--ink)",
} as const;
