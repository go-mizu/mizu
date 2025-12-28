# GitHome UI Specification

## Overview

This document specifies the UI implementation for GitHome, a GitHub-like repository hosting web application. The design follows GitHub's Primer Design System to achieve a pixel-perfect look and feel.

## Design System

### Color Palette (GitHub Primer)

```css
/* Light Theme Colors */
--color-canvas-default: #ffffff;
--color-canvas-subtle: #f6f8fa;
--color-canvas-inset: #eff2f5;
--color-border-default: #d0d7de;
--color-border-muted: #d8dee4;

/* Text Colors */
--color-fg-default: #1f2328;
--color-fg-muted: #656d76;
--color-fg-subtle: #6e7781;
--color-fg-on-emphasis: #ffffff;

/* Accent Colors */
--color-accent-fg: #0969da;
--color-accent-emphasis: #0969da;
--color-success-fg: #1a7f37;
--color-success-emphasis: #1f883d;
--color-danger-fg: #d1242f;
--color-danger-emphasis: #cf222e;

/* Button Colors */
--color-btn-primary-bg: #1f883d;
--color-btn-primary-hover-bg: #1a7f37;
--color-btn-bg: #f6f8fa;
--color-btn-hover-bg: #f3f4f6;

/* Header */
--color-header-bg: #24292f;
--color-header-text: #ffffff;
```

### Typography

- Font Family: `-apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans", Helvetica, Arial, sans-serif`
- Font sizes: 12px (small), 14px (body), 16px (medium), 20px (large), 24px (h2), 32px (h1)
- Font weights: 400 (normal), 500 (medium), 600 (semibold), 700 (bold)

### Spacing Scale

- 0: 0px
- 1: 4px
- 2: 8px
- 3: 16px
- 4: 24px
- 5: 32px
- 6: 40px

## Directory Structure

```
blueprints/githome/assets/
├── embed.go                    # Template and static file embedding
├── static/
│   ├── css/
│   │   └── app.css            # Main stylesheet with GitHub Primer colors
│   ├── js/
│   │   └── app.js             # Client-side JavaScript
│   └── img/
│       └── logo.svg           # GitHome logo
└── views/
    └── default/
        ├── layouts/
        │   ├── default.html    # Main layout with header/sidebar
        │   └── auth.html       # Auth pages layout (login/register)
        └── pages/
            ├── home.html           # Dashboard / landing
            ├── login.html          # Sign in
            ├── register.html       # Sign up
            ├── explore.html        # Browse repositories
            ├── new_repo.html       # Create repository
            ├── user_profile.html   # User profile page
            ├── repo_home.html      # Repository home (README)
            ├── repo_issues.html    # Issues list
            ├── issue_view.html     # Single issue view
            ├── new_issue.html      # Create new issue
            └── repo_settings.html  # Repository settings
```

## Pages Specification

### 1. Default Layout (`layouts/default.html`)

```
+------------------------------------------------------------------+
|  [Logo] GitHome     [Search...]        [+] [Notifications] [Avatar]|
+------------------------------------------------------------------+
|                                                                    |
|                         Main Content Area                          |
|                                                                    |
+------------------------------------------------------------------+
|  Footer: GitHome - A GitHub Clone                                  |
+------------------------------------------------------------------+
```

Components:
- Header with dark background (#24292f)
- Logo (Octocat-style icon)
- Global search bar
- Create new dropdown (+)
- Notifications icon
- User avatar dropdown

### 2. Auth Layout (`layouts/auth.html`)

Centered card with:
- GitHome logo
- Form container
- Sign in/up links

### 3. Home Page (`pages/home.html`)

For authenticated users (Dashboard):
```
+------------------------------------------------------------------+
| Header                                                            |
+------------------------------------------------------------------+
| Sidebar (240px)        | Main Content                            |
|                        |                                         |
| Repositories           | Recent Activity                         |
| - repo-1              | - User X starred repo                    |
| - repo-2              | - User Y created issue                   |
|                        |                                         |
| Your teams             | Explore repositories                    |
| Create repository      | [Trending repos]                        |
+------------------------------------------------------------------+
```

For unauthenticated users (Landing):
```
+------------------------------------------------------------------+
| Header                                                            |
+------------------------------------------------------------------+
|                                                                   |
|  [Hero Section]                                                   |
|  Build and ship software on a single, powerful platform          |
|  [Sign up for free]  [Start a free trial]                       |
|                                                                   |
|  [Explore public repositories]                                    |
|  - Repository cards grid                                         |
+------------------------------------------------------------------+
```

### 4. Login Page (`pages/login.html`)

```
+---------------------------+
|     [GitHome Logo]        |
|                           |
|  Sign in to GitHome       |
|                           |
|  [Email/Username]         |
|  [Password]               |
|  [Sign in]                |
|                           |
|  New to GitHome?          |
|  Create an account        |
+---------------------------+
```

### 5. Register Page (`pages/register.html`)

```
+---------------------------+
|     [GitHome Logo]        |
|                           |
|  Create your account      |
|                           |
|  [Username]               |
|  [Email address]          |
|  [Password]               |
|  [Create account]         |
|                           |
|  Already have an account? |
|  Sign in                  |
+---------------------------+
```

### 6. Explore Page (`pages/explore.html`)

```
+------------------------------------------------------------------+
| Header                                                            |
+------------------------------------------------------------------+
|                                                                   |
| Explore                                                           |
|                                                                   |
| [Search repositories...]                                          |
|                                                                   |
| Trending                 | Filters                                |
| +----------------------+ | Language: [All]                       |
| | repo-name           | | Sort: [Most stars]                    |
| | Description...      | |                                       |
| | Stars: 123          | |                                       |
| +----------------------+ |                                       |
+------------------------------------------------------------------+
```

### 7. New Repository Page (`pages/new_repo.html`)

```
+------------------------------------------------------------------+
| Header                                                            |
+------------------------------------------------------------------+
|                                                                   |
| Create a new repository                                           |
|                                                                   |
| Owner: [dropdown]  / Repository name: [input]                     |
| Description (optional): [input]                                   |
|                                                                   |
| ( ) Public - Anyone can see this repository                      |
| ( ) Private - Only you can see this repository                   |
|                                                                   |
| [x] Add a README file                                            |
| [x] Add .gitignore: [None]                                       |
| [x] Choose a license: [None]                                     |
|                                                                   |
| [Create repository]                                              |
+------------------------------------------------------------------+
```

### 8. User Profile Page (`pages/user_profile.html`)

```
+------------------------------------------------------------------+
| Header                                                            |
+------------------------------------------------------------------+
| Avatar (296px)          | Pinned repositories                     |
| Username                | +----+ +----+ +----+                   |
| @handle                 | |repo| |repo| |repo|                   |
| Bio text                | +----+ +----+ +----+                   |
| [Edit profile]          |                                        |
|                         | [Overview] [Repositories] [Stars]       |
| Followers: 100          |                                        |
| Following: 50           | Repository list                        |
| Location: City          | +----------------------------------+   |
|                         | | repo-name               [Star]   |   |
|                         | | Description...                   |   |
|                         | | Language • Stars • Forks         |   |
|                         | +----------------------------------+   |
+------------------------------------------------------------------+
```

### 9. Repository Home Page (`pages/repo_home.html`)

```
+------------------------------------------------------------------+
| Header                                                            |
+------------------------------------------------------------------+
| owner/repo-name  [Watch] [Fork] [Star ★ 123]                     |
|                                                                   |
| [Code] [Issues] [Pull requests] [Actions] [Settings]              |
|                                                                   |
| +--------------------------------------------------------------+ |
| | Branch: main ▼ | [Go to file] [Add file ▼] [Code ▼]         | |
| +--------------------------------------------------------------+ |
| | README.md                                                     | |
| | +----------------------------------------------------------+ | |
| | | # Repository Name                                        | | |
| | | Description and documentation                            | | |
| | +----------------------------------------------------------+ | |
| +--------------------------------------------------------------+ |
|                                                                   |
| About                                                             |
| Description                                                       |
| ★ Stars: 123                                                     |
| ⑂ Forks: 45                                                      |
+------------------------------------------------------------------+
```

### 10. Issues List Page (`pages/repo_issues.html`)

```
+------------------------------------------------------------------+
| Header                                                            |
+------------------------------------------------------------------+
| owner/repo-name                                                   |
| [Code] [Issues (23)] [Pull requests] [Settings]                   |
|                                                                   |
| [Filters ▼] [Labels] [Milestones] [New issue]                    |
| [x] Open   [ ] Closed                                            |
|                                                                   |
| +--------------------------------------------------------------+ |
| | ○ Issue title #123                                            | |
| |   opened 2 hours ago by username                              | |
| +--------------------------------------------------------------+ |
| | ● Issue title #122                                      [Done]| |
| |   opened yesterday by username                                | |
| +--------------------------------------------------------------+ |
|                                                                   |
| [1] [2] [3] Next                                                 |
+------------------------------------------------------------------+
```

### 11. Issue View Page (`pages/issue_view.html`)

```
+------------------------------------------------------------------+
| Header                                                            |
+------------------------------------------------------------------+
| owner/repo-name > Issues > #123                                   |
|                                                                   |
| Issue title #123                                                  |
| ○ Open   username opened this issue 2 hours ago                  |
|                                                                   |
| +----------------------------------------------+ Sidebar         |
| | Issue description with markdown support     | | Assignees     |
| |                                             | | Labels        |
| +----------------------------------------------| | Milestone     |
| | 👤 Comment by user                          | | Projects      |
| | Comment content...                          | |               |
| +----------------------------------------------+ |               |
| | [Add a comment...]                          | |               |
| | [Comment] [Close issue]                     | |               |
| +----------------------------------------------+ +---------------+
+------------------------------------------------------------------+
```

### 12. New Issue Page (`pages/new_issue.html`)

```
+------------------------------------------------------------------+
| Header                                                            |
+------------------------------------------------------------------+
| owner/repo-name > Issues > New issue                              |
|                                                                   |
| +----------------------------------------------+ Sidebar         |
| | Title: [                                   ] | | Assignees     |
| |                                             | | Labels        |
| | [Write] [Preview]                           | | Projects      |
| | +------------------------------------------+| | Milestone     |
| | | Add a description...                     || |               |
| | |                                          || |               |
| | +------------------------------------------+| |               |
| |                                             | |               |
| | [Submit new issue]                          | |               |
| +----------------------------------------------+ +---------------+
+------------------------------------------------------------------+
```

### 13. Repository Settings Page (`pages/repo_settings.html`)

```
+------------------------------------------------------------------+
| Header                                                            |
+------------------------------------------------------------------+
| owner/repo-name > Settings                                        |
|                                                                   |
| Sidebar                | General                                  |
| +-------------------+  | +-------------------------------------+ |
| | General          |  | | Repository name                     | |
| | Collaborators    |  | | [repo-name]                        | |
| | Branches         |  | |                                     | |
| | Tags             |  | | Description (optional)              | |
| | Danger Zone      |  | | [description text...]               | |
| +-------------------+  | |                                     | |
|                        | | Visibility                          | |
|                        | | ( ) Public  (•) Private            | |
|                        | |                                     | |
|                        | | [Save changes]                     | |
|                        | +-------------------------------------+ |
+------------------------------------------------------------------+
```

## Components

### 1. Header Component

- Height: 62px
- Background: #24292f
- Contains: Logo, search, notifications, avatar

### 2. Repository Card

```html
<div class="repo-card">
  <div class="repo-header">
    <a href="/{owner}/{repo}" class="repo-name">{owner}/{repo}</a>
    <span class="visibility-label">Public</span>
  </div>
  <p class="repo-description">{description}</p>
  <div class="repo-meta">
    <span class="language"><span class="color-dot"></span> {language}</span>
    <span class="stars">★ {stars}</span>
    <span class="forks">⑂ {forks}</span>
    <span class="updated">Updated {timeAgo}</span>
  </div>
</div>
```

### 3. Issue Row

```html
<div class="issue-row">
  <span class="issue-icon open">○</span>
  <div class="issue-content">
    <a href="/{owner}/{repo}/issues/{number}" class="issue-title">{title}</a>
    <div class="issue-meta">
      #{number} opened {timeAgo} by {author}
    </div>
  </div>
  <div class="issue-labels">
    <span class="label" style="background: {color}">{name}</span>
  </div>
</div>
```

### 4. User Avatar

```html
<img src="{avatarUrl}" alt="{username}" class="avatar avatar-{size}">
<!-- Sizes: sm (20px), md (32px), lg (48px), xl (96px) -->
```

### 5. Button Styles

```html
<!-- Primary (green) -->
<button class="btn btn-primary">Create repository</button>

<!-- Default -->
<button class="btn">Cancel</button>

<!-- Outline -->
<button class="btn btn-outline">Star</button>

<!-- Danger -->
<button class="btn btn-danger">Delete</button>
```

## Template Data Types

### LoginData
```go
type LoginData struct {
    Title    string
    Error    string
}
```

### RegisterData
```go
type RegisterData struct {
    Title    string
    Error    string
}
```

### HomeData
```go
type HomeData struct {
    Title        string
    User         *users.User
    Repositories []*repos.Repository
}
```

### ExploreData
```go
type ExploreData struct {
    Title        string
    User         *users.User
    Repositories []*repos.Repository
    Query        string
}
```

### UserProfileData
```go
type UserProfileData struct {
    Title        string
    User         *users.User
    Profile      *users.User
    Repositories []*repos.Repository
    IsOwner      bool
}
```

### RepoHomeData
```go
type RepoHomeData struct {
    Title      string
    User       *users.User
    Owner      *users.User
    Repository *repos.Repository
    IsStarred  bool
    CanEdit    bool
}
```

### RepoIssuesData
```go
type RepoIssuesData struct {
    Title      string
    User       *users.User
    Owner      *users.User
    Repository *repos.Repository
    Issues     []*issues.Issue
    Total      int
    State      string
}
```

### IssueViewData
```go
type IssueViewData struct {
    Title      string
    User       *users.User
    Owner      *users.User
    Repository *repos.Repository
    Issue      *issues.Issue
    Author     *users.User
    Comments   []*issues.Comment
    CanEdit    bool
}
```

### NewIssueData
```go
type NewIssueData struct {
    Title      string
    User       *users.User
    Owner      *users.User
    Repository *repos.Repository
    Labels     []*labels.Label
}
```

### RepoSettingsData
```go
type RepoSettingsData struct {
    Title         string
    User          *users.User
    Owner         *users.User
    Repository    *repos.Repository
    Collaborators []*repos.Collaborator
}
```

### NewRepoData
```go
type NewRepoData struct {
    Title string
    User  *users.User
    Error string
}
```

## JavaScript Functionality

### Required Features

1. **Dropdown menus** - User menu, create menu, filter dropdowns
2. **Modal dialogs** - Delete confirmation, add collaborator
3. **Form validation** - Client-side validation before submit
4. **Star toggle** - AJAX star/unstar without page reload
5. **Tab switching** - Code/Issues/PRs tabs
6. **Search autocomplete** - Repository/user search
7. **Markdown preview** - For issue/comment editing

## CSS Structure

```css
/* app.css */

/* 1. CSS Custom Properties (GitHub Primer colors) */
:root { ... }

/* 2. Reset & Base */
*, *::before, *::after { ... }
body { ... }

/* 3. Typography */
h1, h2, h3 { ... }
a { ... }

/* 4. Layout */
.container { ... }
.main-content { ... }
.sidebar { ... }

/* 5. Components */
.btn { ... }
.btn-primary { ... }
.avatar { ... }
.label { ... }
.repo-card { ... }
.issue-row { ... }

/* 6. Header */
.app-header { ... }
.header-search { ... }

/* 7. Forms */
.form-group { ... }
.form-control { ... }

/* 8. Utilities */
.text-muted { ... }
.d-flex { ... }
.mt-3 { ... }
```

## Implementation Order

1. **Phase 1: Assets Structure**
   - Update `assets/embed.go` with template parsing
   - Create directory structure
   - Add base CSS with Primer colors

2. **Phase 2: Layouts**
   - Create `layouts/default.html` with header
   - Create `layouts/auth.html` for login/register

3. **Phase 3: Auth Pages**
   - Implement `login.html`
   - Implement `register.html`

4. **Phase 4: Core Pages**
   - Implement `home.html`
   - Implement `explore.html`
   - Implement `user_profile.html`

5. **Phase 5: Repository Pages**
   - Implement `repo_home.html`
   - Implement `new_repo.html`
   - Implement `repo_settings.html`

6. **Phase 6: Issues Pages**
   - Implement `repo_issues.html`
   - Implement `issue_view.html`
   - Implement `new_issue.html`

7. **Phase 7: Integration**
   - Update `page.go` handlers to use templates
   - Add JavaScript interactions
   - Write template tests

## Testing

Each template should have a test that:
1. Loads all templates without errors
2. Renders with sample data without errors
3. Contains expected HTML elements

Example test pattern:
```go
func TestLoginTemplate(t *testing.T) {
    templates := loadTemplates(t)
    data := LoginData{Title: "Sign In"}
    var buf bytes.Buffer
    err := templates["login"].Execute(&buf, data)
    if err != nil {
        t.Errorf("Login template error: %v", err)
    }
}
```

## Sources

- [GitHub Primer CSS](https://github.com/primer/css) - GitHub's Design System
- [GitHub Primer Primitives](https://github.com/primer/primitives) - Design tokens
- [Primer Components](https://primer.style/components/) - Component library
- [GitHub Color Palette](https://www.designpieces.com/palette/github-color-palette-hex-and-rgb/) - Official colors
