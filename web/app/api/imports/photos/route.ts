// Photo-batch upload proxy (phase 11 BRIEF §4C): the browser talks only to
// Next, so the photo island POSTs here and this handler forwards the
// multipart stream to the Go API verbatim — the same pass-through shape as
// the Timeline proxy (app/api/imports/route.ts), for the same reason: a
// camera-roll batch must occupy no Node memory. The Go side processes parts
// one at a time and discards each photo's bytes after metadata extraction;
// nothing is retained anywhere on this path except records and thumbnails.
import { api } from "@/lib/api/client";

export async function POST(request: Request) {
  const { data, error, response } = await api.POST("/imports/photos", {
    // The generated type wants the multipart fields; the serializer below
    // replaces the body wholesale with the raw incoming stream.
    body: { file: [] },
    bodySerializer: () => request.body,
    headers: {
      "content-type": request.headers.get("content-type") ?? "",
    },
    // duplex is what Node's fetch requires to send a request body as a
    // stream; openapi-fetch passes unknown init options through to fetch.
    duplex: "half",
  });

  // Pass the Go API's answer through unchanged — 202 PhotoImportResult,
  // 400/409 Error. The island reads the status code.
  return Response.json(data ?? error ?? { error: "the API is not reachable" }, {
    status: response?.status ?? 502,
  });
}
