import type { ReactNode } from "react";

/**
 * Economist-style chart frame: red brand tag, left-aligned bold title,
 * lighter subtitle, the chart itself, then a thin rule + gray source footer.
 * Every chart card wraps its content in this for a consistent editorial look.
 */
export function ChartFrame({
  title,
  subtitle,
  source,
  actions,
  children,
}: {
  title: string;
  subtitle?: string;
  source?: string;
  /** Optional right-aligned controls (e.g. a Segmented toggle) beside the title. */
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="chart-frame">
      <div className="chart-frame-head">
        <div className="chart-frame-titles">
          <span className="red-tag" aria-hidden="true" />
          <h3 className="chart-title">{title}</h3>
          {subtitle && <p className="chart-subtitle">{subtitle}</p>}
        </div>
        {actions && <div className="chart-frame-actions">{actions}</div>}
      </div>

      <div className="chart-frame-body">{children}</div>

      {source && (
        <div className="chart-source-wrap">
          <p className="chart-source">{source}</p>
        </div>
      )}
    </div>
  );
}
