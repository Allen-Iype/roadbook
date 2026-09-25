// The image export's browser half (phase 14 CP3, BRIEF §3A option 1): a
// second, offscreen MapLibre map rendered at the plate's map slot with its
// drawing buffer preserved, read once, composed with the margin that
// lib/plate-image.ts laid out, handed to the browser as a PNG download, and
// destroyed. Browser-only by nature — it needs a WebGL context, a 2D
// canvas, and the document's fonts — so nothing here is unit-tested; the
// layout it paints is (plate-image.test.ts), and the e2e download spec
// proves the pixels.
//
// Why a second map and not the on-screen canvas (BRIEF §1): the browser
// may discard a WebGL drawing buffer once it has been composited, and
// MapLibre lets it — reading the visible canvas after the frame gives
// blank pixels (the phase 6 screenshot trap). The remedies are to ask for
// the buffer to be kept (a per-frame cost the interactive map should not
// pay for the life of the page) or to read inside the render callback.
// The offscreen instance asks for the buffer to be kept, lives for
// seconds, and has deterministic dimensions independent of the viewer's
// window — the image is 2400×1600 on a phone and on a desktop alike.
import { LngLatBounds, MapLibreMap, setWorkerUrl } from "maplibre-gl";

import { bboxOf } from "@/lib/geo";
import {
  MAP_RECT,
  PLATE_SCALE,
  joinAttributions,
  overlayFilename,
  overlayLayout,
  plateFilename,
  plateLayout,
  type FontRole,
  type Op,
  type PlateInput,
  type PlateLayout,
  type RowItem,
} from "@/lib/plate-image";
import {
  FIT_PADDING,
  ROUTE_LAYERS,
  ROUTE_SOURCE,
  highlightPaint,
  routeFeatures,
} from "@/lib/route-layers";
import type { Day } from "@/lib/slice-days";
import { INK } from "@/lib/tokens";
import type { components } from "@/lib/api/schema";

type Journey = components["schemas"]["Journey"];

// Same worker file as the on-screen map (see route-map.tsx); setting it
// here too keeps this module correct even if it is ever loaded first.
setWorkerUrl("/maplibre/maplibre-gl-worker.mjs");

/** How long the basemap may take to settle before the export gives up. */
export const EXPORT_TIMEOUT_MS = 30_000;

export type ExportSpec = {
  journey: Journey;
  days: Day[];
  selectedDay: number | null;
  styleUrl: string;
  name: string;
  plate: number | null;
  startTruncated: boolean;
  endTruncated: boolean;
};

/** Every way the export can fail, each with words the control shows. A
 * failure never downloads anything — there is no partial plate. */
export type ExportFailure =
  | "webgl"
  | "basemap-blocked"
  | "basemap-failed"
  | "timeout"
  | "tainted"
  | "encode";

export class PlateExportError extends Error {
  constructor(
    public readonly kind: ExportFailure,
    message: string,
  ) {
    super(message);
    this.name = "PlateExportError";
  }
}

// MapLibre's AJAXError, duck-typed: the class is not among the package's
// named exports, but its shape is stable — an HTTP status (0 when fetch
// itself failed) and the URL it was fetching.
function ajaxLike(e: unknown): { status: number; url: string } | null {
  if (typeof e !== "object" || e === null) return null;
  const o = e as { status?: unknown; url?: unknown };
  return typeof o.status === "number" && typeof o.url === "string"
    ? { status: o.status, url: o.url }
    : null;
}

function hostOf(url: string): string {
  try {
    return new URL(url, window.location.href).host;
  } catch {
    return url;
  }
}

// The first basemap error the offscreen map reported, as words. A fetch
// that failed outright (status 0) is what a tile server without CORS
// headers looks like from inside a browser — and also what an unreachable
// one looks like; the browser does not tell them apart, so neither can the
// message, and it says both.
function basemapError(e: unknown): PlateExportError {
  const a = ajaxLike(e);
  if (a && a.status === 0) {
    return new PlateExportError(
      "basemap-blocked",
      `The basemap at ${hostOf(a.url)} could not be read: the tile server ` +
        "either did not allow cross-origin use (no CORS headers) or could " +
        "not be reached. The on-screen map may still show, but an image " +
        "cannot be drawn from tiles the browser was not allowed to read. " +
        "Nothing was downloaded.",
    );
  }
  if (a) {
    return new PlateExportError(
      "basemap-failed",
      `A basemap resource failed to load (HTTP ${a.status} from ${hostOf(a.url)}). ` +
        "Try again in a moment. Nothing was downloaded.",
    );
  }
  const msg = e instanceof Error ? e.message : String(e);
  return new PlateExportError(
    "basemap-failed",
    `The basemap could not be loaded (${msg}). Nothing was downloaded.`,
  );
}

/** Renders the plate and returns the PNG. Throws PlateExportError only. */
export async function exportPlateImage(
  spec: ExportSpec,
): Promise<{ blob: Blob; filename: string }> {
  const data = routeFeatures(spec.journey, spec.days);
  const box = bboxOf(data.features);
  if (!box) {
    throw new PlateExportError(
      "encode",
      "This journey has nothing drawn, so there is no plate to export.",
    );
  }

  // The offscreen container: laid out (MapLibre reads clientWidth), out of
  // view, at exactly the map slot's logical size.
  const container = document.createElement("div");
  container.setAttribute("aria-hidden", "true");
  container.style.cssText =
    `position:fixed;top:0;left:-${MAP_RECT.w + 200}px;` +
    `width:${MAP_RECT.w}px;height:${MAP_RECT.h}px;pointer-events:none;`;
  document.body.appendChild(container);

  let map: MapLibreMap | null = null;
  try {
    const errors: unknown[] = [];
    try {
      map = new MapLibreMap({
        container,
        style: spec.styleUrl,
        bounds: new LngLatBounds(box[0], box[1]),
        fitBoundsOptions: { padding: FIT_PADDING },
        pixelRatio: PLATE_SCALE,
        // The trap and its documented remedy (BRIEF §1): keep the drawing
        // buffer so the canvas can be read after the frame.
        canvasContextAttributes: { preserveDrawingBuffer: true },
        interactive: false,
        attributionControl: false,
        // No tile fade-in: idle then means "every tile is drawn at full
        // opacity", not "the fade is still running".
        fadeDuration: 0,
      });
    } catch (e) {
      throw new PlateExportError(
        "webgl",
        "This browser could not create a map for the export (WebGL is " +
          `unavailable: ${e instanceof Error ? e.message : String(e)}). ` +
          "Nothing was downloaded.",
      );
    }
    const m = map;
    m.on("error", (ev) => errors.push(ev.error));

    await withTimeout(m.once("load"), "load");
    m.addSource(ROUTE_SOURCE, { type: "geojson", data });
    for (const layer of ROUTE_LAYERS) m.addLayer(layer);
    for (const c of highlightPaint(spec.selectedDay))
      m.setPaintProperty(c.layer, c.property, c.value);
    await withTimeout(m.once("idle"), "tiles");

    if (errors.length > 0) throw basemapError(errors[0]);

    // The credit line is read from what actually loaded — the style JSON
    // carries no attribution of its own; it arrives in each source's
    // TileJSON — so a self-hoster's different basemap credits itself.
    const attribution = joinAttributions(
      Object.keys(m.getStyle().sources ?? {}).map((id) => {
        const src = m.getSource(id);
        const a = (src as { attribution?: unknown } | undefined)?.attribution;
        return typeof a === "string" ? a : undefined;
      }),
    );
    const b = m.getBounds();
    const input: PlateInput = {
      journey: spec.journey,
      name: spec.name,
      plate: spec.plate,
      startTruncated: spec.startTruncated,
      endTruncated: spec.endTruncated,
      selectedDay: spec.selectedDay,
      attribution,
      mapView: { west: b.getWest(), east: b.getEast(), centerLat: m.getCenter().lat },
    };
    const layout = plateLayout(input);

    const fonts = await plateFonts(layout);
    const canvas = document.createElement("canvas");
    canvas.width = layout.width * layout.scale;
    canvas.height = layout.height * layout.scale;
    const ctx = canvas.getContext("2d");
    if (!ctx) {
      throw new PlateExportError(
        "encode",
        "This browser could not create a drawing surface for the export. Nothing was downloaded.",
      );
    }
    ctx.scale(layout.scale, layout.scale);
    paint(ctx, layout, fonts, m.getCanvas());

    const blob = await toBlob(canvas);
    return { blob, filename: plateFilename(spec.name) };
  } finally {
    // Release the WebGL context — browsers cap live contexts — and the DOM.
    map?.remove();
    container.remove();
  }
}

/** Renders the transparent overlay (CP4, BRIEF §9) and returns the PNG.
 * Tile-free: no map instance, no network — the route is projected from
 * the same features the maps draw. Throws PlateExportError only. */
export async function exportOverlayImage(
  spec: ExportSpec,
): Promise<{ blob: Blob; filename: string }> {
  const data = routeFeatures(spec.journey, spec.days);
  if (!bboxOf(data.features)) {
    throw new PlateExportError(
      "encode",
      "This journey has nothing drawn, so there is no route to export.",
    );
  }
  const layout = overlayLayout(
    {
      journey: spec.journey,
      name: spec.name,
      plate: spec.plate,
      startTruncated: spec.startTruncated,
      endTruncated: spec.endTruncated,
      selectedDay: spec.selectedDay,
    },
    data,
  );
  const fonts = await plateFonts(layout);
  const canvas = document.createElement("canvas");
  canvas.width = layout.width * layout.scale;
  canvas.height = layout.height * layout.scale;
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new PlateExportError(
      "encode",
      "This browser could not create a drawing surface for the export. Nothing was downloaded.",
    );
  }
  ctx.scale(layout.scale, layout.scale);
  // No fill op in the overlay's list: the ground stays transparent.
  paint(ctx, layout, fonts, null);
  const blob = await toBlob(canvas);
  return { blob, filename: overlayFilename(spec.name) };
}

/** Hands the blob to the browser as a download. */
export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  // Revoked on the next tick: some browsers start the download after click
  // returns, and a URL revoked synchronously downloads nothing.
  setTimeout(() => URL.revokeObjectURL(url), 1_000);
}

function withTimeout<T>(p: Promise<T>, what: string): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const timer = setTimeout(
      () =>
        reject(
          new PlateExportError(
            "timeout",
            `The basemap did not finish loading within ${EXPORT_TIMEOUT_MS / 1000} s ` +
              `(waiting for ${what}). Check the connection and try again. Nothing was downloaded.`,
          ),
        ),
      EXPORT_TIMEOUT_MS,
    );
    p.then(
      (v) => {
        clearTimeout(timer);
        resolve(v);
      },
      (e) => {
        clearTimeout(timer);
        reject(e);
      },
    );
  });
}

function toBlob(canvas: HTMLCanvasElement): Promise<Blob> {
  return new Promise((resolve, reject) => {
    try {
      canvas.toBlob((blob) => {
        if (blob) resolve(blob);
        else
          reject(
            new PlateExportError(
              "encode",
              "The browser could not encode the plate as PNG. Nothing was downloaded.",
            ),
          );
      }, "image/png");
    } catch (e) {
      // A SecurityError here means the 2D canvas was tainted: something
      // drawn onto it came from another origin without CORS approval.
      const name = e instanceof Error ? e.name : "";
      reject(
        name === "SecurityError"
          ? new PlateExportError(
              "tainted",
              "The browser refused to read the composed image: the basemap " +
                "was drawn without cross-origin permission (its tile server " +
                "sends no CORS headers), which taints the canvas. Nothing was downloaded.",
            )
          : new PlateExportError(
              "encode",
              `The browser could not encode the plate as PNG (${e instanceof Error ? e.message : String(e)}). Nothing was downloaded.`,
            ),
      );
    }
  });
}

// ---- fonts ----

type FontFamilies = Record<FontRole, string>;

// next/font/local registers the self-hosted faces under generated family
// names and exposes each as a CSS variable on <html>; the canvas needs the
// family names themselves, so they are read off the document at export
// time — never guessed — and awaited through the Font Loading API so the
// first export does not paint in a fallback face. The sans is the theme's
// system stack (globals.css @theme), read the same way.
async function plateFonts(layout: PlateLayout): Promise<FontFamilies> {
  const css = getComputedStyle(document.documentElement);
  const read = (v: string, fallback: string) => {
    const val = css.getPropertyValue(v).trim();
    return val === "" ? fallback : val;
  };
  const families: FontFamilies = {
    display: read("--font-display-face", "Georgia, serif"),
    mono: read("--font-mono-face", "ui-monospace, Menlo, monospace"),
    sans: read("--font-sans", "system-ui, sans-serif"),
  };
  if (typeof document.fonts?.load === "function") {
    const specs = new Set<string>();
    const add = (s: { role: FontRole; size: number; weight: number }) =>
      specs.add(`${s.weight} ${s.size}px ${families[s.role]}`);
    for (const op of layout.ops) {
      if (op.op === "text") {
        add(op);
      } else if (op.op === "row") {
        for (const it of op.items) if (it.item === "text") add(it);
      } else if (op.op === "scale") {
        add({ role: "mono", size: 11, weight: 500 });
      }
    }
    await Promise.all(
      [...specs].map((s) => document.fonts.load(s).catch(() => [])),
    );
  }
  return families;
}

// ---- painting ----

function font(f: FontFamilies, s: { role: FontRole; size: number; weight: number }) {
  return `${s.weight} ${s.size}px ${f[s.role]}`;
}

function setTracking(ctx: CanvasRenderingContext2D, em: number | undefined, size: number) {
  // letterSpacing is a recent canvas property; where unsupported the
  // assignment is ignored and the label sets untracked — legible either way.
  const c = ctx as CanvasRenderingContext2D & { letterSpacing?: string };
  if ("letterSpacing" in c) c.letterSpacing = em ? `${em * size}px` : "0px";
}

function paint(
  ctx: CanvasRenderingContext2D,
  layout: PlateLayout,
  fonts: FontFamilies,
  mapCanvas: HTMLCanvasElement | null,
) {
  ctx.textBaseline = "alphabetic";
  // The overlay's route halo: one translucent paper pass under EVERY leg
  // before any ink — the route's own ground on a photo nobody has seen
  // (BRIEF §9). Ground, not encoding: routed's crisp opaque casing is
  // painted afterwards, per leg, and still reads as a casing on top.
  const paths = layout.ops.filter((op): op is Extract<Op, { op: "path" }> => op.op === "path");
  if (paths.length > 0) {
    ctx.save();
    ctx.strokeStyle = "rgba(245,242,232,0.7)";
    ctx.lineWidth = 11;
    ctx.lineCap = "round";
    ctx.lineJoin = "round";
    for (const p of paths) {
      ctx.globalAlpha = p.alpha;
      strokePolyline(ctx, p.points);
    }
    ctx.restore();
  }
  for (const op of layout.ops) paintOp(ctx, op, fonts, mapCanvas);
}

function strokePolyline(ctx: CanvasRenderingContext2D, pts: [number, number][]) {
  ctx.beginPath();
  pts.forEach(([x, y], i) => (i === 0 ? ctx.moveTo(x, y) : ctx.lineTo(x, y)));
  ctx.stroke();
}

// The overlay's line channel per kind, the map layers' numbers (widths in
// px, MapLibre dash units × width): observed solid 3.5, routed 2.5 over a
// 5.5 paper casing, unknown 2 dashed [4,6], air 2.2 round-dotted.
function strokeKind(ctx: CanvasRenderingContext2D, kind: Extract<Op, { op: "path" }>["kind"], pts: [number, number][]) {
  ctx.lineCap = "round";
  ctx.lineJoin = "round";
  if (kind === "routed") {
    ctx.strokeStyle = INK.paper;
    ctx.lineWidth = 5.5;
    ctx.setLineDash([]);
    strokePolyline(ctx, pts);
  }
  ctx.strokeStyle = INK[kind];
  ctx.lineWidth = kind === "observed" ? 3.5 : kind === "routed" ? 2.5 : kind === "air" ? 2.2 : 2;
  ctx.setLineDash(kind === "unknown" ? [4, 6] : kind === "air" ? [0.1, 4.4] : []);
  strokePolyline(ctx, pts);
  ctx.setLineDash([]);
}

// Paper casing around the glyphs — the overlay's ground for text: a wide
// round-joined paper stroke first, the ink fill on top.
function fillTextHalo(
  ctx: CanvasRenderingContext2D,
  text: string,
  x: number,
  y: number,
  size: number,
  halo: boolean | undefined,
  maxWidth?: number,
) {
  if (halo) {
    ctx.save();
    ctx.strokeStyle = INK.paper;
    // Proportional to the type size: thick enough to isolate the glyph
    // from a busy photo, thin enough not to fill small counters.
    ctx.lineWidth = Math.max(2.4, size * 0.18);
    ctx.lineJoin = "round";
    ctx.miterLimit = 2;
    if (maxWidth !== undefined) ctx.strokeText(text, x, y, maxWidth);
    else ctx.strokeText(text, x, y);
    ctx.restore();
  }
  if (maxWidth !== undefined) ctx.fillText(text, x, y, maxWidth);
  else ctx.fillText(text, x, y);
}

function paintOp(
  ctx: CanvasRenderingContext2D,
  op: Op,
  fonts: FontFamilies,
  mapCanvas: HTMLCanvasElement | null,
) {
  switch (op.op) {
    case "fill":
      ctx.fillStyle = op.color;
      ctx.fillRect(op.x, op.y, op.w, op.h);
      return;
    case "map":
      // The offscreen canvas is the slot's size at the plate's pixel ratio;
      // drawn into the slot it lands pixel for pixel. Absent on the overlay.
      if (mapCanvas) ctx.drawImage(mapCanvas, op.x, op.y, op.w, op.h);
      return;
    case "path":
      ctx.save();
      ctx.globalAlpha = op.alpha;
      strokeKind(ctx, op.kind, op.points);
      ctx.restore();
      return;
    case "point": {
      ctx.save();
      ctx.globalAlpha = op.alpha;
      paintDot(ctx, op.kind, op.x, op.y);
      ctx.restore();
      return;
    }
    case "frame":
      ctx.strokeStyle = op.color;
      ctx.lineWidth = 1;
      // Half-pixel offsets keep 1 px rules crisp at integer coordinates.
      ctx.strokeRect(op.x - 0.5, op.y - 0.5, op.w + 1, op.h + 1);
      ctx.strokeRect(
        op.x - op.offset - 1.5,
        op.y - op.offset - 1.5,
        op.w + 2 * op.offset + 3,
        op.h + 2 * op.offset + 3,
      );
      return;
    case "text": {
      ctx.font = font(fonts, op);
      setTracking(ctx, op.tracking, op.size);
      ctx.fillStyle = op.color;
      ctx.textAlign = op.align;
      // Shrink-to-fit for the one line that can overflow (the name): step
      // the size down until it fits, to a floor, then let maxWidth compress
      // whatever is left.
      let size = op.size;
      if (op.maxWidth !== undefined) {
        while (size > 18 && ctx.measureText(op.text).width > op.maxWidth) {
          size -= 1;
          ctx.font = font(fonts, { ...op, size });
        }
      }
      fillTextHalo(ctx, op.text, op.x, op.y, size, op.halo, op.maxWidth);
      setTracking(ctx, undefined, op.size);
      return;
    }
    case "row": {
      // Inline flow; with `wrap`, an item that would cross maxWidth starts
      // the next line (leading spaces are dropped at a line start).
      let x = op.x;
      let y = op.y;
      ctx.textAlign = "left";
      for (const it of op.items) {
        if (op.wrap) {
          const w = measureRowItem(ctx, it, fonts);
          if (x > op.x && x + w > op.x + op.wrap.maxWidth) {
            x = op.x;
            y += op.wrap.lineHeight;
            if (it.item === "space") continue;
          }
        }
        x = paintRowItem(ctx, it, x, y, fonts);
      }
      return;
    }
    case "bar": {
      ctx.fillStyle = op.track;
      ctx.fillRect(op.x, op.y, op.w, op.h);
      let x = op.x;
      for (const s of op.segments) {
        const w = op.w * s.fraction;
        ctx.fillStyle = INK[s.kind];
        ctx.fillRect(x, op.y, w, op.h);
        x += w;
      }
      return;
    }
    case "scale": {
      // A paper chip under the bar keeps it legible over any basemap; the
      // bar itself is a hairline with end ticks and the distance in mono.
      ctx.font = font(fonts, { role: "mono", size: 11, weight: 500 });
      setTracking(ctx, undefined, 11);
      const labelW = ctx.measureText(op.label).width;
      const w = Math.max(op.px, labelW) + 12;
      ctx.fillStyle = "rgba(245,242,232,0.8)";
      ctx.fillRect(op.x - 6, op.y - 20, w, 26);
      ctx.strokeStyle = INK.ink;
      ctx.lineWidth = 1.5;
      ctx.beginPath();
      ctx.moveTo(op.x, op.y - 6);
      ctx.lineTo(op.x, op.y + 2);
      ctx.lineTo(op.x + op.px, op.y + 2);
      ctx.lineTo(op.x + op.px, op.y - 6);
      ctx.stroke();
      ctx.fillStyle = INK.ink;
      ctx.textAlign = "left";
      ctx.fillText(op.label, op.x, op.y - 8);
      return;
    }
  }
}

/** Paints one inline item at x on baseline y; returns the next x. */
function paintRowItem(
  ctx: CanvasRenderingContext2D,
  it: RowItem,
  x: number,
  y: number,
  fonts: FontFamilies,
): number {
  switch (it.item) {
    case "space":
      return x + it.px;
    case "text": {
      ctx.font = font(fonts, it);
      setTracking(ctx, it.tracking, it.size);
      ctx.fillStyle = it.color;
      fillTextHalo(ctx, it.text, x, y, it.size, it.halo);
      const w = ctx.measureText(it.text).width;
      setTracking(ctx, undefined, it.size);
      return x + w;
    }
    case "sample": {
      // The legend samples draw the actual channel, as the DOM legend's
      // SVG does (components/legend.tsx): 44×8 box, line at its middle.
      const mid = y - 4;
      ctx.lineCap = "round";
      if (it.kind === "routed") {
        ctx.strokeStyle = INK.paper;
        ctx.lineWidth = 7;
        ctx.setLineDash([]);
        ctx.beginPath();
        ctx.moveTo(x + 2, mid);
        ctx.lineTo(x + 42, mid);
        ctx.stroke();
      }
      ctx.strokeStyle = INK[it.kind];
      ctx.lineWidth = it.kind === "observed" ? 3 : 2.5;
      ctx.setLineDash(
        it.kind === "unknown" ? [6, 4] : it.kind === "air" ? [0.1, 5.5] : [],
      );
      ctx.beginPath();
      ctx.moveTo(x + 2, mid);
      ctx.lineTo(x + 42, mid);
      ctx.stroke();
      ctx.setLineDash([]);
      return x + 44;
    }
    case "dot": {
      const r = it.kind === "fix" ? 4 : 5;
      paintDot(ctx, it.kind, x + r, y - 4);
      return x + 2 * r + 2;
    }
  }
}

/** The width an inline item takes, for wrapping. */
function measureRowItem(ctx: CanvasRenderingContext2D, it: RowItem, fonts: FontFamilies): number {
  switch (it.item) {
    case "space":
      return it.px;
    case "sample":
      return 44;
    case "dot":
      return (it.kind === "fix" ? 4 : 5) * 2 + 2;
    case "text": {
      ctx.font = font(fonts, it);
      setTracking(ctx, it.tracking, it.size);
      const w = ctx.measureText(it.text).width;
      setTracking(ctx, undefined, it.size);
      return w;
    }
  }
}

// A fix or stop mark: the map layers' circles — fix in the observed ink
// at radius 4, stop in ink at radius 5, each with a paper stroke and an
// outer ring so it reads on any ground. Shared by the legend samples and
// the overlay's route points.
function paintDot(ctx: CanvasRenderingContext2D, kind: "fix" | "stop", cx: number, cy: number) {
  const r = kind === "fix" ? 4 : 5;
  const ink = kind === "fix" ? INK.observed : INK.ink;
  ctx.beginPath();
  ctx.arc(cx, cy, r, 0, Math.PI * 2);
  ctx.fillStyle = ink;
  ctx.fill();
  ctx.lineWidth = 1.25;
  ctx.strokeStyle = INK.paper;
  ctx.stroke();
  ctx.beginPath();
  ctx.arc(cx, cy, r + 1, 0, Math.PI * 2);
  ctx.lineWidth = 1;
  ctx.strokeStyle = ink;
  ctx.stroke();
}
