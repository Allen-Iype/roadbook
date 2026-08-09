// The provenance bar (DESIGN §6): a hairline stacked bar of a distance's
// kind-composition, attached to every headline distance figure. Invariant 8
// applied to numbers, not just geometry — a total never appears without
// showing how much of it is measurement and how much inference.
//
// Purely presentational; the numbers are km and only their ratios matter.
// The visual is aria-hidden because the composition is always stated in
// adjacent text — the bar makes it scannable, the words make it accessible.
export function ProvenanceBar({
  observed,
  routed,
  unknown,
  air,
  className = "",
}: {
  observed: number;
  routed: number;
  unknown: number;
  air: number;
  className?: string;
}) {
  const total = observed + routed + unknown + air;
  if (total <= 0) return null;
  const parts = [
    { key: "observed", km: observed, cls: "bg-observed" },
    { key: "routed", km: routed, cls: "bg-routed" },
    { key: "unknown", km: unknown, cls: "bg-unknown" },
    { key: "air", km: air, cls: "bg-air" },
  ].filter((p) => p.km / total > 0.002);
  return (
    <span
      aria-hidden
      className={`flex h-1 w-full overflow-hidden bg-rule ${className}`}
    >
      {parts.map((p) => (
        <span
          key={p.key}
          className={`block h-full ${p.cls}`}
          style={{ width: `${(p.km / total) * 100}%` }}
        />
      ))}
    </span>
  );
}
