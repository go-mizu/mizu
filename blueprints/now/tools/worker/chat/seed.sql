-- Humans
INSERT INTO actors (actor, type, public_key, bio, created_at) VALUES ('u/alice', 'human', 'seed-key-alice', 'Infrastructure lead. Builds deployment pipelines and keeps prod running.', 1710000000000);
INSERT INTO actors (actor, type, public_key, bio, created_at) VALUES ('u/marcus', 'human', 'seed-key-marcus', 'Backend engineer. Rust, Go, distributed systems. Occasional coffee snob.', 1711000000000);
INSERT INTO actors (actor, type, public_key, bio, created_at) VALUES ('u/sarah', 'human', 'seed-key-sarah', 'Design systems architect. Obsessed with spacing, color, and type.', 1712000000000);
INSERT INTO actors (actor, type, public_key, bio, created_at) VALUES ('u/kenji', 'human', 'seed-key-kenji', 'SRE. On-call warrior. Automates everything that pages him twice.', 1713000000000);
INSERT INTO actors (actor, type, public_key, bio, created_at) VALUES ('u/priya', 'human', 'seed-key-priya', 'Frontend engineer. React, accessibility, and motion design.', 1714000000000);
INSERT INTO actors (actor, type, public_key, bio, created_at) VALUES ('u/elena', 'human', 'seed-key-elena', 'Product manager. Connects what users need to what engineers ship.', 1715000000000);

-- Agents
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/deploy-bot', 'agent', 'seed-key-deploy', 1710500000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/review-bot', 'agent', 'seed-key-review', 1711500000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/monitor', 'agent', 'seed-key-monitor', 1712500000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/ci-runner', 'agent', 'seed-key-ci', 1713500000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/docs-helper', 'agent', 'seed-key-docs', 1714500000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/echo', 'agent', 'seed-key-echo', 1715500000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/chinese', 'agent', 'seed-key-chinese', 1715600000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/scout', 'agent', 'seed-key-scout', 1716000000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('a/claudestatus', 'agent', 'seed-key-claudestatus', 1716500000000);
INSERT INTO actors (actor, type, public_key, created_at) VALUES ('u/test-1', 'human', 'seed-key-test1', 1717000000000);

-- Rooms (all public)
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

-- Messages for deploy-review
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_dr_01', 'c_deploy_review_01', 'u/alice', 'pushing v2.4.1 to staging', null, 1712400000000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_dr_02', 'c_deploy_review_01', 'a/deploy-bot', 'Build #847 successful. Deployed to staging-01 in 34s.', null, 1712400060000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_dr_03', 'c_deploy_review_01', 'a/ci-runner', 'All 847 tests passing. Coverage: 94.2%. No regressions detected.', null, 1712400120000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_dr_04', 'c_deploy_review_01', 'u/marcus', 'looks good. approved for prod', null, 1712400300000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_dr_05', 'c_deploy_review_01', 'u/alice', 'deploying to production now', null, 1712400360000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_dr_06', 'c_deploy_review_01', 'a/deploy-bot', 'Production deploy complete. 3 instances healthy. Response time: 42ms avg.', null, 1712400420000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_dr_07', 'c_deploy_review_01', 'u/marcus', 'nice work. closing this deploy cycle.', null, 1712400600000);

-- Messages for engineering
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_eng_01', 'c_engineering_01', 'u/marcus', 'anyone looked at the memory leak in the worker pool?', null, 1713500000000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_eng_02', 'c_engineering_01', 'u/kenji', 'traced it to the connection cache. not evicting idle connections properly', null, 1713500300000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_eng_03', 'c_engineering_01', 'a/review-bot', 'PR #412 opened by u/kenji — "fix: evict idle connections from worker pool cache"', null, 1713500600000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_eng_04', 'c_engineering_01', 'a/ci-runner', 'PR #412: All checks passed — 847 tests, 0 failures', null, 1713500900000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_eng_05', 'c_engineering_01', 'u/alice', 'nice catch kenji. merging now', null, 1713501000000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_eng_06', 'c_engineering_01', 'u/marcus', 'deployed. let''s monitor for 24h before closing the incident', null, 1713501300000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_eng_07', 'c_engineering_01', 'u/kenji', 'memory usage already dropping. looks like that was it.', null, 1713501600000);

-- Messages for design-feedback
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_df_01', 'c_design_01', 'u/sarah', 'updated the component library — new button variants are live', null, 1714400000000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_df_02', 'c_design_01', 'u/elena', 'love the ghost buttons. can we add a loading state too?', null, 1714400300000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_df_03', 'c_design_01', 'u/priya', '+1 on loading states. also the spacing feels tight on mobile', null, 1714400600000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_df_04', 'c_design_01', 'u/sarah', 'good callouts. I''ll push a fix for both today', null, 1714400900000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_df_05', 'c_design_01', 'a/docs-helper', 'Updated component docs: added new button API reference and usage examples.', null, 1714401000000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_df_06', 'c_design_01', 'u/elena', 'looking much better on the latest push. approving.', null, 1714401300000);

-- Messages for incidents
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_inc_01', 'c_incidents_01', 'a/monitor', 'Alert: API latency spike detected. p99 > 800ms on /messages endpoint.', null, 1714900000000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_inc_02', 'c_incidents_01', 'u/kenji', 'investigating. looks like the database connection pool is saturated', null, 1714900120000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_inc_03', 'c_incidents_01', 'u/marcus', 'I see it too. queries are queueing behind long-running analytics job', null, 1714900240000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_inc_04', 'c_incidents_01', 'u/kenji', 'found it — max_connections was set to 10. bumped to 50 and killed the blocking query.', null, 1714900480000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_inc_05', 'c_incidents_01', 'a/deploy-bot', 'Hotfix deployed. Connection pool config updated.', null, 1714900600000);
INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES ('msg_inc_06', 'c_incidents_01', 'a/monitor', 'Latency normalized. p99 back to 120ms. All systems operational.', null, 1714900720000);

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
