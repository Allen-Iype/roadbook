// Upload proxy (phase 7 BRIEF §3B): the browser talks only to Next, so the
// upload island POSTs here and this route handler forwards the request to
// the Go API — as a stream. It never parses the body: the multipart bytes
// (and the boundary declared in Content-Type) pass through verbatim, so a
// multi-hundred-MB archive occupies no Node memory. A route handler rather
// than a server action because an action would materialise File objects —
// the whole archive in process memory — and its encoding exists for
// React-invoked mutations, not gigabyte bodies.
//
// The generated client still makes the upstream call (the photos-upload
// precedent: only the body serialisation is custom). `duplex: "half"` is
// what Node's fetch requires to send a request body as a stream — without
// it the call rejects outright, so there is no silent-buffering failure
// mode to fear here.
import { api } from "@/lib/api/client";

export async function POST(request: Request) {
  const { data, error, response } = await api.POST("/imports", {
    // The generated type wants the multipart fields; the serializer below
    // replaces the body wholesale with the raw incoming stream.
    body: { file: "" },
    bodySerializer: () => request.body,
    headers: {
      "content-type": request.headers.get("content-type") ?? "",
    },
    // duplex is what Node's fetch requires to send a request body as a
    // stream; openapi-fetch passes unknown init options through to fetch.
    duplex: "half",
  });

  // Pass the Go API's answer through unchanged — 202 Import, 400
  // ImportRejection, 409/413 Error. The island reads the status code.
  return Response.json(data ?? error ?? { error: "the API is not reachable" }, {
    status: response?.status ?? 502,
  });
}
