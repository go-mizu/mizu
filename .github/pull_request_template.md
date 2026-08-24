<!--
Title: use the conventional-commit shape, for example

  feat(cache): add the memory driver
  fix(router): keep the trailing slash on a wildcard match
  docs: write the errs page
  refactor(errs)!: close the Kind taxonomy

Prefixes: feat, fix, docs, perf, refactor, test, build, ci, chore, sec, revert.
An exclamation mark before the colon marks a breaking change.

A bot reads the title, the paths you touched, and the size, and applies the
type, area, size, and milestone labels. You do not need to add them by hand.
-->

## What this does

<!-- One paragraph. What changes for somebody using the toolkit, not what you
edited. If nothing changes for them, say what changes for whoever maintains it. -->

## Why

<!-- Link the issue if there is one. If there is not, the reason goes here. -->

## How it was checked

<!-- Delete what does not apply, and say what you actually ran. -->

- [ ] `go test ./...` passes
- [ ] `go test -race ./...` passes
- [ ] New behaviour has a test that fails without the change
- [ ] Exported symbols are documented, and the doc comment says what it does not do
- [ ] Benchmarks run, and any performance claim in the description has a number behind it

## What it costs

<!-- Every change costs something: an allocation, an exported symbol that now
has to keep working, a dependency, a concept somebody has to learn. Name it.
"Nothing" is a valid answer when it is true. -->

Checklist:

<!--
If this finishes an item on a milestone tracking issue, put its identifier on
the line above, like this:

  Checklist: M0-03

Several items are comma separated. When this merges, the box is ticked and a
comment goes on the tracking issue. Leave the line empty or delete it if this
change is not a milestone item, which is the normal case.
-->
