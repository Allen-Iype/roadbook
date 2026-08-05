import type { NextConfig } from "next";

const nextConfig: NextConfig = {
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
