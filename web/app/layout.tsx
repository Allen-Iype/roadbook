import type { Metadata } from "next";
import "./globals.css";

// The root layout wraps every route; it is the only place that owns <html> and
// <body>. The scaffold's Google-hosted font was removed deliberately: it made
// every build depend on an external fetch, for no need a system font stack
// doesn't cover.
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
    <html lang="en" className="h-full antialiased">
      <body className="min-h-full flex flex-col">{children}</body>
    </html>
  );
}
