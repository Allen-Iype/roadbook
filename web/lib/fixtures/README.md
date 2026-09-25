# Journey fixtures for vitest

The three demo adventures exactly as `GET /candidates/{id}/journey` served
them from a fresh instance holding only `testdata/demo/demo.json` (the
fictional Reykjavík persona — no real location history is ever committed,
CLAUDE.md invariant 14), confirmed in id order as "South coast drive",
"Westfjords loop", "Akureyri weekend". Unrouted: no OSRM was attached, so
gaps are `unknown` or `air`.

They exist for the cross-language parity test (`summary-parity.test.ts`):
the served `summary.civil_days` against the narrative's `sliceDays`.

Regenerate after a contract change:

```
for i in 1 2 3; do
  curl -s http://127.0.0.1:8080/candidates/$i/journey > web/lib/fixtures/demo-journey-$i.json
done
```

against an instance in that state (the README quickstart produces one; the
API port is whatever the stack publishes).
