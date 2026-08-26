// Rejection as redirection (phase 7 BRIEF §3E): every stable format label the
// sniffer can attach to a rejected upload maps to a section of the front door
// page, so a rejection always lands the user on the next thing to try. This is
// presentation of an API-provided label — the Go taxonomy explains what the
// file was (its message travels with the rejection); this map only says where
// on /welcome the fix lives. It never rewords the taxonomy.
//
// The label list mirrors internal/timeline/parse.go: thirteen synchronous
// sniff labels, plus the two only discoverable mid-parse (truncated,
// json-unrecognised), which arrive on a failed imports row rather than in the
// upload response. rejection-anchors.test.ts walks all fifteen — the
// no-dead-ends regression; it fails visibly if the Go taxonomy grows past
// this map.

// The front door's section ids, the single source of truth: the page renders
// its id= attributes from this object and the map below types its anchors
// against it, so a dead-end anchor cannot be written.
export const WELCOME_SECTIONS = {
  whatYouNeed: "what-you-need",
  neverEnabled: "never-enabled",
  theFile: "the-file",
  export: "export",
  exportAndroid: "export-android",
  exportIphone: "export-iphone",
  upload: "upload",
  photos: "photos",
  next: "what-happens-next",
} as const;

type Anchor = (typeof WELCOME_SECTIONS)[keyof typeof WELCOME_SECTIONS];

export type Redirection = {
  /** Section id on /welcome (no leading #). */
  anchor: Anchor;
  /** Link text: where the fix lives, not a re-explanation of the rejection. */
  link: string;
};

// Four destinations cover the taxonomy: old or adjacent Google products →
// the phone-export walkthroughs; everything that is simply the wrong file →
// what the right file looks like; an incomplete file → export it again; an
// image dropped into the Timeline upload → the photos section, because since
// phase 11 photos are an import source of their own, not a wrong file.
const OLD_FORMAT: Redirection = {
  anchor: WELCOME_SECTIONS.export,
  link: "How to export the current format from your phone",
};
const WRONG_FILE: Redirection = {
  anchor: WELCOME_SECTIONS.theFile,
  link: "What the right file looks like",
};
const INCOMPLETE: Redirection = {
  anchor: WELCOME_SECTIONS.export,
  link: "Export a fresh copy from your phone and upload that",
};
const A_PHOTO: Redirection = {
  anchor: WELCOME_SECTIONS.photos,
  link: "Photos import separately — add them in the photos section",
};

export const REJECTION_REDIRECTS: Record<string, Redirection> = {
  // Old Google products with a current phone-export answer.
  "semantic-history": OLD_FORMAT,
  "records-json": OLD_FORMAT,
  "my-activity": OLD_FORMAT,
  kml: OLD_FORMAT,
  // A photo in the Timeline upload: not wrong, just the other door.
  image: A_PHOTO,
  // Wrong file outright.
  zip: WRONG_FILE,
  gzip: WRONG_FILE,
  pdf: WRONG_FILE,
  html: WRONG_FILE,
  xml: WRONG_FILE,
  binary: WRONG_FILE,
  "not-json": WRONG_FILE,
  empty: WRONG_FILE,
  "json-unrecognised": WRONG_FILE,
  // Passed the sniff but cut off mid-file: the copy is bad, not the format.
  truncated: INCOMPLETE,
};

// Labels this map has never heard of (a future sniffer addition shipping
// ahead of a web update) still land somewhere sensible.
export function redirectFor(format: string | undefined): Redirection {
  return (format && REJECTION_REDIRECTS[format]) || WRONG_FILE;
}
