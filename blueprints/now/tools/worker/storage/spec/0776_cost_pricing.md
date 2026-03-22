# 0776 — Cost & Pricing Analysis

**Date:** 2026-03-21
**Status:** Draft
**Goal:** Estimate infrastructure costs, validate tier pricing, model path to $1M ARR, identify cost overrun risks.

---

## 1. Architecture Overview (Cost-Relevant)

| Component | Service | Role | Billing Model |
|-----------|---------|------|---------------|
| Compute | Cloudflare Workers | API routing, auth, metadata ops | Per-request + CPU time |
| Object Storage | Cloudflare R2 | File blobs (content-addressed) | Per-GB stored + per-op |
| Metadata DB | Cloudflare D1 | Files, actors, sessions, rate limits, audit | Per-rows-read + rows-written |
| Email | Resend | Magic link authentication | Per-email |
| Bot Detection | Cloudflare Turnstile | CAPTCHA on registration/login | Free |
| DNS/CDN | Cloudflare | Edge delivery, custom domain | Free (included) |

### Key Cost Decisions Already Made

- **R2 zero egress** — Eliminates the #1 cost driver in traditional storage (AWS S3 egress = $0.09/GB). We can offer "no egress fees" honestly.
- **Presigned URLs** — Pro/Max uploads and all downloads bypass the Worker. No CPU time for file transfer.
- **Content-addressed blobs** — SHA-256 keying enables per-actor dedup (same file uploaded twice = one R2 object). Cross-actor dedup intentionally skipped for GDPR isolation.
- **D1-backed rate limiting** — No external rate limiting service. Probabilistic cleanup (1% per-request) keeps table small.

---

## 2. Cloudflare Pricing (as of 2026-03)

### Workers (Paid plan: $5/mo base)

| Metric | Free | Paid |
|--------|------|------|
| Requests | 100K/day | 10M/mo included, then $0.30/M |
| CPU time | 10ms/invocation | 30M ms/mo included, then $0.02/M ms |

### R2

| Metric | Rate |
|--------|------|
| Storage | $0.015/GB-month |
| Class A ops (PUT, POST, LIST) | $4.50/M |
| Class B ops (GET, HEAD) | $0.36/M |
| Egress | **$0** |
| Free tier | 10 GB storage, 1M Class A, 10M Class B per month |

### D1

| Metric | Free | Paid |
|--------|------|------|
| Rows read | 5M/day | 25B/mo included, then $0.001/M |
| Rows written | 100K/day | 50M/mo included, then $1.00/M |
| Storage | 5 GB | 5 GB included, then $0.75/GB |

### Resend (Email)

| Tier | Emails/mo | Cost |
|------|-----------|------|
| Free | 100/day | $0 |
| Pro | 50K/mo | $20/mo |
| Business | 200K/mo | $80/mo |
| Scale | Per email | ~$0.00065/email |

---

## 3. Per-User Cost Model

### Assumptions by Tier

| Metric | Free User | Pro User | Max User |
|--------|-----------|----------|----------|
| Storage used (avg) | 200 MB | 30 GB | 200 GB |
| Files stored (avg) | 50 | 2,000 | 15,000 |
| API requests/mo | 5,000 | 100,000 | 500,000 |
| R2 writes/mo | 100 | 3,000 | 20,000 |
| R2 reads/mo | 500 | 15,000 | 80,000 |
| D1 rows read/mo | 10,000 | 200,000 | 1,000,000 |
| D1 rows written/mo | 500 | 5,000 | 30,000 |
| Magic link emails/mo | 4 | 8 | 15 |
| Presigned URL ops/mo | 0 | 10,000 | 50,000 |

### Cost per User per Month

| Component | Free User | Pro User | Max User |
|-----------|-----------|----------|----------|
| **R2 storage** | $0.003 | $0.45 | $3.00 |
| **R2 Class A** | $0.0005 | $0.014 | $0.09 |
| **R2 Class B** | $0.0002 | $0.005 | $0.03 |
| **Workers CPU** | $0.001 | $0.02 | $0.10 |
| **D1 reads** | ~$0 | ~$0 | $0.001 |
| **D1 writes** | ~$0 | ~$0 | $0.03 |
| **Resend email** | $0.003 | $0.005 | $0.01 |
| **Total cost/user/mo** | **~$0.01** | **~$0.50** | **~$3.26** |
| **Revenue/user/mo** | $0 | $20 | $100 |
| **Gross margin** | n/a | **97.5%** | **96.7%** |

### Key Takeaway

Infrastructure costs per user are extremely low. Even at maximum assumed usage, a Max user at 200 GB costs ~$3.26/mo against $100/mo revenue — a 96.7% gross margin. The R2 zero-egress model is the foundation of this.

---

## 4. Tier Limits (Final)

| | Free | Pro ($20/mo) | Max ($100/mo) |
|---|---|---|---|
| Storage | 1 GB | 100 GB | 1 TB |
| Max file size | 50 MB | 500 MB | 5 GB (multipart) |
| Actors | 3 | 25 | 100 |
| API requests/day | 1,000 | 50,000 | 200,000 |
| Presigned uploads | No | Yes | Yes |
| Team sharing | No | No | Yes |
| Support | Community | Email | Priority |

### Why These Numbers

- **1 GB free** — Enough to prototype, test the API, run a couple of agents. Low enough that abuse is bounded ($0.015/mo worst case per free user).
- **100 GB pro** — Covers most production single-team use cases. R2 cost at 100% utilization: $1.50/mo.
- **1 TB max** — Serious teams with large datasets. R2 cost at 100% utilization: $15/mo. Still 85% margin.
- **Actor limits** — Prevent free-tier account farming. 3 is generous for personal use; 25 covers a real team's agent fleet.
- **Daily rate limits** — Prevent runaway scripts. Resets daily so brief spikes don't lock users out for hours.

---

## 5. Revenue Model: Path to $1M ARR

$1M ARR = $83,333/mo recurring revenue.

### Scenario A: Pro-Heavy (Realistic Early Stage)

| Tier | Users | Revenue/mo | Total/mo |
|------|-------|------------|----------|
| Free | 10,000 | $0 | $0 |
| Pro | 3,500 | $20 | $70,000 |
| Max | 135 | $100 | $13,500 |
| **Total** | **13,635** | | **$83,500/mo** |
| **ARR** | | | **$1,002,000** |

**Infrastructure cost at this scale:**

| Component | Monthly Cost |
|-----------|-------------|
| R2 storage (10K×0.2GB + 3.5K×30GB + 135×200GB) | $1,983 |
| R2 operations | $220 |
| Workers (base + overages) | $45 |
| D1 | $10 |
| Resend (Pro plan) | $20 |
| Workers Paid plan | $5 |
| **Total infra** | **~$2,283/mo** |
| **Gross margin** | **97.3%** |

### Scenario B: Max-Heavy (Enterprise Traction)

| Tier | Users | Revenue/mo | Total/mo |
|------|-------|------------|----------|
| Free | 5,000 | $0 | $0 |
| Pro | 1,500 | $20 | $30,000 |
| Max | 535 | $100 | $53,500 |
| **Total** | **7,035** | | **$83,500/mo** |

**Infra cost:** ~$17,800/mo (driven by Max users averaging 200 GB storage each = 107 TB on R2 = $1,605/mo R2 storage alone, plus operations). Gross margin: **78.7%**.

### Scenario C: Blended Growth Path

| Month | Free | Pro | Max | MRR | Infra Cost | Margin |
|-------|------|-----|-----|-----|------------|--------|
| 1 | 100 | 5 | 0 | $100 | $8 | 92% |
| 3 | 500 | 30 | 2 | $800 | $25 | 97% |
| 6 | 2,000 | 150 | 10 | $4,000 | $100 | 97% |
| 12 | 5,000 | 600 | 40 | $16,000 | $500 | 97% |
| 18 | 8,000 | 1,500 | 80 | $38,000 | $1,200 | 97% |
| 24 | 12,000 | 3,000 | 150 | $75,000 | $2,400 | 97% |
| 27 | 14,000 | 3,500 | 200 | $90,000 | $2,900 | 97% |
| **30** | **16,000** | **3,800** | **230** | **$99,000** | **$3,300** | **96.7%** |

**$1M ARR reached in ~month 28-30** (2.5 years) assuming:
- 5% free-to-pro conversion rate
- 3% pro-to-max upgrade rate
- 2% monthly churn on paid plans
- Organic + developer community growth (no paid ads modeled)

---

## 6. Cost Risks & Mitigations

### Risk 1: Free Tier Abuse (Storage Stuffing)

**Threat:** Bots or scripts register thousands of free accounts, each using 1 GB. 10K abusive accounts = 10 TB = $150/mo R2 storage with zero revenue.

**Mitigations already in place:**
- Turnstile CAPTCHA on registration
- Bot guard scoring (datacenter ASN, UA patterns, CF bot score)
- Rate limiting: 5 registrations/hour per IP
- Magic link requires valid email (cost gate: each registration = one email sent)

**Additional mitigations to consider:**
- Email domain allowlist/blocklist (block disposable email providers)
- Require email verification before first upload
- Auto-suspend accounts inactive >90 days (delete after 180 days notice)

**Worst case bounded:** Even 50K abusive free accounts at full 1 GB each = 50 TB = $750/mo. Painful but survivable.

### Risk 2: R2 Storage Growth Outpaces Revenue

**Threat:** Max users store 1 TB each (their full allocation). 500 Max users × 1 TB = 500 TB = $7,500/mo R2 cost against $50,000/mo revenue.

**Assessment:** 85% margin even in this worst case. R2 storage costs are linear and predictable. This is not a real risk — it's the expected cost curve.

**If it becomes a concern:**
- Add overage pricing beyond tier limits (e.g., $0.05/GB/mo beyond 1 TB)
- Introduce annual billing at 15-20% discount (improves cash flow, reduces churn)

### Risk 3: Resend Email Cost Spike

**Threat:** Magic link re-authentication generates high email volume. If every Pro user authenticates 3x/day = 315K emails/mo.

**Assessment:** Resend Business plan at $80/mo covers 200K emails. At 315K, cost is ~$130/mo. Manageable.

**Mitigation:** Session TTL is already 2 hours (human) / 30 days (agent). Most agent traffic is Ed25519 (no email). Email volume is naturally low.

### Risk 4: D1 Row Limits at Scale

**Threat:** D1's free tier is generous (5M reads/day, 100K writes/day) but rate limit table queries run on every request. At 100M API requests/day, D1 reads could hit 500M+ rows/day.

**Assessment:** We won't hit 100M requests/day before $1M ARR. At Scenario A scale (~50M requests/mo), D1 is well within paid limits (25B reads/mo included).

**Mitigation if needed:**
- Switch to Cloudflare KV for rate limiting (simpler key-value, $0.50/M reads)
- Batch rate limit checks (read once per session, not per request)
- Move to Durable Objects for per-actor sharding (already implemented as opt-in)

### Risk 5: Presigned URL Abuse

**Threat:** Pro/Max users generate thousands of presigned URLs, then distribute them publicly. We bear the R2 operation costs; they get a free CDN.

**Assessment:** Presigned URLs expire after 1 hour. R2 reads at scale are cheap ($0.36/M). Even 10M reads/mo = $3.60. Not a real cost risk.

**Mitigation:** Monitor per-account presigned URL generation rates. Already rate-limited (50 shares/hour, 200 uploads/hour).

### Risk 6: Workers CPU Spikes

**Threat:** Complex metadata queries (search, log, list with large results) consume excess CPU.

**Assessment:** Presigned URLs handle all file I/O. Workers only do metadata + auth. CPU per request is typically <5ms. 30M ms/mo included = 6M requests before overage.

**Mitigation:** Cap list/search results (already done: 1,000 max list, 200 max search). Paginate aggressively.

---

## 7. Cost vs. Competitors

| Service | 100 GB Storage | Egress (100 GB/mo) | Total/mo | Our Pro |
|---------|---------------|---------------------|----------|---------|
| AWS S3 | $2.30 | $9.00 | $11.30 | — |
| Google Cloud Storage | $2.00 | $12.00 | $14.00 | — |
| Backblaze B2 | $0.50 | $1.00 | $1.50 | — |
| Cloudflare R2 (raw) | $1.50 | $0 | $1.50 | — |
| **Storage Pro** | — | — | **$20/mo** | Auth, API, agents, UI, support |

**Value proposition:** We charge $20/mo for 100 GB when raw R2 costs $1.50. The $18.50 premium pays for the managed service layer: authentication, actor management, REST API, web UI, share links, presigned uploads, edge delivery, and support.

**Comparable managed services:**
- Supabase Storage Pro: $25/mo (includes 100 GB, but charges $0.021/GB egress)
- Firebase Storage (Blaze): Pay-as-you-go, $0.026/GB stored + $0.12/GB egress
- Uploadthing Pro: $30/mo for 100 GB

Our pricing is competitive and the zero-egress story is a genuine differentiator.

---

## 8. Revenue Optimization Levers

### Not Yet Implemented (Future)

1. **Annual billing** — Offer 2 months free ($200/yr Pro, $1,000/yr Max). Improves cash flow and retention.
2. **Overage pricing** — $0.05/GB/mo beyond tier limits. Captures value from heavy users without forcing tier jumps.
3. **Enterprise tier** — Custom pricing for >1 TB, SSO/SAML, SLAs, audit logs. Could be $500-2,000/mo.
4. **Usage-based add-ons** — Additional actors ($2/actor/mo), additional storage packs ($5/50 GB).
5. **Marketplace integrations** — Charge for premium features (virus scanning, image optimization, CDN custom domains).

### Pricing Sensitivity

At the $1M ARR target (Scenario A):
- Raising Pro from $20 → $25/mo adds $17,500/mo ($210K ARR) with no cost increase
- Raising Max from $100 → $120/mo adds $2,700/mo ($32K ARR)
- Combined: $1.24M ARR from the same user base

Don't raise prices early. Build adoption first, then adjust based on willingness-to-pay signals (upgrade rate, feature requests, support load).

---

## 9. Break-Even Analysis

**Fixed costs (monthly):**

| Item | Cost |
|------|------|
| Cloudflare Workers Paid | $5 |
| Resend Pro | $20 |
| Domain | $1 |
| Monitoring/alerting | $0 (CF observability included) |
| **Total fixed** | **~$26/mo** |

**Variable costs:** ~$0.50/Pro user, ~$3.26/Max user (see Section 3).

**Break-even:** 2 Pro users ($40 revenue - $26 fixed - $1 variable = $13 profit). We are profitable from the second paying customer.

This is the advantage of serverless infrastructure — near-zero fixed costs. There is no server to keep running overnight.

---

## 10. Summary

| Metric | Value |
|--------|-------|
| Gross margin (at $1M ARR) | ~97% |
| Infra cost at $1M ARR | ~$2,300-3,300/mo |
| Break-even | 2 Pro users |
| Time to $1M ARR (modeled) | ~28-30 months |
| Biggest cost risk | Free tier storage abuse |
| Biggest revenue risk | Slow conversion from free to paid |
| Primary competitive advantage | Zero egress + agent-first auth |

The infrastructure economics are strong. R2 zero-egress and Workers serverless pricing mean costs scale linearly with usage at very low per-unit rates. The pricing tiers have 85-97% gross margins across all scenarios modeled.

The primary challenge is not cost management — it's user acquisition and free-to-paid conversion. Focus engineering effort on features that drive upgrades: presigned uploads (Pro gate), team sharing (Max gate), and storage usage visibility (creates natural upgrade pressure).
