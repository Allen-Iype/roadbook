// Sign-out proxy (phase 13 CP3): a plain form POST — no JavaScript needed
// to leave. Go deletes the session row (revocation is a DELETE) and hands
// back the clearing cookie; the browser lands on the sign-in page. The
// session cookie itself reaches Go through the client middleware.
import { api } from "@/lib/api/client";

export async function POST() {
  const { response } = await api.POST("/auth/signout", {});
  const headers = new Headers({ Location: "/signin" });
  const cookie = response?.headers.get("set-cookie");
  if (cookie) headers.set("Set-Cookie", cookie);
  return new Response(null, { status: 303, headers });
}
