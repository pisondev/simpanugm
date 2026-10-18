-- +goose Up
-- set when an administrator edits a row through US-07, from then on the seed inserts missing rows but leaves this one alone
ALTER TABLE zones ADD COLUMN edited_at timestamptz;
ALTER TABLE locations ADD COLUMN edited_at timestamptz;
ALTER TABLE posts ADD COLUMN edited_at timestamptz;
ALTER TABLE categories ADD COLUMN edited_at timestamptz;

-- a pair is switched off rather than deleted, so a later seed run cannot bring it back
ALTER TABLE category_compatibility
    ADD COLUMN active boolean NOT NULL DEFAULT true,
    ADD COLUMN edited_at timestamptz;

-- +goose Down
ALTER TABLE category_compatibility DROP COLUMN edited_at, DROP COLUMN active;
ALTER TABLE categories DROP COLUMN edited_at;
ALTER TABLE posts DROP COLUMN edited_at;
ALTER TABLE locations DROP COLUMN edited_at;
ALTER TABLE zones DROP COLUMN edited_at;
