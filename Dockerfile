# Go image: one static binary that is CLI and server both (phase 5 BRIEF
# §3A). The build stage holds the toolchain and is discarded; the runtime
# stage is alpine, not scratch, because the compose start command
# (migrate-then-serve), the wget healthcheck, and operator debugging all
# want a shell — the size difference is noise at this scale.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO off: pgx, the image pipeline, and the countries data are pure Go, and a
# static binary runs on any base.
RUN CGO_ENABLED=0 go build -o /out/roadbook ./cmd/roadbook

FROM alpine:3.22
# ca-certificates: the route batch and the opt-in geocoder may call HTTPS
# endpoints. tzdata: Go time handling for named zones.
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H roadbook \
    && mkdir -p /photos /uploads && chown roadbook /photos /uploads
COPY --from=build /out/roadbook /usr/local/bin/roadbook
USER roadbook
# /photos is the thumbnail directory — irreplaceable user data, a named
# volume in compose (it inherits this directory's ownership on first mount).
# /uploads is the retained-exports directory (phase 7 BRIEF §3C) — same
# volume-ownership mechanism, same safety class.
ENV ROADBOOK_PHOTOS_DIR=/photos
ENV ROADBOOK_UPLOADS_DIR=/uploads
EXPOSE 8080
CMD ["roadbook", "serve"]
