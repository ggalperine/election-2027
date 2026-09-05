// Package store is the Postgres data-access layer (pgx pool).
package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ggalperine/election2027/internal/models"
)

type Store struct{ pool *pgxpool.Pool }

func New(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	for i := 0; i < 30; i++ {
		if err = pool.Ping(ctx); err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	return &Store{pool: pool}, err
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) upsertPollster(ctx context.Context, name string) (int, error) {
	var id int
	err := s.pool.QueryRow(ctx, `
		INSERT INTO pollsters(name) VALUES($1)
		ON CONFLICT(name) DO UPDATE SET name=EXCLUDED.name
		RETURNING id`, name).Scan(&id)
	return id, err
}

// candidateID returns candidate id, creating/updating party + party colour.
func (s *Store) candidateID(ctx context.Context, name, party string) (int, error) {
	var id int
	err := s.pool.QueryRow(ctx, `
		INSERT INTO candidates(name, party, color)
		VALUES($1, NULLIF($2,''), COALESCE((SELECT color FROM parties WHERE name=$2), '#888888'))
		ON CONFLICT(name) DO UPDATE SET
			party = COALESCE(NULLIF($2,''), candidates.party),
			color = COALESCE((SELECT color FROM parties WHERE name=$2), candidates.color)
		RETURNING id`, name, party).Scan(&id)
	return id, err
}

// SavePoll persists a raw poll + its per-candidate results idempotently.
func (s *Store) SavePoll(ctx context.Context, p models.RawPoll) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	pollsterID, err := s.upsertPollster(ctx, p.Pollster)
	if err != nil {
		return err
	}

	cycle := p.Cycle
	if cycle == "" {
		cycle = "2027"
	}

	var pollID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO polls(external_id, cycle, pollster_id, sponsor, field_start, field_end, sample_size, round, source_url)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT(external_id) DO UPDATE SET sample_size=EXCLUDED.sample_size, sponsor=EXCLUDED.sponsor
		RETURNING id`,
		p.ExternalID, cycle, pollsterID, p.Sponsor, p.FieldStart, p.FieldEnd, p.SampleSize, p.Round, p.SourceURL).Scan(&pollID)
	if err != nil {
		return err
	}

	for cand, pct := range p.Results {
		cid, err := s.candidateID(ctx, cand, p.Parties[cand])
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO poll_results(poll_id, candidate_id, pct)
			VALUES($1,$2,$3)
			ON CONFLICT(poll_id, candidate_id) DO UPDATE SET pct=EXCLUDED.pct`,
			pollID, cid, pct); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// SaveGeoResult persists one official geo result row idempotently.
func (s *Store) SaveGeoResult(ctx context.Context, g models.GeoResult) error {
	cid, err := s.candidateID(ctx, g.Candidate, "")
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO geo_results(election, geo_level, geo_code, geo_name, registered, votes_cast, candidate_id, votes, pct)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT(election, geo_level, geo_code, candidate_id)
		DO UPDATE SET votes=EXCLUDED.votes, pct=EXCLUDED.pct`,
		g.Election, g.GeoLevel, g.GeoCode, g.GeoName, g.Registered, g.VotesCast, cid, g.Votes, g.Pct)
	return err
}

// RawResults returns every (date, candidate, pct, sample) tuple for a cycle+round,
// feeding the in-process stats engine (weighted average, CI, forecast).
func (s *Store) RawResults(ctx context.Context, cycle string, round int) ([]models.RawResult, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.field_end::date, ps.name, c.name, COALESCE(c.party,''), c.color, pr.pct, COALESCE(p.sample_size,0)
		FROM polls p
		JOIN poll_results pr ON pr.poll_id = p.id
		JOIN candidates c ON c.id = pr.candidate_id
		JOIN pollsters ps ON ps.id = p.pollster_id
		WHERE p.cycle = $1 AND p.round = $2
		ORDER BY p.field_end`, cycle, round)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.RawResult
	for rows.Next() {
		var r models.RawResult
		if err := rows.Scan(&r.Date, &r.Pollster, &r.Candidate, &r.Party, &r.Color, &r.Pct, &r.SampleSize); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LatestPoll returns the ingest time, field_end and institut of the most recent
// poll for a cycle+round.
func (s *Store) LatestPoll(ctx context.Context, cycle string, round int) (lastUpdated, fieldEnd time.Time, pollster string, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT (SELECT MAX(created_at) FROM polls WHERE cycle = $1 AND round = $2),
		       p.field_end, ps.name
		FROM polls p JOIN pollsters ps ON ps.id = p.pollster_id
		WHERE p.cycle = $1 AND p.round = $2
		ORDER BY p.field_end DESC, p.created_at DESC
		LIMIT 1`, cycle, round).Scan(&lastUpdated, &fieldEnd, &pollster)
	return
}

// PollsterStats returns per-institut activity for a cycle+round.
func (s *Store) PollsterStats(ctx context.Context, cycle string, round int) ([]models.PollsterStat, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ps.name,
		       COUNT(DISTINCT p.id),
		       COALESCE(ROUND(AVG(NULLIF(p.sample_size,0)))::int, 0),
		       MIN(p.field_end)::date,
		       MAX(p.field_end)::date
		FROM polls p
		JOIN pollsters ps ON ps.id = p.pollster_id
		WHERE p.cycle = $1 AND p.round = $2
		GROUP BY ps.name
		ORDER BY COUNT(DISTINCT p.id) DESC, ps.name`, cycle, round)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.PollsterStat
	for rows.Next() {
		var st models.PollsterStat
		var first, last time.Time
		if err := rows.Scan(&st.Pollster, &st.NPolls, &st.AvgSample, &first, &last); err != nil {
			return nil, err
		}
		st.FirstPoll = first.Format("2006-01-02")
		st.LastPoll = last.Format("2006-01-02")
		out = append(out, st)
	}
	return out, rows.Err()
}

// RecentPolls returns individual polls (with source + sponsor) for the sources table.
func (s *Store) RecentPolls(ctx context.Context, cycle string, round, limit int) ([]models.Poll, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.external_id, ps.name, COALESCE(p.sponsor,''), p.field_end::date,
		       COALESCE(p.sample_size,0), p.round, COALESCE(p.source_url,'')
		FROM polls p
		JOIN pollsters ps ON ps.id = p.pollster_id
		WHERE p.cycle = $1 AND p.round = $2
		ORDER BY p.field_end DESC
		LIMIT $3`, cycle, round, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var polls []models.Poll
	var ids []int64
	idx := map[int64]int{}
	for rows.Next() {
		var id int64
		var p models.Poll
		var fe time.Time
		if err := rows.Scan(&id, &p.ExternalID, &p.Pollster, &p.Sponsor, &fe, &p.SampleSize, &p.Round, &p.SourceURL); err != nil {
			return nil, err
		}
		p.FieldEnd = fe.Format("2006-01-02")
		p.Results = map[string]float64{}
		idx[id] = len(polls)
		ids = append(ids, id)
		polls = append(polls, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return polls, nil
	}
	// fill per-poll candidate results
	rr, err := s.pool.Query(ctx, `
		SELECT pr.poll_id, c.name, pr.pct
		FROM poll_results pr JOIN candidates c ON c.id=pr.candidate_id
		WHERE pr.poll_id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rr.Close()
	for rr.Next() {
		var pid int64
		var name string
		var pct float64
		if err := rr.Scan(&pid, &name, &pct); err != nil {
			return nil, err
		}
		if i, ok := idx[pid]; ok {
			polls[i].Results[name] = pct
		}
	}
	return polls, rr.Err()
}

// ActualResults returns the official final result of a past election (reference line).
func (s *Store) ActualResults(ctx context.Context, cycle string, round int) ([]models.ActualResult, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.candidate, COALESCE(c.color,'#888888'), a.pct, a.won
		FROM actual_results a
		LEFT JOIN candidates c ON c.name = a.candidate
		WHERE a.cycle = $1 AND a.round = $2
		ORDER BY a.pct DESC`, cycle, round)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ActualResult
	for rows.Next() {
		var r models.ActualResult
		if err := rows.Scan(&r.Candidate, &r.Color, &r.Pct, &r.Won); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Cycles lists the election cycles that have poll data, plus which have actual results.
func (s *Store) Cycles(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT p.cycle,
		       EXISTS(SELECT 1 FROM actual_results a WHERE a.cycle = p.cycle) AS has_actual
		FROM polls p ORDER BY p.cycle DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var cycle string
		var hasActual bool
		if err := rows.Scan(&cycle, &hasActual); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"cycle": cycle, "has_actual": hasActual})
	}
	return out, rows.Err()
}

// GeoWinners returns, per geo_code, the leading candidate for a given election+level (for the map).
func (s *Store) GeoWinners(ctx context.Context, election, level string) ([]map[string]any, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (g.geo_code)
		       g.geo_code, g.geo_name, c.name, c.color, g.pct
		FROM geo_results g
		JOIN candidates c ON c.id=g.candidate_id
		WHERE g.election=$1 AND g.geo_level=$2
		ORDER BY g.geo_code, g.pct DESC`, election, level)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var code, name, cand, color string
		var pct float64
		if err := rows.Scan(&code, &name, &cand, &color, &pct); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"geo_code": code, "geo_name": name, "winner": cand, "color": color, "pct": pct,
		})
	}
	return out, rows.Err()
}
