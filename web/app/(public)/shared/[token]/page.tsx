import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

import { AdventureView } from "@/components/adventure/adventure-view";
import { api } from "@/lib/api/client";
import { MAP_STYLE_URL } from "@/lib/basemap";
import { sharedThumbs } from "@/lib/photo-display";

// The shared view (phase 13 CP4): what a share link opens. It lives in the
// public shell because its reader has no account here and needs none —
// the token in the URL is the whole credential, resolved by Go; this page
// consults no session and renders no account slot. What it draws is the
// owner's plate, unchanged: the same component, the same legs with the
// same kinds (invariants 5 and 8 apply to strangers most of all), the same
// photos. Only the surroundings differ — no controls, and the copy names
// whose adventure it is.
//
// Every state a link can be in that does not open — unknown, revoked,
// dismissed since, orphaned by re-detection — is one 404 from the API and
// one not-found page here: the outside learns nothing about which.
export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: "Roadbook — shared adventure",
  // Belt and braces with next.config's X-Robots-Tag header on /shared/*.
  robots: { index: false, follow: false },
};

export default async function SharedAdventurePage({
  params,
}: {
  params: Promise<{ token: string }>;
}) {
  const { token } = await params;

  // An unreachable API is a rejected fetch, not an HTTP error — it throws
  // (the phase 9 lesson: openapi-fetch's {error} is HTTP-only).
  let res: Awaited<ReturnType<typeof fetchShared>> | null;
  try {
    res = await fetchShared(token);
  } catch {
    res = null;
  }
  if (res === null) {
    return (
      <main className="mx-auto w-full max-w-3xl px-4 py-8 sm:px-6">
        <Masthead />
        <p className="mt-10 text-red-700">
          This Roadbook is not reachable right now. Try the link again in a
          little while.
        </p>
      </main>
    );
  }
  if (res.status === 404 || !res.data) notFound();
  const shared = res.data;

  return (
    <main className="mx-auto w-full max-w-[91rem] px-4 py-8 sm:px-6">
      <Masthead />
      <AdventureView
        journey={shared.journey}
        photos={shared.photos}
        importPhotos={shared.import_photos}
        styleUrl={MAP_STYLE_URL}
        plate={null}
        shared={{
          name: shared.name,
          start_truncated: shared.start_truncated,
          end_truncated: shared.end_truncated,
        }}
        thumbs={sharedThumbs(token)}
      />
      <footer className="mt-12 max-w-[58ch] border-t border-rule pt-4 text-sm leading-relaxed text-ink-2">
        This plate was shared from someone&apos;s Roadbook — an atlas drawn
        from their own location history, honest about what was recorded and
        what was inferred.{" "}
        <Link href="/welcome" className="text-ink underline decoration-rule underline-offset-2 hover:decoration-ink">
          How Roadbook works, and how to make your own.
        </Link>
      </footer>
    </main>
  );
}

async function fetchShared(token: string) {
  const { data, response } = await api.GET("/shared/{token}", {
    params: { path: { token } },
  });
  return { data, status: response.status };
}

// The public masthead: the wordmark leads to the pitch, not into an app
// the reader has no account for; the right-hand line says what this page
// is. No navigation, no account slot (the public shell never has one).
function Masthead() {
  return (
    <header className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 border-b border-rule pb-4">
      <Link
        href="/welcome"
        className="-mx-2 -my-2 px-2 py-2 font-display text-lg font-semibold tracking-[0.28em] text-ink no-underline"
      >
        ROADBOOK
      </Link>
      <span className="text-sm text-ink-2">A shared adventure</span>
    </header>
  );
}
