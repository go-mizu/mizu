-- Seed sample Spaces data for test@localhost.com (actor: u/test)
-- References files from the main seed.sql

-- Ensure extra actors exist for collaboration demo
INSERT OR IGNORE INTO actors (actor, type, email, bio, created_at)
VALUES ('u/alice', 'human', 'alice@example.com', 'Engineering Lead', 1710700000000);
INSERT OR IGNORE INTO actors (actor, type, email, bio, created_at)
VALUES ('u/bob', 'human', 'bob@example.com', 'Backend Engineer', 1710700000000);
INSERT OR IGNORE INTO actors (actor, type, email, bio, created_at)
VALUES ('u/carol', 'human', 'carol@example.com', 'Product Manager', 1710700000000);
INSERT OR IGNORE INTO actors (actor, type, email, bio, created_at)
VALUES ('u/dave', 'human', 'dave@example.com', 'DevOps Engineer', 1710700000000);
INSERT OR IGNORE INTO actors (actor, type, public_key, bio, created_at)
VALUES ('a/reports-bot', 'agent', 'ed25519_placeholder_key_for_seed', 'Generates weekly reports and summaries', 1710700000000);
INSERT OR IGNORE INTO actors (actor, type, public_key, bio, created_at)
VALUES ('a/organizer', 'agent', 'ed25519_placeholder_key_for_seed2', 'Auto-organizes files by project', 1710700000000);

-- Clean existing spaces data for u/test
DELETE FROM space_activity WHERE space_id IN (SELECT id FROM spaces WHERE owner = 'u/test');
DELETE FROM space_items WHERE space_id IN (SELECT id FROM spaces WHERE owner = 'u/test');
DELETE FROM space_sections WHERE space_id IN (SELECT id FROM spaces WHERE owner = 'u/test');
DELETE FROM space_members WHERE space_id IN (SELECT id FROM spaces WHERE owner = 'u/test');
DELETE FROM spaces WHERE owner = 'u/test';

-- ═══════════════════════════════════════════
-- SPACE 1: Q1 Product Launch
-- ═══════════════════════════════════════════
INSERT INTO spaces (id, owner, title, description, cover_url, icon, visibility, created_at, updated_at)
VALUES ('sp_001', 'u/test', 'Q1 Product Launch', 'All assets, strategy docs, and design files for the spring product launch campaign.', '', '🚀', 'team', 1710000000000, 1710600000000);

-- Members
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_001', 'sp_001', 'u/alice', 'editor', 1710000000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_002', 'sp_001', 'u/bob', 'editor', 1710000000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_003', 'sp_001', 'u/carol', 'viewer', 1710100000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_004', 'sp_001', 'a/reports-bot', 'viewer', 1710100000000);

-- Sections
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_001', 'sp_001', 'Strategy & Planning', 'Project proposals, roadmaps, and planning documents', 0, 1710000000000, 1710500000000);
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_002', 'sp_001', 'Design Assets', 'Logos, screenshots, and visual identity files', 1, 1710000000000, 1710400000000);
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_003', 'sp_001', 'Meeting Notes', 'Weekly standup and planning session notes', 2, 1710100000000, 1710300000000);

-- Items (referencing real files from seed.sql)
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_001', 'ss_001', 'sp_001', 'file', 'Project Proposal', 'Q2 product roadmap and resource allocation plan', 'o_d001000000000001', 0, 'u/test', 1710000000000, 1710100000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_002', 'ss_001', 'sp_001', 'file', 'Q1 Metrics', 'Quarterly metrics export', 'o_d001000000000005', 1, 'u/alice', 1710100000000, 1710100000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, url, position, added_by, created_at, updated_at)
VALUES ('si_003', 'ss_001', 'sp_001', 'url', 'Competitive Analysis (Notion)', 'External research doc', 'https://notion.so/competitive-analysis-q1', 2, 'u/carol', 1710200000000, 1710200000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_004', 'ss_002', 'sp_001', 'file', 'Dashboard Screenshot', 'Latest dashboard design v2', 'o_d001000000000009', 0, 'u/test', 1710000000000, 1710000000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_005', 'ss_002', 'sp_001', 'file', 'Mobile App Screenshot', 'Mobile app capture', 'o_d001000000000010', 1, 'u/test', 1710000000000, 1710000000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_006', 'ss_002', 'sp_001', 'file', 'Company Logo', 'SVG logo for the campaign', 'o_d001000000000007', 2, 'u/alice', 1710100000000, 1710100000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_007', 'ss_002', 'sp_001', 'file', 'Banner Image', 'Hero banner for landing', 'o_d001000000000008', 3, 'u/alice', 1710100000000, 1710100000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_008', 'ss_003', 'sp_001', 'file', 'March Meeting Notes', 'Weekly standup notes', 'o_d001000000000002', 0, 'u/test', 1710100000000, 1710300000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, note_body, position, added_by, created_at, updated_at)
VALUES ('si_009', 'ss_003', 'sp_001', 'note', 'Launch Checklist', 'Key milestones before go-live', '- [ ] Final design review\n- [ ] API documentation complete\n- [ ] Load testing passed\n- [x] Marketing copy approved\n- [x] Legal review complete', 1, 'u/carol', 1710200000000, 1710500000000);

-- ═══════════════════════════════════════════
-- SPACE 2: Analytics & Reports
-- ═══════════════════════════════════════════
INSERT INTO spaces (id, owner, title, description, cover_url, icon, visibility, created_at, updated_at)
VALUES ('sp_002', 'u/test', 'Analytics & Reports', 'Dashboards, metrics exports, and reporting pipeline outputs.', '', '📊', 'team', 1710100000000, 1710600000000);

INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_005', 'sp_002', 'u/alice', 'viewer', 1710100000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_006', 'sp_002', 'a/reports-bot', 'editor', 1710100000000);

INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_004', 'sp_002', 'Reports', 'Generated reports and reviews', 0, 1710100000000, 1710500000000);
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_005', 'sp_002', 'Raw Data', 'CSV and JSONL data exports', 1, 1710100000000, 1710400000000);
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_006', 'sp_002', 'Dashboards', 'Links to live dashboards', 2, 1710200000000, 1710200000000);

INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_010', 'ss_004', 'sp_002', 'file', 'Annual Review', '2024 annual performance and financial review', 'o_d001000000000006', 0, 'u/test', 1710100000000, 1710200000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_011', 'ss_004', 'sp_002', 'file', 'Q1 Metrics', 'First quarter metrics export', 'o_d001000000000005', 1, 'a/reports-bot', 1710200000000, 1710200000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_012', 'ss_005', 'sp_002', 'file', 'Analytics March Export', 'Monthly analytics data', 'o_d001000000000016', 0, 'a/reports-bot', 1710200000000, 1710200000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_013', 'ss_005', 'sp_002', 'file', 'Users Export', 'User data JSON export', 'o_d001000000000015', 1, 'u/test', 1710200000000, 1710300000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_014', 'ss_005', 'sp_002', 'file', 'Logs March', 'Application logs for March', 'o_d001000000000017', 2, 'a/reports-bot', 1710200000000, 1710200000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, url, position, added_by, created_at, updated_at)
VALUES ('si_015', 'ss_006', 'sp_002', 'url', 'Grafana Dashboard', 'Live metrics dashboard', 'https://grafana.internal/d/main', 0, 'u/dave', 1710200000000, 1710200000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, url, position, added_by, created_at, updated_at)
VALUES ('si_016', 'ss_006', 'sp_002', 'url', 'Datadog APM', 'Application performance monitoring', 'https://app.datadoghq.com/apm', 1, 'u/dave', 1710200000000, 1710200000000);

-- ═══════════════════════════════════════════
-- SPACE 3: Design System
-- ═══════════════════════════════════════════
INSERT INTO spaces (id, owner, title, description, cover_url, icon, visibility, created_at, updated_at)
VALUES ('sp_003', 'u/test', 'Design System', 'Brand assets, style guides, Figma files, and visual identity resources.', '', '🎨', 'public', 1710200000000, 1710500000000);

INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_007', 'sp_003', 'u/alice', 'editor', 1710200000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_008', 'sp_003', 'u/bob', 'viewer', 1710200000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_009', 'sp_003', 'u/carol', 'viewer', 1710200000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_010', 'sp_003', 'u/dave', 'viewer', 1710300000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_011', 'sp_003', 'a/organizer', 'editor', 1710300000000);

INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_007', 'sp_003', 'Logos & Icons', 'Official logo variations and app icons', 0, 1710200000000, 1710400000000);
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_008', 'sp_003', 'Photography', 'Team photos and campaign imagery', 1, 1710200000000, 1710300000000);
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_009', 'sp_003', 'Design Files', 'Figma files and design specifications', 2, 1710200000000, 1710500000000);

INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_017', 'ss_007', 'sp_003', 'file', 'Logo SVG', 'Official logo vector format', 'o_d001000000000007', 0, 'u/test', 1710200000000, 1710200000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_018', 'ss_007', 'sp_003', 'file', 'Banner Image', 'Website hero banner', 'o_d001000000000008', 1, 'u/alice', 1710200000000, 1710200000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_019', 'ss_008', 'sp_003', 'file', 'Team Offsite Photo', 'Q1 offsite in Portland', 'o_d001000000000011', 0, 'u/test', 1710200000000, 1710200000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_020', 'ss_009', 'sp_003', 'file', 'Design System Figma', 'Master design system file', 'o_d001000000000019', 0, 'u/alice', 1710300000000, 1710400000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, note_body, position, added_by, created_at, updated_at)
VALUES ('si_021', 'ss_009', 'sp_003', 'note', 'Brand Colors', 'Primary palette reference', '## Brand Colors\n\n- **Primary**: #171717 (Black)\n- **Accent**: #667eea (Blue)\n- **Success**: #43e97b (Green)\n- **Warning**: #fa709a (Pink)\n- **Background**: #fafafa (Light) / #0a0a0a (Dark)', 1, 'u/alice', 1710300000000, 1710500000000);

-- ═══════════════════════════════════════════
-- SPACE 4: API & Developer Hub
-- ═══════════════════════════════════════════
INSERT INTO spaces (id, owner, title, description, cover_url, icon, visibility, created_at, updated_at)
VALUES ('sp_004', 'u/test', 'API & Developer Hub', 'OpenAPI specs, integration guides, and developer resources for the storage.now platform.', '', '⚡', 'private', 1710300000000, 1710600000000);

INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_012', 'sp_004', 'u/bob', 'editor', 1710300000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_013', 'sp_004', 'u/dave', 'editor', 1710300000000);

INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_010', 'sp_004', 'API Specifications', 'OpenAPI and YAML specs', 0, 1710300000000, 1710500000000);
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_011', 'sp_004', 'Infrastructure', 'Deployment configs and backups', 1, 1710300000000, 1710400000000);
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_012', 'sp_004', 'Models & ML', 'Model weights and training configs', 2, 1710400000000, 1710600000000);

INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_022', 'ss_010', 'sp_004', 'file', 'API Spec (YAML)', 'OpenAPI specification', 'o_d001000000000020', 0, 'u/bob', 1710300000000, 1710300000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_023', 'ss_010', 'sp_004', 'file', 'README', 'Project overview and setup guide', 'o_d001000000000021', 1, 'u/test', 1710300000000, 1710500000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, url, position, added_by, created_at, updated_at)
VALUES ('si_024', 'ss_010', 'sp_004', 'url', 'API Documentation (Live)', 'Hosted API docs', 'https://storage.liteio.dev/docs', 2, 'u/bob', 1710300000000, 1710300000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_025', 'ss_011', 'sp_004', 'file', 'DB Snapshot Mar 15', 'Latest database backup', 'o_d001000000000024', 0, 'u/dave', 1710432000000, 1710432000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_026', 'ss_011', 'sp_004', 'file', 'DB Snapshot Mar 10', 'Previous database backup', 'o_d001000000000023', 1, 'u/dave', 1710000000000, 1710000000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_027', 'ss_012', 'sp_004', 'file', 'Model Weights v2', 'Production model v2.3.1', 'o_d001000000000012', 0, 'u/test', 1710400000000, 1710500000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_028', 'ss_012', 'sp_004', 'file', 'Model Config', 'Training hyperparameters', 'o_d001000000000013', 1, 'u/test', 1710400000000, 1710400000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_029', 'ss_012', 'sp_004', 'file', 'Model Weights v1', 'Previous model version', 'o_d001000000000014', 2, 'u/test', 1710400000000, 1710400000000);

-- ═══════════════════════════════════════════
-- SPACE 5: Client Onboarding
-- ═══════════════════════════════════════════
INSERT INTO spaces (id, owner, title, description, cover_url, icon, visibility, created_at, updated_at)
VALUES ('sp_005', 'u/test', 'Client Onboarding', 'Welcome packets, contracts, and setup checklists for new client engagements.', '', '👋', 'team', 1710400000000, 1710600000000);

INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_014', 'sp_005', 'u/carol', 'editor', 1710400000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_015', 'sp_005', 'u/alice', 'viewer', 1710400000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_016', 'sp_005', 'a/organizer', 'viewer', 1710500000000);

INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_013', 'sp_005', 'Contracts & Legal', 'NDAs, service agreements, and legal templates', 0, 1710400000000, 1710500000000);
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_014', 'sp_005', 'Onboarding Checklists', 'Step-by-step guides for client setup', 1, 1710400000000, 1710600000000);

INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_030', 'ss_013', 'sp_005', 'file', 'NDA Template', 'Standard non-disclosure agreement', 'o_d001000000000003', 0, 'u/test', 1710400000000, 1710400000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, position, added_by, created_at, updated_at)
VALUES ('si_031', 'ss_013', 'sp_005', 'file', 'Service Agreement', 'Master service agreement template', 'o_d001000000000004', 1, 'u/test', 1710400000000, 1710400000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, note_body, position, added_by, created_at, updated_at)
VALUES ('si_032', 'ss_014', 'sp_005', 'note', 'New Client Checklist', 'Standard onboarding steps', '## New Client Onboarding\n\n1. [ ] Send NDA for signature\n2. [ ] Countersign service agreement\n3. [ ] Create shared folder in storage.now\n4. [ ] Add client contacts as viewers\n5. [ ] Schedule kickoff call\n6. [ ] Share brand assets Space\n7. [ ] Set up weekly sync cadence', 0, 'u/carol', 1710400000000, 1710600000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, url, position, added_by, created_at, updated_at)
VALUES ('si_033', 'ss_014', 'sp_005', 'url', 'CRM Client Board', 'Track onboarding status', 'https://crm.internal/boards/onboarding', 1, 'u/carol', 1710500000000, 1710500000000);

-- ═══════════════════════════════════════════
-- SPACE 6: Engineering Wiki (owned by alice, test is member)
-- ═══════════════════════════════════════════
INSERT INTO spaces (id, owner, title, description, cover_url, icon, visibility, created_at, updated_at)
VALUES ('sp_006', 'u/alice', 'Engineering Wiki', 'Architecture decisions, runbooks, and technical documentation maintained by the engineering team.', '', '🔧', 'team', 1710200000000, 1710600000000);

INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_017', 'sp_006', 'u/test', 'editor', 1710200000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_018', 'sp_006', 'u/bob', 'editor', 1710200000000);
INSERT INTO space_members (id, space_id, actor, role, created_at)
VALUES ('sm_019', 'sp_006', 'u/dave', 'admin', 1710200000000);

INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_015', 'sp_006', 'Architecture', 'System architecture docs and diagrams', 0, 1710200000000, 1710500000000);
INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
VALUES ('ss_016', 'sp_006', 'Runbooks', 'Operational runbooks and incident response', 1, 1710300000000, 1710400000000);

INSERT INTO space_items (id, section_id, space_id, item_type, title, description, note_body, position, added_by, created_at, updated_at)
VALUES ('si_034', 'ss_015', 'sp_006', 'note', 'System Architecture Overview', 'High-level architecture', '## Architecture\n\n### Stack\n- **Runtime**: Cloudflare Workers (Hono)\n- **Database**: D1 (SQLite)\n- **Storage**: R2 (S3-compatible)\n- **Auth**: Ed25519 + Magic Links + OAuth 2.0\n\n### Services\n1. storage.now — File storage and sharing\n2. chat.now — Messaging for humans and agents\n\n### Key Decisions\n- Server-side rendering (no client framework)\n- Actor model: u/ for humans, a/ for agents\n- REST-first API design', 0, 'u/alice', 1710200000000, 1710500000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, url, position, added_by, created_at, updated_at)
VALUES ('si_035', 'ss_015', 'sp_006', 'url', 'Cloudflare Dashboard', 'Workers and D1 management', 'https://dash.cloudflare.com', 1, 'u/dave', 1710300000000, 1710300000000);
INSERT INTO space_items (id, section_id, space_id, item_type, title, description, note_body, position, added_by, created_at, updated_at)
VALUES ('si_036', 'ss_016', 'sp_006', 'note', 'Incident Response Runbook', 'Steps for production incidents', '## Incident Response\n\n### P1 — Service Down\n1. Check Cloudflare status page\n2. Review wrangler tail logs\n3. Check D1 query latency in dashboard\n4. If R2: verify bucket health\n5. If auth: check sessions table for corruption\n6. Post update in #incidents Slack channel\n\n### P2 — Degraded Performance\n1. Check rate_limits table for spikes\n2. Review slow D1 queries\n3. Scale R2 multipart thresholds if upload-related', 0, 'u/dave', 1710300000000, 1710400000000);

-- ═══════════════════════════════════════════
-- ACTIVITY (across all spaces)
-- ═══════════════════════════════════════════
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_001', 'sp_001', 'u/test', 'created', 'Q1 Product Launch', 1710000000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_002', 'sp_001', 'u/alice', 'added_item', 'Q1 Metrics to Strategy & Planning', 1710100000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_003', 'sp_002', 'a/reports-bot', 'added_item', 'Analytics March Export to Raw Data', 1710200000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_004', 'sp_003', 'u/alice', 'edited', 'Brand Colors note', 1710300000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_005', 'sp_004', 'u/bob', 'shared', 'API & Developer Hub with u/dave', 1710300000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_006', 'sp_004', 'u/dave', 'added_item', 'DB Snapshot Mar 15 to Infrastructure', 1710432000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_007', 'sp_001', 'u/carol', 'commented', 'on Launch Checklist', 1710450000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_008', 'sp_003', 'a/organizer', 'organized', '4 files into Logos & Icons', 1710460000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_009', 'sp_005', 'u/carol', 'added_item', 'New Client Checklist note', 1710500000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_010', 'sp_006', 'u/dave', 'added_item', 'Incident Response Runbook', 1710500000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_011', 'sp_002', 'a/reports-bot', 'added_item', 'Q1 Metrics to Reports', 1710550000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_012', 'sp_001', 'u/test', 'edited', 'Project Proposal description', 1710580000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_013', 'sp_004', 'u/test', 'added_item', 'Model Weights v2 to Models & ML', 1710590000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_014', 'sp_003', 'u/alice', 'edited', 'Design System Figma description', 1710595000000);
INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
VALUES ('sa_015', 'sp_006', 'u/alice', 'edited', 'System Architecture Overview', 1710600000000);
