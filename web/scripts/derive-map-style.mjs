// Derives the Roadbook light basemap (public/map-style/roadbook-light.json)
// from OpenFreeMap's Liberty style (phase 6 BRIEF §3B).
//
//   node scripts/derive-map-style.mjs [source]
//
// `source` is a URL or local path to a MapLibre style; default is the live
// Liberty style. The output is committed: operators get a working basemap
// with zero build-time network dependency, and the derivation stays
// reproducible — rerun the script to refresh from upstream. Tiles, glyphs
// and sprites keep pointing at OpenFreeMap (the README's tile note is the
// one deliberate external surface; this script changes colors, not hosts).
//
// Why a script and not a Maputnik session: the palette must trace to the
// Atlas ground tokens (DESIGN §6), and a hand-edited 111-layer JSON cannot
// show which of its thousand color literals is a decision and which is an
// accident. Here every recolor is one rule below; the diff against upstream
// IS the design.
//
// The rules:
//   - drop POI symbols, road shields, one-way arrows, 3D buildings, and the
//     shaded-relief raster — workbench noise on a plate whose only loud
//     marks must be the leg inks (RESEARCH §4: the basemap may never
//     compete with the honesty channel);
//   - land fills flatten toward the land token, keeping a whisper of the
//     original hue so parks and woods stay legible as texture;
//   - water becomes the sea token exactly;
//   - roads bleach toward atlas white over rule-toned casings;
//   - boundaries and labels mute toward the ink tokens, halos to paper.

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const DEFAULT_SOURCE = "https://tiles.openfreemap.org/styles/liberty";

// The Atlas ground tokens (DESIGN §6) — the same values as globals.css and
// lib/tokens.ts; the basemap is derived FROM these, which is the point.
const LAND = [0xef, 0xec, 0xe2]; // #efece2
const SEA = "#d9e2e4";
const PAPER = [0xf5, 0xf2, 0xe8]; // #f5f2e8
const INK2 = [0x6e, 0x6a, 0x5e]; // #6e6a5e
const RULE = [0xc9, 0xc3, 0xb2]; // #c9c3b2
const ROAD_WHITE = [0xff, 0xff, 0xff];

const DROP = [
  /^poi_/, // POI symbols
  /shield/, // highway shields, us and otherwise
  /^road_one_way/, // one-way arrows
  /^building-3d$/, // flat plate — no extrusions
  /^natural_earth$/, // shaded relief competes with the paper ground
];

// ---- color handling ---------------------------------------------------

// Parses #rgb/#rrggbb, rgb()/rgba(), hsl()/hsla() into [r,g,b,a]. Returns
// null for anything else (e.g. named colors, which Liberty does not use).
function parseColor(str) {
  const s = str.trim().toLowerCase();
  let m;
  if ((m = /^#([0-9a-f]{3})$/.exec(s))) {
    return [...m[1]].map((c) => parseInt(c + c, 16)).concat(1);
  }
  if ((m = /^#([0-9a-f]{6})$/.exec(s))) {
    return [0, 2, 4].map((i) => parseInt(m[1].slice(i, i + 2), 16)).concat(1);
  }
  if ((m = /^rgba?\(([^)]+)\)$/.exec(s))) {
    const p = m[1].split(",").map((v) => parseFloat(v));
    return [p[0], p[1], p[2], p.length > 3 ? p[3] : 1];
  }
  if ((m = /^hsla?\(([^)]+)\)$/.exec(s))) {
    const p = m[1].split(",").map((v) => parseFloat(v));
    const [r, g, b] = hslToRgb(p[0], p[1] / 100, p[2] / 100);
    return [r, g, b, p.length > 3 ? p[3] : 1];
  }
  return null;
}

function hslToRgb(h, s, l) {
  h = ((h % 360) + 360) % 360;
  const c = (1 - Math.abs(2 * l - 1)) * s;
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1));
  const m = l - c / 2;
  const [r, g, b] =
    h < 60 ? [c, x, 0]
    : h < 120 ? [x, c, 0]
    : h < 180 ? [0, c, x]
    : h < 240 ? [0, x, c]
    : h < 300 ? [x, 0, c]
    : [c, 0, x];
  return [r + m, g + m, b + m].map((v) => Math.round(v * 255));
}

function fmt([r, g, b, a]) {
  const hex = (v) => Math.round(v).toString(16).padStart(2, "0");
  return a >= 1
    ? `#${hex(r)}${hex(g)}${hex(b)}`
    : `rgba(${Math.round(r)},${Math.round(g)},${Math.round(b)},${+a.toFixed(3)})`;
}

// Mixes a color toward a target by t (0 = untouched, 1 = the target),
// preserving the original alpha — Liberty uses alpha for zoom fades.
function blend(str, target, t) {
  const c = parseColor(str);
  if (!c) return str;
  return fmt([
    c[0] + (target[0] - c[0]) * t,
    c[1] + (target[1] - c[1]) * t,
    c[2] + (target[2] - c[2]) * t,
    c[3],
  ]);
}

// Applies fn to every color literal in a paint/layout value — a plain
// string, or any string nested in an expression / legacy stops object.
// Only called for properties whose name ends in "color", so every string
// reached here is a color.
function mapColors(value, fn) {
  if (typeof value === "string") return fn(value);
  if (Array.isArray(value)) return value.map((v) => mapColors(v, fn));
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(value).map(([k, v]) => [k, mapColors(v, fn)]),
    );
  }
  return value;
}

// ---- the per-layer rule -----------------------------------------------

// Given a layer, returns a function color→color, or null to leave the
// layer untouched. One rule per layer category; first match wins.
function ruleFor(layer) {
  const { id, type } = layer;
  const sourceLayer = layer["source-layer"] ?? "";

  if (id === "background") return () => fmt([...LAND, 1]);
  if (sourceLayer === "water" || sourceLayer === "waterway") return () => SEA;
  if (sourceLayer === "water_name") return (c) => blend(c, INK2, 0.5);

  if (type === "symbol") {
    // Handled per-property below (text vs halo), signalled by "symbol".
    return "symbol";
  }
  if (sourceLayer === "boundary") return (c) => blend(c, INK2, 0.6);
  if (sourceLayer === "transportation") {
    if (/casing|hatching/.test(id)) return (c) => blend(c, RULE, 0.55);
    if (/rail|transit/.test(id)) return (c) => blend(c, RULE, 0.6);
    return (c) => blend(c, ROAD_WHITE, 0.6);
  }
  if (type === "fill" || type === "fill-extrusion") {
    return (c) => blend(c, LAND, 0.78);
  }
  if (type === "line") return (c) => blend(c, RULE, 0.6);
  return null;
}

function recolorLayer(layer) {
  const rule = ruleFor(layer);
  if (!rule) return layer;
  const out = structuredClone(layer);
  for (const bag of ["paint", "layout"]) {
    if (!out[bag]) continue;
    for (const [prop, value] of Object.entries(out[bag])) {
      if (!prop.endsWith("color")) continue;
      const fn =
        rule === "symbol"
          ? prop.includes("halo")
            ? () => fmt([...PAPER, 1])
            : (c) => blend(c, INK2, 0.55)
          : rule;
      out[bag][prop] = mapColors(value, fn);
    }
  }
  return out;
}

// ---- main ---------------------------------------------------------------

const source = process.argv[2] ?? DEFAULT_SOURCE;
const raw = source.startsWith("http")
  ? await (await fetch(source)).text()
  : fs.readFileSync(source, "utf8");
const style = JSON.parse(raw);

const kept = style.layers.filter((l) => !DROP.some((re) => re.test(l.id)));
const dropped = style.layers.length - kept.length;
style.layers = kept.map(recolorLayer);

// Drop sources no remaining layer references (the shaded-relief raster).
const used = new Set(style.layers.map((l) => l.source).filter(Boolean));
for (const name of Object.keys(style.sources)) {
  if (!used.has(name)) delete style.sources[name];
}

style.name = "Roadbook Light";
style.metadata = {
  ...style.metadata,
  "roadbook:derived-from": source,
  "roadbook:generator": "web/scripts/derive-map-style.mjs",
};

const outPath = path.join(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "public",
  "map-style",
  "roadbook-light.json",
);
fs.mkdirSync(path.dirname(outPath), { recursive: true });
fs.writeFileSync(outPath, JSON.stringify(style, null, 1) + "\n");
console.log(
  `wrote ${path.relative(process.cwd(), outPath)} — ${style.layers.length} layers kept, ${dropped} dropped`,
);
