import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // A stray lockfile in the home directory makes Next mis-detect the
  // workspace root; web/ is a standalone npm project, so pin it here.
  turbopack: {
    root: __dirname,
  },
};

export default nextConfig;
