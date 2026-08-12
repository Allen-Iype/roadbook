// Preload for the standalone server (phase 8 CP2): relax Node's in-flight
// request timeout, which Next exposes no knob for.
//
// Node's http.Server.requestTimeout defaults to 300 000 ms. A Timeline
// export uploaded through a slow public path (the pilot's funnel relay
// measured ~221 KB/s) can legitimately need longer than five minutes, and
// the default fails it with an opaque 408 mid-upload. Next's standalone
// server.js reads KEEP_ALIVE_TIMEOUT but never touches requestTimeout, so
// this preload wraps http.createServer and stamps the timeout on the server
// instance the moment it is created.
//
// ROADBOOK_REQUEST_TIMEOUT_MS: override in ms; 0 disables the cap entirely.
// Default 3 600 000 (one hour) — far above any plausible upload, while still
// bounding a wedged connection. headersTimeout (60 s) is left alone: it
// guards the pre-body handshake and slow-loris shapes, and uploads never
// trip it.
//
// Loaded via `node --require ./server-timeout.cjs server.js` (web
// Dockerfile CMD). Removal is deleting this file and the flag — the wrap
// touches nothing else.
"use strict";

/* eslint-disable @typescript-eslint/no-require-imports --
   This file is deliberately CommonJS: `node --require` preloads run before
   any bundler or ESM machinery exists, so require() is the only tool. */
const http = require("http");

let timeout = Number.parseInt(process.env.ROADBOOK_REQUEST_TIMEOUT_MS ?? "", 10);
if (Number.isNaN(timeout) || timeout < 0) {
  timeout = 3_600_000;
}

// Inject via constructor options AND set the property afterwards: Node
// derives per-connection deadlines from the options at construction, so a
// later property assignment alone is not reliably honoured.
const createServer = http.createServer;
http.createServer = function (...args) {
  let options = { requestTimeout: timeout };
  let rest = args;
  if (args.length > 0 && typeof args[0] === "object" && args[0] !== null) {
    options = { ...args[0], requestTimeout: timeout };
    rest = args.slice(1);
  }
  const server = createServer.call(this, options, ...rest);
  server.requestTimeout = timeout;
  // One line at startup so a live container proves the preload took effect
  // in the process that owns the listening socket.
  console.log(`server-timeout preload: requestTimeout=${timeout}ms (pid ${process.pid})`);
  return server;
};
