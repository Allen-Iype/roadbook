import "server-only";

import { redirect } from "next/navigation";
import { api } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";

// The session read every gated page starts with (phase 13 CP3). Two layers
// of protection exist and only one lives here: Go answers 401 to any data
// call without a session — that is the enforcement, and it holds no matter
// what this file does. requireUser is the UX half: send a signed-out
// visitor to the sign-in page instead of letting them watch data fetches
// fail. No trust decision is made in this layer (CLAUDE.md invariant 11's
// spirit): a bug here shows someone an empty shell, never data.

export type Session = components["schemas"]["SessionInfo"];

// getSession asks Go who this request is. A null answer means the API is
// unreachable — callers keep their existing API-down rendering for that.
export async function getSession(): Promise<Session | null> {
  try {
    const { data } = await api.GET("/auth/session");
    return data ?? null;
  } catch {
    return null;
  }
}

// requireUser gates a page: in oidc mode a signed-out visitor is
// redirected to /signin. Mode off — the authless single-user reference —
// passes everyone, which is what keeps this a no-op on a plain
// `docker compose up`. Returns the session so pages can hand the account
// slot its email without a second round trip.
export async function requireUser(): Promise<Session | null> {
  const session = await getSession();
  if (session && session.mode === "oidc" && !session.signed_in) {
    redirect("/signin");
  }
  return session;
}
