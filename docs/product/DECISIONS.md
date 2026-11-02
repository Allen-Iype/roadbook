# Copy pass — decision log

Three lines each, written as the work happens. The seven decisions that
shaped the word list are in `words.md`; this log records what each commit
settled on the way.

## 2026-11-02 — Global strings and the legend behind a button

- **Chosen:** one `LegendInfo` client component (button "What the lines
  mean" + the same native popover as `Term`, inline fallback) replaces the
  printed `LegKindLegend` on all four maps; the trip page's Fix and Stop
  markers move inside it, and the bar's caption under the headline
  distance is gone. The maintainer's review of the first build (on the
  phone, 2026-11-02) removed the bar's one-line note from the popover as
  well: the bar stays drawn, unexplained except by the closed fold at the
  foot of the trip page — the recorded/drawn split is not something the
  person is asked to read. One `Unreachable` component
  carries the "isn't answering" sentence with the `roadbook serve` hint
  only when `session.mode === "off"` (decision 7). Nav reads Trips · Trips
  to review · n · Uploads on the header and the map panel. Six Go 404
  messages say "this trip isn't in your list any more — reload the page";
  the seven web fallbacks say "couldn't … (error n)". The date-named
  default becomes "Trip of 24 April 2026" through `fmtDateRange`.
- **Rejected:** keeping the legend's names as `Term` buttons inside the
  popover (a popover in a popover; the descriptions are already there);
  one shared "Legend" string constant for the button label in `lib/`
  (the component exports it, and the e2e helper opens it by role).
- **Would change our mind:** a tester not finding the (i) — then the
  button grows a word ("Legend") or the trip page's margin prints the
  four lines again.
- **Verified:** tsc, lint (0 errors), vitest 128, `go test ./internal/api
  ./internal/timeline`, Playwright 122 passed / 22 skipped at 390/768/1280
  against `next dev -p 3017` over the roadbook-p16 API; screenshots at
  390 and 1280 with the popover open on all four maps.
