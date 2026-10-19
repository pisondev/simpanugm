-- +goose Up
CREATE TABLE duplicate_links (
    id          uuid PRIMARY KEY,
    report_type text NOT NULL CHECK (report_type IN ('lost', 'found')),
    report_a    uuid NOT NULL,
    report_b    uuid NOT NULL,
    score       numeric(6,4) NOT NULL,
    status      text NOT NULL CHECK (status IN ('suggested', 'confirmed', 'rejected')),
    reviewed_by uuid REFERENCES users (id),
    reviewed_at timestamptz,
    created_at  timestamptz NOT NULL,
    CHECK (report_a <> report_b)
);

CREATE UNIQUE INDEX duplicate_links_pair ON duplicate_links (
    report_type,
    LEAST(report_a, report_b),
    GREATEST(report_a, report_b)
);

CREATE INDEX duplicate_links_status ON duplicate_links (status, score DESC);
CREATE INDEX duplicate_links_report_a ON duplicate_links (report_a);
CREATE INDEX duplicate_links_report_b ON duplicate_links (report_b);

-- +goose Down
DROP TABLE duplicate_links;
