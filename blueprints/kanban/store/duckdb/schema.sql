-- Kanban Schema
-- A full-featured project management database schema

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR PRIMARY KEY,
    email VARCHAR UNIQUE NOT NULL,
    username VARCHAR UNIQUE NOT NULL,
    display_name VARCHAR NOT NULL,
    password_hash VARCHAR NOT NULL,
    avatar_url VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- Workspaces table
CREATE TABLE IF NOT EXISTS workspaces (
    id VARCHAR PRIMARY KEY,
    slug VARCHAR UNIQUE NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    avatar_url VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workspaces_slug ON workspaces(slug);

-- Workspace members table
CREATE TABLE IF NOT EXISTS workspace_members (
    id VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR NOT NULL DEFAULT 'member',
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workspace_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_workspace_members_workspace_id ON workspace_members(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_members_user_id ON workspace_members(user_id);

-- Projects table
CREATE TABLE IF NOT EXISTS projects (
    id VARCHAR PRIMARY KEY,
    workspace_id VARCHAR NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR,
    color VARCHAR DEFAULT '#6366f1',
    lead_id VARCHAR REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR DEFAULT 'active',
    issue_counter INTEGER DEFAULT 0,
    start_date DATE,
    target_date DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workspace_id, key)
);

CREATE INDEX IF NOT EXISTS idx_projects_workspace_id ON projects(workspace_id);
CREATE INDEX IF NOT EXISTS idx_projects_key ON projects(key);

-- Labels table
CREATE TABLE IF NOT EXISTS labels (
    id VARCHAR PRIMARY KEY,
    project_id VARCHAR NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    color VARCHAR NOT NULL,
    description VARCHAR,
    UNIQUE(project_id, name)
);

CREATE INDEX IF NOT EXISTS idx_labels_project_id ON labels(project_id);

-- Sprints table
CREATE TABLE IF NOT EXISTS sprints (
    id VARCHAR PRIMARY KEY,
    project_id VARCHAR NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    goal VARCHAR,
    status VARCHAR DEFAULT 'planning',
    start_date DATE,
    end_date DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sprints_project_id ON sprints(project_id);
CREATE INDEX IF NOT EXISTS idx_sprints_status ON sprints(status);

-- Issues table
CREATE TABLE IF NOT EXISTS issues (
    id VARCHAR PRIMARY KEY,
    project_id VARCHAR NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    number INTEGER NOT NULL,
    key VARCHAR NOT NULL,
    title VARCHAR NOT NULL,
    description VARCHAR,
    type VARCHAR DEFAULT 'task',
    status VARCHAR DEFAULT 'backlog',
    priority VARCHAR DEFAULT 'none',
    parent_id VARCHAR REFERENCES issues(id) ON DELETE SET NULL,
    creator_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sprint_id VARCHAR REFERENCES sprints(id) ON DELETE SET NULL,
    due_date DATE,
    estimate INTEGER,
    position INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, number)
);

CREATE INDEX IF NOT EXISTS idx_issues_project_id ON issues(project_id);
CREATE INDEX IF NOT EXISTS idx_issues_key ON issues(key);
CREATE INDEX IF NOT EXISTS idx_issues_status ON issues(status);
CREATE INDEX IF NOT EXISTS idx_issues_sprint_id ON issues(sprint_id);
CREATE INDEX IF NOT EXISTS idx_issues_parent_id ON issues(parent_id);
CREATE INDEX IF NOT EXISTS idx_issues_creator_id ON issues(creator_id);

-- Issue assignees junction table
CREATE TABLE IF NOT EXISTS issue_assignees (
    issue_id VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY(issue_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_issue_assignees_user_id ON issue_assignees(user_id);

-- Issue labels junction table
CREATE TABLE IF NOT EXISTS issue_labels (
    issue_id VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    label_id VARCHAR NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY(issue_id, label_id)
);

-- Issue links table
CREATE TABLE IF NOT EXISTS issue_links (
    id VARCHAR PRIMARY KEY,
    source_issue_id VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    target_issue_id VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    link_type VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_issue_links_source ON issue_links(source_issue_id);
CREATE INDEX IF NOT EXISTS idx_issue_links_target ON issue_links(target_issue_id);

-- Comments table
CREATE TABLE IF NOT EXISTS comments (
    id VARCHAR PRIMARY KEY,
    issue_id VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    author_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content VARCHAR NOT NULL,
    edited_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comments_issue_id ON comments(issue_id);
CREATE INDEX IF NOT EXISTS idx_comments_author_id ON comments(author_id);

-- Activity log table
CREATE TABLE IF NOT EXISTS activities (
    id VARCHAR PRIMARY KEY,
    issue_id VARCHAR NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    actor_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR NOT NULL,
    field VARCHAR,
    old_value VARCHAR,
    new_value VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_activities_issue_id ON activities(issue_id);
CREATE INDEX IF NOT EXISTS idx_activities_actor_id ON activities(actor_id);

-- Notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR NOT NULL,
    issue_id VARCHAR REFERENCES issues(id) ON DELETE CASCADE,
    actor_id VARCHAR REFERENCES users(id) ON DELETE SET NULL,
    content VARCHAR NOT NULL,
    read_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_read_at ON notifications(read_at);
