-- Privacy-friendly daily unique-visitor counter (no cookie). visitor is a daily
-- one-way hash of IP+UA+secret — cannot be linked across days or back to a person.
CREATE TABLE IF NOT EXISTS visits (
    day     DATE NOT NULL,
    visitor TEXT NOT NULL,
    PRIMARY KEY (day, visitor)
);
CREATE INDEX IF NOT EXISTS idx_visits_day ON visits(day);
