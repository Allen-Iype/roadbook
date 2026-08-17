// The public shell (phase 9 BRIEF §6): the front door — pages a cold
// visitor sees before any data of their own exists. /welcome lives here;
// a future marketing surface would too. The route group changes no URL;
// it is the other side of the public/app seam, and it never gains an
// account slot — there is no account to show a visitor.
export default function PublicShellLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
