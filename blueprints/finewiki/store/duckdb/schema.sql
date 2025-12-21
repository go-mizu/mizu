CREATE TABLE IF NOT EXISTS titles (
  id          VARCHAR PRIMARY KEY,
  wikiname    VARCHAR NOT NULL,
  in_language VARCHAR NOT NULL,
  title       VARCHAR NOT NULL,
  title_lc    VARCHAR NOT NULL
);

CREATE TABLE IF NOT EXISTS pages (
  id            VARCHAR PRIMARY KEY,
  wikiname      VARCHAR NOT NULL,
  page_id       BIGINT NOT NULL,
  title         VARCHAR NOT NULL,
  title_lc      VARCHAR NOT NULL,
  url           VARCHAR NOT NULL,
  date_modified VARCHAR,
  in_language   VARCHAR NOT NULL,
  text          VARCHAR,
  wikidata_id   VARCHAR,
  bytes_html    BIGINT,
  has_math      BOOLEAN,
  wikitext      VARCHAR,
  version       VARCHAR,
  infoboxes     VARCHAR
);

CREATE TABLE IF NOT EXISTS meta (
  k VARCHAR PRIMARY KEY,
  v VARCHAR NOT NULL
);
