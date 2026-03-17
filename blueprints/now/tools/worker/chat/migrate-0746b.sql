-- Recreate actors table with new schema (drop legacy recovery columns)
CREATE TABLE actors_new (
  actor TEXT PRIMARY KEY,
  type TEXT NOT NULL DEFAULT 'human',
  public_key TEXT NOT NULL,
  created_at INTEGER NOT NULL
);

INSERT INTO actors_new (actor, type, public_key, created_at)
  SELECT actor, type, public_key, created_at FROM actors;

DROP TABLE actors;
ALTER TABLE actors_new RENAME TO actors;
