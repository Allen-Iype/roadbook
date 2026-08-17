// The app shell (phase 9 BRIEF §6): everything behind this layout is the
// product working on the user's own data — the life map, the adventures
// atlas, triage, imports. The route group changes no URL; it exists so the
// public/app seam is a structural fact of the file layout. When
// authentication arrives (a later, charter-gated phase), its gate wraps
// exactly here — no page moves.
//
// Deliberately a pass-through today: the auth-ready constraint shapes where
// files live, never what gets built. Headers stay per-page (the life map
// floats its own chrome; the workbench pages compose SiteHeader), so this
// layout renders nothing of its own.
export default function AppShellLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
