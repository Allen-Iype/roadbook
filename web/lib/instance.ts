import "server-only";

// The reserved header slot's content (phase 9 BRIEF §6): the app shell
// header carries the instance label today and an account control in some
// later, charter-gated phase. Operator-set, optional, empty by default —
// a single self-hosted instance needs no label. Read at request time on
// the server (every app page is force-dynamic), so compose can set it
// without a rebuild; the public shell never reads it.
export const INSTANCE_LABEL: string =
  process.env.ROADBOOK_INSTANCE_LABEL ?? "";
