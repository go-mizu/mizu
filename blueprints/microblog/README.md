# Microblog

Microblog is a self-hosted platform for short-form posts and conversations. It focuses on timelines, interactions, moderation, and discovery, with a design that can evolve from a single service into multiple cooperating services.

## What you can build with this blueprint

A complete microblogging backend, including:
- Accounts and profiles
- Posts and threads
- Timelines and fanout strategies
- Interactions (likes, reposts, bookmarks)
- Notifications
- Moderation and safety controls
- Discovery (search and trends)

## Product references

Microblog sits in the same product family as:
- [X](https://x.com)
- [Threads](https://threads.com)
- Mastodon-style instances (federation is optional and can be added later)

## Scope

This blueprint aims to cover the core backend for a modern microblogging product:
- Short posts, threaded replies, quote posts
- Follow graph and timeline generation
- Public and private visibility controls
- Media attachments and metadata
- Notifications for social events
- Admin and user-level safety controls
- Basic discovery primitives: search and trends

## Non-goals

This blueprint does not aim to cover every production concern by default:
- Monetization and ads
- Full multi-region architecture
- Heavy ML ranking or recommendation stacks
- Full federation compatibility out of the box

## How to navigate this folder

This blueprint follows the repository conventions:
- `cmd/` runnable binaries
- `app/` application wiring
- `feature/` product capabilities
- `httpx/` HTTP boundary helpers
- `jobs/` background work
- `pkg/` shared primitives
- `store/` infrastructure adapters

Start with the README in each `feature/` folder, then trace how features are wired in `app/`.

## Core capabilities

### Identity and profiles
- Account creation and authentication primitives
- Profile data and public views
- Admin controls for verification and account actions

### Posting and conversations
- Short posts with rich text rules
- Reply chains and thread context
- Quote posts and reposts
- Editing with history (optional per implementation)

### Social graph
- Follow relationships
- Muting and blocking
- Lists (optional per implementation)

### Timelines
- Home timeline based on follow graph
- Local or instance-wide public timeline
- User profile timeline
- Hashtag or topic timelines (optional per implementation)

### Interactions
- Likes
- Reposts
- Bookmarks
- Counts and aggregation strategy defined per store adapter

### Notifications
- Event-driven notifications for follows, replies, mentions, and interactions
- Read and dismissal semantics

### Discovery
- Search across posts and accounts (implementation may vary)
- Trending tags and posts (implementation may vary)
- Suggested accounts (optional)

### Moderation and safety
- Reporting and review workflow (can start minimal)
- Rate limits and abuse controls (as the blueprint matures)
- Admin tooling for enforcement actions

## Status

This blueprint is under active development. Structure is intended to remain stable while implementation details evolve.

## Roadmap markers

- MVP: accounts, posts, follows, home timeline, basic interactions
- Growth: notifications, search, trends, media workflows
- Scale: background fanout, caches, sharding, realtime delivery
- Optional: federation support and cross-instance interoperability

## License

MIT
