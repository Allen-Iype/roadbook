import type { Metadata } from "next";
import Link from "next/link";

import { LegKindLegend } from "@/components/legend";
import { SiteHeader } from "@/components/site-header";
import { UploadImport } from "@/components/upload-import";
import { WelcomePlate } from "@/components/welcome-plate";
import { WELCOME_SECTIONS as S } from "@/lib/rejection-anchors";

// The front door (phase 7 BRIEF §3E): the one page that carries a cold
// visitor — typically on a phone, from a shared link — from "what is this?"
// to their own candidates. Section order is fixed by the cold-visitor test:
// each section answers the question the previous one raises. Pitch first,
// before asking for anything; the honest never-enabled branch before the
// walkthroughs; the upload only after the file exists.
//
// Statically rendered, deliberately: the page reads nothing from the API
// (DECISIONS 2026-08-11) — outcome states live where outcomes show
// (/candidates, the life map), and the embedded upload island carries all
// interactivity. `/` redirects here while the instance has zero imports.
export const metadata: Metadata = {
  title: "Roadbook — start here",
  description:
    "What Roadbook is, how to export your Google Timeline, and where to upload it.",
};

// Section ids come from the same module the rejection-redirection map types
// its anchors against (lib/rejection-anchors.ts) — a dead-end anchor cannot
// be written.
export default function WelcomePage() {
  return (
    <main className="mx-auto w-full max-w-3xl px-4 py-8 sm:px-6">
      <SiteHeader />

      {/* ── 1 · The pitch ─────────────────────────────────────────────── */}
      <section className="mt-10 sm:mt-14">
        <p className="text-[11.5px] uppercase tracking-[0.24em] text-ink-2">
          A road atlas of your own past
        </p>
        <h1 className="mt-2 font-display text-4xl font-semibold leading-[1.05] sm:text-5xl">
          The adventures you actually took.
        </h1>
        <p className="mt-5 max-w-[58ch] text-[15px] leading-relaxed">
          Your phone has been keeping a record of where it went — Google Maps
          calls it Timeline. Roadbook reads one Timeline export and finds the
          journeys that stand out: stretches far from home, with a real
          destination. You confirm the ones that mattered and name them; each
          becomes a route on a life map of your own.
        </p>
        <p className="mt-3 max-w-[58ch] text-[15px] leading-relaxed">
          The maps are honest about what they know. What your phone recorded
          is drawn differently from what was inferred along roads — and where
          nothing is known, the map says so instead of guessing.
        </p>

        {/* The plate: the demo life map, framed the way the app frames its
            maps — double rule, marginalia below. Demo data only. */}
        <figure className="mt-8">
          <div className="border border-ink [outline:1px_solid_var(--color-ink)] [outline-offset:3px]">
            <WelcomePlate />
          </div>
          <figcaption className="mt-1 border border-t-0 border-ink bg-paper px-3 py-2.5 [outline:1px_solid_var(--color-ink)] [outline-offset:3px] sm:px-4">
            <p className="font-display text-[11px] font-semibold tracking-[0.22em]">
              THREE ADVENTURES ACROSS ICELAND · DEMONSTRATION TIMELINE
            </p>
            <div className="mt-2">
              <LegKindLegend />
            </div>
            <p className="mt-2 text-xs text-ink-2">
              Fictional demonstration data. Your map is drawn the same way,
              from your own export.
            </p>
          </figcaption>
        </figure>

        {/* Jump row: a returning visitor with the file in hand should not
            re-read the pitch to find the upload control. */}
        <nav
          aria-label="Sections of this page"
          className="mt-6 flex flex-wrap gap-x-5 gap-y-1 text-sm"
        >
          <a href={`#${S.whatYouNeed}`} className="underline decoration-rule underline-offset-2 hover:text-ink">
            What you need
          </a>
          <a href={`#${S.export}`} className="underline decoration-rule underline-offset-2 hover:text-ink">
            Export it from your phone
          </a>
          <a href={`#${S.upload}`} className="underline decoration-rule underline-offset-2 hover:text-ink">
            Upload
          </a>
        </nav>
      </section>

      {/* ── 2 · What you need ─────────────────────────────────────────── */}
      <section id={S.whatYouNeed} className="mt-12 scroll-mt-6 border-t border-ink pt-6">
        <h2 className="font-display text-2xl font-semibold">What you need</h2>
        <p className="mt-3 max-w-[58ch] text-[15px] leading-relaxed">
          One file: a <strong>Timeline export</strong>
          {/* explicit space: SWC drops the plain one at this boundary */}{" "}
          from Google Maps on your phone. Timeline (formerly Location History) is off unless you —
          or a setup screen, years ago — turned it on. It moved from
          Google&apos;s servers onto the phone itself in 2024, which is why
          the export is
          made on the device.
        </p>

        {/* The honest third branch, stated before the walkthroughs (BRIEF
            §3E): this page never implies data exists that does not. */}
        <div
          id={S.neverEnabled}
          className="mt-5 max-w-prose scroll-mt-6 border-l-[3px] border-rule bg-land px-4 py-3"
        >
          <p className="text-[15px] font-semibold">Never had Timeline on?</p>
          <p className="mt-1 text-sm leading-relaxed text-ink-2">
            Then there is nothing to export — Google holds no location record
            for you, and Roadbook has nothing to read. No tool can recover
            journeys that were never recorded. Turning Timeline on now starts
            the record from today: an export made after your next real
            journey would hold it.
          </p>
        </div>

        <div id={S.theFile} className="mt-6 scroll-mt-6">
          <h3 className="font-display text-lg font-semibold">
            What the right file looks like
          </h3>
          <p className="mt-2 max-w-[58ch] text-sm leading-relaxed">
            A single <code className="font-mono">.json</code> file, usually
            named <code className="font-mono">Timeline.json</code>. On
            Android it lands in the phone&apos;s{" "}
            <strong>Downloads</strong> folder; on iPhone it goes wherever you
            save it from the share sheet, typically the Files app.
          </p>
          <ul className="mt-3 max-w-[58ch] list-disc space-y-1.5 pl-5 text-sm leading-relaxed text-ink-2">
            <li>
              Got a <code className="font-mono">.zip</code> archive? Extract
              it and upload the <code className="font-mono">.json</code> file
              inside.
            </li>
            <li>
              Files from a Google Takeout download — monthly &ldquo;Semantic
              Location History&rdquo; files or{" "}
              <code className="font-mono">Records.json</code> — are the old
              server-side format, which Roadbook does not read. The phone
              export below is the current one.
            </li>
            <li>
              Photos, PDFs, and KML files are not location exports — the
              upload will say so and point back here.
            </li>
          </ul>
        </div>
      </section>

      {/* ── 3 · The walkthroughs ──────────────────────────────────────── */}
      <section id={S.export} className="mt-12 scroll-mt-6 border-t border-ink pt-6">
        <h2 className="font-display text-2xl font-semibold">
          Export it from your phone
        </h2>
        <p className="mt-3 max-w-[58ch] text-sm leading-relaxed text-ink-2">
          The export is created on the phone itself and takes a couple of
          minutes. Do it on the same phone you&apos;re reading this on and the
          file is already where the upload below can find it.
        </p>

        <Walkthrough
          id={S.exportAndroid}
          title="On Android"
          steps={[
            <>
              Open <strong>Settings</strong> →{" "}
              <strong>Location</strong> → <strong>Location Services</strong>{" "}
              → <strong>Timeline</strong>. (Wording differs slightly between
              Android versions; Timeline settings can also be reached from
              Google Maps → your profile picture → Your Timeline.)
            </>,
            <>
              Choose <strong>Export Timeline data</strong> and confirm.
            </>,
            <>
              The phone saves a <code className="font-mono">.json</code> file
              to <strong>Downloads</strong>.
            </>,
            <>
              Come back to this page and{" "}
              <a href={`#${S.upload}`} className="underline decoration-rule underline-offset-2">
                upload it below
              </a>{" "}
              — the file picker opens straight into Downloads.
            </>,
          ]}
        />

        <Walkthrough
          id={S.exportIphone}
          title="On iPhone"
          steps={[
            <>
              Open the <strong>Google Maps</strong> app and tap your profile
              picture → <strong>Your Timeline</strong>.
            </>,
            <>
              Open Timeline settings (the <strong>⋯</strong> or gear button)
              and choose <strong>Export Timeline data</strong>.
            </>,
            <>
              Save the file from the share sheet — <strong>Save to
              Files</strong> is the simplest place to find it again.
            </>,
            <>
              Come back to this page and{" "}
              <a href={`#${S.upload}`} className="underline decoration-rule underline-offset-2">
                upload it below
              </a>
              , picking the file from the Files app.
            </>,
          ]}
        />

        <p className="mt-5 max-w-[58ch] text-sm leading-relaxed text-ink-2">
          Using someone else&apos;s Roadbook link on a computer instead? Export on
          the phone first, then move the file over — or simply open the same
          link on the phone; everything here works there.
        </p>
      </section>

      {/* ── 4 · The upload ────────────────────────────────────────────── */}
      <section id={S.upload} className="mt-12 scroll-mt-6 border-t border-ink pt-6">
        <h2 className="font-display text-2xl font-semibold">Upload it</h2>
        <UploadImport />
      </section>

      {/* ── 5 · What happens next ─────────────────────────────────────── */}
      <section id={S.next} className="mt-12 scroll-mt-6 border-t border-ink pt-6">
        <h2 className="font-display text-2xl font-semibold">
          What happens next
        </h2>
        <ol className="mt-4 max-w-[58ch] space-y-3 text-[15px] leading-relaxed">
          <NextStep n={1} title="Upload">
            The file streams to this Roadbook instance and nowhere else.
            Nothing is recorded until it has fully arrived — if the
            connection drops, just try again.
          </NextStep>
          <NextStep n={2} title="Import">
            Every visit, activity, and recorded point is read in. Your export
            is stored on the instance as-is, so future versions of Roadbook
            can re-read it without asking you for the file again.
          </NextStep>
          <NextStep n={3} title="Detection">
            Roadbook looks for journeys far from home with a real destination
            — somewhere you stayed, not somewhere you passed through — and
            lists them as candidates. Home is worked out from the data
            itself.
          </NextStep>
          <NextStep n={4} title="You decide">
            Confirm the real adventures and name them; dismiss the rest. The
            confirmed ones are drawn on your life map.
          </NextStep>
        </ol>
        <p className="mt-5 max-w-[58ch] text-sm leading-relaxed text-ink-2">
          Once the upload finishes you can close the tab —{" "}
          <Link href="/imports" className="underline decoration-rule underline-offset-2 hover:text-ink">
            Imports
          </Link>{" "}
          keeps every attempt and its outcome, and is the place to come back
          to.
        </p>
      </section>
    </main>
  );
}

// One platform walkthrough. Steps are a real sequence, so an ordered list is
// the honest structure. Each branch carries its verification state as
// visible page text (BRIEF §3E): a drafted-but-unverified walkthrough says
// so rather than carrying a false date — the CP4 device pass replaces the
// marker with "verified <date>, <device>".
function Walkthrough({
  id,
  title,
  steps,
}: {
  id: string;
  title: string;
  steps: React.ReactNode[];
}) {
  return (
    <div id={id} className="mt-6 max-w-prose scroll-mt-6 border border-rule bg-paper">
      <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 border-b border-rule bg-land px-4 py-2.5">
        <h3 className="font-display text-lg font-semibold">{title}</h3>
        <p className="text-[11px] uppercase tracking-[0.14em] text-ink-2">
          Unverified — steps may have changed
        </p>
      </div>
      <ol className="space-y-2.5 px-4 py-3.5 text-sm leading-relaxed">
        {steps.map((step, i) => (
          <li key={i} className="flex gap-3">
            <span className="font-mono text-ink-2">{i + 1}.</span>
            <span>{step}</span>
          </li>
        ))}
      </ol>
      <p className="border-t border-rule px-4 py-2 text-xs text-ink-2">
        Drafted 11 Aug 2026 from documentation, not yet verified on a device
        — Google changes these menus without notice.
      </p>
    </div>
  );
}

function NextStep({
  n,
  title,
  children,
}: {
  n: number;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <li className="flex gap-3">
      <span className="font-mono text-ink-2">{n}.</span>
      <span>
        <strong className="font-semibold">{title}.</strong>{" "}
        <span className="text-ink-2">{children}</span>
      </span>
    </li>
  );
}
