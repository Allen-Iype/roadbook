import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { getSession } from "@/lib/session";

// The sign-in page (phase 13 CP3): the public shell's door onto a
// multi-user instance. It renders exactly one action — the OIDC redirect —
// because there is exactly one way in; no form, no fields, no second
// path. On an authless instance this page does not exist as a surface:
// mode off redirects home, so no dead control is ever shown (the phase-9
// auth-ready rule, kept now that auth is real).
export const metadata: Metadata = {
  title: "Roadbook — sign in",
};

export const dynamic = "force-dynamic";

export default async function SignInPage({
  searchParams,
}: {
  // Next 16: searchParams is a Promise — await it (the params lesson from
  // phase 2, same shape).
  searchParams: Promise<{ error?: string }>;
}) {
  const session = await getSession();
  if (!session || session.mode === "off" || session.signed_in) {
    // Unreachable API falls through to the app's own API-down rendering;
    // an authless instance has nothing to sign in to; a signed-in visitor
    // is already through.
    redirect("/");
  }
  const { error } = await searchParams;

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

        {error && (
          <p
            role="alert"
            className="mt-6 max-w-[58ch] border border-rule bg-paper-2 px-3 py-2 text-sm text-ink"
          >
            Sign-in didn&apos;t complete: {error}
          </p>
        )}

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
      </section>
    </main>
  );
}
