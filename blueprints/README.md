# Blueprints

Real-world backend systems in Go, built one blueprint at a time.

This repository is a growing collection of backend system blueprints written in plain Go. Each project represents a familiar product class and is organized to be easy to read, extend, and learn from.

The aim is to evolve these blueprints toward full, end-to-end feature sets over time. Many projects are still early, and completeness varies by folder.

## What this repository contains

This repository is structured as a library of projects, optimized for breadth and clarity.

It includes over 100 blueprints covering social platforms, realtime systems, storage engines, developer tooling, marketplaces, automation platforms, and more. All projects follow a consistent folder layout so you can move between systems quickly, with code that emphasizes boundaries, data flow, and realistic evolution paths.

These projects are designed both as teaching material and as reference implementations that can be adapted to real systems.

## Philosophy

The goal is not to provide isolated sample code. The goal is to demonstrate how backend systems are shaped over time.

Each blueprint focuses on how services are decomposed, how responsibilities are separated, where state lives, what must be synchronous versus asynchronous, and how a system evolves from a single binary into multiple cooperating services.

The emphasis is on building systems that can become complete. In early stages, structure and direction matter more than perfect polish.

## Important disclaimer

This repository is built to move fast.

Some structure is created quickly and refined later. In early development, many decisions are pragmatic rather than ideal. AI coding agents are used heavily to accelerate scaffolding, repetitive wiring, and baseline endpoints.

All changes are reviewed with the same goals in mind: clear code, readable layout, and steady progress toward complete systems.

## How to navigate

Each project lives in its own top-level folder. Every folder contains a README explaining the system, its goals, and its current state. All projects follow the same directory conventions so the mental model transfers across the entire repository.



## Start here

If you are new, the following projects provide the fastest understanding of how the repository is organized:

**microblog**
Twitter or Threads-style backend
Feeds, follows, timelines, fanout, moderation

**chat**
Discord or Slack-style realtime system
Gateways, presence, events, delivery guarantees

**storage**
S3 or Dropbox-style storage service
Metadata, blobs, consistency models, multipart uploads



## Blueprint structure

All blueprints follow the same core structure:

- `cmd` for runnable binaries
- `app` for application wiring
- `feature` for product capabilities
- `httpx` for HTTP boundary helpers
- `jobs` for background work
- `pkg` for shared primitives
- `store` for infrastructure adapters

This consistency is deliberate. It is the fastest way to make a large collection of systems navigable.



## All blueprints

### Social and Discussion

- microblog – Twitter, Threads
- forum – Reddit, Discourse
- news – Hacker News, Lobsters
- social – Facebook
- photos – Instagram
- video – Short-video platforms
- community – Community platforms
- groups – Group-based social systems
- comments – Comment systems
- qa – Stack Overflow-style Q and A

### Messaging and Realtime

- chat – Discord, Slack
- messaging – WhatsApp, Telegram
- securechat – Signal
- presence – Online status systems
- voice – Voice signaling systems
- videochat – Zoom, Meet
- broadcast – Channel and broadcast messaging
- rooms – Live audio rooms
- notifications – In-app notifications
- realtime – PubSub and gateway systems

### Content and Knowledge

- blog – Medium, Dev.to
- newsletter – Substack
- cms – WordPress, Ghost
- wiki – Wikipedia
- docs – Documentation platforms
- notes – Note systems
- knowledge – Knowledge bases
- writing – Writing platforms
- reading – Read-it-later systems
- publishing – Publishing pipelines

### Developer Platforms

- git – GitHub, GitLab
- ci – Continuous integration systems
- registry – Package registries
- backend – Backend-as-a-service
- deploy – Deployment platforms
- hosting – Hosting platforms
- monitoring – Observability systems
- logging – Log aggregation
- metrics – Metrics pipelines
- apis – API tooling

### Productivity and Workflows

- issues – Issue tracking
- tasks – Task management
- boards – Kanban boards
- projects – Project management
- calendar – Calendaring systems
- scheduler – Scheduling tools
- workflow – Workflow engines
- automation – Automation platforms
- forms – Form systems
- tables – Spreadsheet-database hybrids

### Media and Streaming

- streaming – Live streaming
- vod – Video on demand
- music – Music streaming
- audio – Audio platforms
- podcasts – Podcast systems
- gallery – Media galleries
- images – Image hosting
- media – Media servers
- transcoding – Media pipelines
- playlists – Media curation

### Marketplaces

- marketplace – General marketplaces
- commerce – Merchant platforms
- payments – Payment systems
- subscriptions – Subscription platforms
- crowdfunding – Crowdfunding systems
- auctions – Auction platforms
- rides – Ride dispatch
- delivery – Delivery platforms
- booking – Booking systems
- tickets – Ticketing systems

### Storage and Data

- storage – Object storage
- files – File sync systems
- drive – Cloud drive systems
- backup – Backup services
- sync – Data sync engines
- search – Search services
- indexing – Indexing systems
- analytics – Analytics platforms
- warehouse – Data warehouses
- pipelines – Data pipelines

### Finance and Business

- billing – Billing systems
- invoicing – Invoicing platforms
- accounting – Accounting software
- expenses – Expense tracking
- payroll – Payroll systems
- trading – Trading platforms
- crypto – Crypto exchanges
- banking – Banking integrations
- equity – Equity management
- finance – General fintech

### Learning and Lifestyle

- learning – Learning platforms
- courses – Course marketplaces
- language – Language learning
- education – Education platforms
- fitness – Fitness tracking
- health – Health systems
- mindfulness – Mindfulness apps
- books – Reading platforms
- reviews – Review systems
- housing – Real estate platforms



## Design principles

- plain Go
- explicit dependencies
- small packages with clear names
- consistent structure across projects
- readable code over clever abstractions

## Status

Early development. Many blueprints are under active construction. The direction is to evolve each folder toward a complete, full-featured backend for its product class while keeping the code easy to follow and consistent across the repository.
