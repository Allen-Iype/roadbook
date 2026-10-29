import type { Metadata } from "next";
import Link from "next/link";

import { LegKindLegend } from "@/components/legend";
import { OpenOnHash } from "@/components/open-on-hash";
import { PhotoUploadImport } from "@/components/photo-upload-import";
import { SiteHeader } from "@/components/site-header";
import { Term } from "@/components/term";
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
    "What Roadbook is, how to export your Google Timeline, and where to upload it.",
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
          The adventures you actually took.
        </h1>
        <p className="mt-4 max-w-[58ch] text-[15px] leading-relaxed">
          Roadbook reads the record your phone already keeps of where it went,
          draws the trips that stood out, and leaves it to you to say which
          were adventures.
        </p>

        {/* The plate: the demo life map, framed the way the app frames its
            maps — double rule, marginalia below. Demo data only. */}
        <figure className="mt-6">
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
              Fictional demonstration data. Yours is drawn the same way, from
              your own record.
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
            minutes.
          </Step>
          <Step n={2}>Upload the file here.</Step>
          <Step n={3}>
            Confirm the trips that were adventures; the rest stay off the map.
          </Step>
        </ol>

        <div id={S.upload} className="mt-6 scroll-mt-6">
          <UploadImport variant="door" />
        </div>

        {/* The walkthroughs, folded (BRIEF §4 CP2): each platform is a
            <details>; its verification state stays visible page text
            (phase 7 BRIEF §3E) on the summary line, so a stale date is
            still read before the steps are. */}
        <div id={S.export} className="mt-8 scroll-mt-6">
          <h3 className="font-display text-lg font-semibold">
            Don&apos;t have the file yet?
          </h3>
          <p className="mt-1 max-w-[58ch] text-sm leading-relaxed text-ink-2">
            Google makes the export on the phone itself. Do it on the phone
            you&apos;re reading this on and the file is already where the
            button above can find it.
          </p>

          <Walkthrough
            id={S.exportAndroid}
            title="On Android"
            verified="Verified 11 Aug 2026"
            verifiedNote="Walked 11 Aug 2026 on an Android device from a WhatsApp-opened link: the steps matched the phone's menus and the upload completed in the in-app browser."
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

          {/* The honest third branch (phase 7 BRIEF §3E): the page never
              implies data exists that does not. Since phase 11 it points
              at the second door instead of ending. */}
          <p
            id={S.neverEnabled}
            className="mt-4 max-w-[58ch] scroll-mt-6 text-sm leading-relaxed text-ink-2"
          >
            <strong className="font-semibold text-ink">Never had Timeline on?</strong>{" "}
            Then there is nothing to export — Google holds no location record
            for you.{" "}
            <a
              href={`#${S.photos}`}
              className="underline decoration-rule underline-offset-2 hover:text-ink"
            >
              Start from your photos
            </a>{" "}
            instead.
          </p>
        </div>
      </section>

      {/* ── 3 · The second door: photos (phase 11) ────────────────────── */}
      <section id={S.photos} className="mt-12 scroll-mt-6 border-t border-ink pt-6">
        <h2 className="font-display text-2xl font-semibold">
          Or start from photos
        </h2>
        <p className="mt-3 max-w-[58ch] text-[15px] leading-relaxed">
          A photo straight from the camera carries where and when it was
          taken, and that is enough to find your trips — no export step, the
          picker opens your photo library.{" "}
          <Term id="originals">Original photos</Term> work best.
        </p>
        <div className="mt-5">
          <PhotoUploadImport variant="door" />
        </div>

        <Fold title="Which photos work" last>
          <ul className="list-disc space-y-1.5 pl-5">
            <li>
              Camera originals, and downloads from your own Google Photos.
              iPhone photos are usually HEIC — upload them as they are; if
              the picker offers JPEG, either works.
            </li>
            <li>
              Photos received in a chat almost never carry a location —
              messaging apps strip it. Screenshots never have one.
            </li>
            <li>
              A Google Photos Takeout includes small{" "}
              <code className="font-mono">.json</code> files beside the
              photos. Upload them together: they can restore a location the
              photo file lost.
            </li>
            <li>
              On their own, photos need a little context: home is worked out
              from where your photos recur, so include a spread from ordinary
              weeks too, not only the trips. With a Timeline export already
              in, photos simply add detail to the adventures it found.
            </li>
          </ul>
          <p className="mt-3 text-xs text-ink-2">
            Photo intake verified 26 Aug 2026 with a real camera roll. Not
            yet walked on a phone — the iPhone notes come from documentation,
            not a device.
          </p>
        </Fold>
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
              Got a <code className="font-mono">.zip</code> archive? Extract
              it and upload the <code className="font-mono">.json</code> file
              inside.
            </li>
            <li>
              Files from a Google Takeout download — monthly &ldquo;Semantic
              Location History&rdquo; files or{" "}
              <code className="font-mono">Records.json</code> — are the old
              server-side format, which Roadbook does not read. The phone
              export is the current one.
            </li>
            <li>
              Photos, PDFs and KML files are not location exports — the
              upload will say so and point back here.
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
            Your file streams to this Roadbook and nowhere else, and is kept
            here as-is so later versions can re-read it without asking you
            for the file again. Photos are not kept — only the position, the
            time and a small thumbnail of each. Everything you upload is
            listed on{" "}
            <Link href="/imports" className="underline decoration-rule underline-offset-2 hover:text-ink">
              Imports
            </Link>
            , and can be deleted from there.
          </p>
        </Fold>

        <Fold id={S.next} title="What happens after you upload" last>
          <ol className="space-y-2">
            <Step n={1}>
              <strong className="font-semibold text-ink">Upload.</strong>{" "}
              Nothing is recorded until the whole file has arrived — if the
              connection drops, just try again.
            </Step>
            <Step n={2}>
              <strong className="font-semibold text-ink">Import.</strong>{" "}
              Every visit, journey and recorded position is read in; each
              geotagged photo contributes the position it carries.
            </Step>
            <Step n={3}>
              <strong className="font-semibold text-ink">Detection.</strong>{" "}
              Roadbook looks for journeys far from home with a real
              destination — somewhere you stayed, not somewhere you passed
              through — and lists them as{" "}
              <Term id="candidate">candidates</Term>. Home is worked out from
              the data itself.
            </Step>
            <Step n={4}>
              <strong className="font-semibold text-ink">You decide.</strong>{" "}
              Confirm the real adventures and name them; dismiss the rest.
              The confirmed ones are drawn on your life map and shelved as{" "}
              <Term id="plate">plates</Term> in your atlas.
            </Step>
          </ol>
          <p className="mt-3">
            Once the upload finishes you can close the tab —{" "}
            <Link href="/imports" className="underline decoration-rule underline-offset-2 hover:text-ink">
              Imports
            </Link>{" "}
            keeps every attempt and its outcome.
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
// ordered list is the honest structure. Each branch carries its
// verification state as visible page text (phase 7 BRIEF §3E) on the
// summary line, before the fold: a real device pass stamps `verified`; a
// drafted-but-unwalked branch says "unverified" rather than carrying a
// false date. A stale verified date is itself information — Google changes
// these menus without notice, and the date says when they last matched.
function Walkthrough({
  id,
  title,
  steps,
  verified,
  verifiedNote,
}: {
  id: string;
  title: string;
  steps: React.ReactNode[];
  /** Summary-line stamp once the device pass has walked this branch. */
  verified?: string;
  /** Footer sentence recording what the pass checked. */
  verifiedNote?: string;
}) {
  return (
    <details id={id} className="group mt-3 max-w-prose scroll-mt-6 border border-rule bg-paper">
      <summary className="flex cursor-pointer list-none flex-wrap items-baseline gap-x-3 gap-y-1 bg-land px-4 py-2.5 marker:hidden">
        <span aria-hidden className="font-mono text-sm text-ink-2 transition-transform group-open:rotate-90">
          ▸
        </span>
        <span className="font-display text-lg font-semibold">{title}</span>
        <span className="ml-auto text-[11px] uppercase tracking-[0.14em] text-ink-2">
          {verified ?? "Unverified"}
        </span>
      </summary>
      <ol className="space-y-2.5 border-t border-rule px-4 py-3.5 text-sm leading-relaxed">
        {steps.map((step, i) => (
          <li key={i} className="flex gap-3">
            <span className="font-mono text-ink-2">{i + 1}.</span>
            <span>{step}</span>
          </li>
        ))}
      </ol>
      <p className="border-t border-rule px-4 py-2 text-xs text-ink-2">
        {verifiedNote ??
          "Drafted 11 Aug 2026 from documentation, not yet verified on a device — Google changes these menus without notice."}
      </p>
    </details>
  );
}
