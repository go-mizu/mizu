# QA Blueprint

A full-featured Q&A system inspired by Stack Overflow. This blueprint covers questions, answers, comments, tags, voting, reputation, badges, moderation, and a production-grade UI.

## Features

### Core
- Questions with tags, views, favorites, and accepted answers
- Answers with scoring and acceptance
- Comments on questions and answers
- Voting and score calculation

### Discovery
- Tag browsing and tag wiki
- Sorting: newest, active, score, unanswered
- Search across questions, tags, and users

### Reputation & Badges
- Reputation events for votes and accepts
- Bronze/Silver/Gold badges
- User profiles with stats

### Moderation
- Close/reopen, lock, delete
- Flags and review queues (simplified)

## Architecture

```
qa/
|-- cmd/qa/               # CLI entry point
|-- cli/                  # serve/init/seed commands
|-- app/web/              # HTTP server and handlers
|-- feature/              # Domain features
|-- store/duckdb/         # DuckDB-backed stores
|-- assets/               # Embedded static assets + templates
`-- pkg/                  # Shared utilities
```

## Status

Early development. UI is complete and matches Stack Overflow layout. Backend services and store are scaffolded to support core flows.
