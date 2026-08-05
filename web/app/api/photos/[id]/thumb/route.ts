// Thumbnail proxy (phase 4 BRIEF §1.3): the browser talks only to Next, so
// <img src> points here and this route handler streams the bytes from the Go
// API. A route handler rather than a server action because this is a plain
// HTTP resource — an image URL — not a mutation: actions are POST-only
// functions invoked from React; a handler answers GET with arbitrary bytes.
//
// The generated client still makes the upstream call (parseAs: "stream"
// passes the body through untouched) — the one access path to the Go API
// survives even for binary responses.
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

  const { data, response } = await api.GET("/photos/{id}/thumbnail", {
    params: { path: { id } },
    parseAs: "stream",
  });
  if (!data) {
    return new Response("not found", { status: response.status || 404 });
  }
  return new Response(data, {
    headers: {
      "Content-Type": "image/jpeg",
      // Thumbnails are immutable in practice (content-hash-named on disk;
      // a photo id never changes its pixels), so let the browser keep them
      // for a day. A deleted photo's cached thumbnail is harmless: nothing
      // references its URL any more.
      "Cache-Control": "private, max-age=86400",
    },
  });
}
