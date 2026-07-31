// loading.tsx is a file convention: while the async server component in
// page.tsx is fetching, Next streams this fallback immediately, then replaces
// it when the page resolves. Kept skeletal — the fetch is local and fast; the
// state exists so a slow API degrades to "visibly loading" rather than a
// blank tab.
export default function Loading() {
  return (
    <main className="mx-auto w-full max-w-5xl px-6 py-10">
      <h1 className="text-2xl font-semibold">Roadbook</h1>
      <p className="mt-6 text-sm text-neutral-500">Loading candidates…</p>
    </main>
  );
}
