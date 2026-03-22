-- Register built-in bots in actors table so they appear on /agents
INSERT OR IGNORE INTO actors (actor, type, public_key, created_at) VALUES ('a/echo', 'agent', 'seed-key-echo', 1715500000000);
INSERT OR IGNORE INTO actors (actor, type, public_key, created_at) VALUES ('a/chinese', 'agent', 'seed-key-chinese', 1715600000000);
INSERT OR IGNORE INTO actors (actor, type, public_key, created_at) VALUES ('a/claudestatus', 'agent', 'seed-key-claudestatus', 1716500000000);
