-- Pollsters (Ifop, Ipsos, OpinionWay, Elabe, ...)
CREATE TABLE IF NOT EXISTS pollsters (
    id   SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

-- Candidates for the 2027 French presidential election
CREATE TABLE IF NOT EXISTS candidates (
    id      SERIAL PRIMARY KEY,
    name    TEXT UNIQUE NOT NULL,
    party   TEXT,
    color   TEXT DEFAULT '#888888'
);

-- One poll = one survey by one pollster on one date
CREATE TABLE IF NOT EXISTS polls (
    id            BIGSERIAL PRIMARY KEY,
    external_id   TEXT UNIQUE,               -- dedupe key from source
    pollster_id   INT REFERENCES pollsters(id),
    field_start   DATE,
    field_end     DATE NOT NULL,
    sample_size   INT,
    round         SMALLINT NOT NULL DEFAULT 1, -- 1 or 2 (tour)
    source_url    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_polls_field_end ON polls(field_end);

-- Voting intention per candidate within a poll (percentage 0..100)
CREATE TABLE IF NOT EXISTS poll_results (
    poll_id      BIGINT REFERENCES polls(id) ON DELETE CASCADE,
    candidate_id INT REFERENCES candidates(id),
    pct          NUMERIC(4,1) NOT NULL,
    PRIMARY KEY (poll_id, candidate_id)
);

-- Rolling aggregate (moving average) computed by the aggregator service
CREATE TABLE IF NOT EXISTS aggregates (
    as_of        DATE NOT NULL,
    candidate_id INT REFERENCES candidates(id),
    round        SMALLINT NOT NULL DEFAULT 1,
    avg_pct      NUMERIC(4,1) NOT NULL,
    n_polls      INT NOT NULL,
    PRIMARY KEY (as_of, candidate_id, round)
);

-- Official results by geography from elections.interieur.gouv.fr
-- geo_level: 'departement' | 'commune' | 'region'
CREATE TABLE IF NOT EXISTS geo_results (
    id           BIGSERIAL PRIMARY KEY,
    election     TEXT NOT NULL,              -- e.g. 'presidentielle-2022-t1'
    geo_level    TEXT NOT NULL,
    geo_code     TEXT NOT NULL,             -- INSEE code (dept '75', commune '75056')
    geo_name     TEXT,
    registered   INT,                       -- inscrits
    votes_cast   INT,                       -- votants
    candidate_id INT REFERENCES candidates(id),
    votes        INT NOT NULL,
    pct          NUMERIC(5,2),
    UNIQUE (election, geo_level, geo_code, candidate_id)
);
CREATE INDEX IF NOT EXISTS idx_geo_results_lookup ON geo_results(election, geo_level, geo_code);

-- Seed a few known likely candidates (editable)
INSERT INTO candidates (name, party, color) VALUES
    ('Jordan Bardella',       'RN',      '#0d378a'),
    ('Gabriel Attal',         'Renaissance', '#ffd700'),
    ('Jean-Luc Mélenchon',    'LFI',     '#cc2443'),
    ('Édouard Philippe',      'Horizons', '#00b0f0'),
    ('Marine Le Pen',         'RN',      '#0d378a'),
    ('Raphaël Glucksmann',    'PS',      '#ff8080')
ON CONFLICT (name) DO NOTHING;
