-- Humans
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('u/alice', 'human', 'seed-key-alice', 1710000000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('u/marcus', 'human', 'seed-key-marcus', 1711000000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('u/sarah', 'human', 'seed-key-sarah', 1712000000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('u/kenji', 'human', 'seed-key-kenji', 1713000000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('u/priya', 'human', 'seed-key-priya', 1714000000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('u/elena', 'human', 'seed-key-elena', 1715000000000);

-- Agents
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/deploy-bot', 'agent', 'seed-key-deploy', 1710500000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/review-bot', 'agent', 'seed-key-review', 1711500000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/monitor', 'agent', 'seed-key-monitor', 1712500000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/ci-runner', 'agent', 'seed-key-ci', 1713500000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/docs-helper', 'agent', 'seed-key-docs', 1714500000000);

-- Rooms
INSERT INTO chats (id, kind, title, creator, visibility, created_at) VALUES ('c_deploy_review_01', 'room', 'deploy-review', 'u/alice', 'public', 1712000000000);
INSERT INTO chats (id, kind, title, creator, visibility, created_at) VALUES ('c_engineering_01', 'room', 'engineering', 'u/marcus', 'public', 1713000000000);
INSERT INTO chats (id, kind, title, creator, visibility, created_at) VALUES ('c_design_01', 'room', 'design-feedback', 'u/sarah', 'public', 1714000000000);
INSERT INTO chats (id, kind, title, creator, visibility, created_at) VALUES ('c_incidents_01', 'room', 'incidents', 'u/kenji', 'public', 1714500000000);

-- Members for deploy-review
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_deploy_review_01', 'u/alice', 'admin', 1712000000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_deploy_review_01', 'u/marcus', 'member', 1712100000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_deploy_review_01', 'a/deploy-bot', 'member', 1712200000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_deploy_review_01', 'a/ci-runner', 'member', 1712300000000);

-- Members for engineering
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_engineering_01', 'u/marcus', 'admin', 1713000000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_engineering_01', 'u/alice', 'member', 1713100000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_engineering_01', 'u/kenji', 'member', 1713200000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_engineering_01', 'a/review-bot', 'member', 1713300000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_engineering_01', 'a/ci-runner', 'member', 1713400000000);

-- Members for design-feedback
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_design_01', 'u/sarah', 'admin', 1714000000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_design_01', 'u/elena', 'member', 1714100000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_design_01', 'u/priya', 'member', 1714200000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_design_01', 'a/docs-helper', 'member', 1714300000000);

-- Members for incidents
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_incidents_01', 'u/kenji', 'admin', 1714500000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_incidents_01', 'u/marcus', 'member', 1714600000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_incidents_01', 'a/monitor', 'member', 1714700000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_incidents_01', 'a/deploy-bot', 'member', 1714800000000);
