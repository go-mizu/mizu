# 0758 — Design System

Console-grade visual consistency across every page of storage.now.

## Design Philosophy

Terminal/console aesthetic. Flat, sharp, monochrome. Every pixel intentional.
No rounded corners. No shadows. No gradients (except shimmer text effect).
Information density over whitespace. Engineered, not decorated.

---

## Audit: Current Inconsistencies

### 1. CSS Variable Names

| Page | Variable convention |
|------|-------------------|
| layout.css | `--text`, `--text-2`, `--text-3`, `--surface`, `--surface-alt`, `--border`, `--ink` |
| home.css | `--text`, `--text-2`, `--text-3`, `--surface`, `--surface-alt`, `--border`, `--ink` |
| pricing.css | `--text`, `--text-2`, `--text-3`, `--surface`, `--surface-alt`, `--border`, `--ink` |
| ai.ts (inline) | `--text`, `--text-2`, `--text-3`, `--surface`, `--surface-alt`, `--border`, `--ink` |
| cli.ts (inline) | `--text`, `--text-2`, `--text-3`, `--surface`, `--surface-alt`, `--surface-2`, `--border`, `--ink` + colors |
| developers.css | `--tx`, `--tx2`, `--tx3`, `--sf`, `--sf2`, `--sf3`, `--bd`, `--ink` + colors |

**Problem**: developers.css uses completely different variable names. cli.ts adds `--surface-2`.

### 2. Color Token Values (Light Mode)

| Token | layout.css | home.css | pricing.css | developers.css | cli.ts | ai.ts |
|-------|-----------|---------|------------|---------------|-------|------|
| text | `#09090B` | **`#18181B`** | `#09090B` | `#09090B` | **`#18181B`** | `#09090B` |
| surface (dark) | `#18181B` | `#18181B` | `#18181B` | **`#111113`** | `#18181B` | `#18181B` |
| border (dark) | `#27272A` | `#27272A` | `#27272A` | **`#1E1E21`** | `#27272A` | `#27272A` |

**Problem**: home.css and cli.ts use `#18181B` for text instead of `#09090B`. developers.css uses different dark mode surface/border.

### 3. Nav Height

| Page | Height | Mobile height |
|------|--------|--------------|
| layout.css | 56px | 52px |
| home.css | **64px** | 56px |
| pricing.css | **64px** | 52px |
| developers.css | 56px | 48px |
| cli.ts | 56px | 48px |
| ai.ts | **64px** | 52px |

**Problem**: Three different nav heights (56px, 64px). Three different mobile heights (48px, 52px, 56px).

### 4. Nav Container (max-width / padding)

| Page | max-width | padding |
|------|-----------|---------|
| layout.css | 1400px | 0 60px |
| home.css | none | 0 48px |
| pricing.css | none | 0 48px |
| developers.css | 1280px | 0 40px |
| cli.ts | 1100px | 0 60px |
| ai.ts | none | 0 48px |

**Problem**: Six different nav container configurations.

### 5. Logo Font

| Page | Font | Weight | Size |
|------|------|--------|------|
| layout.css | JetBrains Mono | 500 | 14px |
| home.css | Inter | 700 | 15px |
| pricing.css | Inter | 700 | 15px |
| developers.css | (inherited) | 700 | 15px |
| cli.ts | JetBrains Mono | 700 | 15px |
| ai.ts | Inter | 700 | 15px |

**Problem**: Logo uses Inter on some pages, JetBrains Mono on others. Different weights and sizes.

### 6. Nav Link Font

| Page | Font | Size | Weight | Spacing |
|------|------|------|--------|---------|
| layout.css | JetBrains Mono | 12px | — | 0.5px |
| home.css | Inter | 14px | 500 | — |
| pricing.css | Inter | 14px | 500 | — |
| developers.css | Inter | 13px | 500 | — |
| cli.ts | Inter | 13px | 500 | — |
| ai.ts | Inter | 14px | 500 | — |

**Problem**: layout.css uses mono 12px while others use sans 13-14px.

### 7. Nav Links (Which Links Appear)

| Page | Links shown |
|------|------------|
| home | developers, api, cli, pricing |
| developers | developers, api, cli, ai, pricing |
| pricing | developers, api, pricing |
| cli | developers, api, cli, ai, pricing |
| ai | developers, api, cli, pricing |
| api | (inline — unknown, need to check) |

**Problem**: Inconsistent navigation. Some pages missing links. AI page doesn't link to itself. Pricing missing cli and ai.

### 8. Theme Toggle

| Page | Border | Padding |
|------|--------|---------|
| layout.css | `1px solid var(--border)` | `6px 10px` |
| home.css | none | 8px |
| pricing.css | none | 8px |
| developers.css | none | 6px |
| cli.ts | none | 8px |
| ai.ts | none | 8px |

**Problem**: layout.css has bordered theme toggle; all others are borderless.

### 9. Content Max-width

| Page | Max-width | Padding |
|------|-----------|---------|
| layout.css (container) | 1100px | 24px 48px 80px |
| home.css (sections) | 1080px | 0 48px |
| developers.css (inner) | 1080px | 0 40px |
| pricing.css (section-inner) | 1100px | 0 48px |
| cli.ts (wrap) | 1100px | 0 60px |
| ai.ts (s-inner) | 1100px | 0 48px |

**Problem**: 1080px vs 1100px. Padding varies between 40px, 48px, 60px.

### 10. Grid Background

| Page | Pattern | Grid size | Opacity (light/dark) |
|------|---------|-----------|---------------------|
| home.css | dots | 32px | .25 / .08 |
| pricing.css | dots | 24px | .40 / .20 |
| developers.css | lines + dots | 80px | .25 / .06 |
| ai.ts | dots | 24px | .35 / .12 |
| cli.ts | none | — | — |

**Problem**: Four different grid background implementations. CLI has none.

### 11. Hero Title

| Page | Size | Weight | Spacing |
|------|------|--------|---------|
| home | 56px | 800 | -2.5px |
| developers | 42px | 900 | -1.8px |
| cli | 42px | 900 | -2px |
| pricing | 36px | 500 | -0.5px |
| ai | 48px | 500 | -1.5px |

**Problem**: Five different hero title treatments.

### 12. Section Labels

| Page | Size | Spacing | Weight |
|------|------|---------|--------|
| developers | 10px | 2px | 500 |
| cli | 10px | 2px | 500 |
| ai | 11px | 2.5px | 500 |

**Problem**: AI page uses slightly different label size/spacing.

### 13. Button Styles

| Page | Size | Padding | Weight |
|------|------|---------|--------|
| home | 15px | 12px 28px | 500 |
| developers | 13px | 10px 24px | 500 |
| ai | 14px | 14px 32px | 500 |

**Problem**: Three different button sizes.

### 14. Rounded Corners (Violations)

| Location | Element | Radius |
|----------|---------|--------|
| home.css | `.mock-chat-pill` | `18px` |
| home.css | `.mock-connector-field` | `8px` |
| home.css | `.mock-connector-cancel` | `8px` |
| home.css | `.mock-connector-add` | `8px` |
| home.css | `.spinner` | `50%` (acceptable — circular) |
| browse.css | various elements | likely present |

### 15. Shadows (Violations)

| Location | Element | Shadow |
|----------|---------|--------|
| layout.css | `--shadow` variable | defined but used |
| layout.css | `.mode-switch` | `box-shadow` |

### 16. CSS Architecture

| Page | Style location |
|------|---------------|
| home | External `/home.css` |
| developers | External `/developers.css` |
| pricing | External `/pricing.css` |
| browse | External `/browse.css` + `/browse.js` |
| cli | **Inline `<style>` in cli.ts** (~300 lines) |
| ai | **Inline `<style>` in ai.ts** (~228 lines) |
| privacy | **Inline `<style>` in privacy.ts** (~30 lines) |
| api reference | **Inline `<style>` in index.ts** |

**Problem**: Major duplication. cli.ts and ai.ts each contain full standalone CSS. Every nav/theme-toggle/grid-bg re-implemented.

### 17. Sign-out Button

| Page | Font | Size | Padding |
|------|------|------|---------|
| layout.css | JetBrains Mono | 11px | 5px 12px |
| home.css | Inter | 13px | 6px 16px |
| cli.ts | Inter | 12px | 5px 14px |

---

## Design System Specification

### Color Tokens

```css
:root {
  --bg: #FAFAF9;
  --surface: #FFF;
  --surface-alt: #F4F4F5;
  --text: #09090B;
  --text-2: #52525B;
  --text-3: #A1A1AA;
  --border: #E4E4E7;
  --ink: #09090B;
  --green: #22C55E;
  --blue: #3B82F6;
  --amber: #F59E0B;
  --red: #EF4444;
}
html.dark {
  --bg: #09090B;
  --surface: #18181B;
  --surface-alt: #18181B;
  --text: #FAFAF9;
  --text-2: #A1A1AA;
  --text-3: #52525B;
  --border: #27272A;
  --ink: #FAFAF9;
  --green: #4ADE80;
  --blue: #60A5FA;
  --amber: #FBBF24;
  --red: #F87171;
}
```

**Rules**:
- `--text` is always `#09090B` in light mode (not `#18181B`)
- `--surface` (dark) is always `#18181B` (not `#111113`)
- `--border` (dark) is always `#27272A` (not `#1E1E21`)
- developers.css variable names (`--tx`, `--bd`, etc.) must be renamed to match

### Typography

| Role | Font | Weight | Size | Spacing |
|------|------|--------|------|---------|
| Body | Inter | 400 | 14px | — |
| Logo | JetBrains Mono | 500 | 14px | -0.3px |
| Nav links | JetBrains Mono | 400 | 12px | 0.5px |
| Section label | JetBrains Mono | 500 | 10px | 2px, uppercase |
| Section heading | Inter | 700 | 28px | -0.8px |
| Hero title (primary) | Inter | 800 | 48px | -2px |
| Hero title (secondary) | Inter | 700 | 36px | -1.5px |
| Hero subtitle | Inter | 400 | 16px | — |
| Code / mono | JetBrains Mono | 400 | 12px | — |
| Code inline | JetBrains Mono | 400 | 0.85em | — |
| Button | Inter | 500 | 13px | — |
| Button primary | Inter | 600 | 13px | — |

### Spacing Scale

Base unit: 4px. Use multiples: 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80.

| Token | Value | Usage |
|-------|-------|-------|
| Content max-width | 1100px | All page content |
| Nav max-width | 1100px | Nav inner container |
| Content padding (desktop) | 0 48px | Horizontal gutter |
| Content padding (mobile) | 0 20px | Mobile gutter |
| Section padding | 80px 0 | Vertical section spacing |
| Nav height | 56px | Desktop |
| Nav height (mobile) | 48px | Mobile breakpoint |

### Nav Bar

```
Height: 56px (desktop), 48px (mobile <=640px)
Max-width: 1100px, centered
Padding: 0 48px (desktop), 0 20px (mobile)
Background: color-mix(in srgb, var(--bg) 85%, transparent)
Backdrop-filter: blur(16px)
Border: none (no bottom border)
```

**Logo**: JetBrains Mono, 500 weight, 14px, -0.3px spacing.
Square dot (6x6px, var(--text), no border-radius).

**Links** (consistent across ALL pages):
```
developers | api | cli | ai | pricing
```
Font: JetBrains Mono, 12px, 0.5px letter-spacing, color var(--text-3).
Active state: color var(--text).

**Theme toggle**: No border. Padding 6px. color var(--text-3), hover var(--text).

**Sign-out button**: JetBrains Mono, 11px, border 1px solid var(--border), padding 5px 12px.

### Buttons

```css
.btn {
  font-size: 13px;
  font-weight: 500;
  padding: 10px 24px;
  border: 1px solid var(--border);
  border-radius: 0;           /* NEVER rounded */
  box-shadow: none;            /* NEVER shadow */
  display: inline-flex;
  align-items: center;
  gap: 8px;
  transition: all .15s;
  color: var(--text-2);
  cursor: pointer;
}
.btn:hover {
  border-color: var(--text-3);
  color: var(--text);
}
.btn--primary {
  background: var(--ink);
  border-color: var(--ink);
  color: var(--bg);
  font-weight: 600;
}
.btn--primary:hover {
  opacity: 0.85;
  color: var(--bg);
}
```

### Grid Background

Consistent across all pages that use it:

```css
.grid-bg {
  position: fixed; inset: 0;
  pointer-events: none; z-index: 0;
  background-image: radial-gradient(circle, var(--border) 1px, transparent 1px);
  background-size: 24px 24px;
  opacity: .3;
}
html.dark .grid-bg { opacity: .1; }
```

### Borders & Edges

- `border-radius: 0` — always, everywhere. No exceptions.
- `box-shadow: none` — always. No shadows.
- Standard border: `1px solid var(--border)`
- Grid gap pattern: Use `gap: 1px; background: var(--border)` for card grids (children have `background: var(--bg)`).

### Code Blocks

```css
code {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.85em;
  padding: 2px 6px;
  border: 1px solid var(--border);
  background: var(--surface-alt);
  border-radius: 0;
}
```

### Terminal Mockups

```css
.term {
  border: 1px solid var(--border);
  background: var(--surface);
  border-radius: 0;
  box-shadow: none;
  overflow: hidden;
}
.term-bar {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border);
  gap: 10px;
  background: var(--surface-alt);
}
.term-dots {
  display: flex; gap: 5px;
}
.term-dots span {
  width: 8px; height: 8px;
  background: var(--border);
  border-radius: 0;           /* Square dots, not circles */
}
html.dark .term-dots span {
  background: var(--text-3);
}
.term-title {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10px;
  color: var(--text-3);
  flex: 1;
  text-align: center;
}
.term-body {
  padding: 18px 20px;
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  line-height: 1.8;
  overflow-x: auto;
}
```

### Section Pattern

```css
.section {
  position: relative;
  z-index: 1;
  width: 100%;
  padding: 80px 0;
  opacity: 0;
  transform: translateY(20px);
  transition: opacity .7s ease, transform .7s ease;
}
.section.visible {
  opacity: 1;
  transform: none;
}
.section-inner {
  max-width: 1100px;
  margin: 0 auto;
  padding: 0 48px;
}
.section-label {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10px;
  letter-spacing: 2px;
  color: var(--text-3);
  font-weight: 500;
  text-transform: uppercase;
  margin-bottom: 12px;
}
.section-heading {
  font-size: 28px;
  font-weight: 700;
  letter-spacing: -0.8px;
  line-height: 1.2;
  margin-bottom: 12px;
}
.section-sub {
  font-size: 15px;
  color: var(--text-2);
  line-height: 1.7;
  margin-bottom: 32px;
  max-width: 520px;
}
```

### Card Grid Pattern

```css
.card-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1px;
  background: var(--border);
  border: 1px solid var(--border);
  overflow: hidden;
}
.card-grid > * {
  background: var(--bg);
  padding: 24px;
  transition: background .15s;
}
.card-grid > *:hover {
  background: var(--surface-alt);
}
```

### Responsive Breakpoints

| Breakpoint | Changes |
|-----------|---------|
| <=1024px | Grids collapse to 1-2 columns. Split layouts stack. |
| <=768px | Hero titles shrink. Section padding reduces. |
| <=640px | Nav collapses to hamburger. Content padding 0 20px. Nav height 48px. |

### Forbidden

- `border-radius` on any UI element (exception: only spinner animation)
- `box-shadow` anywhere
- `text-shadow` anywhere
- Gradient backgrounds (exception: shimmer text animation)
- `--shadow` CSS variable
- Different variable names per file (`--tx` vs `--text`)
- Inline `<style>` blocks > 20 lines (extract to external CSS)

---

## Implementation Tasks

### Task 1: Extract shared base CSS (`/public/base.css`)

Create a single shared CSS file containing:
- CSS reset
- Color tokens (`:root` + `html.dark`)
- Typography base (body, a, code)
- Nav styles (complete, canonical)
- Grid background
- Theme toggle
- Button system (`.btn`, `.btn--primary`, `.btn--ghost`)
- Section pattern (`.section`, `.section-inner`, `.section-label`, `.section-heading`, `.section-sub`)
- Terminal mockup (`.term`, `.term-bar`, `.term-dots`, `.term-body`)
- Card grid (`.card-grid`)
- Mode switch
- Machine view / markdown view
- Responsive breakpoints for shared components

All pages will `<link>` this file first, then their page-specific CSS.

### Task 2: Fix home page (`home.css` + `home.ts`)

- Change `--text` from `#18181B` to `#09090B`
- Remove nav styles (use base.css)
- Remove grid-bg styles (use base.css)
- Remove mode-switch styles (use base.css)
- Fix logo from Inter to JetBrains Mono
- Fix nav link font from Inter 14px to JetBrains Mono 12px
- Fix nav height from 64px to 56px
- Fix content max-width from 1080px to 1100px
- Remove `border-radius: 18px` from `.mock-chat-pill`
- Remove `border-radius: 8px` from connector mockup elements
- Add missing `ai` link to nav in `home.ts`
- Remove `--shadow` usage

### Task 3: Fix developers page (`developers.css` + `developers.ts`)

- Rename all CSS variables: `--tx` -> `--text`, `--tx2` -> `--text-2`, `--tx3` -> `--text-3`, `--sf` -> `--surface`, `--sf2` -> `--surface-alt`, `--sf3` -> (remove or map), `--bd` -> `--border`
- Fix dark `--surface` from `#111113` to `#18181B`
- Fix dark `--border` from `#1E1E21` to `#27272A`
- Remove nav styles (use base.css)
- Remove grid-bg styles — switch from line grid to dot grid (use base.css)
- Fix nav max-width from 1280px to 1100px
- Fix content padding from 0 40px to 0 48px
- Fix button size from 13px/10px 24px to match spec
- Fix section heading from 30px to 28px

### Task 4: Fix pricing page (`pricing.css` + `pricing.ts`)

- Remove nav styles (use base.css)
- Remove grid-bg styles (use base.css)
- Fix nav height from 64px to 56px
- Fix logo from Inter to JetBrains Mono
- Fix nav link font from Inter 14px to JetBrains Mono 12px
- Fix grid-bg opacity from .4/.2 to .3/.1
- Add missing `cli` and `ai` links to nav in `pricing.ts`

### Task 5: Fix CLI page (extract inline CSS from `cli.ts`)

- Extract all CSS from inline `<style>` to `/public/cli.css`
- Remove duplicated nav/toggle/grid-bg styles (use base.css)
- Fix `--text` from `#18181B` to `#09090B`
- Fix nav height to 56px (was mixed)
- Fix logo font to JetBrains Mono 500 14px
- Add grid background (currently missing)
- Fix nav link font to JetBrains Mono 12px
- Fix content padding from 0 60px to 0 48px
- Link base.css + cli.css in `<head>`

### Task 6: Fix AI page (extract inline CSS from `ai.ts`)

- Extract all CSS from inline `<style>` to `/public/ai.css`
- Remove duplicated nav/toggle/grid-bg styles (use base.css)
- Fix nav height from 64px to 56px
- Fix logo from Inter to JetBrains Mono 500 14px
- Fix nav link font from Inter 14px to JetBrains Mono 12px
- Fix section label from 11px/2.5px to 10px/2px
- Fix hero title weight from 500 to 800
- Add `ai` link (active) to nav
- Link base.css + ai.css in `<head>`

### Task 7: Fix API reference page (inline CSS in `index.ts`)

- Ensure it uses base.css
- Fix nav to match spec
- Add all nav links (developers, api, cli, ai, pricing)
- Ensure no border-radius or shadows

### Task 8: Fix browse page (`browse.css` + `browse.ts` + `browse.js`)

- Audit for border-radius and shadows, remove
- Ensure color tokens match spec
- Ensure nav matches spec

### Task 9: Fix privacy page (`privacy.ts`)

- Fix dark `--surface` from `#111113` to `#18181B`
- Fix dark `--border` from `#1E1E21` to `#27272A`
- Minimal page — add a simple back-link style, keep inline CSS short

### Task 10: Fix layout.css (shared directory/immersive layout)

- Fix nav to match spec (height, padding, fonts)
- Remove `--shadow` variable
- Remove `box-shadow` from `.mode-switch`
- Fix nav max-width from 1400px to 1100px
- Fix nav padding from 0 60px to 0 48px
- Fix theme toggle to remove border

### Task 11: Delete layout.css, replace with base.css references

After base.css is created, layout.css should only contain styles specific to directory/immersive layouts (machine view, mode switch). All generic nav/theme/container styles move to base.css.

---

## Verification Checklist

After all tasks are complete, every page must pass:

- [ ] Uses `<link rel="stylesheet" href="/base.css">` as first stylesheet
- [ ] No inline `<style>` blocks > 20 lines
- [ ] Color tokens identical to spec (diff `:root` and `html.dark` blocks)
- [ ] Nav height 56px desktop, 48px mobile
- [ ] Nav links: developers | api | cli | ai | pricing (all five, correct active state)
- [ ] Logo: JetBrains Mono, 500, 14px
- [ ] Nav links: JetBrains Mono, 12px, 0.5px spacing
- [ ] Theme toggle: no border
- [ ] Content max-width: 1100px
- [ ] Content padding: 0 48px desktop, 0 20px mobile
- [ ] Grid background: dots, 24px, opacity .3/.1
- [ ] Zero `border-radius` in computed styles (grep for `radius`)
- [ ] Zero `box-shadow` in computed styles (grep for `shadow`)
- [ ] Section labels: JetBrains Mono, 10px, 2px spacing
- [ ] Buttons: 13px, 10px 24px padding
