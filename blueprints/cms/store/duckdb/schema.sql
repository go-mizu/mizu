-- CMS Schema - WordPress-compatible Content Management System
-- Full compatibility with WordPress database structure

-- ============================================================================
-- USERS
-- ============================================================================

CREATE TABLE IF NOT EXISTS wp_users (
    ID VARCHAR PRIMARY KEY,
    user_login VARCHAR(60) NOT NULL,
    user_pass VARCHAR(255) NOT NULL,
    user_nicename VARCHAR(50) NOT NULL,
    user_email VARCHAR(100) NOT NULL,
    user_url VARCHAR(100) DEFAULT '',
    user_registered TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_activation_key VARCHAR(255) DEFAULT '',
    user_status INTEGER DEFAULT 0,
    display_name VARCHAR(250) DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_login ON wp_users(LOWER(user_login));
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_email ON wp_users(LOWER(user_email));
CREATE INDEX IF NOT EXISTS idx_user_nicename ON wp_users(user_nicename);

-- User Meta
CREATE TABLE IF NOT EXISTS wp_usermeta (
    umeta_id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    meta_key VARCHAR(255),
    meta_value TEXT
);

CREATE INDEX IF NOT EXISTS idx_usermeta_user_id ON wp_usermeta(user_id);
CREATE INDEX IF NOT EXISTS idx_usermeta_key ON wp_usermeta(meta_key);

-- Sessions (extended for token-based auth)
CREATE TABLE IF NOT EXISTS wp_sessions (
    session_id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    token VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    payload TEXT,
    last_activity TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_session_user ON wp_sessions(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_session_token ON wp_sessions(token);

-- Application Passwords (REST API auth)
CREATE TABLE IF NOT EXISTS wp_application_passwords (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL,
    uuid VARCHAR(36) NOT NULL,
    app_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used TIMESTAMP,
    last_ip VARCHAR(45)
);

CREATE INDEX IF NOT EXISTS idx_app_password_user ON wp_application_passwords(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_app_password_uuid ON wp_application_passwords(uuid);

-- ============================================================================
-- POSTS (posts, pages, attachments, revisions, custom post types)
-- ============================================================================

CREATE TABLE IF NOT EXISTS wp_posts (
    ID VARCHAR PRIMARY KEY,
    post_author VARCHAR NOT NULL DEFAULT '',
    post_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    post_date_gmt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    post_content TEXT NOT NULL DEFAULT '',
    post_title TEXT NOT NULL DEFAULT '',
    post_excerpt TEXT NOT NULL DEFAULT '',
    post_status VARCHAR(20) NOT NULL DEFAULT 'publish',
    comment_status VARCHAR(20) NOT NULL DEFAULT 'open',
    ping_status VARCHAR(20) NOT NULL DEFAULT 'open',
    post_password VARCHAR(255) NOT NULL DEFAULT '',
    post_name VARCHAR(200) NOT NULL DEFAULT '',
    to_ping TEXT NOT NULL DEFAULT '',
    pinged TEXT NOT NULL DEFAULT '',
    post_modified TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    post_modified_gmt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    post_content_filtered TEXT NOT NULL DEFAULT '',
    post_parent VARCHAR NOT NULL DEFAULT '',
    guid VARCHAR(255) NOT NULL DEFAULT '',
    menu_order INTEGER NOT NULL DEFAULT 0,
    post_type VARCHAR(20) NOT NULL DEFAULT 'post',
    post_mime_type VARCHAR(100) NOT NULL DEFAULT '',
    comment_count BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_post_name ON wp_posts(post_name);
CREATE INDEX IF NOT EXISTS idx_post_type_status_date ON wp_posts(post_type, post_status, post_date, ID);
CREATE INDEX IF NOT EXISTS idx_post_parent ON wp_posts(post_parent);
CREATE INDEX IF NOT EXISTS idx_post_author ON wp_posts(post_author);
CREATE INDEX IF NOT EXISTS idx_post_date ON wp_posts(post_date DESC);
CREATE INDEX IF NOT EXISTS idx_post_modified ON wp_posts(post_modified DESC);

-- Post Meta
CREATE TABLE IF NOT EXISTS wp_postmeta (
    meta_id VARCHAR PRIMARY KEY,
    post_id VARCHAR NOT NULL DEFAULT '',
    meta_key VARCHAR(255),
    meta_value TEXT
);

CREATE INDEX IF NOT EXISTS idx_postmeta_post_id ON wp_postmeta(post_id);
CREATE INDEX IF NOT EXISTS idx_postmeta_key ON wp_postmeta(meta_key);

-- ============================================================================
-- TERMS (categories, tags, custom taxonomies)
-- ============================================================================

CREATE TABLE IF NOT EXISTS wp_terms (
    term_id VARCHAR PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(200) NOT NULL,
    term_group BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_term_slug ON wp_terms(slug);
CREATE INDEX IF NOT EXISTS idx_term_name ON wp_terms(name);

-- Term Taxonomy (relates terms to taxonomies)
CREATE TABLE IF NOT EXISTS wp_term_taxonomy (
    term_taxonomy_id VARCHAR PRIMARY KEY,
    term_id VARCHAR NOT NULL DEFAULT '',
    taxonomy VARCHAR(32) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    parent VARCHAR NOT NULL DEFAULT '',
    count BIGINT NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_term_id_taxonomy ON wp_term_taxonomy(term_id, taxonomy);
CREATE INDEX IF NOT EXISTS idx_taxonomy ON wp_term_taxonomy(taxonomy);

-- Term Relationships (posts to terms)
CREATE TABLE IF NOT EXISTS wp_term_relationships (
    object_id VARCHAR NOT NULL DEFAULT '',
    term_taxonomy_id VARCHAR NOT NULL DEFAULT '',
    term_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (object_id, term_taxonomy_id)
);

CREATE INDEX IF NOT EXISTS idx_term_taxonomy_id ON wp_term_relationships(term_taxonomy_id);

-- Term Meta
CREATE TABLE IF NOT EXISTS wp_termmeta (
    meta_id VARCHAR PRIMARY KEY,
    term_id VARCHAR NOT NULL DEFAULT '',
    meta_key VARCHAR(255),
    meta_value TEXT
);

CREATE INDEX IF NOT EXISTS idx_termmeta_term_id ON wp_termmeta(term_id);
CREATE INDEX IF NOT EXISTS idx_termmeta_key ON wp_termmeta(meta_key);

-- ============================================================================
-- COMMENTS
-- ============================================================================

CREATE TABLE IF NOT EXISTS wp_comments (
    comment_ID VARCHAR PRIMARY KEY,
    comment_post_ID VARCHAR NOT NULL DEFAULT '',
    comment_author TEXT NOT NULL,
    comment_author_email VARCHAR(100) NOT NULL DEFAULT '',
    comment_author_url VARCHAR(200) NOT NULL DEFAULT '',
    comment_author_IP VARCHAR(100) NOT NULL DEFAULT '',
    comment_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    comment_date_gmt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    comment_content TEXT NOT NULL,
    comment_karma INTEGER NOT NULL DEFAULT 0,
    comment_approved VARCHAR(20) NOT NULL DEFAULT '1',
    comment_agent VARCHAR(255) NOT NULL DEFAULT '',
    comment_type VARCHAR(20) NOT NULL DEFAULT 'comment',
    comment_parent VARCHAR NOT NULL DEFAULT '',
    user_id VARCHAR NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_comment_post_ID ON wp_comments(comment_post_ID);
CREATE INDEX IF NOT EXISTS idx_comment_approved_date ON wp_comments(comment_approved, comment_date_gmt);
CREATE INDEX IF NOT EXISTS idx_comment_parent ON wp_comments(comment_parent);
CREATE INDEX IF NOT EXISTS idx_comment_author_email ON wp_comments(comment_author_email);
CREATE INDEX IF NOT EXISTS idx_comment_date ON wp_comments(comment_date DESC);

-- Comment Meta
CREATE TABLE IF NOT EXISTS wp_commentmeta (
    meta_id VARCHAR PRIMARY KEY,
    comment_id VARCHAR NOT NULL DEFAULT '',
    meta_key VARCHAR(255),
    meta_value TEXT
);

CREATE INDEX IF NOT EXISTS idx_commentmeta_comment_id ON wp_commentmeta(comment_id);
CREATE INDEX IF NOT EXISTS idx_commentmeta_key ON wp_commentmeta(meta_key);

-- ============================================================================
-- OPTIONS (site settings)
-- ============================================================================

CREATE TABLE IF NOT EXISTS wp_options (
    option_id VARCHAR PRIMARY KEY,
    option_name VARCHAR(191) NOT NULL,
    option_value TEXT NOT NULL,
    autoload VARCHAR(20) NOT NULL DEFAULT 'yes'
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_option_name ON wp_options(option_name);
CREATE INDEX IF NOT EXISTS idx_autoload ON wp_options(autoload);

-- ============================================================================
-- LINKS (blogroll - deprecated but supported)
-- ============================================================================

CREATE TABLE IF NOT EXISTS wp_links (
    link_id VARCHAR PRIMARY KEY,
    link_url VARCHAR(255) NOT NULL DEFAULT '',
    link_name VARCHAR(255) NOT NULL DEFAULT '',
    link_image VARCHAR(255) NOT NULL DEFAULT '',
    link_target VARCHAR(25) NOT NULL DEFAULT '',
    link_description VARCHAR(255) NOT NULL DEFAULT '',
    link_visible VARCHAR(20) NOT NULL DEFAULT 'Y',
    link_owner VARCHAR NOT NULL DEFAULT '',
    link_rating INTEGER NOT NULL DEFAULT 0,
    link_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    link_rel VARCHAR(255) NOT NULL DEFAULT '',
    link_notes TEXT NOT NULL DEFAULT '',
    link_rss VARCHAR(255) NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_link_visible ON wp_links(link_visible);

-- ============================================================================
-- MENUS (navigation menus)
-- Note: Menus are stored as terms with taxonomy='nav_menu'
-- Menu items are stored as posts with post_type='nav_menu_item'
-- ============================================================================

-- ============================================================================
-- WIDGETS
-- Note: Widgets are stored in options as serialized data
-- ============================================================================

-- ============================================================================
-- REVISIONS
-- Note: Revisions are stored as posts with post_type='revision'
-- ============================================================================

-- ============================================================================
-- CUSTOM TABLES (for this implementation)
-- ============================================================================

-- Nonces (for CSRF protection)
CREATE TABLE IF NOT EXISTS wp_nonces (
    nonce_id VARCHAR PRIMARY KEY,
    user_id VARCHAR,
    action VARCHAR(255) NOT NULL,
    nonce VARCHAR(64) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_nonce ON wp_nonces(nonce);
CREATE INDEX IF NOT EXISTS idx_nonce_expires ON wp_nonces(expires_at);

-- Cron jobs (WP-Cron compatible)
CREATE TABLE IF NOT EXISTS wp_cron (
    cron_id VARCHAR PRIMARY KEY,
    hook VARCHAR(255) NOT NULL,
    args TEXT,
    schedule VARCHAR(255),
    interval_seconds INTEGER,
    next_run TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cron_next_run ON wp_cron(next_run);
CREATE INDEX IF NOT EXISTS idx_cron_hook ON wp_cron(hook);

-- Transients (temporary cached data)
CREATE TABLE IF NOT EXISTS wp_transients (
    transient_id VARCHAR PRIMARY KEY,
    transient_key VARCHAR(191) NOT NULL,
    transient_value TEXT,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_transient_key ON wp_transients(transient_key);
CREATE INDEX IF NOT EXISTS idx_transient_expires ON wp_transients(expires_at);

-- ============================================================================
-- MULTISITE TABLES (optional, for network installations)
-- ============================================================================

-- Sites (blogs in multisite)
CREATE TABLE IF NOT EXISTS wp_blogs (
    blog_id VARCHAR PRIMARY KEY,
    site_id VARCHAR NOT NULL DEFAULT '',
    domain VARCHAR(200) NOT NULL DEFAULT '',
    path VARCHAR(100) NOT NULL DEFAULT '/',
    registered TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    public INTEGER NOT NULL DEFAULT 1,
    archived INTEGER NOT NULL DEFAULT 0,
    mature INTEGER NOT NULL DEFAULT 0,
    spam INTEGER NOT NULL DEFAULT 0,
    deleted INTEGER NOT NULL DEFAULT 0,
    lang_id INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_blog_domain_path ON wp_blogs(domain, path);

-- Network sites
CREATE TABLE IF NOT EXISTS wp_site (
    id VARCHAR PRIMARY KEY,
    domain VARCHAR(200) NOT NULL DEFAULT '',
    path VARCHAR(100) NOT NULL DEFAULT '/'
);

CREATE INDEX IF NOT EXISTS idx_site_domain ON wp_site(domain, path);

-- Site Meta
CREATE TABLE IF NOT EXISTS wp_sitemeta (
    meta_id VARCHAR PRIMARY KEY,
    site_id VARCHAR NOT NULL DEFAULT '',
    meta_key VARCHAR(255),
    meta_value TEXT
);

CREATE INDEX IF NOT EXISTS idx_sitemeta_site_id ON wp_sitemeta(site_id);
CREATE INDEX IF NOT EXISTS idx_sitemeta_key ON wp_sitemeta(meta_key);

-- Seed mappings (for idempotent external data seeding)
CREATE TABLE IF NOT EXISTS seed_mappings (
    source VARCHAR NOT NULL,
    entity_type VARCHAR NOT NULL,
    external_id VARCHAR NOT NULL,
    local_id VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (source, entity_type, external_id)
);

CREATE INDEX IF NOT EXISTS idx_seed_mappings_local ON seed_mappings(local_id);
