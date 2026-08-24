// Milestonebot keeps the repository's labels, milestone tracking issues, and
// pull requests in agreement with each other.
//
// The problem it solves is small and constant. A roadmap lives in a tracking
// issue as a checklist, work lands as pull requests, and the two drift apart
// within a week because ticking a box by hand is the first thing anybody
// forgets. So a pull request names the checklist item it finishes, and when it
// merges this tool ticks the box and says so in the thread.
//
// Four commands, each doing one thing:
//
//	milestonebot sync-labels    reconcile the repository labels with .github/labels.yml
//	milestonebot label-pr       put type, area, size, and milestone labels on a pull request
//	milestonebot on-merge       tick the checklist item a merged pull request finished
//	milestonebot status         print how far along every tracking issue is
//
// Every command takes -repo in owner/name form, defaulting to GITHUB_REPOSITORY
// so it needs no flags inside a workflow, and every command takes -n to print
// what it would do without doing it. Authentication is GITHUB_TOKEN, which is
// the token GitHub Actions provides.
//
// # Three repositories, one tool
//
// This lives in go-mizu/mizu and also runs in go-mizu/docs and go-mizu/shizuku,
// whose workflows check it out rather than keeping a copy. Two things vary by
// repository and nothing else does: each has its own .github/labels.yml, and
// the area labels come from a rule set chosen by repository name, because a
// path like content/ means nothing in the toolkit and everything in the site.
//
// The site's checklist is the one place work and tracking come apart. Its items
// are on a tracking issue in go-mizu/mizu and its pull requests merge in the
// other two, so on-merge takes -tracking-repo and reads the tracking repository
// with TRACKING_TOKEN when that is set. Two tokens rather than one, because a
// workflow token only ever reaches its own repository.
//
// # How a pull request names its checklist item
//
// Each line of a tracking issue's checklist carries a stable identifier:
//
//   - [ ] `M0-03` Config: TOML loading, layering, env expansion, strict unknown-key errors
//
// A pull request that finishes that item says so in its description:
//
//	Checklist: M0-03
//
// The identifier is stable because it is written once when the tracking issue
// is opened and never renumbered. Reordering the list, rewording an item, or
// splitting one into two leaves every other identifier alone, so a pull
// request opened three months ago still points at the right box.
//
// A pull request that finishes more than one item lists them separated by
// commas. A pull request that finishes none says nothing, and on-merge exits
// quietly, because most pull requests are not milestone items.
package main
