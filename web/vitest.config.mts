// Vitest arrives in phase 6 checkpoint 2 (BRIEF §0, §4.2) as the web side's
// first test runner. Its targets are the pure modules under lib/ — layer
// specs, geometry, later sliceDays — which need no browser and no DOM: the
// default node environment is enough, which is exactly why those seams were
// built pure.
import { defineConfig } from "vitest/config";
import path from "node:path";

export default defineConfig({
  resolve: {
    // Mirror tsconfig's "@/*" → "./*" so tests import the app's own modules
    // by their real names.
    alias: { "@": path.resolve(__dirname) },
  },
  test: {
    include: ["lib/**/*.test.ts"],
  },
});
