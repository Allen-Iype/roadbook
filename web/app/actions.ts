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
