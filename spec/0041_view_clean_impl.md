# View Package Cleanup - Implementation Plan

This document outlines the implementation plan for simplifying the view package according to spec/0041_view_clean.md.

## Goal

Transform the view package from a custom rendering framework into "Go templates with layouts" - simple, teachable, and aligned with standard Go template patterns.

## Current State Analysis

### Files to Modify
- `view/view.go` (731 lines) - Main implementation
- `view/view_test.go` (678 lines) - Tests

### Test Templates Using Features to Remove
- `testdata/views/pages/home.html` - Uses `{{define "title"}}` for slot extraction
- `testdata/views/pages/with-layout.html` - Uses `{{define "layout"}}bare{{end}}` magic
- `testdata/views/pages/with-component.html` - Uses `{{component "button" ...}}`
- `testdata/views/pages/with-partial.html` - Uses `{{partial "sidebar"}}`
- `testdata/views/layouts/default.html` - Uses `{{slot}}` and `{{stack}}`
- `testdata/views/layouts/bare.html` - Uses `{{slot "content"}}`

## Implementation Tasks

### Phase 1: Core Simplification (view.go)

#### 1.1 Remove Config Fields
**Lines 48-59**: Remove `StrictMode` and `DedupeStacks` from Config struct.

```go
// BEFORE
type Config struct {
    Dir           string
    FS            fs.FS
    Extension     string
    DefaultLayout string
    Funcs         template.FuncMap
    Delims        [2]string
    Development   bool
    StrictMode    bool           // REMOVE
    DedupeStacks  bool           // REMOVE
}

// AFTER
type Config struct {
    Dir           string
    FS            fs.FS
    Extension     string
    DefaultLayout string
    Funcs         template.FuncMap
    Delims        [2]string
    Development   bool
}
```

#### 1.2 Remove renderCtx Type and Methods
**Lines 382-459**: Delete the entire `renderCtx` struct and all methods:
- `renderCtx` struct
- `newRenderCtx()`
- `setSlot()`
- `slot()`
- `push()`
- `stack()`
- `setChildren()`
- `children()`

#### 1.3 Remove parseWithCtx Function
**Lines 336-380**: Delete `parseWithCtx` - use `parse` everywhere instead.

#### 1.4 Remove Engine.Component and Engine.Partial
**Lines 208-248**: Delete these methods:
- `Engine.Component()`
- `Engine.Partial()`
- Internal `component()` helper
- Internal `partial()` helper

#### 1.5 Simplify Engine.Render
**Lines 127-206**: Rewrite to remove:
- `renderCtx` creation (line 134)
- Layout extraction via `tmpl.Lookup("layout")` (lines 150-159)
- Slot extraction loop (lines 175-185)
- `ctx.setSlot()` calls

New render flow:
1. Load page content
2. Parse page with standard `parse()`
3. Execute page content to buffer
4. Load layout content
5. Parse layout
6. Execute layout with page content as `.Content` in data

#### 1.6 Update Load() Method
**Lines 110-118**: Remove "components" and "partials" from directories to load.

#### 1.7 Update load() Kind Mapping
**Lines 271-289**: Remove "component" and "partial" kind mappings.

### Phase 2: Template Functions Cleanup (baseFuncs)

#### 2.1 Remove Placeholders
**Lines 546-553**: Delete these stub functions:
- `slot`
- `stack`
- `push`
- `component`
- `partial`
- `children`

#### 2.2 Remove Unsafe Functions
**Lines 562-565**: Remove footgun functions:
- `safeCSS`
- `safeJS`
- `safeURL`

Keep only `safeHTML` (commonly needed, document carefully).

#### 2.3 Remove Deprecated/Problematic Functions
**Line 570**: Remove `title` (deprecated, culturally problematic)

#### 2.4 Remove Math/Numeric Functions
**Lines 584-606**: Remove all comparison and math functions:
- `lt`, `le`, `gt`, `ge` (keep `eq`, `ne` as they use reflect.DeepEqual)
- `add`, `sub`, `mul`, `div`, `mod`

#### 2.5 Remove Conditional Helpers
**Lines 580-581**: Remove:
- `ternary`
- `coalesce`

#### 2.6 Remove Type Conversion Functions
**Lines 672-730**: Delete:
- `toFloat64()`
- `toInt64()`
- `ternaryFunc()`
- `coalesceFunc()`

### Phase 3: Error Type Simplification

#### 3.1 Update Error.Kind Comment
**Line 26**: Update comment to remove "component" and "partial":
```go
Kind string // "page", "layout", "template"
```

#### 3.2 Remove Line Field (Optional)
**Line 28**: The `Line` field is never set - consider removing or keeping for future use.

### Phase 4: Mizu Integration Cleanup

#### 4.1 Rename Handler to Middleware
**Lines 498-507**: Rename `Handler()` to `Middleware()` for consistency.

#### 4.2 Remove Component Package Function
**Lines 534-542**: Delete `Component()` package-level function.

### Phase 5: Template Files Update

Update test templates to use standard Go template patterns:

#### 5.1 layouts/default.html
```html
<!-- BEFORE -->
<title>{{slot "title" "Default Title"}}</title>
{{slot "content"}}

<!-- AFTER -->
<title>{{.Page.Title}}</title>
{{.Content}}
```

#### 5.2 layouts/bare.html
```html
<!-- BEFORE -->
{{slot "content"}}

<!-- AFTER -->
{{.Content}}
```

#### 5.3 pages/home.html
```html
<!-- BEFORE -->
{{define "title"}}Home Page{{end}}
{{define "content"}}...{{end}}

<!-- AFTER -->
<h1>Welcome, {{.Data.Name}}</h1>
<p>This is the home page.</p>
```

#### 5.4 pages/with-layout.html
```html
<!-- BEFORE -->
{{define "layout"}}bare{{end}}
{{define "content"}}...{{end}}

<!-- AFTER -->
<p>Using bare layout</p>
```
(Layout selection via `Layout()` option only)

#### 5.5 pages/with-component.html
```html
<!-- BEFORE -->
{{component "button" (dict "Label" "Click Me" "Variant" "primary")}}

<!-- AFTER -->
{{template "components/button" (dict "Label" "Click Me" "Variant" "primary")}}
```

#### 5.6 pages/with-partial.html
```html
<!-- BEFORE -->
{{partial "sidebar"}}

<!-- AFTER -->
{{template "partials/sidebar" .}}
```

### Phase 6: Test Updates

#### 6.1 Remove Tests for Deleted Features
- `TestEngine_Component` - Remove or convert to standard template test
- `TestEngine_ComponentInPage` - Update to use `{{template}}`
- `TestEngine_PartialInPage` - Update to use `{{template}}`
- `TestEngine_Slots` - Remove (slots no longer exist)
- `TestComponent_Handler` - Remove

#### 6.2 Remove Template Function Tests
- `TestTernaryFunc` - Remove
- `TestCoalesceFunc` - Remove
- `TestComparisonFuncs` - Keep only eq/ne tests
- `TestMathFuncs` - Remove

#### 6.3 Update Error Tests
- Update `TestErrors` to remove component/partial kind tests

### Phase 7: New Architecture

The new simplified architecture:

```go
// Engine parses all templates into a single template.Template set
type Engine struct {
    cfg   Config
    fs    fs.FS
    mu    sync.RWMutex
    cache map[string]string
    funcs template.FuncMap
    tmpl  *template.Template  // Single template set (optional, for full preload)
}

// Load parses all templates into a single set
func (e *Engine) Load() error {
    // Walk all directories, parse into single template
    // Name templates as: "layouts/default", "pages/home", "components/button"
}

// Render executes page within layout
func (e *Engine) Render(w io.Writer, page string, data any, opts ...option) error {
    // 1. Parse/get page template
    // 2. Execute page to get content
    // 3. Parse/get layout template
    // 4. Execute layout with {Page, Data, Content}
}
```

## Summary: Lines to Remove/Modify

| Component | Lines | Action |
|-----------|-------|--------|
| Config.StrictMode, DedupeStacks | 57-58 | Remove |
| renderCtx type + methods | 382-459 | Delete entire section |
| parseWithCtx | 336-380 | Delete |
| Engine.Component | 208-212 | Delete |
| Engine.Partial | 214-218 | Delete |
| component() helper | 220-233 | Delete |
| partial() helper | 235-248 | Delete |
| Layout extraction | 150-159 | Remove from Render |
| Slot extraction loop | 175-185 | Remove from Render |
| baseFuncs placeholders | 547-553 | Delete |
| safeCSS/JS/URL | 563-565 | Delete |
| title | 570 | Delete |
| lt/le/gt/ge | 586-589 | Delete |
| add/sub/mul/div/mod | 592-606 | Delete |
| ternary/coalesce | 580-581 | Delete |
| toFloat64/toInt64 | 672-730 | Delete |
| ternaryFunc/coalesceFunc | 656-670 | Delete |
| Component package func | 534-542 | Delete |
| Handler rename | 498 | Rename to Middleware |

## Estimated Impact

- **Before**: ~730 lines in view.go, ~678 lines in view_test.go
- **After**: ~350-400 lines (estimated 50% reduction)
- **Concepts removed**: slots, stacks, children, components, partials, layout-by-template
- **Concepts kept**: layouts, pages, standard `{{template}}` composition

## Implementation Order

1. Start with baseFuncs cleanup (safe, isolated)
2. Remove renderCtx and parseWithCtx
3. Simplify Render method
4. Remove Component/Partial methods
5. Update Config
6. Rename Handler to Middleware
7. Update test templates
8. Update tests
9. Run tests, fix any issues

---

## Implementation Completed

All phases have been implemented successfully.

### Final Results

| File | Before | After | Reduction |
|------|--------|-------|-----------|
| view/view.go | 731 lines | 404 lines | 45% |
| view/view_test.go | 678 lines | 457 lines | 33% |

### Changes Made

1. **Config**: Removed `StrictMode` and `DedupeStacks` fields
2. **Error**: Removed `Line` field (never set), updated Kind comment
3. **renderCtx**: Deleted entire type and all methods
4. **parseWithCtx**: Deleted function
5. **Engine.Component/Partial**: Deleted methods and helpers
6. **Render**: Simplified to page-content-layout flow using `.Content`
7. **Load()**: Now only loads `layouts` and `pages` directories
8. **baseFuncs()**: Reduced to minimal set:
   - Data helpers: `dict`, `list`, `default`, `empty`
   - Safe content: `safeHTML` only
   - String helpers: `upper`, `lower`, `trim`, `contains`, `replace`, `split`, `join`, `hasPrefix`, `hasSuffix`
   - Comparisons: `eq`, `ne` only
9. **Handler()**: Renamed to `Middleware()`
10. **Component()**: Removed package-level function

### Test Templates Updated

- `layouts/default.html`: Uses `{{.Content}}` instead of `{{slot "content"}}`
- `layouts/bare.html`: Uses `{{.Content}}` instead of `{{slot "content"}}`
- `pages/home.html`: Simple content without slot definitions
- `pages/with-layout.html`: Layout via `Layout()` option only
- `pages/with-component.html`: Inline button content
- `pages/with-partial.html`: Inline sidebar content

### Tests Updated

- Removed: `TestEngine_Component`, `TestComponent_Handler`, `TestEngine_Slots`
- Removed: `TestTernaryFunc`, `TestCoalesceFunc`, `TestMathFuncs`
- Updated: `TestComparisonFuncs` (only eq/ne), `TestErrors` (removed component/partial kinds)
- Updated: All tests using `Handler()` now use `Middleware()`

All tests pass.
