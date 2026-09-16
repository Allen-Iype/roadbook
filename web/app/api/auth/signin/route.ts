// Sign-in proxy (phase 13 CP3): the browser navigates here; Go answers
// with the provider redirect and the state cookie, and both pass through
// verbatim. `redirect: "manual"` keeps fetch from following the 302 into
// the provider — the BROWSER must make that hop, carrying its own cookies.
// A 409 means the instance runs authless; there is nothing to sign in to,
// so the visitor goes home.
import { api } from "@/lib/api/client";

export async function GET() {
  const { response } = await api.GET("/auth/signin", { redirect: "manual" });
  if (response.status !== 302) {
    return new Response(null, { status: 303, headers: { Location: "/" } });
  }
  const headers = new Headers({
    Location: response.headers.get("location") ?? "/",
  });
  const cookie = response.headers.get("set-cookie");
  if (cookie) headers.set("Set-Cookie", cookie);
  return new Response(null, { status: 302, headers });
}
