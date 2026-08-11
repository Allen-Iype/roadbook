-- +goose Up
-- Browser upload path (phase 7 BRIEF §4, migration 00009).
--
-- content_hash: SHA-256 of the uploaded file, which is also the retained
-- file's name in the uploads directory — row↔file association exactly as
-- photos established it (phase 4). NULL on CLI imports and every historical
-- row: the CLI reads from a path the operator owns and retains nothing.
--
-- detect_status: the automatic detection that follows a successful upload
-- import (BRIEF §3D). Persisted — not held in serve memory — because the
-- front door polls it and a restart must not turn "detection failed" into
-- silence. NULL means no auto-detect was triggered (CLI imports, failed
-- imports). A detection failure never marks the import failed: the import
-- row's status describes the import, whose semantics predate this phase.
-- inserted: how many observations were genuinely new (the ON CONFLICT
-- skip count's complement). The per-type counters record what the file
-- contained; without this, a duplicate upload's row is indistinguishable
-- from its original, and the front door cannot say "nothing new". NULL on
-- historical rows: the number was printed to a terminal and not kept.
ALTER TABLE imports ADD COLUMN content_hash text;
ALTER TABLE imports ADD COLUMN detect_status text
    CHECK (detect_status IN ('running', 'completed', 'failed'));
ALTER TABLE imports ADD COLUMN inserted integer;

-- +goose Down
ALTER TABLE imports DROP COLUMN inserted;
ALTER TABLE imports DROP COLUMN detect_status;
ALTER TABLE imports DROP COLUMN content_hash;
