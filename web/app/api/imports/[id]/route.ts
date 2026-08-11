// Import status proxy — the polling half of the upload loop (phase 7 BRIEF
// §1.2). The upload island asks this route every couple of seconds while an
// import runs; it forwards to GET /imports/{id} on the Go API through the
// generated client (the one access path). A plain HTTP resource, so a route
// handler, exactly like the thumbnail proxy.
import { api } from "@/lib/api/client";

export async function GET(
  _req: Request,
  { params }: { params: Promise<{ id: string }> },
) {
  const { id: rawId } = await params;
  const id = Number(rawId);
  if (!Number.isInteger(id)) {
    return Response.json({ error: "not found" }, { status: 404 });
  }

  const { data, error, response } = await api.GET("/imports/{id}", {
    params: { path: { id } },
  });
  return Response.json(data ?? error ?? { error: "the API is not reachable" }, {
    status: response?.status ?? 502,
    // Polling must always see the row's current state.
    headers: { "Cache-Control": "no-store" },
  });
}
