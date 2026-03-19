-- Seed Spaces for test@localhost.com (actor: u/test.2ks9) on remote
-- 3 spaces per spec/0756: storage.now, API Reference, Sprint 12
-- Uses real object IDs from deployed database. No emoji.

-- Ensure collaborator actors exist
INSERT OR IGNORE INTO actors (actor, type, email, bio, created_at)
VALUES ('u/alice', 'human', 'alice@example.com', 'Engineering Lead', 1710700000000);
INSERT OR IGNORE INTO actors (actor, type, email, bio, created_at)
VALUES ('u/bob', 'human', 'bob@example.com', 'Backend Engineer', 1710700000000);
INSERT OR IGNORE INTO actors (actor, type, email, bio, created_at)
VALUES ('u/carol', 'human', 'carol@example.com', 'Product Manager', 1710700000000);
INSERT OR IGNORE INTO actors (actor, type, public_key, bio, created_at)
VALUES ('a/codegen', 'agent', 'ed25519_placeholder_key_codegen', 'AI code generation agent', 1710700000000);
INSERT OR IGNORE INTO actors (actor, type, public_key, bio, created_at)
VALUES ('a/summarizer', 'agent', 'ed25519_placeholder_key_summarizer', 'Auto-summarization agent', 1710700000000);

-- Clean existing spaces for this user
DELETE FROM space_activity WHERE space_id IN (SELECT id FROM spaces WHERE owner = 'u/test.2ks9');
DELETE FROM space_items WHERE space_id IN (SELECT id FROM spaces WHERE owner = 'u/test.2ks9');
DELETE FROM space_sections WHERE space_id IN (SELECT id FROM spaces WHERE owner = 'u/test.2ks9');
DELETE FROM space_members WHERE space_id IN (SELECT id FROM spaces WHERE owner = 'u/test.2ks9');
DELETE FROM spaces WHERE owner = 'u/test.2ks9';
-- Also clean alice-owned spaces
DELETE FROM space_activity WHERE space_id IN (SELECT id FROM spaces WHERE owner = 'u/alice');
DELETE FROM space_items WHERE space_id IN (SELECT id FROM spaces WHERE owner = 'u/alice');
DELETE FROM space_sections WHERE space_id IN (SELECT id FROM spaces WHERE owner = 'u/alice');
DELETE FROM space_members WHERE space_id IN (SELECT id FROM spaces WHERE owner = 'u/alice');
DELETE FROM spaces WHERE owner = 'u/alice';

-- ═══════════════════════════════════════════════════════
-- SPACE 1: storage.now
-- A real project space. Code, config, images, docs.
-- Shows how a development team organizes a project.
-- ═══════════════════════════════════════════════════════
INSERT INTO spaces (id, owner, title, description, cover_url, icon, visibility, created_at, updated_at)
VALUES ('sp_r01', 'u/test.2ks9', 'storage.now', 'Source code, configuration, assets, and documentation for the storage platform.', '', '', 'team', 1710000000000, 1711200000000);

INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_r01', 'sp_r01', 'u/alice', 'editor', 1710000000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_r02', 'sp_r01', 'u/bob', 'editor', 1710000000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_r03', 'sp_r01', 'a/codegen', 'viewer', 1710100000000);

-- Section: Source Code
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_r01', 'sp_r01', 'Source Code', 'TypeScript and Go source files', 0, 1710000000000, 1711200000000);

-- Section: Configuration
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_r02', 'sp_r01', 'Configuration', 'Build and project configs', 1, 1710000000000, 1711100000000);

-- Section: Assets
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_r03', 'sp_r01', 'Assets', 'Images, icons, and visual assets', 2, 1710100000000, 1711000000000);

-- Section: Documentation
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_r04', 'sp_r01', 'Documentation', 'READMEs and project docs', 3, 1710000000000, 1711100000000);

-- Items: Source Code
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_r01', 'ss_r01', 'sp_r01', 'file', 'index.ts', 'Main entry point — Hono router and middleware', 'o_73b14a449d7dc836e82350e4', 0, 'u/test.2ks9', 1710000000000, 1711200000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_r02', 'ss_r01', 'sp_r01', 'file', 'hello.go', 'Go server example', 'o_8ff2b90eb65803bc24d12e47', 1, 'u/bob', 1710100000000, 1710800000000);

-- Items: Configuration
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_r03', 'ss_r02', 'sp_r01', 'file', 'tsconfig.json', 'TypeScript compiler configuration', 'o_fe94604a7205e77d128d5a38', 0, 'u/test.2ks9', 1710000000000, 1710500000000);

-- Items: Assets
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_r04', 'ss_r03', 'sp_r01', 'file', 'gopher.png', 'Go mascot — primary logo', 'o_36692dcf0e04e57de215d549', 0, 'u/test.2ks9', 1710100000000, 1710100000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_r05', 'ss_r03', 'sp_r01', 'file', 'gopher_2.png', 'Alternate gopher variant', 'o_31a7593f11160da95c570408', 1, 'u/test.2ks9', 1710100000000, 1710100000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_r06', 'ss_r03', 'sp_r01', 'file', 'pixel.png', 'Pixel art asset', 'o_390db585124832c9d0c32422', 2, 'u/alice', 1710200000000, 1710200000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_r07', 'ss_r03', 'sp_r01', 'file', 'Team photo', 'Team photo from Q1 offsite', 'o_f282a7067b19550196444fdd', 3, 'u/test.2ks9', 1710200000000, 1710200000000);

-- Items: Documentation
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_r08', 'ss_r04', 'sp_r01', 'file', 'README.md', 'Project overview and getting started', 'o_756a751a3c6ed92de9e8a3cd', 0, 'u/test.2ks9', 1710000000000, 1710800000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_r09', 'ss_r04', 'sp_r01', 'file', 'Webapp README', 'Webapp-specific documentation', 'o_92fdb3047f4d295962935e45', 1, 'u/test.2ks9', 1710000000000, 1710600000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, note_body, position, added_by, created_at, updated_at)
VALUES ('si_r10', 'ss_r04', 'sp_r01', 'note', 'Architecture Notes', 'Key decisions and stack overview', '## Architecture

- **Runtime**: Cloudflare Workers
- **Framework**: Hono (TypeScript)
- **Database**: D1 (SQLite at edge)
- **Storage**: R2 (S3-compatible)

### Key Decisions
- Server-side HTML rendering, no client framework
- REST-first API with MCP support
- Actor model: u/ for humans, a/ for agents
- Monochrome design language, sharp corners, 1px borders', 2, 'u/alice', 1710200000000, 1711000000000);

-- ═══════════════════════════════════════════════════════
-- SPACE 2: API Reference
-- Documentation space. Notes, URLs, markdown.
-- Shows knowledge-base use case.
-- ═══════════════════════════════════════════════════════
INSERT INTO spaces (id, owner, title, description, cover_url, icon, visibility, created_at, updated_at)
VALUES ('sp_r02', 'u/test.2ks9', 'API Reference', 'API documentation, endpoint references, and integration guides for storage.now.', '', '', 'public', 1710100000000, 1711200000000);

INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_r04', 'sp_r02', 'u/alice', 'editor', 1710100000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_r05', 'sp_r02', 'a/codegen', 'editor', 1710200000000);

-- Section: Endpoints
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_r05', 'sp_r02', 'Endpoints', 'API endpoint documentation and examples', 0, 1710100000000, 1711100000000);

-- Section: Authentication
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_r06', 'sp_r02', 'Authentication', 'Auth flows and security', 1, 1710100000000, 1711000000000);

-- Section: External References
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_r07', 'sp_r02', 'External References', 'Links to related documentation', 2, 1710200000000, 1711200000000);

-- Items: Endpoints
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, note_body, position, added_by, created_at, updated_at)
VALUES ('si_r11', 'ss_r05', 'sp_r02', 'note', 'Files API', 'Upload, download, and manage files', '## Files API

### Upload
PUT /files/{path}
Content-Type: application/octet-stream

### Download
GET /files/{path}

### Delete
DELETE /files/{path}

### Head (metadata only)
HEAD /files/{path}

All endpoints require Bearer token or session cookie.', 0, 'u/test.2ks9', 1710100000000, 1711100000000);

INSERT INTO space_items (id, section_id, space_id, item_type, title, description, note_body, position, added_by, created_at, updated_at)
VALUES ('si_r12', 'ss_r05', 'sp_r02', 'note', 'Spaces API', 'Manage collaborative workspaces', '## Spaces API

### List spaces
GET /spaces/list

### Get space detail
GET /spaces/{id}
Returns: space, sections, items (with file metadata), members, activity

### Create space
POST /spaces
Body: { title, description?, visibility? }

### Add section
POST /spaces/{id}/sections
Body: { title, description? }

### Add item
POST /spaces/{id}/items
Body: { section_id, item_type, title, object_id?, url?, note_body? }', 1, 'u/alice', 1710200000000, 1711200000000);

-- Items: Authentication
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, note_body, position, added_by, created_at, updated_at)
VALUES ('si_r13', 'ss_r06', 'sp_r02', 'note', 'Auth Flows', 'Authentication methods', '## Authentication

### Magic Link (humans)
1. POST /auth/magic-link { email }
2. User clicks link in email
3. GET /auth/magic/:token sets session cookie

### Ed25519 Challenge (agents)
1. POST /auth/challenge { actor }
2. Sign challenge with private key
3. POST /auth/verify { actor, signature }
4. Returns bearer token

### OAuth 2.0
Standard authorization code flow via /oauth/* endpoints.
Supports dynamic client registration (RFC 7591).', 0, 'u/test.2ks9', 1710100000000, 1711000000000);

-- Items: External References
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, url, position, added_by, created_at, updated_at)
VALUES ('si_r14', 'ss_r07', 'sp_r02', 'url', 'Hono Framework', 'Web framework documentation', 'https://hono.dev/docs', 0, 'u/alice', 1710200000000, 1710200000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, url, position, added_by, created_at, updated_at)
VALUES ('si_r15', 'ss_r07', 'sp_r02', 'url', 'Cloudflare Workers Docs', 'Runtime documentation', 'https://developers.cloudflare.com/workers', 1, 'u/alice', 1710200000000, 1710200000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, url, position, added_by, created_at, updated_at)
VALUES ('si_r16', 'ss_r07', 'sp_r02', 'url', 'D1 Database', 'SQLite at the edge', 'https://developers.cloudflare.com/d1', 2, 'u/test.2ks9', 1710300000000, 1710300000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, url, position, added_by, created_at, updated_at)
VALUES ('si_r17', 'ss_r07', 'sp_r02', 'url', 'MCP Specification', 'Model Context Protocol', 'https://modelcontextprotocol.io/specification', 3, 'a/codegen', 1710400000000, 1710400000000);

-- ═══════════════════════════════════════════════════════
-- SPACE 3: Sprint 12
-- Time-boxed collaboration. Mixed content, multiple
-- contributors including AI agents. Active teamwork.
-- ═══════════════════════════════════════════════════════
INSERT INTO spaces (id, owner, title, description, cover_url, icon, visibility, created_at, updated_at)
VALUES ('sp_r03', 'u/test.2ks9', 'Sprint 12', 'Current sprint: Spaces feature, file preview, AI integration. Ends March 22.', '', '', 'team', 1710200000000, 1711200000000);

INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_r06', 'sp_r03', 'u/alice', 'editor', 1710200000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_r07', 'sp_r03', 'u/bob', 'editor', 1710200000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_r08', 'sp_r03', 'u/carol', 'editor', 1710200000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_r09', 'sp_r03', 'a/summarizer', 'editor', 1710300000000);

-- Section: Goals & Planning
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_r08', 'sp_r03', 'Goals & Planning', 'Sprint objectives and tracking', 0, 1710200000000, 1711200000000);

-- Section: In Progress
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_r09', 'sp_r03', 'In Progress', 'Work currently being done', 1, 1710200000000, 1711200000000);

-- Section: References
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_r10', 'sp_r03', 'References', 'Links, notes, and background', 2, 1710300000000, 1711100000000);

-- Items: Goals & Planning
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, note_body, position, added_by, created_at, updated_at)
VALUES ('si_r18', 'ss_r08', 'sp_r03', 'note', 'Sprint Goals', 'What we are shipping', '## Sprint 12 Goals

1. [x] Deploy Spaces feature (listing + detail)
2. [x] Redesign spaces page (monochrome, no emoji)
3. [ ] File preview in space detail view
4. [ ] AI agent auto-summarization
5. [ ] Drag-and-drop section reordering

**Deadline**: March 22, 2026
**Owner**: test.2ks9', 0, 'u/test.2ks9', 1710200000000, 1711200000000);

INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_r19', 'ss_r08', 'sp_r03', 'file', 'standup-march.md', 'Weekly standup notes', 'o_0e216c165587c1c901e626c2', 1, 'u/carol', 1710300000000, 1711100000000);

INSERT INTO space_items (id, section_id, space_id, item_type, title, description, url, position, added_by, created_at, updated_at)
VALUES ('si_r20', 'ss_r08', 'sp_r03', 'url', 'Sprint Board', 'Track progress on Linear', 'https://linear.app/team/sprints/12', 2, 'u/carol', 1710300000000, 1710300000000);

-- Items: In Progress
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_r21', 'ss_r09', 'sp_r03', 'file', 'todo.md', 'Current action items and tasks', 'o_070cd5d8205661e9f3f24a74', 0, 'u/test.2ks9', 1710200000000, 1711200000000);

INSERT INTO space_items (id, section_id, space_id, item_type, title, description, note_body, position, added_by, created_at, updated_at)
VALUES ('si_r22', 'ss_r09', 'sp_r03', 'note', 'AI Summary', 'Auto-generated sprint summary', '## Sprint 12 Status (auto-generated)

**Progress**: 2/5 goals complete (40%)
**Velocity**: On track

### Completed
- Spaces feature deployed with listing and detail views
- Monochrome redesign applied to spaces page

### Blockers
- File preview requires R2 presigned URL changes
- Agent API needs rate limit adjustments

Last updated: March 18, 2026 by a/summarizer', 1, 'a/summarizer', 1710800000000, 1711200000000);

-- Items: References
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, note_body, position, added_by, created_at, updated_at)
VALUES ('si_r23', 'ss_r10', 'sp_r03', 'note', 'Design System', 'Visual language reference', '## Design System

- **Colors**: Monochrome — #FAFAF9 (light), #09090B (dark)
- **Borders**: 1px solid, zero radius, no shadows
- **Fonts**: Inter (body), JetBrains Mono (labels, code)
- **Grid**: 1px gap pattern, auto-fill columns
- **Identity**: Typography-based (monograms, language badges), no emoji

### Principles
1. Information density over decoration
2. Typography IS the interface
3. Monochrome with zero exceptions', 0, 'u/alice', 1710400000000, 1711000000000);

INSERT INTO space_items (id, section_id, space_id, item_type, title, description, url, position, added_by, created_at, updated_at)
VALUES ('si_r24', 'ss_r10', 'sp_r03', 'url', 'Cloudflare Dashboard', 'Workers and D1 management', 'https://dash.cloudflare.com', 1, 'u/bob', 1710400000000, 1710400000000);

-- ═══════════════════════════════════════════════════════
-- ACTIVITY (rich history across all 3 spaces)
-- ═══════════════════════════════════════════════════════

-- storage.now activity
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r01', 'sp_r01', 'u/test.2ks9', 'created', 'storage.now', 1710000000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r02', 'sp_r01', 'u/test.2ks9', 'added_item', 'index.ts to Source Code', 1710000000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r03', 'sp_r01', 'u/bob', 'added_item', 'hello.go to Source Code', 1710100000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r04', 'sp_r01', 'u/alice', 'added_item', 'pixel.png to Assets', 1710200000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r05', 'sp_r01', 'u/alice', 'added_item', 'Architecture Notes', 1710200000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r06', 'sp_r01', 'u/test.2ks9', 'shared', 'with a/codegen', 1710800000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r07', 'sp_r01', 'u/test.2ks9', 'edited', 'index.ts', 1711200000000);

-- API Reference activity
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r08', 'sp_r02', 'u/test.2ks9', 'created', 'API Reference', 1710100000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r09', 'sp_r02', 'u/alice', 'added_item', 'Spaces API', 1710200000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r10', 'sp_r02', 'a/codegen', 'added_item', 'MCP Specification', 1710400000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r11', 'sp_r02', 'u/alice', 'edited', 'Spaces API', 1711200000000);

-- Sprint 12 activity
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r12', 'sp_r03', 'u/test.2ks9', 'created', 'Sprint 12', 1710200000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r13', 'sp_r03', 'u/carol', 'added_item', 'standup-march.md', 1710300000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r14', 'sp_r03', 'u/carol', 'added_item', 'Sprint Board link', 1710300000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r15', 'sp_r03', 'u/alice', 'added_item', 'Design System note', 1710400000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r16', 'sp_r03', 'a/summarizer', 'added_item', 'AI Summary', 1710800000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r17', 'sp_r03', 'u/test.2ks9', 'edited', 'Sprint Goals', 1711200000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at) VALUES ('sa_r18', 'sp_r03', 'a/summarizer', 'edited', 'AI Summary', 1711200000000);
