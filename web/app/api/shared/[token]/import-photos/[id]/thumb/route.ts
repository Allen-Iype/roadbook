// Token-scoped thumbnail proxy for photo-import records (phase 13 CP4) —
// the sibling of ../../photos/[id]/thumb. 404 for a record outside the
// shared adventure's span, and for a record with no thumbnail (HEIC).
import { api } from "@/lib/api/client";

export async function GET(
  _req: Request,
  { params }: { params: Promise<{ token: string; id: string }> },
) {
  const { token, id: rawId } = await params;
  const id = Number(rawId);
  if (!Number.isInteger(id)) {
    return new Response("not found", { status: 404 });
  }

  const { data, response } = await api.GET(
    "/shared/{token}/import-photos/{id}/thumbnail",
    { params: { path: { token, id } }, parseAs: "stream" },
  );
  if (!data) {
    return new Response("not found", { status: response.status || 404 });
  }
  return new Response(data, {
    headers: {
      "Content-Type": "image/jpeg",
      "Cache-Control": "private, max-age=86400",
    },
  });
}
