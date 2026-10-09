---
name: labops-art-direction
description: Apply the lab.bingo art direction, a clean modern interface set in a maritime world with a pirate streak. Use when designing or restyling anything a visitor sees on the status page (the sea band, service rows, day marks, logbook, illustrations, mascot, favicon) or when writing its copy.
metadata:
  short-description: Follow the lab.bingo visual identity
---

# LabOps art direction

The theme is the world of piracy, applied to `lab.bingo`: a modern, uncluttered
interface that sails in maritime waters. The interface stays a tool first. The
sea is in the details and the pirate is in the wink.

The status page is one application of that world. The same direction is
written for the whole lab in the `labops` repository
(`.agents/skills/labops-art-direction`) and for the portal in its own; keep the
three in agreement on the layers, the shared colours and the mascot. Use it
together with the `frontend-design` skill: that one asks for deliberate
choices, this one says which are already made.

Read `web/src/styles.css` before designing: its `:root` tokens are the only
source of the palette and type, in light and in dark. Reuse them; when a value
is missing, derive it with `color-mix` and add a token rather than a literal
colour.

## Three layers

Keep these apart, in this order of priority.

1. **Interface: modern and sober.** Plain surfaces, legible type, slightly
   rounded shapes, generous spacing. Accent colours mark the important actions
   and states, and mean the same thing everywhere.
2. **World: maritime.** Compasses, ship's wheels, anchors, rhumb lines and
   sea-chart patterns, used as discreet background or small detail.
3. **Signature: pirate.** Carried by names, illustrations and a few winks, not
   by the chrome. The eye patch is the recurring mark of the identity.

The page must never turn into a historical set: no parchment, wood or rope
textures, no blackletter or "treasure map" fonts, no pirate speech in
headlines, states, errors or help text. A visitor comes to learn whether a
service works: every sentence that answers that is plain. The maritime world
may name a section ("Journal de bord", "Logbook") when the name still says what
the section holds.

## Shared with the other lab.bingo applications

| Role | Value |
| --- | --- |
| Sea blue (band, links) | `#1443d6`, token `--sea` |
| Deep blue (end of the band gradient) | `#0a2480`, token `--sea-deep` |
| Navy detail (eyes, patch) | `#0b2a94` |
| Mint (collar, logo dot) | `#8af2bf` |

- Blue is the sea and the brand. The state colours (`--ok`, `--warn`, `--down`,
  `--unknown`) are reserved for state, and a state is never shown by colour
  alone: it has a glyph, a word, or a shape.
- Pirate elements are drawn in navy, not black. No other brand accent.
- The band stays blue whatever the state.
- The lockup is the rounded-square "L" mark with its mint dot, next to the name.

## The status page's own character

- Type: Red Hat Display for headings, Red Hat Text for body, as on the login
  page, the other public face of the lab. Both are bundled; nothing is fetched
  from a CDN.
- **The swell is the one bold element.** The lower edge of the sea band is a
  moving wave whose height is the state of the lab: nearly flat when every
  service responds, rougher as services stop. Keep everything around it quiet,
  and do not add a second animated or decorative idea to the page.
- Below the band the page is a record, not a dashboard: no cards, rows
  separated by a hairline, text aligned left on one measure.
- Day marks read like soundings along a route: a day without trouble stands
  full height in soft blue, an interrupted one is shorter and takes a state
  colour, an unmeasured one is hollow.
- Light and dark are both designed. Check every change in the two themes.
- Copy exists in French and in English (`web/src/i18n.ts`); a new string needs
  both.

## Mascot and illustrations

The mascot is a white cat with a mint collar and a navy eye patch over its left
eye (on the viewer's right), held by a thin diagonal strap. Keep those four
traits whenever the cat is drawn. It is white, so it always sits on blue: here
it looks through a porthole in the footer (`Mascot` in `web/src/App.tsx`). One
cat per screen.

Line drawings (`CalmSea`) share one stroke on a 64 unit grid: link-blue line,
tinted fill, one accent. Pirate attributes are added one at a time, never as a
full costume.

## Motion

Animations are light and slow: the swell, a blink, an ear twitching. Nothing
moves across the reading area or delays an answer. Every animation stops under
`prefers-reduced-motion: reduce`.

## Constraints

- The page is public: it shows names, states and durations, never an internal
  address, a probe condition or an error message.
- Decoration never lowers text contrast and never overlaps a control.
- Assets are inline SVG, data URIs or bundled files.

## Before finishing

Run `make demo`, then `STATUS_DEMO=outage ./bin/status`, and look at both, in
light and dark, wide and on a phone, in French and in English. Would the page
still answer "does it work?" in two seconds with the decoration removed? Is
the swell still the only thing that moves the eye? Then update the
"Apparence" section of `README.md` if what a visitor sees has changed.
