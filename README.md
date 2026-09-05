# Présidentielle 2027 — Poll Aggregator & Results Map

Go microservices + RabbitMQ + Postgres + React. Aggregates French 2027
presidential polls (rolling averages + charts) and maps official results by
department from open data (elections.interieur.gouv.fr / data.gouv.fr).

## Architecture

```
poll-ingester ───fetch polls────┐
results-ingester ─fetch gov CSV─┐│
                                ▼▼
scheduler (CRON 24h) ─triggers─► RabbitMQ (topic: election.events) ─► aggregator ─► Postgres
                                                                                        ▲
web (React/Vite) ◄── REST ── gateway ───────────────────────────────────────────────┘
```

| Service            | Role                                                                 |
|--------------------|---------------------------------------------------------------------|
| `gateway`          | REST API for the frontend (`/api/aggregates`, `/api/timeseries`, `/api/map`) |
| `aggregator`       | Consumes `poll.raw` + `geo.result`, persists, recomputes averages   |
| `poll-ingester`    | Fetches polls, publishes `poll.raw`                                  |
| `results-ingester` | Fetches official gov results, publishes `geo.result`                 |
| `scheduler`        | CRON (default daily 06:00), publishes `cmd.fetch.*`                  |
| `web`              | React UI: bar chart, trend chart, France choropleth map             |

Message flow uses a single RabbitMQ **topic exchange** `election.events`:
- `poll.raw` — one poll → aggregator
- `geo.result` — one geo result row → aggregator
- `cmd.fetch.polls` / `cmd.fetch.geo` — scheduler → ingesters

## Run everything

```bash
docker compose up --build
```

- Web UI:            http://localhost:5173
- Gateway API:       http://localhost:8080/api/aggregates
- RabbitMQ console:  http://localhost:15672  (election / election)
- Postgres:          localhost:5432          (election / election)

By default the ingesters emit **sample data** so the whole pipeline runs with no
external dependencies — charts and map populate immediately.

## Wire real data

No single free API exists for French polls; use a community CSV or a scraper.

```bash
# Poll feed (CSV with header: external_id,pollster,field_end,sample_size,round,source_url,<candidate cols>)
export POLL_CSV_URL="https://.../nsppolls.csv"

# Official results export from data.gouv.fr / elections.interieur.gouv.fr
# CSV header: election,geo_level,geo_code,geo_name,registered,votes_cast,candidate,votes,pct
export GOV_CSV_URL="https://.../resultats-presidentielle.csv"
```

Set these in `docker-compose.yml` under the ingester services, then the sample
sources are bypassed. Adapt the parsers in `internal/poll/source.go` and
`internal/results/source.go` to the real column layout.

## CRON

`scheduler` runs `robfig/cron`. Override the schedule:

```yaml
scheduler:
  environment:
    CRON_SPEC: "0 6 * * *"   # every 24h at 06:00
```

## Local dev (without Docker)

Backend needs Go 1.23+. Start Postgres + RabbitMQ (via compose), then:

```bash
go mod tidy
go run ./cmd/gateway
go run ./cmd/aggregator
go run ./cmd/poll-ingester
go run ./cmd/results-ingester
go run ./cmd/scheduler
```

Frontend:

```bash
cd web
npm install
npm run dev      # proxies /api → localhost:8080
```

## API

| Endpoint                        | Returns                                             |
|---------------------------------|-----------------------------------------------------|
| `GET /api/aggregates?round=1`   | Latest snapshot: avg % per candidate                |
| `GET /api/timeseries?round=1`   | 30-day rolling average time series per candidate     |
| `GET /api/map?election=&level=` | Winner + color per department (choropleth)          |
| `GET /health`                   | Health check                                        |

## Data model

See `migrations/001_init.sql`: `pollsters`, `candidates`, `polls`,
`poll_results`, `aggregates`, `geo_results`.

## Notes / TODO

- Sample candidate list is editable in the migration and `internal/poll/sample.go`.
- Second round (`round=2`) supported in schema/API; add a source that provides it.
- Map GeoJSON pulled at runtime from the public `france-geojson` repo (dept codes).
- Add auth + rate limiting on the gateway before public deployment.
```
