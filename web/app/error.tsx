"use client";

// error.tsx is a file convention: an error boundary for *unexpected* thrown
// errors during render. It must be a client component because error recovery
// (the reset callback) is interactive. Expected failures — API down — are
// handled in page.tsx as ordinary rendering, not here.
export default function ErrorBoundary({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <main className="mx-auto w-full max-w-5xl px-6 py-10">
      <h1 className="font-display text-2xl font-semibold">Roadbook</h1>
      <p className="mt-6 text-red-700">Something failed while rendering.</p>
      <p className="mt-1 font-mono text-xs text-ink-2">{error.message}</p>
      <button
        onClick={reset}
        className="mt-4 rounded border border-rule px-3 py-1 text-sm hover:bg-land"
      >
        Try again
      </button>
    </main>
  );
}
