-- 0755: Spaces — collaborative content workspaces
-- Spaces, sections, items, members, activity

CREATE TABLE IF NOT EXISTS spaces (
  id          TEXT PRIMARY KEY,
  owner       TEXT NOT NULL,
  title       TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  cover_url   TEXT NOT NULL DEFAULT '',
  icon        TEXT NOT NULL DEFAULT '',
  visibility  TEXT NOT NULL DEFAULT 'private' CHECK(visibility IN ('private','team','public')),
  created_at  INTEGER NOT NULL,
  updated_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_spaces_owner ON spaces(owner);

CREATE TABLE IF NOT EXISTS space_members (
  id         TEXT PRIMARY KEY,
  space_id   TEXT NOT NULL,
  actor      TEXT NOT NULL,
  role       TEXT NOT NULL DEFAULT 'viewer' CHECK(role IN ('viewer','editor','admin')),
  created_at INTEGER NOT NULL,
  FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
  UNIQUE(space_id, actor)
);
CREATE INDEX IF NOT EXISTS idx_space_members_actor ON space_members(actor);

CREATE TABLE IF NOT EXISTS space_sections (
  id          TEXT PRIMARY KEY,
  space_id    TEXT NOT NULL,
  title       TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  position    INTEGER NOT NULL DEFAULT 0,
  created_at  INTEGER NOT NULL,
  updated_at  INTEGER NOT NULL,
  FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_space_sections_space ON space_sections(space_id);

CREATE TABLE IF NOT EXISTS space_items (
  id          TEXT PRIMARY KEY,
  section_id  TEXT NOT NULL,
  space_id    TEXT NOT NULL,
  item_type   TEXT NOT NULL CHECK(item_type IN ('file','url','note')),
  title       TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  object_id   TEXT,
  url         TEXT,
  note_body   TEXT,
  position    INTEGER NOT NULL DEFAULT 0,
  added_by    TEXT NOT NULL,
  created_at  INTEGER NOT NULL,
  updated_at  INTEGER NOT NULL,
  FOREIGN KEY (section_id) REFERENCES space_sections(id) ON DELETE CASCADE,
  FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_space_items_section ON space_items(section_id);
CREATE INDEX IF NOT EXISTS idx_space_items_space ON space_items(space_id);

CREATE TABLE IF NOT EXISTS space_activity (
  id         TEXT PRIMARY KEY,
  space_id   TEXT NOT NULL,
  actor      TEXT NOT NULL,
  action     TEXT NOT NULL,
  target     TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_space_activity_space ON space_activity(space_id, created_at);
CREATE INDEX IF NOT EXISTS idx_space_activity_actor ON space_activity(actor, created_at);
