-- Fetch-pipeline heartbeat: one row per cron/startup run of the scheduler.
-- Powers the API's "Dernière mise à jour" (last cron), which must reflect the
-- last data refresh attempt, not the newest poll's insert time.
CREATE TABLE IF NOT EXISTS ingest_runs (
    id     BIGSERIAL PRIMARY KEY,
    kind   TEXT NOT NULL,          -- 'cron' | 'startup'
    ran_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
