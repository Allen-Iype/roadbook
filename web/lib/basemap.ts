// The basemap style URL, resolved once for every map page. server-only for
// the same reason as the API client: the env var is server configuration,
// and the browser receives the resolved value as a prop, never the
// environment itself.
import "server-only";

// Default: the committed Roadbook light style, served from our own origin
// (public/map-style/, derived by scripts/derive-map-style.mjs — see the
// header there). Its tiles, glyphs and sprites still come from OpenFreeMap;
// the README's tile note covers that external surface. ROADBOOK_MAP_STYLE
// still lets any operator point at any style URL (invariant 7's spirit:
// visible degradation beats hidden dependence — a self-hoster can swap the
// basemap without touching code).
// `||`, not `??`: compose passes ROADBOOK_MAP_STYLE as an empty string when
// no override is configured, and empty means unset here.
export const MAP_STYLE_URL =
  process.env.ROADBOOK_MAP_STYLE || "/map-style/roadbook-light.json";
