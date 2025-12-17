# Implementation Plan: View Package Cleanup R2

**Status: COMPLETED**

Based on spec `0041_view_clean_r2.md`, this document outlines the concrete changes to simplify the view package.

## Summary of Changes

### 1. Remove `Status` option (line 279-281)
**Rationale**: `Engine.Render` ignores status entirely. Status control belongs at HTTP handler layer via `Render(c, ...)`.

**Changes**:
- Delete `Status(code int) option` function
- Remove `status int` field from `renderCfg` struct
- Keep status handling only in `Render(c *mizu.Ctx, ...)` function

**Files**: `view/view.go`

### 2. Remove `Title` and `CSRF` fields from pageData (lines 256-268)
**Rationale**: Never set, never used. Premature API surface that invites confusion.

**Changes**:
- Remove `Title string` from `pageMeta` struct
- Remove `CSRF string` from `pageData` struct
- Update testdata templates that reference `.Page.Title`

**Files**: `view/view.go`, `view/testdata/views/layouts/default.html`

### 3. Remove `safeHTML` from base funcs (line 340)
**Rationale**: Footgun that causes accidental XSS. Users can opt-in via `Config.Funcs`.

**Changes**:
- Remove `safeHTML` from `baseFuncs()` function

**Files**: `view/view.go`

### 4. Remove `empty` and `default` funcs (lines 376-403)
**Rationale**: Adds reflection complexity for little value. Keep only `dict` and `list`.

**Changes**:
- Remove `defaultFunc` and `emptyFunc` functions
- Remove "default" and "empty" from `baseFuncs()` map
- Update/remove related tests

**Files**: `view/view.go`, `view/view_test.go`

### 5. Remove `eq` and `ne` funcs (lines 354-355)
**Rationale**: Uses `reflect.DeepEqual` which surprises with numeric types. Users can use Go's built-in comparisons or provide their own via `Config.Funcs`.

**Changes**:
- Remove "eq" and "ne" from `baseFuncs()` map
- Update/remove related tests

**Files**: `view/view.go`, `view/view_test.go`

### 6. Cache parsed templates instead of raw strings (lines 77-194)
**Rationale**: Current implementation caches content strings but parses on every render. This is surprising: "why did Load not speed up Render?"

**Changes**:
- Change cache from `map[string]string` to `map[string]*template.Template`
- In prod: cache parsed `*template.Template` for pages and layouts
- In dev: parse every time (no caching)
- Update `Load()` to parse and cache templates
- Update `Render()` to use cached templates when available

**Files**: `view/view.go`

## Files to Modify

1. **view/view.go** - Main implementation changes
2. **view/view_test.go** - Update tests for removed functions
3. **view/testdata/views/layouts/default.html** - Remove `.Page.Title` reference

## Implementation Order

1. Remove `Status` option
2. Remove `Title` and `CSRF` fields + update templates
3. Remove `safeHTML` from base funcs
4. Remove `empty` and `default` funcs + tests
5. Remove `eq` and `ne` funcs + tests
6. Implement parsed template caching
7. Run tests to verify all changes

## Breaking Changes

- `Status()` option removed (use handler-level status control)
- `.Page.Title` no longer available in templates
- `.CSRF` no longer available in templates
- `safeHTML` no longer available by default (opt-in via `Config.Funcs`)
- `default`, `empty`, `eq`, `ne` no longer available by default

## Minimal v1 API Surface (Post-Cleanup)

**Types**:
- `type Engine struct`
- `type Config struct`
- `type Data = map[string]any`
- `var ErrNotFound error`
- `type Error struct { Kind, Name string; Err error }`

**Functions/Methods**:
- `func New(Config) *Engine`
- `func (e *Engine) Load() error`
- `func (e *Engine) Clear()`
- `func (e *Engine) Render(w io.Writer, page string, data any, opts ...option) error`
- `func (e *Engine) Middleware() mizu.Middleware`
- `func From(c *mizu.Ctx) *Engine`
- `func Render(c *mizu.Ctx, page string, data any, opts ...option) error`

**Options**:
- `func Layout(name string) option`
- `func NoLayout() option`

**Template Functions (baseFuncs)**:
- `dict` - Create map from key/value pairs
- `list` - Create slice from items
- String helpers: `upper`, `lower`, `trim`, `contains`, `replace`, `split`, `join`, `hasPrefix`, `hasSuffix`

**Template Contract**:
- Page receives: `.Page{Name, Layout}`, `.Data`, `.Content` (empty for pages)
- Layout receives: `.Page{Name, Layout}`, `.Data`, `.Content` (rendered page HTML)

## Implementation Summary

All changes have been implemented and verified:

1. **view/view.go**:
   - Removed `Status` option and `status` field from `renderCfg`
   - Removed `Title` from `pageMeta` and `CSRF` from `pageData`
   - Removed `safeHTML`, `empty`, `default`, `eq`, `ne` from `baseFuncs()`
   - Removed `emptyFunc` and `defaultFunc` helper functions
   - Removed unused `reflect` import
   - Changed cache type from `map[string]string` to `map[string]*template.Template`
   - Added new `template()` method for cached template retrieval
   - Updated `loadDir()` to cache parsed templates
   - Updated `Render()` to use cached templates via `template()` method

2. **view/view_test.go**:
   - Removed `TestRender_Status` test
   - Removed `TestDefaultFunc` test
   - Removed `TestEmptyFunc` test
   - Removed `TestComparisonFuncs` test

3. **view/testdata/views/layouts/default.html**:
   - Simplified title to static "Default Title" (removed `.Page.Title` reference)

All tests pass successfully.
