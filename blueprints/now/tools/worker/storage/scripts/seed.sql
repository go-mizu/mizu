-- Seed realistic test data for storage.now file browser
-- Owner: u/test (test@example.com)
-- Also creates agents a/inference and actors u/alice, u/bob for shares

-- Ensure share target actors exist
INSERT OR IGNORE INTO actors (actor, type, email, created_at)
VALUES ('a/inference', 'agent', NULL, 1710700000000);
INSERT OR IGNORE INTO actors (actor, type, email, created_at)
VALUES ('u/alice', 'human', 'alice@example.com', 1710700000000);

-- Clean existing test objects for u/test to avoid duplicates
DELETE FROM shares WHERE owner = 'u/test';
DELETE FROM objects WHERE owner = 'u/test';

-- ═══════════════════════════════════════════
-- FOLDERS (14 folders)
-- ═══════════════════════════════════════════

INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_f001000000000001', 'u/test', 'documents/', 'documents', 1, '', 0, '', 0, 1709200000000, 1709200000000);

INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_f001000000000002', 'u/test', 'documents/contracts/', 'contracts', 1, '', 0, '', 0, 1709200000000, 1709200000000);

INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_f001000000000003', 'u/test', 'documents/reports/', 'reports', 1, '', 0, '', 0, 1709300000000, 1709300000000);

INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_f001000000000004', 'u/test', 'images/', 'images', 1, '', 0, '', 0, 1709400000000, 1709400000000);

INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_f001000000000005', 'u/test', 'images/screenshots/', 'screenshots', 1, '', 0, '', 0, 1709400000000, 1709400000000);

INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_f001000000000006', 'u/test', 'images/photos/', 'photos', 1, '', 0, '', 0, 1709400000000, 1709400000000);

INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_f001000000000007', 'u/test', 'models/', 'models', 1, '', 0, '', 0, 1709500000000, 1709500000000);

INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_f001000000000008', 'u/test', 'models/v1/', 'v1', 1, '', 0, '', 0, 1709500000000, 1709500000000);

INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_f001000000000009', 'u/test', 'models/v2/', 'v2', 1, '', 0, '', 0, 1709600000000, 1709600000000);

INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_f001000000000010', 'u/test', 'data/', 'data', 1, '', 0, '', 0, 1709700000000, 1709700000000);

INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_f001000000000011', 'u/test', 'data/exports/', 'exports', 1, '', 0, '', 0, 1709700000000, 1709700000000);

INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_f001000000000012', 'u/test', 'archive/', 'archive', 1, '', 0, '', 0, 1709800000000, 1709800000000);

INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_f001000000000013', 'u/test', 'shared-projects/', 'shared-projects', 1, '', 0, '', 0, 1709900000000, 1709900000000);

INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_f001000000000014', 'u/test', 'backups/', 'backups', 1, '', 0, '', 0, 1710000000000, 1710000000000);

-- ═══════════════════════════════════════════
-- FILES (22 files with realistic sizes)
-- ═══════════════════════════════════════════

-- documents/project-proposal.pdf (2.4 MB) — starred
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, description, created_at, updated_at)
VALUES ('o_d001000000000001', 'u/test', 'documents/project-proposal.pdf', 'project-proposal.pdf', 0, 'application/pdf', 2516582, 'u/test/documents/project-proposal.pdf', 1, 1710500000000, 'Q2 product roadmap and resource allocation plan', 1709200000000, 1710100000000);

-- documents/meeting-notes-march.md (12 KB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000002', 'u/test', 'documents/meeting-notes-march.md', 'meeting-notes-march.md', 0, 'text/markdown', 12288, 'u/test/documents/meeting-notes-march.md', 0, 1710400000000, 1710000000000, 1710300000000);

-- documents/contracts/nda-2024.pdf (890 KB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000003', 'u/test', 'documents/contracts/nda-2024.pdf', 'nda-2024.pdf', 0, 'application/pdf', 911360, 'u/test/documents/contracts/nda-2024.pdf', 0, 1710200000000, 1709300000000, 1709300000000);

-- documents/contracts/service-agreement.pdf (1.2 MB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000004', 'u/test', 'documents/contracts/service-agreement.pdf', 'service-agreement.pdf', 0, 'application/pdf', 1258291, 'u/test/documents/contracts/service-agreement.pdf', 0, 1710100000000, 1709400000000, 1709400000000);

-- documents/reports/q1-metrics.csv (45 KB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000005', 'u/test', 'documents/reports/q1-metrics.csv', 'q1-metrics.csv', 0, 'text/csv', 46080, 'u/test/documents/reports/q1-metrics.csv', 0, 1710600000000, 1710000000000, 1710000000000);

-- documents/reports/annual-review.pdf (3.8 MB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, description, created_at, updated_at)
VALUES ('o_d001000000000006', 'u/test', 'documents/reports/annual-review.pdf', 'annual-review.pdf', 0, 'application/pdf', 3984588, 'u/test/documents/reports/annual-review.pdf', 0, 1710300000000, '2024 annual performance and financial review', 1709500000000, 1710200000000);

-- images/logo.svg (4 KB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000007', 'u/test', 'images/logo.svg', 'logo.svg', 0, 'image/svg+xml', 4096, 'u/test/images/logo.svg', 0, 1710500000000, 1709400000000, 1709400000000);

-- images/banner.png (340 KB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000008', 'u/test', 'images/banner.png', 'banner.png', 0, 'image/png', 348160, 'u/test/images/banner.png', 0, 1710400000000, 1709500000000, 1710100000000);

-- images/screenshots/dashboard-v2.png (1.8 MB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000009', 'u/test', 'images/screenshots/dashboard-v2.png', 'dashboard-v2.png', 0, 'image/png', 1887436, 'u/test/images/screenshots/dashboard-v2.png', 0, 1710500000000, 1710000000000, 1710000000000);

-- images/screenshots/mobile-app.png (920 KB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000010', 'u/test', 'images/screenshots/mobile-app.png', 'mobile-app.png', 0, 'image/png', 942080, 'u/test/images/screenshots/mobile-app.png', 0, 1710300000000, 1709800000000, 1709800000000);

-- images/photos/team-offsite.jpg (4.2 MB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, description, created_at, updated_at)
VALUES ('o_d001000000000011', 'u/test', 'images/photos/team-offsite.jpg', 'team-offsite.jpg', 0, 'image/jpeg', 4404019, 'u/test/images/photos/team-offsite.jpg', 0, 1710200000000, 'Team photo from Q1 offsite in Portland', 1709900000000, 1709900000000);

-- models/v2/weights.bin (47 MB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, description, created_at, updated_at)
VALUES ('o_d001000000000012', 'u/test', 'models/v2/weights.bin', 'weights.bin', 0, 'application/octet-stream', 49283072, 'u/test/models/v2/weights.bin', 0, 1710600000000, 'Production model weights — v2.3.1', 1709600000000, 1710500000000);

-- models/v2/config.json (2 KB) — starred
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000013', 'u/test', 'models/v2/config.json', 'config.json', 0, 'application/json', 2048, 'u/test/models/v2/config.json', 1, 1710600000000, 1709600000000, 1710400000000);

-- models/v1/weights.bin (38 MB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000014', 'u/test', 'models/v1/weights.bin', 'weights.bin', 0, 'application/octet-stream', 39845888, 'u/test/models/v1/weights.bin', 0, 1709800000000, 1709500000000, 1709500000000);

-- data/users-export.json (156 KB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000015', 'u/test', 'data/users-export.json', 'users-export.json', 0, 'application/json', 159744, 'u/test/data/users-export.json', 0, 1710400000000, 1709700000000, 1710300000000);

-- data/exports/analytics-march.csv (89 KB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000016', 'u/test', 'data/exports/analytics-march.csv', 'analytics-march.csv', 0, 'text/csv', 91136, 'u/test/data/exports/analytics-march.csv', 0, 1710500000000, 1710100000000, 1710100000000);

-- data/exports/logs-2024-03.jsonl (12 MB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000017', 'u/test', 'data/exports/logs-2024-03.jsonl', 'logs-2024-03.jsonl', 0, 'application/jsonl', 12582912, 'u/test/data/exports/logs-2024-03.jsonl', 0, 1710200000000, 1710100000000, 1710100000000);

-- archive/old-website.zip (23 MB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, description, created_at, updated_at)
VALUES ('o_d001000000000018', 'u/test', 'archive/old-website.zip', 'old-website.zip', 0, 'application/zip', 24117248, 'u/test/archive/old-website.zip', 0, 1709900000000, 'Legacy marketing site backup — pre-rebrand', 1709800000000, 1709800000000);

-- shared-projects/design-system.fig (8.5 MB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000019', 'u/test', 'shared-projects/design-system.fig', 'design-system.fig', 0, 'application/octet-stream', 8912896, 'u/test/shared-projects/design-system.fig', 0, 1710500000000, 1709900000000, 1710400000000);

-- shared-projects/api-spec.yaml (34 KB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000020', 'u/test', 'shared-projects/api-spec.yaml', 'api-spec.yaml', 0, 'application/yaml', 34816, 'u/test/shared-projects/api-spec.yaml', 0, 1710400000000, 1709900000000, 1710300000000);

-- README.md (3 KB) — starred
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, description, created_at, updated_at)
VALUES ('o_d001000000000021', 'u/test', 'README.md', 'README.md', 0, 'text/markdown', 3072, 'u/test/README.md', 1, 1710600000000, 'Project overview and getting started guide', 1709200000000, 1710500000000);

-- .env.example (512 B)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, created_at, updated_at)
VALUES ('o_d001000000000022', 'u/test', '.env.example', '.env.example', 0, 'text/plain', 512, 'u/test/.env.example', 0, 1709200000000, 1709200000000);

-- backups/db-snapshot-2024-03-10.sql.gz (5.6 MB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000023', 'u/test', 'backups/db-snapshot-2024-03-10.sql.gz', 'db-snapshot-2024-03-10.sql.gz', 0, 'application/gzip', 5872025, 'u/test/backups/db-snapshot-2024-03-10.sql.gz', 0, 1710000000000, 1710000000000, 1710000000000);

-- backups/db-snapshot-2024-03-15.sql.gz (5.9 MB)
INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, accessed_at, created_at, updated_at)
VALUES ('o_d001000000000024', 'u/test', 'backups/db-snapshot-2024-03-15.sql.gz', 'db-snapshot-2024-03-15.sql.gz', 0, 'application/gzip', 6185574, 'u/test/backups/db-snapshot-2024-03-15.sql.gz', 0, 1710500000000, 1710432000000, 1710432000000);

-- ═══════════════════════════════════════════
-- SHARES (4 cross-actor shares)
-- ═══════════════════════════════════════════

-- a/inference has read access to models/v2/weights.bin
INSERT INTO shares (id, object_id, owner, grantee, permission, created_at)
VALUES ('sh_s001000000000001', 'o_d001000000000012', 'u/test', 'a/inference', 'read', 1710000000000);

-- a/inference has read access to models/v2/config.json
INSERT INTO shares (id, object_id, owner, grantee, permission, created_at)
VALUES ('sh_s001000000000002', 'o_d001000000000013', 'u/test', 'a/inference', 'read', 1710000000000);

-- u/alice has read access to documents/reports/q1-metrics.csv
INSERT INTO shares (id, object_id, owner, grantee, permission, created_at)
VALUES ('sh_s001000000000003', 'o_d001000000000005', 'u/test', 'u/alice', 'read', 1710200000000);

-- u/bob has write access to shared-projects/ folder
INSERT INTO shares (id, object_id, owner, grantee, permission, created_at)
VALUES ('sh_s001000000000004', 'o_f001000000000013', 'u/test', 'u/bob', 'write', 1710300000000);
