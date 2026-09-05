import type { ReactNode } from "react";

/* Section wrapper with a kicker + heading + optional sub-line. */
export function Section({
  kicker,
  title,
  sub,
  children,
}: {
  kicker: string;
  title: string;
  sub?: string;
  children: ReactNode;
}) {
  return (
    <section className="section">
      <div className="section-head">
        <p className="kicker">{kicker}</p>
        <h2>{title}</h2>
        {sub && <p className="sub">{sub}</p>}
      </div>
      {children}
    </section>
  );
}

/* Segmented (pill) toggle. */
export function Segmented<T extends string>({
  value,
  options,
  onChange,
}: {
  value: T;
  options: { value: T; label: string; disabled?: boolean }[];
  onChange: (v: T) => void;
}) {
  return (
    <div className="segmented" role="tablist">
      {options.map((o) => (
        <button
          key={o.value}
          role="tab"
          aria-selected={o.value === value}
          disabled={o.disabled}
          className={o.value === value ? "active" : ""}
          onClick={() => onChange(o.value)}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}

/* Loading / error / empty placeholder inside a card. */
export function AsyncState({
  loading,
  error,
  empty,
  emptyText = "Aucune donnée disponible.",
}: {
  loading: boolean;
  error?: string;
  empty?: boolean;
  emptyText?: string;
}) {
  if (loading) return <div className="state">Chargement…</div>;
  if (error) return <div className="state error">Erreur : {error}</div>;
  if (empty) return <div className="state">{emptyText}</div>;
  return null;
}
