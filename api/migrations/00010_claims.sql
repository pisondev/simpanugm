-- +goose Up
CREATE TABLE claims (
    id                       uuid PRIMARY KEY,
    code                     text NOT NULL UNIQUE,
    claimant_id              uuid REFERENCES users (id),
    created_by               uuid NOT NULL REFERENCES users (id),
    lost_report_id           uuid NOT NULL REFERENCES lost_reports (id),
    found_report_id          uuid NOT NULL REFERENCES found_reports (id),
    match_id                 uuid REFERENCES matches (id),
    source                   text NOT NULL CHECK (source IN ('candidate', 'listing', 'officer_invite', 'proxy')),
    statement                text NOT NULL CHECK (char_length(statement) BETWEEN 30 AND 1000),
    answers                  jsonb NOT NULL DEFAULT '[]',
    preferred_visit          text CHECK (char_length(preferred_visit) <= 100),
    status                   text NOT NULL CHECK (status IN ('submitted', 'under_review', 'info_requested', 'on_hold', 'approved',
                                                             'handed_over', 'rejected', 'withdrawn', 'expired')),
    reviewer_id              uuid REFERENCES users (id),
    decision_reason          text CHECK (decision_reason IN ('details_mismatch', 'insufficient_evidence', 'item_already_returned',
                                                             'duplicate_claim', 'suspected_fraud', 'other')),
    decision_note            text CHECK (char_length(decision_note) <= 1000),
    internal_note            text CHECK (char_length(internal_note) <= 2000),
    decided_by               uuid REFERENCES users (id),
    decided_at               timestamptz,
    handover_nonce            bytea,
    handover_code_issued_at  timestamptz,
    handover_failed_attempts smallint NOT NULL DEFAULT 0,
    handover_locked_at       timestamptz,
    approved_expires_at      timestamptz,
    expiring_notified_at     timestamptz,
    info_requested_at        timestamptz,
    handed_over_at           timestamptz,
    handed_over_by           uuid REFERENCES users (id),
    recipient_id_type        text CHECK (recipient_id_type IN ('ktm', 'ktp', 'sim', 'passport', 'other')),
    recipient_id_last4       text CHECK (char_length(recipient_id_last4) = 4),
    recipient_name_confirmed boolean,
    reopened_at              timestamptz,
    version                  integer NOT NULL DEFAULT 1,
    created_at               timestamptz NOT NULL,
    updated_at               timestamptz NOT NULL
);

-- BR-CLM-02: one open claim per member and found report, and two approvals for one item are impossible
CREATE UNIQUE INDEX claims_one_open ON claims (claimant_id, found_report_id)
    WHERE claimant_id IS NOT NULL AND status IN ('submitted', 'under_review', 'info_requested', 'approved', 'on_hold');
CREATE UNIQUE INDEX claims_one_approved ON claims (found_report_id) WHERE status = 'approved';
CREATE INDEX claims_claimant ON claims (claimant_id, status);
CREATE INDEX claims_found ON claims (found_report_id, status);
CREATE INDEX claims_lost ON claims (lost_report_id, status);
CREATE INDEX claims_status_updated ON claims (status, updated_at);

CREATE TABLE claim_messages (
    id          uuid PRIMARY KEY,
    claim_id    uuid NOT NULL REFERENCES claims (id) ON DELETE CASCADE,
    author_id   uuid REFERENCES users (id),
    author_role text NOT NULL CHECK (author_role IN ('member', 'officer', 'admin', 'system')),
    body        text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
    created_at  timestamptz NOT NULL
);

CREATE INDEX claim_messages_claim ON claim_messages (claim_id, created_at);

-- AI-08: the suggestion at decision time and the officer's result stay with the claim for later audit
CREATE TABLE claim_checks (
    claim_id   uuid NOT NULL REFERENCES claims (id) ON DELETE CASCADE,
    item_key   text NOT NULL,
    position   smallint NOT NULL,
    label      text NOT NULL,
    required   boolean NOT NULL,
    info_only  boolean NOT NULL DEFAULT false,
    suggestion jsonb NOT NULL DEFAULT '{}',
    result     text NOT NULL DEFAULT 'pending' CHECK (result IN ('pending', 'pass', 'fail', 'not_applicable')),
    note       text CHECK (char_length(note) <= 500),
    decided_by uuid REFERENCES users (id),
    decided_at timestamptz,
    PRIMARY KEY (claim_id, item_key)
);

CREATE TABLE dispositions (
    found_report_id uuid PRIMARY KEY REFERENCES found_reports (id),
    kind            text NOT NULL CHECK (kind IN ('donated', 'destroyed', 'transferred_to_unit', 'returned_to_finder')),
    recipient       text CHECK (char_length(recipient) <= 200),
    note            text NOT NULL CHECK (char_length(note) BETWEEN 5 AND 500),
    officer_id      uuid NOT NULL REFERENCES users (id),
    created_at      timestamptz NOT NULL
);

-- evidence images of a claim, FR-CLAIM-02
ALTER TABLE media ADD COLUMN claim_id uuid REFERENCES claims (id);
ALTER TABLE media DROP CONSTRAINT media_purpose_check;
ALTER TABLE media ADD CONSTRAINT media_purpose_check CHECK (purpose IN ('lost_report', 'found_report', 'claim_evidence'));
-- +goose StatementBegin
DO $$
DECLARE c record;
BEGIN
    FOR c IN SELECT conname FROM pg_constraint WHERE conrelid = 'media'::regclass AND contype = 'c' AND pg_get_constraintdef(oid) LIKE '%num_nonnulls%' LOOP
        EXECUTE format('ALTER TABLE media DROP CONSTRAINT %I', c.conname);
    END LOOP;
END $$;
-- +goose StatementEnd
ALTER TABLE media ADD CONSTRAINT media_one_owner CHECK (num_nonnulls(lost_report_id, found_report_id, claim_id) <= 1);
ALTER TABLE media ADD CONSTRAINT media_ready_has_owner CHECK (state <> 'ready' OR num_nonnulls(lost_report_id, found_report_id, claim_id) = 1);
CREATE UNIQUE INDEX media_claim_sha ON media (claim_id, sha256) WHERE claim_id IS NOT NULL AND state <> 'deleted';
CREATE INDEX media_claim ON media (claim_id, position) WHERE claim_id IS NOT NULL;

-- +goose Down
DROP INDEX media_claim;
DROP INDEX media_claim_sha;
ALTER TABLE media DROP CONSTRAINT media_ready_has_owner;
ALTER TABLE media DROP CONSTRAINT media_one_owner;
ALTER TABLE media DROP COLUMN claim_id;
ALTER TABLE media DROP CONSTRAINT media_purpose_check;
ALTER TABLE media ADD CONSTRAINT media_purpose_check CHECK (purpose IN ('lost_report', 'found_report'));
ALTER TABLE media ADD CHECK (num_nonnulls(lost_report_id, found_report_id) <= 1);
ALTER TABLE media ADD CHECK (state <> 'ready' OR num_nonnulls(lost_report_id, found_report_id) = 1);
DROP TABLE dispositions;
DROP TABLE claim_checks;
DROP TABLE claim_messages;
DROP TABLE claims;
