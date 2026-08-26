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

export type BulkDecideResult =
  | { ok: true; decided: number }
  | { ok: false; error: string };

// Bulk triage (phase 11 §6.1): one atomic API call for a whole selection.
// All-or-nothing is the contract's promise — a failure means nothing landed,
// and the bar can say so in one sentence.
export async function decideBulk(
  decisions: { id: number; action: Action; name?: string }[],
): Promise<BulkDecideResult> {
  const { data, error, response } = await api.POST("/candidates/decisions", {
    body: { decisions },
  });
  if (error || !data) {
    return {
      ok: false,
      error: error?.error ?? `bulk decide failed (HTTP ${response.status})`,
    };
  }
  revalidatePath("/candidates");
  revalidatePath("/");
  return { ok: true, decided: data.decided };
}

export type ConfirmSuggestedResult =
  | { ok: true; confirmed: number; suggested: number; dateNamed: number }
  | { ok: false; error: string };

// Confirm a selection using suggested names (phase 11 §6.1's quick-confirm
// at selection scale). Suggestions are best-effort; where none exists the
// candidate is named by its start date — "Journey of 2026-06-16", the same
// convention unconfirmed covers already display — and confirmed anyway
// (maintainer's decision at the CP4 review, DECISIONS 2026-08-26: one pass
// clears the whole selection; renaming is one click via the card's or the
// adventure page's change control).
export async function confirmSelectedWithSuggestions(
  ids: number[],
): Promise<ConfirmSuggestedResult> {
  // The date fallback needs each candidate's span start.
  const list = await api.GET("/candidates");
  const startById = new Map<number, string>();
  for (const c of list.data?.candidates ?? []) {
    startById.set(c.id, c.span_start);
  }

  // Sequential, deliberately: with a real geocoder behind the seam these
  // are external requests, and the public Nominatim policy is ~1/s — a
  // 60-wide parallel burst is how instances get blocked. Natural request
  // latency spaces the calls; the null suggester answers instantly, so a
  // geocoder-less instance pays nothing for the loop.
  let suggested = 0;
  let dateNamed = 0;
  const decisions: { id: number; action: Action; name: string }[] = [];
  for (const id of ids) {
    const { data } = await api.GET("/candidates/{id}/name-suggestion", {
      params: { path: { id } },
    });
    const name = data?.name?.trim();
    if (name) {
      suggested++;
      decisions.push({ id, action: "confirmed", name });
    } else {
      dateNamed++;
      // slice(0, 10): the civil date the traveller experienced, never
      // shifted through the viewer's timezone (the page-wide convention).
      const date = startById.get(id)?.slice(0, 10) ?? "unknown date";
      decisions.push({ id, action: "confirmed", name: `Journey of ${date}` });
    }
  }
  const res = await decideBulk(decisions);
  if (!res.ok) return res;
  return { ok: true, confirmed: res.decided, suggested, dateNamed };
}

// Name suggestion for the confirm step (BRIEF §1.7). Best-effort by design:
// the null suggester, a stale candidate, or an unreachable geocoder all
// return no name, and the confirm flow proceeds with an empty input exactly
// as it would without the seam. The suggestion is prefill — never applied
// to a decision by itself.
export async function suggestName(
  candidateId: number,
): Promise<{ name?: string }> {
  const { data } = await api.GET("/candidates/{id}/name-suggestion", {
    params: { path: { id: candidateId } },
  });
  return { name: data?.name };
}

type PhotoUploadResults = components["schemas"]["PhotoUploadResults"];

export type UploadPhotosResult =
  | { ok: true; results: PhotoUploadResults["results"] }
  | { ok: false; error: string };

// Photo upload: the browser hands the action a FormData of files; the action
// forwards it to the Go API as multipart. The generated client still types
// the path and the response — only the body serialisation is custom, because
// multipart bytes pass through rather than being JSON-encoded. Original
// bytes exist here transiently in memory; the Go side stores a thumbnail and
// discards them (phase 4 BRIEF §1.3, §3B).
export async function uploadPhotos(
  candidateId: number,
  formData: FormData,
): Promise<UploadPhotosResult> {
  const { data, error, response } = await api.POST("/candidates/{id}/photos", {
    params: { path: { id: candidateId } },
    body: {},
    bodySerializer: () => formData,
  });
  if (error || !data) {
    // 404 (stale candidate) and 409 (not confirmed) both carry { error }.
    return {
      ok: false,
      error: error?.error ?? `upload failed (HTTP ${response.status})`,
    };
  }
  revalidatePath(`/adventure/${candidateId}`);
  return { ok: true, results: data.results };
}

export type DeletePhotoResult = { ok: true } | { ok: false; error: string };

export async function deletePhoto(
  photoId: number,
  candidateId: number,
): Promise<DeletePhotoResult> {
  const { error, response } = await api.DELETE("/photos/{id}", {
    params: { path: { id: photoId } },
  });
  if (error) {
    return {
      ok: false,
      error: error.error ?? `delete failed (HTTP ${response.status})`,
    };
  }
  revalidatePath(`/adventure/${candidateId}`);
  return { ok: true };
}
