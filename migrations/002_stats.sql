-- Cycle + sponsor on polls so 2022 and 2027 data coexist and stay separable.
ALTER TABLE polls ADD COLUMN IF NOT EXISTS cycle   TEXT NOT NULL DEFAULT '2027';
ALTER TABLE polls ADD COLUMN IF NOT EXISTS sponsor TEXT;
CREATE INDEX IF NOT EXISTS idx_polls_cycle ON polls(cycle, round, field_end);

-- Party colour reference so charts colour candidates by party.
CREATE TABLE IF NOT EXISTS parties (
    name  TEXT PRIMARY KEY,
    color TEXT NOT NULL
);
INSERT INTO parties(name, color) VALUES
    ('Rassemblement national', '#0d378a'),
    ('Reconquête',             '#243156'),
    ('Les Républicains',       '#0066cc'),
    ('Renaissance',            '#ffb600'),
    ('Horizons',               '#00b0f0'),
    ('MoDem',                  '#ff7f00'),
    ('Parti socialiste',       '#f0426d'),
    ('Place publique',         '#f0426d'),
    ('Europe Écologie Les Verts','#26a65b'),
    ('La France insoumise',    '#cc2443'),
    ('France insoumise',       '#cc2443'),
    ('Parti communiste français','#d70000'),
    ('Debout la France',       '#12428c'),
    ('Lutte ouvrière',         '#b30000'),
    ('Nouveau Parti anticapitaliste','#c0392b')
ON CONFLICT (name) DO UPDATE SET color = EXCLUDED.color;

-- Official final results of past elections, for poll-vs-outcome comparison.
CREATE TABLE IF NOT EXISTS actual_results (
    cycle        TEXT NOT NULL,
    round        SMALLINT NOT NULL,
    candidate    TEXT NOT NULL,
    pct          NUMERIC(4,1) NOT NULL,
    won          BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (cycle, round, candidate)
);

-- 2022 first round (official, Ministry of the Interior).
INSERT INTO actual_results(cycle, round, candidate, pct, won) VALUES
    ('2022', 1, 'Emmanuel Macron', 27.9, true),
    ('2022', 1, 'Marine Le Pen', 23.2, false),
    ('2022', 1, 'Jean-Luc Mélenchon', 22.0, false),
    ('2022', 1, 'Éric Zemmour', 7.1, false),
    ('2022', 1, 'Valérie Pécresse', 4.8, false),
    ('2022', 1, 'Yannick Jadot', 4.6, false),
    ('2022', 1, 'Jean Lassalle', 3.1, false),
    ('2022', 1, 'Fabien Roussel', 2.3, false),
    ('2022', 1, 'Nicolas Dupont-Aignan', 2.1, false),
    ('2022', 1, 'Anne Hidalgo', 1.7, false),
    ('2022', 1, 'Philippe Poutou', 0.8, false),
    ('2022', 1, 'Nathalie Arthaud', 0.6, false),
    -- 2022 second round.
    ('2022', 2, 'Emmanuel Macron', 58.5, true),
    ('2022', 2, 'Marine Le Pen', 41.5, false),
    -- 2017 first round.
    ('2017', 1, 'Emmanuel Macron', 24.0, true),
    ('2017', 1, 'Marine Le Pen', 21.3, false),
    ('2017', 1, 'François Fillon', 20.0, false),
    ('2017', 1, 'Jean-Luc Mélenchon', 19.6, false),
    ('2017', 1, 'Benoît Hamon', 6.4, false),
    ('2017', 2, 'Emmanuel Macron', 66.1, true),
    ('2017', 2, 'Marine Le Pen', 33.9, false)
ON CONFLICT (cycle, round, candidate) DO UPDATE SET pct = EXCLUDED.pct, won = EXCLUDED.won;
