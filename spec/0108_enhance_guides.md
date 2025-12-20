# Spec 0108: Enhance Guides and Core Documentation

## Overview

Reorganize the Guides and Core tabs in the documentation to provide a smoother first-user experience and better developer experience (DX).

## Current Issues

### 1. Duplication and Confusion
- **Guides/Welcome** and **Core/Overview/Intro** both introduce Mizu
- **Guides/What is Mizu** and **Core/Overview/Why** both explain design philosophy
- **Guides/Installation** is in Guides but **Quick Start** is in Core - confusing flow

### 2. Unclear Entry Point
- Users land on Guides but need to jump to Core for quick-start
- Two "Overview" sections exist in different tabs

### 3. Building Blocks Not Actionable
- Lists Core, Middlewares, Contract, View, Frontend, Mobile, CLI
- These just link to other tabs - redundant navigation

### 4. Missing Icons
- Only a few pages in Core (intro, why, features, use-cases) have icons
- All Guides pages and most Core pages are missing icons

### 5. Scattered Learning Path
- Welcome → What is Mizu → Installation → (jump to Core) Quick Start
- Not smooth for first-time users

## Proposed Structure

### Guides Tab (Primary Learning Path)

The Guides tab becomes the main entry point and learning path for users.

```
Guides
├── Getting Started
│   ├── welcome (icon: house) - Simplified intro
│   ├── installation (icon: download) - Setup instructions
│   └── quick-start (icon: rocket) - MOVED from Core
│
├── Core Concepts (MOVED from Core tab)
│   ├── overview (icon: book-open) - Concepts intro
│   ├── app (icon: server) - App lifecycle
│   ├── routing (icon: route) - URL patterns
│   ├── handler (icon: code) - Request handlers
│   ├── context (icon: box) - Ctx wrapper
│   ├── request (icon: arrow-down-to-line) - Reading requests
│   ├── response (icon: arrow-up-from-line) - Writing responses
│   ├── static (icon: folder-open) - Static files
│   ├── logging (icon: scroll) - Structured logging
│   ├── error (icon: bug) - Error handling
│   └── middleware (icon: layer-group) - Middleware patterns
│
├── Building Your App
│   ├── first-api (icon: code) - Build REST API tutorial
│   ├── first-website (icon: globe) - Build website tutorial
│   ├── first-fullstack (icon: layer-group) - Full-stack tutorial
│   ├── project-structure (icon: sitemap) - NEW: Recommended structure
│   ├── testing (icon: vial) - NEW: Testing patterns
│   └── deployment (icon: cloud) - MOVED from Core
│
└── Learn More
    ├── features (icon: sparkles) - Feature summary
    ├── use-cases (icon: lightbulb) - When to use Mizu
    ├── architecture (icon: diagram-project) - Design overview
    ├── roadmap (icon: road) - Future plans
    ├── faq (icon: circle-question) - Common questions
    └── community (icon: users) - Connect and contribute
```

### Overview Tab → Removed

The Overview tab is removed entirely. Its content is merged into the Guides tab under "Learn More" for a single, unified learning path:

```
Guides
└── Learn More
    ├── features (icon: sparkles) - Feature summary
    ├── use-cases (icon: lightbulb) - When to use Mizu
    ├── architecture (icon: diagram-project) - Design overview
    ├── roadmap (icon: road) - Future plans
    ├── faq (icon: circle-question) - Common questions
    └── community (icon: users) - Connect and contribute
```

## Changes Summary

### Removed
- `guides/what-is-mizu` - Merged into welcome
- `guides/core`, `guides/middlewares`, `guides/contract`, `guides/view`, `guides/frontend`, `guides/mobile`, `guides/cli` - Building Blocks section (just links to other tabs)

### Moved
- `get-started/quick-start` → `guides/quick-start`
- `get-started/deployment` → `guides/deployment`
- `concepts/*` → `guides/*` (all concepts pages)

### Added
- `guides/project-structure` - Best practices for project organization
- `guides/testing` - Testing patterns for Mizu apps

### Removed Tab
- "Core"/"Overview" tab → Merged into Guides "Learn More" section

## Icon Assignments

All pages now have icons from FontAwesome library:

| Page | Icon |
|------|------|
| welcome | house |
| installation | download |
| quick-start | rocket |
| overview (concepts) | book-open |
| app | server |
| routing | route |
| handler | code |
| context | box |
| request | arrow-down-to-line |
| response | arrow-up-from-line |
| static | folder-open |
| logging | scroll |
| error | bug |
| middleware | layer-group |
| first-api | code |
| first-website | globe |
| first-fullstack | layer-group |
| project-structure | sitemap |
| testing | vial |
| deployment | cloud |
| architecture | diagram-project |
| faq | circle-question |
| community | users |
| intro | book-open |
| why | heart |
| features | sparkles |
| use-cases | lightbulb |
| roadmap | road |

## User Journey

### Before
1. User lands on Welcome
2. Reads "What is Mizu" (overlaps with Welcome)
3. Goes to Installation
4. Has to find Quick Start in Core tab
5. Confusion about where to learn concepts

### After
1. User lands on Welcome (clear, concise intro)
2. Goes to Installation
3. Follows Quick Start (same tab!)
4. Explores Core Concepts in order
5. Builds with Tutorials
6. Deploys with Deployment guide

## Implementation

1. Update `docs/docs.json` with new structure
2. Add icons to all page frontmatter
3. Create new pages: `project-structure.mdx`, `testing.mdx`
4. Simplify `welcome.mdx` to be more concise
5. Move/rename pages as needed
6. Remove redundant Building Blocks pages
