Below is a “remove as much as possible” review of this `view` package, with a concrete reduction target: make it teachable as “Go templates with layouts”, not as a new rendering framework.

## The largest sources of complexity

### 1) Slots, stacks, children

This subsystem is the biggest conceptual tax and also currently inconsistent:

* `push` in templates is a stub and never records anything (`func(string) string { return "" }`).
* `renderCtx.push` exists but is unreachable from templates.
* `children` exists but is never set (`setChildren` is unused).
* Slot extraction executes every defined template to capture output, which is surprising and expensive.

Recommendation: delete all of it in v1.

Remove:

* `Config.StrictMode`, `Config.DedupeStacks`
* `renderCtx` type and all methods
* `parseWithCtx`
* baseFuncs placeholders for slot/stack/push/children
* the loop over `tmpl.Templates()` that extracts “slot definitions”
* the special `children()` function

Net effect: learners only need to know standard `{{define}}` and `{{template}}`.

### 2) “Layout chosen by template” magic

Executing a `{{define "layout"}}default{{end}}` template to select layout is clever but hidden behavior.

Recommendation: remove layout auto-detection and keep one rule:

* layout comes from `Layout()` option or `DefaultLayout`.

Remove:

* `tmpl.Lookup("layout")` extraction logic
* `"layout"` as a special template name

### 3) Separate parsing per file instead of a single template set

Today each page/layout/component is parsed in isolation, and “components” are rendered by parsing their own file at call time. This prevents standard Go template composition across files and forces you to invent `component()` and `partial()` functions.

Recommendation: parse a single `*template.Template` set that includes all files, then render by executing named templates.

This lets you delete:

* `Engine.Component`, `Engine.Partial`, and the whole `component/partial` call stack
* special directories as a concept (they can still exist on disk, but the engine does not need separate behaviors per kind)

## Minimal API and behavior that is easiest to learn

### Keep only these exported concepts

* `type Engine`
* `type Config` (trimmed)
* `func New(Config) *Engine`
* `func (e *Engine) Load() error` (parse all templates into one set)
* `func (e *Engine) Render(w io.Writer, page string, data any, opts ...Option) error`
* `func (e *Engine) Middleware() mizu.Middleware` (or keep `Handler`, but one name)
* `func Render(c *mizu.Ctx, page string, data any, opts ...Option) error` (optional convenience)

Remove:

* package-level `Component`
* `Engine.Component`, `Engine.Partial`
* `Data` alias (optional; keep only if you see it used everywhere)

### Standardize template names

A simple, unsurprising naming scheme:

* Layout templates are named: `layout/<name>`
* Page templates are named: `page/<name>`
* Components are named: `component/<name>` (optional)

Render rule:

* Execute `layout/<layout>` which calls `{{template "page/<page>" .}}` or `{{template "content" .}}` if you choose that pattern.

This removes the need for “content” special casing in Go code.

## Config: remove fields that teach extra rules

Your current `Config` teaches too many levers:

* `Development` can stay.
* `FS`, `Dir`, `Extension`, `Delims`, `Funcs` can stay.
* `DefaultLayout` can stay.

Remove from v1:

* `StrictMode` (only matters for slots/components you are removing)
* `DedupeStacks` (ditto)

So v1 config becomes:

```go
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

## Base template functions: cut aggressively

The current `baseFuncs()` is a mini standard library. It slows learning, increases surface area, and encourages logic in templates.

Minimal useful set for v1:

Keep:

* `dict`, `list` (optional)
* `default` (optional)
* common string helpers: `upper`, `lower`, `trim`, `contains`, `replace`, `split`, `join`, `hasPrefix`, `hasSuffix`

Remove:

* `title` (deprecated; also culturally problematic)
* `safeCSS`, `safeJS`, `safeURL` (footguns). If you keep anything, keep only `safeHTML` and document it.
* all math and numeric comparisons (`lt/le/gt/ge/add/sub/mul/div/mod`)
* `ternary`, `coalesce` (nice but nonessential)
* `empty` (optional; if you keep `default`, you can keep `empty` internally)

Also remove `toFloat64/toInt64` conversions. Silent coercion to 0 is a classic surprise source.

## Errors: simplify and align with what you actually use

* `Error.Kind` currently includes "component" and "partial". If you remove those features, remove those kinds too.
* `Line` is never set anywhere. Remove it unless you implement line extraction.
* `ErrNotFound` can stay, but avoid `Error.Is` special casing by setting `Err: ErrNotFound` and using `errors.Is`.

For example, missing template becomes:

```go
return &Error{Kind: "page", Name: name, Err: ErrNotFound}
```

Then `errors.Is(err, ErrNotFound)` just works.

## Mizu integration: reduce to one clear hook

You currently mutate the request pointer:

```go
*c.Request() = *c.Request().WithContext(ctx)
```

That is a bit unusual and makes readers wonder if it is safe.

If `mizu.Ctx` has a context setter or a standard pattern, prefer that. If not, keep it, but simplify naming:

* `Handler()` -> `Middleware()` (consistent with return type)
* Keep `From(c)` and package-level `Render(c, ...)` if you want the ergonomic helper.

Remove `Status()` option if you want to keep view purely “render to writer”. Status codes are a transport concern. If you keep it for DX, keep it, but note it is not part of template engine.

## Concrete “deletion list” (what you can remove with minimal redesign)

If you want the smallest diff that still reduces learning load:

1. Delete `renderCtx` and everything that references it.
2. Delete `parseWithCtx`, and use `parse` everywhere.
3. Delete layout extraction via `tmpl.Lookup("layout")`.
4. Delete slot extraction loop over `tmpl.Templates()`.
5. Delete `Engine.Component`, `Engine.Partial`, and the `component/partial` funcs injected into templates.
6. Reduce `Config` fields to remove StrictMode and DedupeStacks.
7. Reduce `baseFuncs()` to only a small set of helpers.

That turns this into “read file, parse, execute”, which is the exact model learners expect.

## One caution: your current system does not compose templates across files

Once you delete component/partial special functions, you should provide a composition mechanism. The simplest is: parse all templates once in `Load()` into a single template set and then `ExecuteTemplate` the page inside the layout. That is the standard Go solution and it is the easiest to teach.

If you want, I can propose a concrete minimal rewrite of the engine structure (still in one file) showing how to:

* parse all `.html` files under `views/`
* name templates by path
* execute `layout/default` which includes `page/home`
  with no slots, no stacks, no custom component rendering, and a very small FuncMap.
