import Link from "next/link";

// The shared page header for the workbench pages (adventures, candidates,
// imports, adventure detail) and the front door. The life map page is not a
// consumer: it floats its own chrome over the map (BRIEF §0), so this stays
// a per-page component rather than living in the shell layout.
//
// `instanceLabel` is the reserved account slot (phase 9 BRIEF §6): the app
// pages pass the operator-set label through; the public shell passes
// nothing and the slot renders nothing. No auth surface — a label, not a
// control.
export function SiteHeader({
  undecided,
  active,
  instanceLabel,
}: {
  undecided?: number;
  active?: "adventures" | "candidates" | "imports";
  instanceLabel?: string;
}) {
  return (
    <header className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 border-b border-rule pb-4">
      {/* Negative-margin padding grows every link to the 44px a thumb
          needs without moving the rendered text (the phase 7 pattern; the
          padding matches the line height — text-lg 28px + py-2, text-sm
          20px + py-3). */}
      <Link
        href="/"
        className="-mx-2 -my-2 px-2 py-2 font-display text-lg font-semibold tracking-[0.28em] text-ink no-underline"
      >
        ROADBOOK
      </Link>
      <span className="flex flex-wrap items-baseline gap-x-5 gap-y-1">
        <nav className="flex gap-5 text-sm text-ink-2">
          <NavLink href="/adventures" current={active === "adventures"}>
            Adventures
          </NavLink>
          <NavLink href="/candidates" current={active === "candidates"}>
            Candidates
            {undecided !== undefined && undecided > 0 && (
              <> — {undecided} undecided</>
            )}
          </NavLink>
          <NavLink href="/imports" current={active === "imports"}>
            Imports
          </NavLink>
        </nav>
        {instanceLabel ? (
          <span className="border border-rule px-2 py-0.5 font-mono text-xs text-ink-2">
            {instanceLabel}
          </span>
        ) : null}
      </span>
    </header>
  );
}

function NavLink({
  href,
  current,
  children,
}: {
  href: string;
  current: boolean;
  children: React.ReactNode;
}) {
  return (
    <Link
      href={href}
      aria-current={current ? "page" : undefined}
      className={
        current
          ? "-mx-2 -my-3 px-2 py-3 font-semibold text-ink underline decoration-ink underline-offset-2"
          : "-mx-2 -my-3 px-2 py-3 underline decoration-rule underline-offset-2 hover:text-ink"
      }
    >
      {children}
    </Link>
  );
}
