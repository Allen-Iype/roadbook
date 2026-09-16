// Token-scoped thumbnail proxy (phase 13 CP4): the shared view's tiles and
// markers load through here. Same shape as /api/photos/[id]/thumb, but the
// upstream operation is the share-token one — the Go side authorises by
// the token and answers 404 for any photo not on the shared adventure, and
// that 404 passes through unchanged. No session is involved at any layer.
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
    "/shared/{token}/photos/{id}/thumbnail",
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
