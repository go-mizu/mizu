# Contributing

The short version: open a pull request with a conventional-commit title, say what it costs, and if it finishes a milestone item put `Checklist: M0-03` in the description.
Everything below is detail.

## Setup

```bash
git clone https://github.com/go-mizu/mizu
cd mizu
go build ./...
go test ./...
go test -race ./...
```

Go 1.26 or later.
There is nothing else to install.
If a change ever requires a tool that is not in the standard distribution, that is a decision to argue for in an issue before writing the code.

The repository tooling lives in its own module, so it is not in `go test ./...` at the root:

```bash
go -C tools/milestonebot test ./...
```

## How the work is organised

The roadmap is 11 milestones, each with a tracking issue carrying the full checklist, the effort estimate, the acceptance criteria, and the risks.
They are the issues labelled [`tracking`](https://github.com/go-mizu/mizu/labels/tracking), and they are the source of truth for what is planned and what is done.

Each checklist item has an identifier that is assigned once and never renumbered:

```
- [ ] `M0-03` Config: TOML loading, layering, env expansion, strict unknown-key errors
```

Reordering the list, rewording an item, or splitting one into two leaves every other identifier alone.
That is why a pull request refers to an item by identifier and not by position or by text.

### Claiming something

Say so in the tracking issue's thread before you start, and a maintainer will add `status/in-progress` and assign it.
This is the only coordination step and it exists so that two people do not spend a week on the same eager-loading strategy.

A pull request that nobody claimed is still welcome.
It is just more likely to collide.

## Pull requests

**The title is a conventional commit.**

```
feat(cache): add the memory driver
fix(router): keep the trailing slash on a wildcard match
docs: write the errs page
refactor(errs)!: close the Kind taxonomy
```

The prefixes are `feat`, `fix`, `docs`, `perf`, `refactor`, `test`, `build`, `ci`, `chore`, `sec`, and `revert`.
An exclamation mark before the colon marks a breaking change.
CI fails on a title it cannot read, and the message tells you what to write instead.

**The labels are applied for you.**
A bot reads the title, the paths you touched, and the size, and applies the type, area, size, and milestone labels.
Do not add them by hand.
If it gets one wrong, add or remove the label yourself and say so in the thread, since the tool never removes a label a person added.

**If it finishes a milestone item, name it.**

```
Checklist: M0-03
```

Several items are comma separated.
When the pull request merges, the box is ticked on the tracking issue and a comment lands in that thread with the pull request, the item, and what is left.
Naming an identifier that does not exist fails the workflow on purpose, because the alternative is a checklist that quietly stops matching reality.

Most pull requests finish no checklist item, and for those the line is left out.

**Say what it costs.**
Every change costs something: an allocation, an exported symbol that now has to keep working, a dependency, a concept somebody has to learn.
The pull request template asks for it because a change whose cost nobody named is a change whose cost somebody pays later.
"Nothing" is a valid answer when it is true.

## What the code has to look like

The bar is the Go standard library, not because that is a nice thing to say but because it is a set of specific, checkable habits.

**Simple beats clever.**
If a reviewer has to reconstruct why something works, it is too clever, whatever it costs in lines.

**Composable beats configurable.**
An option struct with 14 fields is usually four types wearing a trenchcoat.
Prefer a small interface somebody can implement over a flag that switches behaviour inside yours.

**Every package stands alone.**
No package imports the composition root, and CI asserts it.
Every package with a public constructor has a compiled standalone example under `examples/standalone/` and passes the import-closure assertion.
This is not a cleanup pass before 1.0, it is part of the milestone that introduces the package, because the property cannot be retrofitted.

**No `internal/`.**
If something is exported it is documented and supported, and if it should not be either of those then it should not be exported.
Hiding a package behind `internal/` is a way of avoiding that decision rather than making it.

**Doc comments say what it does not do.**
The sentence that saves somebody an hour is almost never the one describing the happy path.

**Errors name the thing that went wrong and what to do about it.**
An error message is documentation, reviewed as documentation, and a message somebody had to read three times is a bug in the message.

**A behaviour without a test does not exist.**
New behaviour comes with a test that fails without the change.
`go test -race ./...` passes.
A performance claim comes with a benchmark, and a benchmark comes with a budget.

**Generated code is checked in and deterministic.**
`mizu gen --check` is clean, and the same input produces byte-identical output across platforms, architectures, `GOMAXPROCS` values, and input order.

## Documentation comes first

A package is not done until its page is.

That rule takes effect at Phase 1 of the [site checklist](https://github.com/go-mizu/mizu/issues/12), and from that point a package and its page ship in the same week.
The reason is not tidiness.
A page written six months later, by somebody reconstructing what the package does from its signatures, is a page that is subtly wrong in the places that matter most.

Pages live in [go-mizu/docs](https://github.com/go-mizu/docs) and every code sample on the site is a region in compiled, tested Go under `examples/docs/` here.
No sample is written in markdown.
If a sample would not compile, the site build fails, which is the entire point.

The site checklist stays in this repository even though the work merges over there, because the roadmap is one list and splitting it across three repositories would mean nobody could see the whole thing.
So a pull request in go-mizu/docs or go-mizu/shizuku writes `Checklist: site-04` exactly as one here would, and the same tool ticks the box on [issue #12](https://github.com/go-mizu/mizu/issues/12).
Those repositories check `tools/milestonebot` out of this one rather than keeping a copy, so a fix to the labelling rules lands in all three at once.

## Reporting a security problem

Open a [private advisory](https://github.com/go-mizu/mizu/security/advisories/new) rather than an issue.
You will get an answer within three working days.

## What gets declined

Being clear about this up front is fairer than being vague and then saying no.

**A feature with no cost stated.**
Not because the answer is no, but because the conversation cannot start.

**A package that needs the composition root.**
If it cannot stand alone, either the design is wrong or it belongs somewhere else, and both are worth finding out before the code is written.

**A new dependency, usually.**
The core's transitive graph is the standard library and `golang.org/x/*`, and a test asserts it.
A driver in its own module is where a third-party dependency belongs.

**Reflection on a hot path.**
The toolkit generates code instead, so that the compiler and the reader both know what is happening.
A profile-based test fails if `reflect.Value` shows up on the ORM read path.

**An exported symbol added quietly.**
Every exported name is a promise to keep it working.
Adding one is a proposal, and `mizu apidiff` will find it in CI anyway.

**A performance claim with no measurement.**
The `needs-benchmark` label exists for this and it is applied often.
