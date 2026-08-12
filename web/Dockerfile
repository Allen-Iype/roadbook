# Next image (phase 5 BRIEF §3A): the build stage needs full node_modules;
# the runtime runs the standalone output — server.js plus only the traced
# node_modules subset (next.config.ts `output: "standalone"`).
FROM node:24-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
# npm's prebuild hook runs copy:maplibre first, so the MapLibre worker files
# land in public/ before the build — the phase 2 silent-blank-map trap,
# honoured inside the image.
RUN npm run build

FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production
COPY --from=build /app/.next/standalone ./
# The standalone server does not include public/ or .next/static — the
# documented manual copy. public/ is taken from the build stage, after the
# prebuild hook, so it contains the worker files.
COPY --from=build /app/public ./public
COPY --from=build /app/.next/static ./.next/static
# Request-timeout preload (phase 8 CP2): Node caps in-flight requests at
# 5 minutes and Next exposes no knob; large uploads over slow public paths
# need longer. See server-timeout.cjs.
COPY --from=build /app/server-timeout.cjs ./server-timeout.cjs
USER node
# Listen on all container interfaces; host exposure is the compose port
# mapping's decision (loopback-only, BRIEF §3A).
ENV HOSTNAME=0.0.0.0 PORT=3000
EXPOSE 3000
CMD ["node", "--require", "./server-timeout.cjs", "server.js"]
