-- +goose Up
-- Import bookkeeping (phase 5 BRIEF §3B): a failed import must be visible in
-- the product, not only in a closed terminal. The import command now writes
-- its row first (status 'running') and finalises it to 'completed' or
-- 'failed'; existing rows all predate failure recording and were successful,
-- hence the 'completed' default backfill.
--
-- detected_format is the sniffer's stable format label, recorded on every
-- import, successful or failed. It is deliberately separate from error: the
-- user-facing message is prose and gets reworded; evidence must be queryable
-- (select detected_format, count(*) from imports where status = 'failed'
-- group by 1). This column is what makes the legacy-formats backlog trigger
-- a count rather than an anecdote (DECISIONS 2026-08-06). NULL means the
-- failure happened before the input was recognised (unreadable file, empty).
ALTER TABLE imports ADD COLUMN status text NOT NULL DEFAULT 'completed'
    CHECK (status IN ('running', 'completed', 'failed'));
ALTER TABLE imports ADD COLUMN error text;
ALTER TABLE imports ADD COLUMN detected_format text;

-- +goose Down
ALTER TABLE imports DROP COLUMN detected_format;
ALTER TABLE imports DROP COLUMN error;
ALTER TABLE imports DROP COLUMN status;
