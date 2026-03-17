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
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/scout', 'agent', 'seed-key-scout', 1716000000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('u/test-1', 'human', 'seed-key-test1', 1717000000000);

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

-- Scout test conversations for u/test-1
INSERT INTO chats (id, kind, title, creator, visibility, created_at) VALUES ('c_test1_scout_01', 'direct', '', 'u/test-1', 'private', 1717100000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_test1_scout_01', 'u/test-1', 'member', 1717100000000);
INSERT INTO members (chat_id, actor, role, joined_at) VALUES ('c_test1_scout_01', 'a/scout', 'member', 1717100000000);

-- Conversation: table query
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_t1s_01', 'c_test1_scout_01', 'u/test-1', 'Premier League table', null, 1717100001000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_t1s_02', 'c_test1_scout_01', 'a/scout', '## 🏆 Premier League Table

| # | Team | P | W | D | L | GD | Pts |
|---|------|---|---|---|---|----|-----|
| 1 | **Liverpool** | 29 | 21 | 5 | 3 | +42 | **68** |
| 2 | Arsenal | 29 | 20 | 4 | 5 | +28 | 64 |
| 3 | Chelsea | 29 | 17 | 5 | 7 | +22 | 56 |
| 4 | Nottm Forest | 29 | 16 | 6 | 7 | +11 | 54 |
| 5 | Manchester City | 29 | 15 | 6 | 8 | +14 | 51 |', null, 1717100002000);

-- Conversation: fixture query
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_t1s_03', 'c_test1_scout_01', 'u/test-1', 'when is Arsenal next match?', null, 1717100003000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_t1s_04', 'c_test1_scout_01', 'a/scout', '## 📅 Upcoming Fixtures

| Date | Home | Away |
|------|------|------|
| Mar 22 | Arsenal | Chelsea |
| Mar 25 | Arsenal | Bayern Munich |', null, 1717100004000);

-- Conversation: team info query
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_t1s_05', 'c_test1_scout_01', 'u/test-1', 'tell me about Barcelona', null, 1717100005000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_t1s_06', 'c_test1_scout_01', 'a/scout', '## ⚽ Barcelona

| | |
|---|---|
| 🏟 Stadium | Estadi Olímpic Lluís Companys |
| 👔 Manager | Hansi Flick |
| 🏆 League  | La Liga |', null, 1717100006000);
