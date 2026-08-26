// A module-level external store for the triage selection (phase 11 §6.1).
// Why not React state: the gallery's cards are server-rendered HTML — the
// route-shape SVGs must not ship as client props — so the checkbox on each
// card and the action bar are separate small islands with no shared React
// ancestor. useSyncExternalStore over this module is the idiomatic bridge:
// every island subscribes to one process-wide Set, and a change re-renders
// exactly the islands.
//
// The snapshot is an immutable copy per mutation so React's identity check
// sees changes; reads between mutations return the same reference.

let selected: ReadonlySet<number> = new Set();
const listeners = new Set<() => void>();

function emit() {
  for (const l of listeners) l();
}

export function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function getSelected(): ReadonlySet<number> {
  return selected;
}

export function toggle(id: number): void {
  const next = new Set(selected);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  selected = next;
  emit();
}

export function clear(): void {
  if (selected.size === 0) return;
  selected = new Set();
  emit();
}

// Select-all (phase 11 §6.1 review addition): replaces the selection with
// the given ids. The caller passes UNDECIDED candidates only — a bulk
// action must never silently re-decide rows the user already curated.
export function selectAll(ids: number[]): void {
  selected = new Set(ids);
  emit();
}
