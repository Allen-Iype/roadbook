// Thumbnail proxy for photo-import records (phase 11 CP4) — the same shape
// as /api/photos/[id]/thumb: the browser talks only to Next, the generated
// client streams the bytes from the Go API. The Go side answers 404 both for
// an unknown record and for a record with no thumbnail (HEIC), and that 404
// passes through unchanged.
import { api } from "@/lib/api/client";

export async function GET(
  _req: Request,
  { params }: { params: Promise<{ id: string }> },
) {
  const { id: rawId } = await params;
  const id = Number(rawId);
  if (!Number.isInteger(id)) {
    return new Response("not found", { status: 404 });
  }

  const { data, response } = await api.GET("/import-photos/{id}/thumbnail", {
    params: { path: { id } },
    parseAs: "stream",
  });
  if (!data) {
    return new Response("not found", { status: response.status || 404 });
  }
  return new Response(data, {
    headers: {
      "Content-Type": "image/jpeg",
      // Content-hash-named on disk; a record id never changes its pixels.
      "Cache-Control": "private, max-age=86400",
    },
  });
}
