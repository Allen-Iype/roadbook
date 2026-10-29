# Landing page — reference board (phase 16 CP2b, Step 1)

The maintainer's verdict on the CP2 landing (2026-10-29): cleaner, not
clean. Three named problems: **flat hierarchy** (every section is eyebrow +
heading + prose; nothing leads), **too many devices** (rules, boxes, roman
numerals, mono caps, middle dots), **still too wordy** (597 words on a
phone). This board is the inspiration step agreed before any mockup:
reference pages looked at the same way, what each does in its first screen,
how it shows its product, how it states custody, how many words it spends,
one thing to take, one thing to refuse. Then five candidate principles for
the discussion STOP.

Method: each page fetched and screenshotted at 390 and 1280 px on
2026-10-29 (screenshots kept in the session scratchpad, not the repository
— third-party pages do not enter git). Word counts are `body.innerText`
whitespace-split at phone width, the same measure `web/scripts/copy-count.cjs`
uses, so they compare with ours: landing 851 before CP2, 597 after.

## Group 1 — the same job: a personal record, drawn as a map

**Polarsteps** — https://www.polarsteps.com/ · 467 words at 390 px.
First screen: a full-bleed photograph of a traveller, one headline ("One
travel app for all your adventures"), one button, store badges; the phone
with the route map rises into the fold from below. The product is shown as
phone mockups and photographs throughout; the map is a cinematic image, not
a live map. Custody is one section far down ("Full privacy control": only
you / friends / the world). A cookie banner covers a third of the phone's
first screen. *Take:* one headline, one button, the payoff (the route on the
map) visible in the first screen on a phone. *Refuse:* the photograph as
hero — our product is the drawing, not the destination; and the banner,
which we structurally cannot have.

**Dawarich** — https://dawarich.app/ · 2,130 words at 390 px.
The direct comparable: "The Private Alternative to Google Timeline — Your
Private Timeline, Forever", a product screenshot slanted behind the
headline, two buttons plus a Google sign-in, a trust-badge row, a counter
strip, a seven-row comparison table against Google, six alternating feature
blocks with screenshots, a retention-warning banner, a privacy stack,
testimonials, three pricing cards, FAQ, newsletter. Custody is stated five
times in five places. *Take:* the one sentence that names the loss ("Google
killed browser Timeline and is limiting data retention") — the reason a
person acts now. *Refuse:* everything else about its length: the page is the
anti-pattern for ours, and it still runs an analytics consent bar.

**Route posters (Strava heatmap, Active Maps, Cartorion, Papertrails)** —
https://www.strava.com/maps (renders only with script; not captured),
Active Maps and Papertrails did not resolve on 2026-10-29, Cartorion from
search results only. The pattern across them: one drawn image of the
person's own routes carries the whole page; the copy is a caption. *Take:*
route-as-art — the drawing is the product and the pitch at once. *Refuse:*
the poster shop's framing (sizes, prices, mockups on a wall).

## Group 2 — the same audience: self-hostable, privacy-stated software

**Protomaps** — https://protomaps.com/ · 402 words at 390 px.
First screen: a headline of nine words, one sentence, one button, and the
product itself as a scattered mosaic of map tiles around the words — the
map is seen before anything is read. Sections are short and alternate
text-only with image-plus-copy; use cases are headed "For companies / For
newsrooms / …". No cookie banner, no counter strip. *Take:* the product
shown as its own artefact, around the headline, before the copy; 400 words
for a whole page. *Refuse:* the rotated-tile scatter as a device (ours is a
plate, framed, square to the page).

**Plausible** — https://plausible.io/ · 1,092 words at 390 px.
First screen: headline with the comparison in it ("privacy-friendly Google
Analytics alternative"), a two-sentence subline that carries custody ("No
cookies, just insights. Made and hosted in the EU"), two buttons, then the
product screenshot full width. Custody appears in the hero line and again
in one dedicated subsection; a founder letter signed by two people gives the
page a voice. *Take:* custody in one line in the first screen, and the
longer statement once, later. *Refuse:* the feature grid and the pricing
cards; the SaaS kit.

**Immich** — https://immich.app/ · 199 words in the first screen.
Dark page, logo, a two-line headline ("Self-hosted photo and video
management solution"), one sentence, Download + Open Demo, a Discord link,
then a full-width product screenshot. Everything a self-hoster needs is in
the first screen. *Take:* the brevity and the order — what it is, where to
get it, what it looks like. *Refuse:* the dark SaaS register for our paper
product (but see direction B below — the dark option is being mocked).

**Kagi** — https://kagi.com/ · 920 words at 390 px. One search box in a
dark field and one line: "100% user funded. 0% advertising." *Take:* the
business model stated in six words where the eye lands. *Refuse:* the rest
(a search page, not a landing).

**Day One** — https://dayoneapp.com/ · 538 words at 390 px. A headline of
four words ("Your journal for life."), store badges, awards, then a
screenshot. *Take:* how little a confident product says above its picture.
*Refuse:* five store badges and three award laurels in the first screen —
decoration standing in for a product shot.

## Group 3 — print ancestry: the product's own vernacular

**Ordnance Survey covers, 1919–1940 (Ellis Martin and others)** —
https://commons.wikimedia.org/wiki/Category:Ordnance_Survey_map_covers ;
the Charles Close Society's index at https://charlesclosesociety.org/coversintro
(text only). The convention across two decades of covers: a title block at
the top ("ORDNANCE SURVEY · ONE-INCH MAP OF …"), **one picture in a frame**
(a walker, a view, a tourer — the use, not the map), the series and price
in a band below, two or three inks on buff card. *Take:* one framed
picture, a title block above it, one band of facts below — three elements,
and the frame is the only device. *Refuse:* the pictorial illustration; our
picture is the plate itself.

**Michelin sheet maps (the yellow series, 1910s–1950s)** —
https://commons.wikimedia.org/wiki/Category:Michelin_maps . A sheet number
in a circle at the corner, the title in a band ("MONS-LUXEMBOURG"), one map
vignette with the sheet's extent marked, the scale stated as a sentence
("1 cm pour 2 km"), two inks on yellow. *Take:* the number as the one
ornament, the scale said in words, the extent shown on a small map — a
cover that is an index to its own content. *Refuse:* nothing; this is the
closest print ancestor to a plate cover.

**The rally roadbook (the namesake)** — https://en.wikipedia.org/wiki/Roadbook
shows a page of one: columns for total distance, interval distance, the
tulip diagram (a ball where you come from, an arrow where you go), and
notes. A pure sequence device: numbers carry order because order is the
content; nothing on the page decorates. *Take:* the discipline — a device
only where it encodes something. *Refuse:* the table form for a landing.

**Atlas plates** (National Geographic, Times) — no open image set found
in the time taken (the Commons category for the Times atlas is empty). The
idiom is already in DESIGN.md §6 as the plate margin: legend, scale bar and
plate label set as marginalia. Not re-researched here.

## What the board says about our three problems

| problem | what the references do instead | the principle |
|---|---|---|
| flat hierarchy | Protomaps, Immich, Plausible, Polarsteps: one headline, one sentence, one action, and the product itself in the first screen; the OS cover: three elements, one framed | **1. One thing leads.** The plate at full sheet width is the hero; everything after it is quiet and smaller. Hierarchy by scale, not by labels. |
| too many devices | Michelin: the number is the one ornament; the roadbook: a device only where it encodes order; OS: the frame is the only device | **2. One label treatment, one rule weight.** The plate label (serif, tracked caps) is the only small-caps style; 1 px ink is the only rule; the double rule means "a drawn plate" and appears nowhere else; numerals only on the three steps (a real sequence). No eyebrows, no mono caps strings, no boxes except the form. |
| weak product proof | Protomaps and Immich show the artefact, not a greyish screenshot; the route posters are nothing but the drawing | **3. Show drawn plates, not screenshots.** The three demo adventures as covers (route art, name, dates, km, provenance bar) in place of the phase 9 screenshots. |
| still too wordy | Protomaps 402, Polarsteps 467, Day One 538 for a whole page; Dawarich 2,130 as the warning | **4. A copy budget of about 350 words on a phone**, measured; custody as one sentence in the first screen (Plausible) and one paragraph once, later. |
| the way in | Plausible and Immich: the action in the first screen and once more at the end; nothing else asks | **5. The form is the only box.** Sign-in in the masthead and the plate margin; the waitlist form at the foot is the one framed block because it is the one thing to fill in. |

Open for the discussion: whether the hero is the life map (as now) or one
adventure's cover; whether drawn covers can replace the screenshots
entirely or the life-map screenshot stays as the one "this is the app"
proof; how much custody is said in the first screen; and whether direction
B (phase 6's parked dark sheet) is mocked as the third option — agreed in
principle on 2026-10-29, to be judged on the mockup.
