# Trello Theme for Kanban Blueprint

## Overview

This document specifies the implementation of a Trello-style theme for the Kanban blueprint, providing an authentic Trello-like UI experience.

## Research Summary

### Trello Design System

Trello uses a design system called **Nachos** internally. The key design elements have been extracted from public resources.

### Color Palette

#### Primary Colors
| Color Name | Hex Code | RGB | Usage |
|------------|----------|-----|-------|
| Trello Blue 500 | #0079bf | rgb(0, 121, 191) | Primary brand color, buttons |
| Trello Blue 600 | #026aa7 | rgb(2, 106, 167) | Hover states, darker accents |
| Trello Blue 700 | #055a8c | rgb(5, 90, 140) | Header backgrounds |
| Dark Blue | #0067a3 | rgb(0, 103, 163) | Masthead/App bar |

#### Board Background Colors
| Color | Hex Code | Usage |
|-------|----------|-------|
| Blue | #0079bf | Default board background |
| Orange | #d29034 | Board background option |
| Green | #519839 | Board background option |
| Red | #b04632 | Board background option |
| Purple | #89609e | Board background option |
| Pink | #cd5a91 | Board background option |
| Lime | #4bbf6b | Board background option |
| Sky | #00c2e0 | Board background option |
| Gray | #838c91 | Board background option |

#### UI Colors
| Element | Hex Code | Usage |
|---------|----------|-------|
| List Background | #ebecf0 | List container background |
| Card Background | #ffffff | Card background |
| Card Shadow | rgba(9,30,66,.25) | Card drop shadow |
| Text Primary | #172b4d | Main text color |
| Text Secondary | #5e6c84 | Secondary/muted text |
| Border | #dfe1e6 | Subtle borders |
| Button Primary | #0079bf | Primary action buttons |
| Button Danger | #eb5a46 | Destructive actions |
| Success | #61bd4f | Success states, labels |
| Warning | #f2d600 | Warning states, labels |

### Typography

- **Primary Font**: `-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Noto Sans', Ubuntu, 'Droid Sans', 'Helvetica Neue', sans-serif`
- **Base Font Size**: 14px
- **Font Weight Regular**: 400
- **Font Weight Bold**: 700

### Component Structure

#### CSS Naming Convention (BEM-like)
```
.component-descendant-descendant
.component.mod-modifier
.component.is-state
.u-utility-class
.js-javascript-hook
```

### Layout Architecture

#### Grid Structure
```
┌─────────────────────────────────────────────────────────────┐
│ App Bar (4rem height)                                       │
├─────────────────────────────────────────────────────────────┤
│ Board Bar (3rem height) - Board title, star, menu           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐           │
│  │  List   │ │  List   │ │  List   │ │  List   │ ← →       │
│  │         │ │         │ │         │ │         │           │
│  │ ┌─────┐ │ │ ┌─────┐ │ │ ┌─────┐ │ │ ┌─────┐ │           │
│  │ │Card │ │ │ │Card │ │ │ │Card │ │ │ │Card │ │           │
│  │ └─────┘ │ │ └─────┘ │ │ └─────┘ │ │ └─────┘ │           │
│  │ ┌─────┐ │ │ ┌─────┐ │ │         │ │         │           │
│  │ │Card │ │ │ │Card │ │ │         │ │         │           │
│  │ └─────┘ │ │ └─────┘ │ │         │ │         │           │
│  │         │ │         │ │         │ │         │           │
│  │[+ Add]  │ │[+ Add]  │ │[+ Add]  │ │[+ Add]  │           │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘           │
│                                                             │
│  (Horizontal scroll when lists overflow)                    │
└─────────────────────────────────────────────────────────────┘
```

#### CSS Layout Properties
```css
/* Board container - horizontal scroll */
.board-canvas {
  display: flex;
  overflow-x: auto;
  overflow-y: hidden;
  padding: 8px 4px;
}

/* List - fixed width, vertical scroll */
.list {
  width: 272px;
  min-width: 272px;
  background: #ebecf0;
  border-radius: 3px;
  margin: 0 4px;
  max-height: 100%;
  display: flex;
  flex-direction: column;
}

/* Cards container - vertical scroll */
.list-cards {
  overflow-y: auto;
  overflow-x: hidden;
  padding: 0 4px;
  flex: 1 1 auto;
  min-height: 0;
}

/* Card */
.card {
  background: #fff;
  border-radius: 3px;
  box-shadow: 0 1px 0 rgba(9,30,66,.25);
  padding: 6px 8px;
  margin-bottom: 8px;
  cursor: pointer;
}
```

### Component Specifications

#### App Bar (Header)
- Height: 40px (4rem at 10px base)
- Background: `#0067a3` (slightly darker than board)
- Contains: Logo, Search, Create button, Notifications, User menu

#### Board Bar
- Height: 30px-40px
- Background: `rgba(0,0,0,0.24)` over board background
- Contains: Board title, Star button, Members, Menu button

#### List
- Width: 272px
- Background: `#ebecf0`
- Border-radius: 3px
- Header: List name, menu button (on hover)
- Footer: "Add a card" button

#### Card
- Background: `#ffffff`
- Border-radius: 3px
- Box-shadow: `0 1px 0 rgba(9,30,66,.25)`
- Padding: 6px 8px
- Contains:
  - Labels (colored badges)
  - Title
  - Badges (due date, attachments, comments, checklist)
  - Members (avatars on bottom-right)

#### Labels (Color Chips)
| Color Name | Hex Code |
|------------|----------|
| Green | #61bd4f |
| Yellow | #f2d600 |
| Orange | #ff9f1a |
| Red | #eb5a46 |
| Purple | #c377e0 |
| Blue | #0079bf |
| Sky | #00c2e0 |
| Lime | #51e898 |
| Pink | #ff78cb |
| Black | #344563 |

## Implementation Plan

### 1. File Structure
```
assets/
├── views/
│   └── trello/
│       ├── layouts/
│       │   ├── default.html     # Main app layout
│       │   └── auth.html        # Login/register layout
│       └── pages/
│           ├── login.html
│           ├── register.html
│           ├── boards.html      # Board list (home)
│           ├── board.html       # Kanban board view
│           ├── card.html        # Card detail modal/page
│           └── settings.html    # Settings page
├── static/
│   └── js/
│       └── trello.js           # Trello-specific interactions
```

### 2. Handler Routes (`/t/` prefix)
```
GET  /t/                        # Redirect to boards
GET  /t/login                   # Login page
GET  /t/register                # Register page
GET  /t/{workspace}             # Boards list
GET  /t/{workspace}/b/{boardId} # Board view
GET  /t/{workspace}/c/{cardKey} # Card detail
```

### 3. Template Data Structures

#### BoardsData (Home/Workspace view)
```go
type TrelloBoardsData struct {
    User       *users.User
    Workspace  *workspaces.Workspace
    Workspaces []*workspaces.Workspace
    Starred    []*projects.Project
    Recent     []*projects.Project
    All        []*projects.Project
}
```

#### BoardData (Kanban view)
```go
type TrelloBoardData struct {
    User      *users.User
    Workspace *workspaces.Workspace
    Board     *projects.Project
    Lists     []*TrelloList
    Members   []*users.User
    Labels    []*fields.Field
}

type TrelloList struct {
    *columns.Column
    Cards []*TrelloCard
}

type TrelloCard struct {
    *issues.Issue
    Labels    []*values.Value
    Members   []*users.User
    HasDueDate bool
    IsOverdue bool
    CommentCount int
}
```

### 4. Key UI Features to Implement

1. **App Header**
   - Logo (left)
   - Search bar (center-left)
   - Create board button
   - User avatar with dropdown menu

2. **Board Header**
   - Board name (editable)
   - Star button
   - Visibility toggle
   - Board menu (right)

3. **Lists**
   - Draggable columns
   - Card count
   - Add card button at bottom
   - List menu on header hover

4. **Cards**
   - Color labels (top)
   - Card title
   - Due date badge
   - Comment count badge
   - Attachment count badge
   - Assignee avatars (bottom-right)
   - Drag and drop

5. **Card Detail Modal**
   - Title (editable)
   - Description (markdown)
   - Labels
   - Members
   - Due date
   - Attachments
   - Checklist
   - Comments
   - Activity log

### 5. Interactive Features

- Drag and drop cards between lists
- Drag and drop to reorder lists
- Inline editing of card/list titles
- Quick add cards
- Keyboard shortcuts
- Real-time updates (optional)

## CSS Variables

```css
:root {
  /* Primary colors */
  --trello-blue-100: #e4f0f6;
  --trello-blue-200: #bcd9ea;
  --trello-blue-300: #8bbdd9;
  --trello-blue-400: #5ba4cf;
  --trello-blue-500: #0079bf;
  --trello-blue-600: #026aa7;
  --trello-blue-700: #055a8c;

  /* Neutral colors */
  --trello-neutral-0: #ffffff;
  --trello-neutral-10: #fafbfc;
  --trello-neutral-20: #f4f5f7;
  --trello-neutral-30: #ebecf0;
  --trello-neutral-40: #dfe1e6;
  --trello-neutral-100: #c1c7d0;
  --trello-neutral-200: #a5adba;
  --trello-neutral-500: #5e6c84;
  --trello-neutral-800: #172b4d;

  /* Semantic colors */
  --trello-success: #61bd4f;
  --trello-warning: #f2d600;
  --trello-danger: #eb5a46;

  /* Label colors */
  --trello-label-green: #61bd4f;
  --trello-label-yellow: #f2d600;
  --trello-label-orange: #ff9f1a;
  --trello-label-red: #eb5a46;
  --trello-label-purple: #c377e0;
  --trello-label-blue: #0079bf;
  --trello-label-sky: #00c2e0;
  --trello-label-lime: #51e898;
  --trello-label-pink: #ff78cb;
  --trello-label-black: #344563;

  /* Layout */
  --header-height: 40px;
  --board-header-height: 36px;
  --list-width: 272px;
  --card-border-radius: 3px;
  --list-border-radius: 3px;

  /* Shadows */
  --card-shadow: 0 1px 0 rgba(9,30,66,.25);
  --popup-shadow: 0 8px 16px -4px rgba(9,30,66,.25);

  /* Font */
  --font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Noto Sans', Ubuntu, sans-serif;
  --font-size-small: 12px;
  --font-size-base: 14px;
  --font-size-large: 16px;
}
```

## Testing Requirements

### page_trello_test.go

Tests should cover:

1. **Authentication flows**
   - Login page renders
   - Register page renders
   - Redirect to login when not authenticated

2. **Boards list**
   - Displays user's workspaces
   - Displays boards in workspace
   - Starred boards section

3. **Board view**
   - Renders all lists
   - Renders cards in correct lists
   - Displays card labels, badges
   - Member avatars show

4. **Card detail**
   - Title and description display
   - Labels display
   - Due date shows
   - Comments load
   - Members show

5. **Error handling**
   - 404 for non-existent board
   - 404 for non-existent card
   - Unauthorized redirect

## References

- [Trello Power-Up Style Guide](https://developer.atlassian.com/cloud/trello/guides/power-ups/style-guide/)
- [Trello CSS Guide (Bobby Grace)](https://gist.github.com/bobbygrace/9e961e8982f42eb91b80)
- [Trello Color Palette](https://colorswall.com/palette/44)
- [CSS Grid/Flexbox Layout Tutorial](https://www.geeksforgeeks.org/css/how-to-create-a-trello-layout-with-css-grid-and-flexbox/)
- [Trello Theming with CSS Variables](https://atlassian.com/engineering/colorful-and-accessible-theming-in-trello)
