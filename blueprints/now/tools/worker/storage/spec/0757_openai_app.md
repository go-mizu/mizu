# 0757 — OpenAI ChatGPT App Submission Checklist

Storage MCP server compliance checklist for the ChatGPT App Directory.
Source: https://developers.openai.com/apps-sdk/deploy/submission
        https://developers.openai.com/apps-sdk/app-submission-guidelines
        https://developers.openai.com/apps-sdk/guides/security-privacy

---

## 1. Organization & Account

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| 1.1 | Organization identity verified (individual or business) | ⬜ | OpenAI Platform Dashboard → General Settings |
| 1.2 | Publishing under verified name | ⬜ | Must match verification — rejection if mismatched |
| 1.3 | Owner role in organization | ⬜ | Required for verification + submission |
| 1.4 | Support contact details provided | ⬜ | Email or URL for end-user support |

## 2. MCP Server Requirements

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| 2.1 | Server hosted on publicly accessible domain | ✅ | `https://storage.liteio.dev/mcp` |
| 2.2 | Not using local or testing endpoints | ✅ | Production Cloudflare Worker |
| 2.3 | Content Security Policy (CSP) defined | ⬜ | **MISSING** — must allow exact domains we fetch from |
| 2.4 | OAuth 2.1 with PKCE (S256) | ✅ | Implemented in `src/routes/oauth.ts` |
| 2.5 | Dynamic client registration (RFC 7591) | ✅ | `POST /oauth/register` |
| 2.6 | `.well-known/oauth-authorization-server` | ✅ | Returns server metadata |
| 2.7 | `.well-known/oauth-protected-resource` | ✅ | Returns resource metadata |
| 2.8 | Tokens validated on every tool call | ✅ | `auth` middleware on all MCP routes |
| 2.9 | Invalid tokens return 401 | ✅ | Auth middleware rejects with 401 |

## 3. Tool Annotations (Top Rejection Reason)

**CRITICAL**: All tools must have `readOnlyHint`, `destructiveHint`, and `openWorldHint` annotations.
Currently **NONE** of our 8 tools have annotations — this will cause immediate rejection.

| Tool | readOnlyHint | destructiveHint | openWorldHint | Status |
|------|-------------|-----------------|---------------|--------|
| `storage_list` | `true` | `false` | `false` | ✅ Added |
| `storage_read` | `true` | `false` | `false` | ✅ Added |
| `storage_write` | `false` | `false` | `false` | ✅ Added |
| `storage_delete` | `false` | `true` | `false` | ✅ Added |
| `storage_search` | `true` | `false` | `false` | ✅ Added |
| `storage_move` | `false` | `false` | `false` | ✅ Added |
| `storage_share` | `false` | `false` | `true` | ✅ Added |
| `storage_stats` | `true` | `false` | `false` | ✅ Added |

### Annotation Rationale

- **`storage_delete`** `destructiveHint: true` — Permanently deletes files, irreversible. Recursive folder deletion possible.
- **`storage_share`** `openWorldHint: true` — Creates a publicly accessible URL that anyone on the internet can access without authentication. This changes publicly visible internet state.
- **`storage_move`** `destructiveHint: false` — Moves file to new path. Overwrites destination if exists, but the data is preserved (not destroyed). Edge case: could overwrite a different file at the destination.
- **`storage_write`** `destructiveHint: false` — Creates or overwrites file. Data is in private storage, not publicly accessible. Overwrite behavior is standard for file storage.

## 4. Data Minimization (Second Top Rejection Reason)

Tools must return ONLY data necessary for the user's request. No internal identifiers, telemetry, or debug payloads.

| # | Issue | Status | Fix Required |
|---|-------|--------|-------------|
| 4.1 | `storage_stats` returns `actor` field (internal ID like `h/alice`) | ✅ | **REMOVED** — actor field stripped from response |
| 4.2 | `storage_read` returns `etag` field | ✅ | **REMOVED** — etag stripped from response metadata |
| 4.3 | `storage_write` returns `updated_at` as epoch ms | ✅ | Keep — useful creation timestamp |
| 4.4 | `storage_list` returns `updated_at` as epoch ms | ✅ | Keep — useful for sorting |
| 4.5 | No session IDs, trace IDs, or request IDs in responses | ✅ | We don't return these |
| 4.6 | No auth tokens or secrets in responses | ✅ | We don't return these |
| 4.7 | Console log includes actor + IP | ⚠️ | OK for server logs, but redact IP from any user-facing output |

## 5. Tool Definitions Quality

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| 5.1 | Human-readable, specific, action-descriptive names | ✅ | `storage_list`, `storage_read`, etc. |
| 5.2 | Unique names within app | ✅ | All 8 names are unique |
| 5.3 | No misleading or promotional language | ✅ | Descriptions are factual |
| 5.4 | Descriptions clearly explain purpose | ✅ | Detailed descriptions for all tools |
| 5.5 | Descriptions don't favor competing apps | ✅ | No competitive references |
| 5.6 | Only necessary input fields requested | ✅ | Minimal, purpose-driven inputs |
| 5.7 | No full conversation history requested | ✅ | We don't request chat logs |
| 5.8 | No GPS/location data requested | ✅ | We don't request location |
| 5.9 | Predictable, auditable behavior | ✅ | Tools do what descriptions say |
| 5.10 | No hidden side effects | ✅ | All effects are documented |
| 5.11 | `storage_delete` description warns about permanence | ✅ | "permanent and cannot be undone" |

## 6. Privacy & Legal

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| 6.1 | Privacy policy page published | ✅ | Created at `/privacy` — `src/pages/privacy.ts` |
| 6.2 | Privacy policy covers: data collected | ⬜ | Email, files, usage metadata |
| 6.3 | Privacy policy covers: use purposes | ⬜ | Storage, sharing, auth |
| 6.4 | Privacy policy covers: recipient categories | ⬜ | Share link recipients |
| 6.5 | Privacy policy covers: user controls | ⬜ | Delete, export, account removal |
| 6.6 | Company URL provided | ⬜ | Need to provide in submission |
| 6.7 | No PCI DSS data collected | ✅ | We don't handle payment cards |
| 6.8 | No PHI collected | ✅ | We don't collect health info |
| 6.9 | No government IDs collected | ✅ | We don't collect SSNs etc. |
| 6.10 | No access credentials stored/returned | ✅ | Tokens are not in tool responses |
| 6.11 | No behavioral profiling | ✅ | No tracking beyond auth |

## 7. App Submission Fields

| # | Field | Status | Value |
|---|-------|--------|-------|
| 7.1 | App name | ⬜ | "Storage" |
| 7.2 | Logo | ⬜ | Need to create/provide |
| 7.3 | Description | ⬜ | "Store, organize, and share files from ChatGPT. Upload files, browse folders, search by name, and generate share links — all through natural language." |
| 7.4 | Company URL | ⬜ | `https://storage.liteio.dev` |
| 7.5 | Privacy policy URL | ⬜ | `https://storage.liteio.dev/privacy` (must create) |
| 7.6 | MCP server URL | ✅ | `https://storage.liteio.dev/mcp` |
| 7.7 | Screenshots | ⬜ | Need 3+ screenshots of ChatGPT using Storage |
| 7.8 | Test prompts + expected responses | ⬜ | See §9 below |
| 7.9 | Test credentials (demo account) | ⬜ | **MUST CREATE** — working demo account, no 2FA/email verification |
| 7.10 | Localization info | ⬜ | English only initially |

## 8. Test Credentials & Demo Account

OpenAI reviewers need a fully functional demo account with sample data, accessible without 2FA, SMS, or email verification.

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| 8.1 | Demo account with login credentials | ⬜ | Need API key or direct bearer token |
| 8.2 | No 2FA/MFA required | ⬜ | Magic links require email — **PROBLEM** |
| 8.3 | No SMS/email verification during login | ⬜ | Our auth IS email-based — need API key bypass |
| 8.4 | Sample data pre-loaded | ⬜ | Seed demo account with example files |
| 8.5 | Credentials don't expire during review | ⬜ | API keys last 90 days ✅, but need to ensure |
| 8.6 | Test on both ChatGPT web and mobile | ⬜ | Verify MCP works on both |

**Solution for 8.2-8.3**: Provide an API key as the demo credential. API keys bypass email verification and work as bearer tokens. Create a dedicated `demo-reviewer` actor with pre-loaded sample files.

## 9. Test Prompts

| # | Prompt | Expected Behavior | Status |
|---|--------|-------------------|--------|
| 9.1 | "What files do I have?" | Calls `storage_list`, shows file listing | ⬜ |
| 9.2 | "Save this as notes.md: Hello world" | Calls `storage_write`, confirms save | ⬜ |
| 9.3 | "Read my notes.md file" | Calls `storage_read`, shows content | ⬜ |
| 9.4 | "Share notes.md with a link" | Calls `storage_share`, shows URL | ⬜ |
| 9.5 | "Search for files named 'report'" | Calls `storage_search`, shows results | ⬜ |
| 9.6 | "How much storage am I using?" | Calls `storage_stats`, shows usage | ⬜ |
| 9.7 | "Move notes.md to docs/notes.md" | Calls `storage_move`, confirms move | ⬜ |
| 9.8 | "Delete the old draft file" | Asks confirmation, calls `storage_delete` | ⬜ |

## 10. Safety & Content

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| 10.1 | Suitable for general audiences (13+) | ✅ | File storage is general-purpose |
| 10.2 | Does not target children under 13 | ✅ | No age-specific targeting |
| 10.3 | No prohibited commerce (digital subscriptions) | ⚠️ | **RISK** — Pricing page shows $20/mo and $100/mo digital subscriptions. OpenAI currently prohibits digital products/subscriptions through apps. Keep pricing external, don't sell through ChatGPT. |
| 10.4 | No advertisements served | ✅ | No ads |
| 10.5 | App provides standalone utility | ✅ | File storage + sharing |
| 10.6 | No scraping/unauthorized API access | ✅ | `storage_write` URL mode fetches user-provided URLs only |
| 10.7 | Not an unofficial third-party connector | ✅ | We ARE the first-party service |
| 10.8 | Does not bypass API restrictions | ✅ | Standard HTTP behavior |

## 11. Security

| # | Requirement | Status | Notes |
|---|-------------|--------|-------|
| 11.1 | OAuth 2.1 with PKCE S256 | ✅ | Implemented |
| 11.2 | Scopes enforced on every call | ⚠️ | Default scope is `*` (all) — should scope tools to requested permissions |
| 11.3 | Server-side input validation | ✅ | Path validation, size limits |
| 11.4 | SQL injection prevention | ✅ | Prepared statements with bindings |
| 11.5 | No stored long-lived secrets | ✅ | Tokens are session-based |
| 11.6 | Confirmation for destructive actions | ✅ | `storage_delete` description says "confirm with user" |
| 11.7 | Dependencies up to date | ⬜ | Check for known vulnerabilities |

---

## Priority Fix List

### P0 — Will cause immediate rejection

1. ~~**Add tool annotations** to all 8 tools (`readOnlyHint`, `destructiveHint`, `openWorldHint`)~~ ✅ DONE
2. ~~**Create privacy policy page** at `/privacy`~~ ✅ DONE
3. **Create demo account** with API key for reviewers (no email verification)
4. ~~**Remove `actor` from `storage_stats`** response (undisclosed internal ID)~~ ✅ DONE
5. ~~**Remove `etag` from `storage_read`** response (internal metadata)~~ ✅ DONE

### P1 — Should fix before submission

6. **Add CSP headers** to MCP responses
7. **Prepare test prompts** with expected outputs
8. **Create screenshots** of ChatGPT using Storage
9. **Prepare app logo**
10. **Organization verification** on OpenAI Platform

### P2 — Good to have

11. Scope enforcement per OAuth grant (currently `*`)
12. Rate limiting per OAuth client
13. Terms of service page at `/terms`
14. Audit log for tool calls
