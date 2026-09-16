import Link from "next/link";

// A link that does not open (phase 13 CP4). One page for every reason —
// revoked, never existed, the adventure dismissed or re-detected away —
// because the reader cannot act on the difference, and telling would leak
// which links exist. What they CAN do is ask for a fresh one.
export default function SharedNotFound() {
  return (
    <main className="mx-auto w-full max-w-3xl px-4 py-8 sm:px-6">
      <header className="border-b border-rule pb-4">
        <Link
          href="/welcome"
          className="-mx-2 -my-2 px-2 py-2 font-display text-lg font-semibold tracking-[0.28em] text-ink no-underline"
        >
          ROADBOOK
        </Link>
      </header>
      <section className="mt-14 sm:mt-20">
        <h1 className="font-display text-4xl font-semibold leading-[1.05]">
          This link doesn&apos;t open anything.
        </h1>
        <p className="mt-5 max-w-[58ch] text-[15px] leading-relaxed">
          Whoever shared it may have revoked it, or the adventure it led to is
          no longer available. If someone sent you this link, ask them for a
          fresh one.
        </p>
        <p className="mt-8 text-sm">
          <Link href="/welcome" className="text-ink underline decoration-rule underline-offset-2 hover:decoration-ink">
            What Roadbook is
          </Link>
        </p>
      </section>
    </main>
  );
}
