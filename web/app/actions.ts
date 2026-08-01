"use server";

// Server actions: functions that run only on the server but are callable from
// client components as if they were local async functions — Next serialises
// the call into a POST behind the scenes. This file-level "use server" marks
// every export. This is how writes keep the architecture rule intact: the
// browser invokes the action, the action (server-side) calls the Go API, and
// no API access ever ships to the client (CLAUDE.md invariant 11).

import { revalidatePath } from "next/cache";
import { api } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";

type Action = components["schemas"]["DecisionRequest"]["action"];

// Result type instead of thrown errors: a failed decide is an expected
// outcome (API down, stale candidate id after re-detection), and expected
// outcomes are data, not exceptions. The client renders r.error inline.
export type DecideResult = { ok: true } | { ok: false; error: string };

export async function decideCandidate(
  candidateId: number,
  action: Action,
  name?: string,
): Promise<DecideResult> {
  const { error, response } = await api.POST("/candidates/{id}/decision", {
    params: { path: { id: candidateId } },
    body: { action, name },
  });
  if (error) {
    // The contract's 400/404 bodies both carry { error: string }.
    return {
      ok: false,
      error: error.error ?? `decide failed (HTTP ${response.status})`,
    };
  }
  // Invalidate the server-rendered list so the next render reflects the
  // database. The action's response and the refreshed page arrive in one
  // round trip.
  revalidatePath("/");
  return { ok: true };
}
