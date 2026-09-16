// OIDC callback proxy (phase 13 CP3): the provider redirected the browser
// here; the code+state query and the state cookie forward to Go, which
// does the verification and answers with the session cookie. Success
// passes through; failure becomes a redirect to the sign-in page with the
// reason — a stranded JSON error body is not a page a person can act on.
import { api } from "@/lib/api/client";
import { cookies } from "next/headers";

export async function GET(request: Request) {
  const url = new URL(request.url);
  const state = (await cookies()).get("roadbook_auth_state");
  const { error, response } = await api.GET("/auth/callback", {
    params: {
      query: {
        code: url.searchParams.get("code") ?? undefined,
        state: url.searchParams.get("state") ?? undefined,
        error: url.searchParams.get("error") ?? undefined,
      },
    },
    headers: state
      ? { cookie: `roadbook_auth_state=${state.value}` }
      : undefined,
    redirect: "manual",
  });

  if (response.status === 302) {
    const headers = new Headers({
      Location: response.headers.get("location") ?? "/",
    });
    const cookie = response.headers.get("set-cookie");
    if (cookie) headers.set("Set-Cookie", cookie);
    return new Response(null, { status: 302, headers });
  }
  const reason = error?.error ?? "sign-in failed";
  return new Response(null, {
    status: 303,
    headers: { Location: "/signin?error=" + encodeURIComponent(reason) },
  });
}
