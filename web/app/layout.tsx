import type { Metadata } from "next";
import localFont from "next/font/local";
import "./globals.css";

// Self-hosted fonts (phase 6 BRIEF §3C): the woff2 files live in the repo
// (OFL-licensed, LICENSE alongside each), are served from our own origin, and
// next/font generates the @font-face CSS at build time with size-adjusted
// fallbacks — no request ever leaves the machine for a font, and no layout
// shift when they load. The `variable` option puts each family on <html> as a
// CSS custom property, which globals.css maps into the Tailwind font tokens.
const display = localFont({
  src: [
    { path: "./fonts/source-serif-4/latin-400-normal.woff2", weight: "400", style: "normal" },
    { path: "./fonts/source-serif-4/latin-600-normal.woff2", weight: "600", style: "normal" },
    { path: "./fonts/source-serif-4/latin-400-italic.woff2", weight: "400", style: "italic" },
  ],
  variable: "--font-display-face",
  display: "swap",
});

const mono = localFont({
  src: [
    { path: "./fonts/ibm-plex-mono/latin-400-normal.woff2", weight: "400", style: "normal" },
    { path: "./fonts/ibm-plex-mono/latin-500-normal.woff2", weight: "500", style: "normal" },
  ],
  variable: "--font-mono-face",
  display: "swap",
});

// The root layout wraps every route; it is the only place that owns <html>
// and <body>. Body text stays a system sans stack (globals.css) — its role is
// to be quiet; the committed faces carry the display and data roles.
export const metadata: Metadata = {
  title: "Roadbook",
  description: "The adventures you actually took.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`h-full antialiased ${display.variable} ${mono.variable}`}
    >
      <body className="min-h-full flex flex-col">{children}</body>
    </html>
  );
}
