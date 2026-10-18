-- +goose Up
CREATE TABLE checks (
    id         TEXT PRIMARY KEY, -- and its ping URL's
    project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    period     BIGINT NOT NULL, -- seconds between pings
    grace      BIGINT NOT NULL, -- seconds a ping may be late
    status     TEXT NOT NULL,   -- new, up, down or paused
    last_ping  BIGINT,          -- NULL before the first
    started_at BIGINT,          -- the start of the job's run, NULL when none runs
    due_at     BIGINT,          -- when it goes down without a ping, if it is up
    created_at BIGINT NOT NULL
);
CREATE INDEX checks_project ON checks (project_id, created_at);
CREATE INDEX checks_due ON checks (status, due_at);

CREATE TABLE pings (
    id         INTEGER PRIMARY KEY,
    check_id   TEXT NOT NULL REFERENCES checks (id) ON DELETE CASCADE,
    kind       TEXT NOT NULL,   -- success, start or fail
    body       TEXT NOT NULL,   -- the job's output, its first 10 KB
    user_agent TEXT NOT NULL,
    duration   BIGINT,          -- microseconds since the run's start, if a start ping began it
    created_at BIGINT NOT NULL
);
CREATE INDEX pings_check ON pings (check_id, id);
CREATE INDEX pings_created_at ON pings (created_at);

-- +goose Down
DROP TABLE pings;
DROP TABLE checks;
