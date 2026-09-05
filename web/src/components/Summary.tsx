import { getSummary } from "../lib/api";
import { useAsync } from "../lib/useAsync";
import { pct, frDate, frDateTime, safeColor } from "../lib/format";
import { AsyncState } from "./ui";

export function Summary({
  cycle,
  round,
  window,
}: {
  cycle: string;
  round: number;
  window: number;
}) {
  const { data, error, loading } = useAsync(
    () => getSummary(cycle, round, window),
    [cycle, round, window]
  );

  const s = data;
  const empty = !loading && (!s || s.n_polls === 0);

  return (
    <div className="summary-strip">
      <AsyncState
        loading={loading}
        error={error}
        empty={empty}
        emptyText={`Aucun sondage pour ce tour (${cycle}, tour ${round}).`}
      />

      {s && s.n_polls > 0 && (
        <div className="stat-tiles">
          <div className="tile leader">
            <span
              className="accent-bar"
              style={{ background: safeColor(s.leader_color) }}
            />
            <div className="tile-label">Leader</div>
            <div className="tile-value">
              <span
                className="lead-name"
                style={{ color: safeColor(s.leader_color) }}
              >
                {s.leader}
              </span>{" "}
              <span className="tnum lead-pct">
                {pct(s.leader_pct)}
                <span className="unit">&nbsp;%</span>
              </span>
            </div>
          </div>

          <div className="tile">
            <div className="tile-label">Avance</div>
            <div className="tile-value tnum">
              +{pct(s.margin)} <span className="unit">pts</span>
            </div>
          </div>

          <div className="tile">
            <div className="tile-label">Sondages</div>
            <div className="tile-value tnum">{s.n_polls}</div>
          </div>

          <div className="tile">
            <div className="tile-label">Instituts</div>
            <div className="tile-value tnum">{s.n_pollsters}</div>
          </div>

          <div className="tile">
            <div className="tile-label">Dernier sondage</div>
            <div className="tile-value latest-poll-value">
              <span className="latest-poll-pollster">{s.latest_pollster}</span>{" "}
              <span className="arrow-sep">·</span>{" "}
              <span className="latest-poll-date tnum">{frDate(s.latest_poll)}</span>
            </div>
          </div>

          <div className="tile period">
            <div className="tile-label">Période couverte</div>
            <div className="tile-value period-value tnum">
              {frDate(s.first_poll)} <span className="arrow-sep">→</span>{" "}
              {frDate(s.last_poll)}
            </div>
          </div>
        </div>
      )}

      {s && s.n_polls > 0 && (
        <p className="summary-updated">
          Mis à jour le {frDateTime(s.last_updated)}
        </p>
      )}
    </div>
  );
}
