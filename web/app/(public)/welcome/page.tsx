import type { Metadata } from "next";
import Link from "next/link";

import { LegendInfo } from "@/components/legend-info";
import { OpenOnHash } from "@/components/open-on-hash";
import { SiteHeader } from "@/components/site-header";
import { UploadImport } from "@/components/upload-import";
import { WelcomePlate } from "@/components/welcome-plate";
import { WELCOME_SECTIONS as S } from "@/lib/rejection-anchors";

// The front door (phase 7 BRIEF §3E; reshaped in phase 16 CP2). The one
// page that carries a cold visitor — typically on a phone, from a shared
// link — from "what is this?" to their own candidates. Phase 16's research
// measured it at eleven hundred words before the upload control
// (RESEARCH §2); the maintainer's finding was that the page has to make a
// person WANT the next step, not explain it harder (RESEARCH §9.1). So the
// order is now: the plate (the payoff, drawn) with one line; the three
// steps to a plate of your own, with the one primary action — the
// Timeline file picker — directly under them; everything that used to be
// a section is a native <details> below, opened on demand. The photos door
// stays a door of its own, second.
//
// Statically rendered, deliberately: the page reads nothing from the API
// (DECISIONS 2026-08-11) — outcome states live where outcomes show
// (/candidates, the life map), and the two upload islands carry all
// interactivity. The one other script is OpenOnHash, which opens a folded
// section when a rejection link targets it. `/` redirects here while the
// instance has zero imports.
export const metadata: Metadata = {
  title: "Roadbook — start here",
  description:
    "Every trip you've taken, found from the location history your phone already keeps.",
};

// Section ids come from the same module the rejection-redirection map types
// its anchors against (lib/rejection-anchors.ts) — a dead-end anchor cannot
// be written. Every id still has a real target; the folded ones are
// <details> elements, which OpenOnHash opens when a link lands on them.
export default function WelcomePage() {
  return (
    <main className="mx-auto w-full max-w-3xl px-4 py-8 sm:px-6">
      <SiteHeader />
      <OpenOnHash />

      {/* ── 1 · The plate, with one line ──────────────────────────────── */}
      <section className="mt-8 sm:mt-12">
        <h1 className="font-display text-4xl font-semibold leading-[1.05] sm:text-5xl">
          Every trip you&apos;ve taken, on one map.
        </h1>
        <p className="mt-4 max-w-[58ch] text-[15px] leading-relaxed">
          Your phone has quietly kept a record of everywhere you&apos;ve
          been. Upload it and your trips are found for you — no diary, no
          logging. You choose which ones to keep.
        </p>

        {/* The plate: the demo life map, framed the way the app frames its
            maps — double rule, marginalia below. Demo data only. */}
        <figure className="mt-6">
          <div className="border border-ink [outline:1px_solid_var(--color-ink)] [outline-offset:3px]">
            <WelcomePlate />
          </div>
          <figcaption className="mt-1 border border-t-0 border-ink bg-paper px-3 py-2.5 [outline:1px_solid_var(--color-ink)] [outline-offset:3px] sm:px-4">
            <p className="font-display text-[11px] font-semibold tracking-[0.22em]">
              SAMPLE · THREE TRIPS ACROSS ICELAND
            </p>
            <div className="mt-2">
              <LegendInfo />
            </div>
            <p className="mt-2 text-xs text-ink-2">
              Sample trips. Yours will look like this.
            </p>
          </figcaption>
        </figure>
      </section>

      {/* ── 2 · Your own, in three steps + the primary action ─────────── */}
      <section id={S.whatYouNeed} className="mt-12 scroll-mt-6 border-t border-ink pt-6">
        <h2 className="font-display text-2xl font-semibold">
          Your own, in three steps
        </h2>
        {/* A real sequence, so the numbers carry order. */}
        <ol className="mt-4 max-w-[58ch] space-y-2 text-[15px] leading-relaxed">
          <Step n={1}>
            Export your Timeline from Google Maps on your phone — a couple of
            minutes; how is below.
          </Step>
          <Step n={2}>Upload the file here.</Step>
          <Step n={3}>
            Keep the trips that mattered; the rest stay off the map.
          </Step>
        </ol>

        <div id={S.upload} className="mt-6 scroll-mt-6">
          <UploadImport variant="door" />
        </div>

        {/* The walkthroughs, folded (BRIEF §4 CP2): each platform is a
            <details>. The copy pass (maintainer's review, 2026-11-02) took
            the subheading, the intro and the summary-line stamp away — the
            steps say the same thing — and the check date now sits inside
            each fold as its last line. */}
        <div id={S.export} className="mt-6 scroll-mt-6">

          <Walkthrough
            id={S.exportAndroid}
            title="On Android"
            verifiedNote="Checked 11 Aug 2026 on an Android phone, opened from a WhatsApp link: the menus matched these steps and the upload worked from the browser inside WhatsApp."
            steps={[
              <>
                Open <strong>Settings</strong> → <strong>Location</strong> →{" "}
                <strong>Location Services</strong> → <strong>Timeline</strong>.
                (Wording differs slightly between Android versions; Timeline
                settings can also be reached from Google Maps → your profile
                picture → Your Timeline.)
              </>,
              <>
                Choose <strong>Export Timeline data</strong> and confirm.
              </>,
              <>
                The phone saves a <code className="font-mono">.json</code>{" "}
                file to <strong>Downloads</strong>.
              </>,
              <>
                Come back here and{" "}
                <a href={`#${S.upload}`} className="underline decoration-rule underline-offset-2">
                  choose the file
                </a>{" "}
                with the button above — the picker opens straight into
                Downloads.
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
                Come back here and{" "}
                <a href={`#${S.upload}`} className="underline decoration-rule underline-offset-2">
                  choose the file
                </a>{" "}
                with the button above, picking it from the Files app.
              </>,
            ]}
          />

        </div>
      </section>


      {/* ── 4 · The rest, folded ──────────────────────────────────────── */}
      <section className="mt-12 border-t border-ink pt-2">
        <Fold id={S.theFile} title="What the right file looks like">
          <p>
            A single <code className="font-mono">.json</code> file, usually
            named <code className="font-mono">Timeline.json</code>. On
            Android it lands in the phone&apos;s <strong>Downloads</strong>{" "}
            folder; on iPhone it goes wherever you saved it from the share
            sheet, typically the Files app.
          </p>
          <ul className="mt-3 list-disc space-y-1.5 pl-5">
            <li>
              Got a <code className="font-mono">.zip</code> file? Open it and
              upload the <code className="font-mono">.json</code> file inside.
            </li>
            <li>
              Files from an older Google download (Takeout: monthly
              &ldquo;Semantic Location History&rdquo; files, or{" "}
              <code className="font-mono">Records.json</code>) are a format
              Roadbook can&apos;t read. The file from the phone is the one
              that works.
            </li>
            <li>
              A photo, a PDF or a map file isn&apos;t your Timeline — the
              upload will say so and point you back here.
            </li>
            <li>
              Using someone else&apos;s Roadbook link on a computer? Export
              on the phone first and move the file over — or open the same
              link on the phone; everything here works there.
            </li>
          </ul>
        </Fold>

        <Fold title="Where your data goes">
          <p>
            Your file goes straight to this Roadbook and nowhere else. It is
            kept as it is, so you never have to upload it again. Photos are
            not kept — only where and when each was taken, and a small
            preview. Everything you&apos;ve uploaded is listed under{" "}
            <Link href="/imports" className="underline decoration-rule underline-offset-2 hover:text-ink">
              Uploads
            </Link>
            , and can be deleted from there.
          </p>
        </Fold>

        <Fold id={S.next} title="What happens next" last>
          <ol className="space-y-2">
            <Step n={1}>
              <strong className="font-semibold text-ink">Upload.</strong>{" "}
              Nothing is saved until the whole file has arrived — if the
              connection drops, just try again.
            </Step>
            <Step n={2}>
              <strong className="font-semibold text-ink">Your trips are found.</strong>{" "}
              Roadbook reads the file and picks out the trips: time well away
              from home, with somewhere you stayed. Home is worked out from
              your own data.
            </Step>
            <Step n={3}>
              <strong className="font-semibold text-ink">You choose.</strong>{" "}
              Keep the real trips and name them; leave out the rest. The kept
              ones are drawn on your map.
            </Step>
          </ol>
          <p className="mt-3">
            Once the upload finishes you can close the tab —{" "}
            <Link href="/imports" className="underline decoration-rule underline-offset-2 hover:text-ink">
              Uploads
            </Link>{" "}
            keeps every attempt and how it went.
          </p>
        </Fold>
      </section>
    </main>
  );
}

// One numbered line. Mono numeral, hanging — the same mark the adventure
// page's day headings use for sequence.
function Step({ n, children }: { n: number; children: React.ReactNode }) {
  return (
    <li className="flex gap-3">
      <span className="font-mono text-ink-2">{n}.</span>
      <span>{children}</span>
    </li>
  );
}

// A folded section: native <details>, closed by default, no script. The
// summary is the one line a person reads; its padding makes it a 44px
// target at phone width (text-lg 28px + py-2.5). The id, when given, is a
// rejection anchor — OpenOnHash opens the fold when a link lands on it.
function Fold({
  id,
  title,
  last = false,
  children,
}: {
  id?: string;
  title: string;
  /** The last fold before a section rule: no rule of its own. */
  last?: boolean;
  children: React.ReactNode;
}) {
  return (
    <details
      id={id}
      className={`group mt-4 scroll-mt-6 pb-4 first:mt-0 ${last ? "" : "border-b border-rule"}`}
    >
      <summary className="flex cursor-pointer list-none items-baseline gap-3 py-2.5 font-display text-lg font-semibold marker:hidden">
        <span aria-hidden className="font-mono text-sm text-ink-2 transition-transform group-open:rotate-90">
          ▸
        </span>
        {title}
      </summary>
      <div className="max-w-[58ch] pl-6 text-sm leading-relaxed text-ink-2">
        {children}
      </div>
    </details>
  );
}

// One platform walkthrough, folded. Steps are a real sequence, so an
// ordered list is the honest structure. Each branch still carries its
// verification state as visible page text (phase 7 BRIEF §3E), now as the
// fold's last line rather than a stamp on the summary (copy pass,
// maintainer's review 2026-11-02): a real device pass says when it was
// checked; a drafted-but-unwalked branch says so rather than carrying a
// false date. Google changes these menus without notice, and the date
// says when they last matched.
function Walkthrough({
  id,
  title,
  steps,
  verifiedNote,
}: {
  id: string;
  title: string;
  steps: React.ReactNode[];
  /** Footer sentence recording what the device pass checked. */
  verifiedNote?: string;
}) {
  return (
    <details id={id} className="group mt-2 max-w-prose scroll-mt-6 border-t border-rule">
      <summary className="flex cursor-pointer list-none items-baseline gap-x-3 py-3 marker:hidden">
        <span aria-hidden className="font-mono text-sm text-ink-2 transition-transform group-open:rotate-90">
          ▸
        </span>
        <span className="font-display text-lg font-semibold">{title}</span>
      </summary>
      <ol className="space-y-2.5 pb-3.5 pl-7 text-sm leading-relaxed">
        {steps.map((step, i) => (
          <li key={i} className="flex gap-3">
            <span className="font-mono text-ink-2">{i + 1}.</span>
            <span>{step}</span>
          </li>
        ))}
      </ol>
      <p className="pb-3 pl-7 text-xs text-ink-2">
        {verifiedNote ??
          "Written 11 Aug 2026 from Google's own instructions, not yet checked on an iPhone — Google changes these menus without notice."}
      </p>
    </details>
  );
}
