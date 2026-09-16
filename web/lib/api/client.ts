// The typed client for the Go API. `schema.d.ts` is generated from
// api/openapi.yaml (`make generate`) and never edited; `paths` describes every
// endpoint, so a call to a wrong path, wrong method, or wrong body shape is a
// compile error, not a runtime surprise (CLAUDE.md invariant 10).
//
// server-only: importing this from a client component fails the build. The
// browser talks only to Next; Next talks to the Go API (CLAUDE.md
// architecture). Keeping the import server-side is what enforces that.
import "server-only";

import { cookies } from "next/headers";
import createClient from "openapi-fetch";
import type { paths } from "./schema";

const baseUrl = process.env.ROADBOOK_API_URL ?? "http://localhost:8080";

export const api = createClient<paths>({
  baseUrl,
  // The candidate list must reflect the database on every request — decisions
  // change it. `no-store` opts this fetch out of Next's data cache.
  cache: "no-store",
});

// Session forwarding (phase 13 CP3): the browser's session cookie names the
// user, and Go is the only place that resolves it — so every server-side
// call carries it upstream. Exactly one cookie crosses the boundary: Go has
// no use for any other, and not forwarding the rest keeps the API boundary
// as narrow as the architecture drawing says it is. A route handler that
// sets its own cookie header (the auth callback forwarding the state
// cookie) wins — the middleware only fills silence.
api.use({
  async onRequest({ request }) {
    if (request.headers.has("cookie")) return request;
    const session = (await cookies()).get("roadbook_session");
    if (session?.value) {
      request.headers.set("cookie", `roadbook_session=${session.value}`);
    }
    return request;
  },
});
