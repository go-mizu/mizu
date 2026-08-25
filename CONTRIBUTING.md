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

Go 1.27 or later.
There is nothing else to install.
If a change ever requires a tool that is not in the standard distribution, that is a decision to argue for in an issue before writing the code.

The repository's own CLI is the shorter way to run the same things:

```bash
go run ./cmd/mizu check    # type check and vet, in about a second
go run ./cmd/mizu verify   # gen, fmt, vet, build, test, doctor, in that order
```

`check` is what you run while you are working and `verify` is what you run before you push.
`verify` takes about a minute on this repository, most of it the test suite, and it fails on the first stage that fails with the diagnostics from that stage rather than a wall of output.
On a project `mizu new` wrote this morning it takes under a second, and a test holds it to ten.

CI runs `go run ./cmd/mizu verify` in a job of its own, so the answer you get before you push is the answer the merge waits for, and a stage added to `verify` is a stage CI runs without anybody editing a workflow.
The other jobs are the part `verify` does not do: four platforms, the race detector, coverage, the nested modules, and the cross compiles.

The repository tooling and the benchmarks live in modules of their own, so neither is in `go test ./...` at the root:

```bash
go -C tools/milestonebot test ./...
go -C bench test ./...
```

The benchmarks are in `bench/`, along with the budget that says what each operation is meant to cost:

```bash
go -C bench test -run='^$' -bench=. ./micro/  # run them
go -C bench run ./cmd/benchrun check          # every budgeted operation has a benchmark
go -C bench run ./cmd/benchrun lint           # the rules that make two runs comparable
```

`bench/doc.go` explains what the numbers are for and what the rules are.
Most of the budget describes packages that do not exist yet, and those rows say which milestone brings them.

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

**The standard library and `golang.org/x`, and nothing else.**
`go get github.com/go-mizu/mizu` pulls in a graph you can read in one screen.
No third-party upgrade to schedule, no advisory to read that is not ours or the Go team's, no diamond conflict with whatever else you depend on.
`deps_test.go` checks both the import graph and the `require` list, so a dependency cannot arrive through a test either.

The `golang.org/x` repositories count as the standard library here.
They go through the same proposal process, they are reviewed by the same people, they are where standard library packages are incubated, and they are in the same vulnerability database.
The rule was absolute until M0-07, when it meant writing argon2id and BLAKE2b by hand, and then Blowfish after that to read a bcrypt hash out of an existing database.
A password hash written by somebody who has not written one before is code where a bug is silent and the tests pass either way, so the rule was costing security rather than buying it.
`D-075` in the decision register has the argument in full.

Adding one is still a change with a reason attached, in `allowedModules` and in the review.
Today the list is `golang.org/x/crypto`, the `golang.org/x/sys` it brings with it, `golang.org/x/text`, and `golang.org/x/term`.
`x/text` arrived with `mizu/str`, because `strings.Title` has been deprecated since Go 1.18 for capitalising the wrong letter in half the world's languages and nothing replaced it in the standard library.
The deprecation notice points at `x/text/cases`, and the alternative is copying the Unicode casing tables in by hand.
`x/term` arrived with `mizu/console`, which decides colour and prompt behaviour from whether the other end is a terminal, and it replaced a check in `mizu/log` that answered yes for `/dev/null`.

Everything else is a third-party dependency and stays out of the core.
A library that needs one goes in a module of its own, the way `tools/milestonebot` does with its YAML parser and `bench` does with `golang.org/x/tools`.
A nested module keeps its dependencies to itself, and somebody who imports the toolkit never sees them.
That is also where database drivers and cloud clients will live.

**Doc comments say what it does not do.**
The sentence that saves somebody an hour is almost never the one describing the happy path.

**Every package has a `doc.go` and an example.**
The package comment goes in `doc.go` rather than on top of whichever file the package started as, it opens with `Package <name>`, and the package has at least one `Example`.
`docs_test.go` at the root checks those three, so a package that skips one fails the build rather than getting a documentation pass six months later by somebody reconstructing what it does from its signatures.
An example that can assert carries an `// Output:` comment and is run as a test.
The ones in the fixture packages cannot, because an assertion needs a test to fail and an example has none, so they are compiled and shown rather than run, and each of those packages says so.

**Every command answers `--json`.**
The flag is global, so a new command takes it without doing anything, and a command that takes it and then prints a table for a person is worse than one that rejects it.
`cmd/mizu/json_test.go` walks the registry, runs each command with `--json`, and fails on one whose output is not a single JSON document.
Adding a command means adding the line to that table that says how to run it, which is the point at which somebody decides what document it writes.

**The project `mizu new` writes is a project.**
A scaffold that compiles is not the same as a scaffold somebody can use, and the difference shows up at the first thing anybody types at a binary they did not build.
Both presets answer `version` and `--version`, and the HTTP one answers them on the command line rather than opening a port.
`cmd/mizu/project_test.go` writes each preset into a temporary directory, builds it, runs the binary, and checks that `mizu gen --check` is clean in a fresh project and names the file and the line once it is not.

**Errors name the thing that went wrong and what to do about it.**
An error message is documentation, reviewed as documentation, and a message somebody had to read three times is a bug in the message.

**A behaviour without a test does not exist.**
New behaviour comes with a test that fails without the change.
`go test -race ./...` passes.
A performance claim comes with a benchmark, and a benchmark comes with a budget.

**A parser gets a fuzz target, and the target gets an hour.**
Anything that reads bytes somebody else wrote is fuzzed: the two key formats in `crypt`, the two password hash formats in `hash`, the TOML parser, and the pattern matcher in `str`.
On a pull request each target runs its seed corpus, which takes a second, because nobody waits for the other version.
The hour is nightly, in `.github/workflows/fuzz.yml`, one job per target so that each of them gets an hour rather than a share of one.
`fuzz_test.go` at the root reads that file back and fails on a `FuzzXxx` it does not name, and the failure prints the row to add.

**Generated code is checked in and deterministic.**
`mizu gen --check` is clean, and the same input produces byte-identical output across platforms, architectures, `GOMAXPROCS` values, and input order.
None of that is taken on trust.
`cmd/mizu/determinism_test.go` copies a fixture package into siblings, hands them to each generator in eight orders at two `GOMAXPROCS` values, and compares the bytes that land on disk, and the `Generator determinism` job runs `mizu gen --check` over this repository on Linux and macOS on both architectures.
The failure this is looking for is a map ranged over on the way to the output, which is why the same order is tried more than once.

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
