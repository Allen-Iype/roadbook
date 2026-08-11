import Link from "next/link";

// The shared page header for the workbench pages (candidates, imports,
// adventure detail). The life map page is not a consumer: it floats its own
// chrome over the map (BRIEF §0), so this stays a per-page component rather
// than living in the root layout.
export function SiteHeader({ undecided }: { undecided?: number }) {
  return (
    <header className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 border-b border-rule pb-4">
      <Link
        href="/"
        className="font-display text-lg font-semibold tracking-[0.28em] text-ink no-underline"
      >
        ROADBOOK
      </Link>
      <nav className="flex gap-5 text-sm text-ink-2">
        <Link
          href="/candidates"
          className="underline decoration-rule underline-offset-2 hover:text-ink"
        >
          Candidates
          {undecided !== undefined && undecided > 0 && <> — {undecided} undecided</>}
        </Link>
        <Link
          href="/imports"
          className="underline decoration-rule underline-offset-2 hover:text-ink"
        >
          Imports
        </Link>
      </nav>
    </header>
  );
}
