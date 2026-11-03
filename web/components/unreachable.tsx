// The one sentence every page shows when the Go service does not answer.
// Decision 7 of the copy pass (docs/product/words.md, 2026-11-02): the
// command to start the service is for the operator, and in a hosted
// account the person signed in is not the operator — they get the plain
// sentence. `operator` is true when the session runs without sign-in (one
// person, their own machine).
export function Unreachable({
  operator,
  className = "mt-6 text-red-700",
}: {
  operator: boolean;
  className?: string;
}) {
  return (
    <p className={className}>
      Roadbook isn&apos;t answering right now.{" "}
      {operator ? (
        <>
          Start it with <code className="font-mono">roadbook serve</code> and
          reload.
        </>
      ) : (
        "Try again in a minute."
      )}
    </p>
  );
}
