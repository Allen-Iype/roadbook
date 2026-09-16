-- +goose Up
-- Phase 13 CP4: share links. A share link is a capability URL (BRIEF §1):
-- the token IS the permission — anyone holding it may read one confirmed
-- adventure, signed in or not, on an authless instance or a hosted one.
-- The table stores only the token's SHA-256, exactly like sessions
-- (00012): a leaked row cannot be turned back into a working link, and the
-- raw token is shown to its creator once, at creation.
--
-- A link is tied to a DECISION, not a candidate: the decision is the
-- adventure's durable identity (phase 1 anchored matching; photos ride it
-- the same way, 00006). Re-detection renumbers candidates and the link
-- keeps working through the recomputed match; a decision that is dismissed
-- again, or currently orphaned, makes its links answer 404 — a link only
-- ever reaches a CONFIRMED adventure of the latest run.
--
-- Revocation is a DELETE (the sessions rule): there is no soft state to
-- reason about, and "revoked" and "never existed" are deliberately
-- indistinguishable from the outside.
--
-- user_id is redundant with decisions.user_id and carried anyway: every
-- user-data table names its owner directly so the store's per-user filter
-- is uniform (CP1 rule — a missed filter is visible in the diff), and
-- CP5's per-user deletion counts rows by it. ON DELETE CASCADE on both
-- references because a link is derived permission over user data, never
-- user data itself: when the adventure or the account goes, so must every
-- link to it.
CREATE TABLE share_links (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     text   NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    decision_id bigint NOT NULL REFERENCES decisions (id) ON DELETE CASCADE,
    token_hash  text   NOT NULL UNIQUE,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX share_links_decision_idx ON share_links (user_id, decision_id, id);

-- +goose Down
DROP TABLE share_links;
