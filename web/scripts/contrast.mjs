// Contrast ratios for the Atlas token pairs in use — the reproduction
// command behind every ratio stated in a DECISIONS or LOG entry
// (invariant 13: no number the repository cannot regenerate).
//
//   node scripts/contrast.mjs            all standing pairs
//   node scripts/contrast.mjs '#8a5c0b'  audition a hex against the grounds
//
// WCAG 2.1 thresholds: 4.5:1 for normal text (AA), 3:1 for large text and
// for graphical objects / meaningful glyphs (1.4.11 non-text contrast).
const TOKENS = {
  paper: "#f5f2e8",
  land: "#efece2",
  ink: "#26251f",
  ink2: "#6e6a5e",
  observed: "#a81e22",
  routed: "#2a5da8",
  unknown: "#8a8375",
  air: "#3f7069",
  flag: "#8a5c0b",
};

function luminance(hex) {
  const c = hex.replace("#", "");
  const [r, g, b] = [0, 2, 4].map((i) => {
    const v = parseInt(c.slice(i, i + 2), 16) / 255;
    return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function ratio(a, b) {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

const fmt = (a, b) => ratio(a, b).toFixed(2) + ":1";

const audition = process.argv[2];
if (audition) {
  console.log(`${audition} on paper ${fmt(audition, TOKENS.paper)}`);
  console.log(`${audition} on land  ${fmt(audition, TOKENS.land)}`);
  process.exit(0);
}

for (const [name, hex] of Object.entries(TOKENS)) {
  if (name === "paper" || name === "land") continue;
  console.log(
    `${name.padEnd(9)} ${hex}  on paper ${fmt(hex, TOKENS.paper)}  on land ${fmt(hex, TOKENS.land)}`,
  );
}
