import "server-only";

// Where a sign-in refused by the user cap is pointed (phase 15 BRIEF §9):
// the waitlist form's URL, on the public landing page in the hosted
// deployment. Operator-set, optional, empty by default — a self-hosted
// instance has no waitlist and no cap. Read at request time on the server
// (the sign-in page is force-dynamic), so compose can set it without a
// rebuild. This is read by the public shell on purpose: the full state is
// shown to someone who is not, and cannot become, a user.
export const WAITLIST_URL: string = process.env.ROADBOOK_WAITLIST_URL ?? "";
