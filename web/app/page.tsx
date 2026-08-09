// Transitional (phase 6 checkpoint 1): / still shows the triage table, now
// living at /candidates. Checkpoint 2 replaces this file with the life map.
// The segment config must be declared here, not re-exported — Next parses it
// statically per route file.
export const dynamic = "force-dynamic";
export { default } from "@/app/candidates/page";
