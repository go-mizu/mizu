# Notes for coding agents

This file is for AI coding agents working in this repository.
Humans should read [CONTRIBUTING.md](CONTRIBUTING.md) instead, which covers the same ground in the order a person needs it.

Everything here is true of this repository today.
If something below contradicts what you find in the code, the code wins and the contradiction is worth reporting as a documentation issue.

## What this repository is

A Go toolkit, module path `github.com/go-mizu/mizu`, currently at `v0.1.0`, which is a thin layer over `net/http`.
The design it is being built towards is much larger and is written out in full in the issues labelled `tracking`.

`go-mizu.dev` is the website domain.
It never appears in an import path.

## Commands

```bash
go build ./...                      # the toolkit
go test ./...                       # tests
go test -race ./...                 # what CI runs
go test -bench=. -run=XXX ./...     # benchmarks, no tests
go run ./cmd/mizu version           # the CLI
go -C tools/milestonebot test ./... # the repository tooling, a separate module
gofmt -l .                          # must print nothing
go vet ./...
scripts/tag-release.sh v0.7.0 --dry-run  # what a release would tag
```

There is no Makefile and no task runner.
If a command is not in the list above, it does not exist yet.

## Rules that are enforced rather than suggested

These are checked by CI or by a test, so breaking one produces a red build rather than a review comment.

- **No `internal/`.** Anywhere. If it should not be exported, do not write it.
- **No package imports the composition root.** Every package works standalone. This is the rule the whole toolkit claim rests on.
- **The core depends on the standard library and `golang.org/x`, and nothing else.** `deps_test.go` asserts the import graph and the `require` list, so a test-only dependency fails too. The list is `golang.org/x/crypto` and the `golang.org/x/sys` it brings with it. Adding another `golang.org/x` repository means an entry in `allowedModules` with the reason next to it. Anything outside `golang.org/x` goes in its own module, the way `tools/milestonebot` does, and needs an entry in the decision register.
- **Generated code is checked in and byte-identical across platforms, architectures, `GOMAXPROCS` values, and input order.**
- **`gofmt` is clean and `go vet` passes.**
- **New behaviour has a test that fails without the change.**

## Conventions that are not enforced but are still expected

- Doc comments on exported symbols, and the comment says what the thing does not do as well as what it does.
- Error messages name what went wrong and what to do about it. They are documentation and are reviewed as documentation.
- Prefer a small interface somebody can implement over a boolean that switches behaviour inside yours.
- A performance claim comes with a benchmark. Without one it is an opinion.

## Prose style, including in comments and commit messages

This applies to everything written in this repository and in [go-mizu/docs](https://github.com/go-mizu/docs).

- Plain English. Short sentences. Write the way a developer explains something to a colleague.
- No em dashes.
- No horizontal rules.
- One sentence per line in markdown source, so that diffs are readable.
- Do not use: simply, just, easy, obviously, trivially, of course, magic, blazingly fast, powerful, robust, seamless, elegant, delightful, leverage, utilize. The docs repository fails its build on these, and the same standard applies here.
- Do not describe something as done when it is planned. The roadmap is public, so say "planned" and link the milestone.

## How work is tracked

Eleven tracking issues, one per milestone, each holding a checklist.
Every item has an identifier that is assigned once and never renumbered, like `M0-03`.

A pull request that finishes an item says so in its description:

```
Checklist: M0-03
```

On merge, `tools/milestonebot` ticks the box and comments on the tracking issue with what is left.
Naming an identifier that does not exist fails the workflow, on purpose.

Pull request titles are conventional commits, and a bot derives the type, area, size, and milestone labels from the title, the paths, and the description.
Do not add those labels by hand.

```
feat(cache): add the memory driver
fix(router): keep the trailing slash on a wildcard match
refactor(errs)!: close the Kind taxonomy
```

The same tool runs in go-mizu/docs and go-mizu/shizuku, which check it out of here rather than keeping a copy.
Each of the three has its own `.github/labels.yml`, and the area rules are chosen by repository name in `classify.go`, because `content/` means nothing here and everything on the site.
The site checklist stays on issue #12 in this repository even though that work merges in the other two.

To see where things stand without reading eleven issues:

```bash
GITHUB_TOKEN=$(gh auth token) go -C tools/milestonebot run . status -repo go-mizu/mizu
```

## Things that will waste your time if you do not know them

**`v0.1.0` is not the design.**
The published API predates the milestones, so a pattern you find in `app.go` is not evidence of how the toolkit is meant to be shaped.
Read the tracking issue for the area you are working in first.

**Documentation is written before the code, not after.**
If you are adding a package, the page describing it is part of the work.
Every code sample on the site is compiled and tested Go under `examples/docs/`, never a snippet typed into markdown.

**Say what a change costs.**
The pull request template asks for it and a review will ask again.
An allocation, an exported symbol that now has to keep working, a dependency, a concept to learn. Name it.

**Do not invent numbers.**
If you have not measured it, do not write a figure. The `needs-benchmark` label exists for exactly this and it gets used.

## Repositories

| Repository | What is in it |
|---|---|
| [go-mizu/mizu](https://github.com/go-mizu/mizu) | The toolkit, the CLI, and the roadmap |
| [go-mizu/docs](https://github.com/go-mizu/docs) | go-mizu.dev, an Astro site built from published artefacts |
| [go-mizu/shizuku](https://github.com/go-mizu/shizuku) | The design system the site is built from, plain CSS with tokens |
