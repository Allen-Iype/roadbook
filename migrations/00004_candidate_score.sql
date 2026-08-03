-- +goose Up
-- Confidence score with its per-component breakdown (BRIEF §1.6). Candidates
-- are regenerated wholesale each run, so new columns are cheap. Nullable by
-- design: rows from runs before scoring existed read as "not scored", which
-- is different information than a low score — the same honesty rule that
-- redistributes a missing component's weight instead of scoring it zero.
ALTER TABLE candidates ADD COLUMN score double precision;
ALTER TABLE candidates ADD COLUMN score_breakdown jsonb;

-- +goose Down
ALTER TABLE candidates DROP COLUMN score_breakdown;
ALTER TABLE candidates DROP COLUMN score;
