import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Output File Tracing (phase 5 BRIEF §3A): `next build` emits
  // .next/standalone — server.js plus only the node_modules subset it
  // traced — which is what the Docker image runs. public/ and .next/static
  // are copied in by the Dockerfile after the build (the documented manual
  // step; public/ must be copied after so the MapLibre worker files from
  // the prebuild copy:maplibre hook are included). Dev (`next dev`) is
  // unaffected.
  output: "standalone",
  // A stray lockfile in the home directory makes Next mis-detect the
  // workspace root; web/ is a standalone npm project, so pin it here.
  turbopack: {
    root: __dirname,
  },
  experimental: {
    serverActions: {
      // Photo uploads travel through a server action as multipart FormData;
      // the default limit is 1 MB — below a single camera JPEG. 64 MB
      // accommodates a realistic batch (phone JPEGs run 2–8 MB) plus
      // multipart overhead; the real enforcement is the Go API's named
      // per-file and per-request limits (phase 4 BRIEF §1.3) — this knob
      // only has to not get in their way. A batch over this fails whole,
      // and the upload island says to send fewer at once.
      bodySizeLimit: "64mb",
    },
  },
};

export default nextConfig;
