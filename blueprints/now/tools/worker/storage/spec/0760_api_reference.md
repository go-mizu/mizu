# 0760 — API Reference Page Redesign

**Status:** Draft
**Date:** 2026-03-20
**Inspired by:** [OpenAI API Reference](https://developers.openai.com/api/reference/overview)

---

## Problem

The `/api` page at `https://storage.liteio.dev/api` is broken. The `renderMarkdownDocs()` function generates markdown-like text but wraps it in a `<pre>` tag — headers, code fences, bold markers, and bullets all render as literal characters. Additional issues:

- `$ref` not resolved → error responses show raw `{ "$ref": "#/components/schemas/Error" }`
- No navigation or table of contents
- No base URL shown
- Auth mechanism not documented (just "Auth required" badge)
- Parameter required/optional not shown
- Enum values, defaults, patterns, descriptions ignored
- `POST /files/mkdir` missing from OpenAPI spec entirely
- CSS targets elements (`h2`, `h3`) that never exist inside `<pre>`

## Design Decisions

### D1: Single-page, dynamically generated from OpenAPI spec

Keep the current approach: the `/api` route calls `app.getOpenAPIDocument()` at runtime and renders HTML. No static site generator, no build step, no external framework. One self-contained function in the worker.

**Rationale:** The OpenAPI spec is auto-generated from route code. A dynamic page guarantees docs are always in sync with deployed code. Zero additional infrastructure.

### D2: Two-column layout inspired by OpenAI

```
┌──────────────────────────────────────────────────┐
│  Storage API Reference              [dark/light] │
├────────────┬─────────────────┬───────────────────┤
│  Sidebar   │  Description    │  Examples         │
│            │                 │                   │
│  Overview  │  ## Files       │                   │
│            │                 │                   │
│  Files     │  ### List files │  curl example     │
│   List     │  GET /files     │                   │
│   Upload   │                 │  Response JSON    │
│   Download │  Query params:  │                   │
│   Delete   │  - prefix       │                   │
│   ...      │  - limit        │                   │
│            │                 │                   │
│  Uploads   │  Returns:       │                   │
│   ...      │  FileList obj   │                   │
│            │                 │                   │
│  Auth      │  ---            │                   │
│   ...      │                 │                   │
│            │  ### Upload     │  curl example     │
│  Keys      │  POST /files/   │                   │
│   ...      │  uploads        │  Response JSON    │
└────────────┴─────────────────┴───────────────────┘
```

- **Sidebar** (~220px): Fixed, scrollable. Grouped by tag. Links anchor to each endpoint.
- **Left column**: Endpoint description, parameters table, returns schema.
- **Right column**: Curl example + JSON response example. Sticky-positioned so examples stay visible while scrolling description.
- **Responsive**: Collapses to single column below 1024px (examples move below description).

### D3: Endpoint naming convention

Follow OpenAI's "Verb a noun" pattern in sidebar and headings:

| Current tag + summary | New display name |
|---|---|
| files / List files | List files |
| files / Upload a file (presigned) | Upload a file |
| files / Complete upload | Complete an upload |
| files / Download a file | Retrieve a file |
| files / File metadata | Retrieve file metadata |
| files / Delete a file | Delete a file |
| files / Search files | Search files |
| files / Storage stats | Retrieve storage stats |
| files / Move / rename | Move a file |
| files / Create share link | Share a file |
| files / Create folder | Create a folder |
| uploads / Initiate multipart | Create a multipart upload |
| uploads / Complete multipart | Complete a multipart upload |
| uploads / Abort multipart | Abort a multipart upload |

### D4: Per-endpoint structure

Each endpoint section contains (in order):

1. **Heading**: `List files` (h3)
2. **Method + path**: `GET /files` — method in colored badge (GET=green, POST=blue, PUT=orange, DELETE=red, HEAD=gray)
3. **Description**: One terse sentence. "Returns a list of files in the authenticated user's storage."
4. **Auth badge**: 🔒 if `security` is set. Link to auth section.
5. **Parameters table** (if any):
   - Grouped: Path Parameters, Query Parameters, Body Parameters
   - Each param: `name` (monospace) + `type` + required/optional + description
   - Enum values shown inline: `"asc" | "desc"`
   - Defaults shown: `(default: 10000)`
6. **Returns section**:
   - Response object shape with each field documented
   - Type shown for each field
7. **Right column — curl example**:
   - Complete, copy-pasteable curl command
   - Uses `$STORAGE_API_KEY` placeholder for auth
   - Base URL: `https://storage.liteio.dev`
8. **Right column — response example**:
   - Realistic JSON with example values (not empty strings / zeros)

### D5: Writing style (matching OpenAI's tone)

- **Terse, imperative, technical.** No conversational filler.
- **Descriptions**: Start with a verb. "Returns a list of files." "Uploads a file using a presigned URL." "Deletes a file permanently."
- **Parameters**: One sentence max. "A folder prefix to filter results (e.g. `docs/`)." "Maximum number of results to return."
- **Required fields**: Omit the word "required" — absence of "optional" implies required (OpenAI pattern).
- **Optional fields**: Show `optional` keyword before type.

### D6: Overview section

The page starts with an overview section (before any endpoints) containing:

1. **Base URL**: `https://storage.liteio.dev`
2. **Authentication**: Bearer token via `Authorization: Bearer <key>`. Keys created via `POST /auth/keys` or the dashboard. Brief curl example.
3. **Content type**: JSON request/response bodies use `application/json`. File downloads return `302` redirect to presigned R2 URL (or JSON metadata with `Accept: application/json`).
4. **Errors**: Standard `{ "error": "code", "message": "Human-readable description" }` shape. Common codes: `unauthorized`, `not_found`, `conflict`, `bad_request`.
5. **Rate limits**: Mention if applicable, or omit.

### D7: Color scheme and typography

CSS-only, no external dependencies. Dark-friendly with `prefers-color-scheme` media query.

**Light mode:**
- Background: `#ffffff`, Sidebar bg: `#f9fafb`
- Text: `#1a1a2e`, Muted: `#6b7280`
- Code bg: `#f3f4f6`, Code text: `#1a1a2e`
- Accent (links, GET badge): `#10b981`

**Dark mode:**
- Background: `#0f0f1a`, Sidebar bg: `#1a1a2e`
- Text: `#e5e7eb`, Muted: `#9ca3af`
- Code bg: `#1e1e30`, Code text: `#e5e7eb`

**Typography:**
- Font: `system-ui, -apple-system, sans-serif`
- Code: `'SF Mono', 'Fira Code', 'Cascadia Code', monospace`
- Base size: 15px, line-height: 1.6

### D8: $ref resolution

Resolve `$ref` pointers before rendering. Walk the spec and replace `{ "$ref": "#/components/schemas/Error" }` with the actual schema from `components.schemas.Error`. This is a simple recursive function since our spec only has one level of refs.

### D9: Realistic examples via schema `example` values

Use `example` values from Zod schemas (already present for many fields via `.openapi({ example: "..." })`). Fall back to type-based placeholders only when no example exists. Provide richer fallbacks:

| Type | Fallback |
|---|---|
| `string` | `"string"` (not `""`) |
| `number` / `integer` | `0` |
| `boolean` | `true` |
| `array` | `[]` |

### D10: No external dependencies

The entire page is generated as a self-contained HTML string. No CDN links, no JS frameworks, no fetching external CSS. The page must work offline and load instantly.

Exception: minimal inline JS for:
- Sidebar toggle on mobile
- Copy-to-clipboard on curl examples
- Smooth scroll to anchor

---

## Implementation Tasks

### Task 1: Fix OpenAPI spec completeness

**File:** `src/routes/files-v2.ts`

- Add `registerPath()` for `POST /files/mkdir`
- Review all existing `registerPath()` calls — ensure descriptions are filled in for every parameter and response field
- Add `example` values to Zod schemas where missing
- Ensure consistent tag naming and summaries matching D3 naming convention

### Task 2: Rewrite `renderMarkdownDocs()` → `renderApiReference()`

**File:** `src/index.ts`

Replace the broken `renderMarkdownDocs` + `schemaToExample` functions with a new `renderApiReference(spec)` function that:

1. Resolves all `$ref` pointers in the spec
2. Generates the overview section (base URL, auth, errors)
3. Iterates paths grouped by tag
4. For each operation, renders the two-column layout (description left, examples right)
5. Generates curl examples from the operation metadata
6. Generates response JSON examples using `example` values
7. Renders parameters with type, required/optional, description, defaults, enums
8. Renders response schemas with field documentation
9. Generates the sidebar navigation from tags and operations

Output: A single HTML string with embedded `<style>` and minimal `<script>`.

### Task 3: Build the sidebar navigation

Part of Task 2, but deserves explicit attention:

- Extract tags and operations from spec
- Generate anchor IDs: `tag-{tag}` for section headers, `op-{method}-{path-slug}` for endpoints
- Render as nested `<nav>` with tag groups
- Highlight current section on scroll (IntersectionObserver, ~15 lines of JS)
- Mobile: hamburger toggle to show/hide sidebar

### Task 4: Style the two-column layout

Part of Task 2 CSS. Key requirements:

- Three-panel grid: `grid-template-columns: 220px 1fr 1fr`
- Right column (examples) uses `position: sticky; top: 20px` so examples stay visible
- Method badges with HTTP-method-specific colors
- Parameter table with alternating subtle backgrounds
- Responsive breakpoint at 1024px → single column, sidebar becomes collapsible top nav
- `prefers-color-scheme` media query for dark mode
- Print-friendly: hide sidebar, single column

### Task 5: Generate curl examples

Part of Task 2. For each operation:

1. Start with `curl https://storage.liteio.dev{path}`
2. Add `-X {METHOD}` (omit for GET)
3. Add `-H "Authorization: Bearer $STORAGE_API_KEY"` if `security` is set
4. Add `-H "Content-Type: application/json"` if request body exists
5. Add `-d '{...}'` with example body JSON (from schema examples)
6. For query parameters, append `?param=example` to URL
7. For path parameters, substitute `{param}` with example values

### Task 6: QA and edge cases

- Verify every endpoint renders correctly
- Test with no auth (overview page should still load)
- Test dark mode
- Test mobile responsive layout
- Verify curl examples are copy-pasteable and work
- Check that anchor links from sidebar work
- Confirm the page loads fast (target: <50ms render time since it's edge-computed)

---

## Out of Scope

- **Multiple language examples** (TypeScript, Python SDK tabs) — we only have curl for now. Can add later.
- **Try-it / interactive playground** — future enhancement.
- **Versioned docs** — single version for now.
- **Search** — page is small enough to Ctrl+F.

## Open Questions

1. Should the `/api` route stay at `/api` or move to `/docs/api` or `/reference`?
2. Should we add a "Copy as curl" button per endpoint? (Proposed: yes, included in Task 2)
3. The `/docs` route currently serves Swagger UI. Keep both `/docs` (Swagger) and `/api` (our reference)? Or replace Swagger UI with our page?
