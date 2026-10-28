import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { getSession } from "@/lib/session";
import { WAITLIST_URL } from "@/lib/waitlist";

// The sign-in page (phase 13 CP3): the public shell's door onto a
// multi-user instance. It renders exactly one action — the OIDC redirect —
// because there is exactly one way in; no form, no fields, no second
// path. On an authless instance this page does not exist as a surface:
// mode off redirects home, so no dead control is ever shown (the phase-9
// auth-ready rule, kept now that auth is real).
//
// The full state (phase 15 BRIEF §9): the API answers a first sign-in at
// the user cap with a redirect here carrying `full=1` and no cookie. The
// page then shows the sentence and the waitlist link and NO sign-in
// control — the provider is asked for `openid email` without
// prompt=select_account, so a second press would be auto-picked and
// refused again in a silent loop. No number is shown: this layer does not
// know the cap, and must not state what it cannot reproduce.
export const metadata: Metadata = {
  title: "Roadbook — sign in",
};

export const dynamic = "force-dynamic";

export default async function SignInPage({
  searchParams,
}: {
  // Next 16: searchParams is a Promise — await it (the params lesson from
  // phase 2, same shape).
  searchParams: Promise<{ error?: string; full?: string }>;
}) {
  const session = await getSession();
  if (!session || session.mode === "off" || session.signed_in) {
    // Unreachable API falls through to the app's own API-down rendering;
    // an authless instance has nothing to sign in to; a signed-in visitor
    // is already through.
    redirect("/");
  }
  const { error, full } = await searchParams;
  const isFull = Boolean(full);

  return (
    <main className="mx-auto w-full max-w-3xl px-4 py-8 sm:px-6">
      <header className="border-b border-rule pb-4">
        <span className="font-display text-lg font-semibold tracking-[0.28em] text-ink">
          ROADBOOK
        </span>
      </header>

      <section className="mt-14 sm:mt-20">
        <p className="text-[11.5px] uppercase tracking-[0.24em] text-ink-2">
          A road atlas of your own past
        </p>
        <h1 className="mt-2 font-display text-4xl font-semibold leading-[1.05]">
          Sign in
        </h1>
        <p className="mt-5 max-w-[58ch] text-[15px] leading-relaxed">
          This Roadbook keeps each person&apos;s adventures their own. Sign
          in and you&apos;ll only ever see yours.
        </p>

        {isFull && (
          <div
            role="status"
            className="mt-6 max-w-[58ch] border border-rule bg-paper-2 px-3 py-2 text-sm text-ink"
          >
            <p className="font-semibold">Roadbook is full for now.</p>
            <p className="mt-1">
              Every seat on this instance is taken, so your sign-in created
              nothing. Join the waitlist and one email arrives when a seat
              opens. Nothing else is ever sent.
            </p>
            {WAITLIST_URL && (
              <p className="mt-3">
                <a
                  href={WAITLIST_URL}
                  className="inline-block border border-ink px-5 py-3 font-display text-sm font-semibold tracking-[0.08em] text-ink no-underline hover:bg-paper"
                >
                  Join the waitlist
                </a>
              </p>
            )}
          </div>
        )}

        {!isFull && error && (
          <p
            role="alert"
            className="mt-6 max-w-[58ch] border border-rule bg-paper-2 px-3 py-2 text-sm text-ink"
          >
            Sign-in didn&apos;t complete: {error}
          </p>
        )}

        {!isFull && (
          <p className="mt-8">
            {/* A plain link, not a button-with-handler: the whole flow is
                browser navigation (this route answers with the provider
                redirect), so zero JavaScript is involved in getting in. */}
            <a
              href="/api/auth/signin"
              className="inline-block border border-ink px-5 py-3 font-display text-sm font-semibold tracking-[0.08em] text-ink no-underline hover:bg-paper-2"
            >
              Sign in to continue
            </a>
          </p>
        )}
      </section>
    </main>
  );
}
