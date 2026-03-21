# Storage Pricing

> Simple, predictable pricing. No egress fees. No bandwidth metering.

Start free, scale when you're ready.

## Plans

| Plan | Price | For |
|------|-------|-----|
| Free | $0 | Personal projects, prototypes, getting started |
| Pro | $20/mo per account | Production apps, multi-agent workflows |
| Max | $100/mo per account | Teams, fine-grained sharing, priority support |

## Free

- Storage for personal projects
- Full REST API access
- Web file browser
- A few actors (human + agents)
- Ed25519 & magic link auth
- Standard rate limits

## Pro — $20/mo

Everything in Free, plus:

- More storage
- Larger file uploads
- Presigned direct uploads (bypass worker)
- More actors for your team
- Higher rate limits
- Email support

## Max — $100/mo

Everything in Pro, plus:

- Even more storage
- Large file uploads (multipart)
- Team sharing and permissions
- More actors
- Higher rate limits
- Priority support & usage analytics

## Every Plan Includes

- **No egress fees** — Read and download without bandwidth charges. Built into the infrastructure.
- **Global edge network** — Files served from 300+ locations. Sub-50ms metadata lookups.
- **Passwordless auth** — Ed25519 challenge-response for agents. Magic links for humans.
- **Plain REST API** — Works with curl, fetch, any HTTP client. No SDK required.
- **Humans & agents** — Same API for both. Share files between people and AI agents.
- **Web file browser** — Upload, browse, preview, and manage files from any browser.

## FAQ

**Why are there no egress fees?**
Our storage backend doesn't charge for egress. We pass that through. This is structural, not promotional.

**What counts as an "actor"?**
Each human user or AI agent identity on your account. Higher plans support more actors.

**What happens when I hit a limit?**
Clear error with the specific limit. Existing files stay accessible. Upgrade or free up space.

**Can AI agents use the free plan?**
Yes. Register with an Ed25519 key and start storing files immediately.

**Can I switch plans anytime?**
Yes. Upgrade or downgrade at any time. No long-term contracts.

**What if I need more than what Max offers?**
Contact us for custom arrangements with larger capacity, SLAs, and dedicated support.

## Links

- [Developer Guide](https://storage.liteio.dev/developers) — API docs, code examples
- [API Reference](https://storage.liteio.dev/api) — Full endpoint documentation
- [CLI](https://storage.liteio.dev/cli) — Terminal interface
