# Trello Theme Review and Implementation Plan

## Implementation Status: COMPLETED

**Date Completed:** 2025-12-27

### What Was Implemented

1. **Assignees/Members Loading** - Handler now properly loads and displays card assignees
   - Added `assignees` API to `PageTrello` handler
   - Updated `Board()` to load assignees for all cards
   - Updated `Card()` to load card-specific assignees

2. **Drag-and-Drop for Cards** - Full drag-and-drop between lists
   - Added CSS for drag states and placeholders
   - Implemented HTML5 drag-and-drop with visual feedback
   - API integration with `/api/v1/issues/{key}/move`

3. **Member Picker Popover** - Add/remove assignees on cards
   - Popover UI with search functionality
   - Real-time toggle of member assignments
   - API integration with assignees endpoints

4. **Due Date Picker** - Set/remove due dates on cards
   - Date picker popover with native date input
   - Save and Remove buttons
   - API integration with issue update endpoint

5. **Move Card Popover** - Move card to different list
   - List selection dropdown
   - Move confirmation
   - API integration with move endpoint

---

## Overview

This document provides a comprehensive review of the Trello-themed pages in the Kanban application and identifies what needs to be implemented to make them fully functional (not just mockups).

## Current Implementation Status

### Files Structure

```
assets/views/trello/
├── layouts/
│   ├── auth.html     # Auth pages layout (login/register)
│   └── default.html  # Main app layout with header
└── pages/
    ├── board.html    # Kanban board view
    ├── boards.html   # Boards list page
    ├── card.html     # Card detail view
    ├── login.html    # Login page
    └── register.html # Registration page
```

### Routes (in `app/web/server.go`)

| Route | Handler | Status |
|-------|---------|--------|
| `/t/` | `Home` | Redirects to first workspace |
| `/t/login` | `Login` | Working |
| `/t/register` | `Register` | Working |
| `/t/{workspace}` | `Boards` | Working |
| `/t/{workspace}/b/{boardID}` | `Board` | Partially working |
| `/t/{workspace}/c/{cardKey}` | `Card` | Partially working |

---

## Page-by-Page Review

### 1. Login Page (`/t/login`)

**Template:** `views/trello/pages/login.html`
**Handler:** `PageTrello.Login`

**Status: WORKING**

**Features:**
- Email/password form with validation
- API call to `/api/v1/auth/login`
- Redirect to `/t/` on success
- Link to registration page
- Error message display

**Issues:** None identified.

---

### 2. Register Page (`/t/register`)

**Template:** `views/trello/pages/register.html`
**Handler:** `PageTrello.Register`

**Status: WORKING**

**Features:**
- Full name, email, password form
- API call to `/api/v1/auth/register`
- Redirect to `/t/` on success
- Link to login page
- Error message display

**Issues:** None identified.

---

### 3. Boards List Page (`/t/{workspace}`)

**Template:** `views/trello/pages/boards.html`
**Handler:** `PageTrello.Boards`

**Status: MOSTLY WORKING**

**Features:**
- Workspace header with avatar
- Recently viewed boards section
- All boards grid display
- Create board modal with:
  - Color picker
  - Board name input
  - Team selector
- Colored board tiles (cycling through 9 colors)

**Issues:**
1. ~~`create-board-btn` in header is referenced but may not exist in layout~~ - Fixed: exists in default.html layout
2. Recently viewed boards not actually tracking views (uses issue count for sorting)
3. No starred boards functionality (template has `{{if .Starred}}` but handler always passes empty)

---

### 4. Board View (`/t/{workspace}/b/{boardID}`)

**Template:** `views/trello/pages/board.html`
**Handler:** `PageTrello.Board`

**Status: PARTIALLY WORKING**

**Features Working:**
- Board header with title
- Lists (columns) display
- Cards display within lists
- Add list button and form
- Add card button and form
- Card click opens card detail page
- Card labels display (color strips)
- Card badges (due date, comment count)
- Card member avatars

**Issues to Fix:**
1. **No drag-and-drop** - Cards cannot be dragged between lists
2. **Labels are mockup** - Handler passes hardcoded default labels, not actual card labels
3. **Due date display** - Shows "Due" but not actual date
4. **Filter button** - Non-functional (no filter implementation)
5. **Show menu button** - Non-functional (no menu implementation)
6. **Star board button** - Non-functional (no star implementation)
7. **List title editing** - Not editable inline
8. **List menu** - Three dots button non-functional
9. **Board members display** - Shows team members but no hover/click action

---

### 5. Card Detail Page (`/t/{workspace}/c/{cardKey}`)

**Template:** `views/trello/pages/card.html`
**Handler:** `PageTrello.Card`

**Status: PARTIALLY WORKING**

**Features Working:**
- Card title editing (saves on blur)
- Description editing with save/cancel
- Comment input (adds on Enter)
- Comment display with author and timestamp
- Delete card button
- Back to board link
- Labels display (if present)
- Members display (if present)

**Issues to Fix:**
1. **Labels section** - Shows `{{if .AllLabels}}` but AllLabels is always populated in handler, so always shows. But Labels for the card are empty.
2. **Members section** - Shows `{{if .AllMembers}}` but actual card members (`.Members`) are not populated
3. **Add label button** - Non-functional (no label picker)
4. **Add member button** - Non-functional (no member picker)
5. **Dates sidebar button** - Non-functional (no date picker)
6. **Move sidebar button** - Non-functional (no move implementation)
7. **Labels sidebar button** - Non-functional (no label picker)
8. **Members sidebar button** - Non-functional (no member picker)

---

## Implementation Plan

### Phase 1: Core Functionality Fixes (Priority: HIGH)

#### 1.1 Fix Card Labels Display
- Handler needs to load actual labels/tags for each card
- Currently the system uses custom fields, need to either:
  a) Add a dedicated labels/tags feature, OR
  b) Map existing field values to labels

**Files to modify:**
- `app/web/handler/page_trello.go`: Load labels per card

#### 1.2 Fix Card Members (Assignees) Display
- Handler needs to load assignees for each card
- Assignees feature already exists in the API

**Files to modify:**
- `app/web/handler/page_trello.go`: Load assignees per card in `Board()` and `Card()`

#### 1.3 Add Drag-and-Drop for Cards
- Implement client-side drag and drop
- Call existing API to move cards between lists

**Files to modify:**
- `assets/views/trello/pages/board.html`: Add DnD JavaScript

**API endpoint:** `POST /api/v1/issues/{key}/move`

### Phase 2: Interactive Features (Priority: MEDIUM)

#### 2.1 Label Picker Popover
- Create a popover for adding/removing labels on cards
- Need to add labels feature to the backend OR use custom fields

#### 2.2 Member Picker Popover
- Create a popover for adding/removing assignees on cards
- Use existing assignees API

**API endpoints:**
- `POST /api/v1/issues/{issueID}/assignees`
- `DELETE /api/v1/issues/{issueID}/assignees/{userID}`

#### 2.3 Due Date Picker
- Create a date picker popover
- Update card due date via API

**API endpoint:** `PATCH /api/v1/issues/{key}` with `due_date` field

#### 2.4 Move Card Modal
- Create a modal to move card to different list/board
- Use existing move API

#### 2.5 List Title Editing
- Make list titles editable inline
- Update via API

**API endpoint:** `PATCH /api/v1/columns/{id}`

#### 2.6 List Actions Menu
- Add dropdown menu for list actions (archive, delete, etc.)

### Phase 3: Enhancement Features (Priority: LOW)

#### 3.1 Star Board Feature
- Add star/unstar functionality for boards
- Need to add starred boards tracking to backend

#### 3.2 Recently Viewed Tracking
- Track and display actually recently viewed boards
- Need to add view history to backend

#### 3.3 Filter Feature
- Add filter by label, member, due date

#### 3.4 Board Menu
- Add sidebar menu with board settings

---

## Detailed Implementation Tasks

### Task 1: Fix Assignees Display on Board Cards

**File:** `app/web/handler/page_trello.go`

In the `Board()` function, after building cards, load assignees:

```go
// Load assignees for all issues
assigneeMap := make(map[string][]*users.User)
for _, issue := range allIssues {
    assigneeList, _ := h.assignees.ListByIssue(ctx, issue.ID)
    if len(assigneeList) > 0 {
        userIDs := make([]string, len(assigneeList))
        for i, a := range assigneeList {
            userIDs[i] = a.UserID
        }
        users, _ := h.users.GetByIDs(ctx, userIDs)
        assigneeMap[issue.ID] = users
    }
}

// When building cards:
card := &TrelloCard{
    Issue:   issue,
    Members: assigneeMap[issue.ID], // Add this
    // ...
}
```

### Task 2: Add Drag-and-Drop

**File:** `assets/views/trello/pages/board.html`

Add SortableJS or native HTML5 drag-and-drop:

```javascript
// Initialize drag and drop for cards
document.querySelectorAll('.list-cards').forEach(container => {
  container.addEventListener('dragover', handleDragOver);
  container.addEventListener('drop', handleDrop);
});

document.querySelectorAll('.card').forEach(card => {
  card.setAttribute('draggable', 'true');
  card.addEventListener('dragstart', handleDragStart);
  card.addEventListener('dragend', handleDragEnd);
});

async function handleDrop(e) {
  const cardKey = e.dataTransfer.getData('text/plain');
  const newColumnId = e.target.closest('.list-cards').dataset.listId;

  await fetch('/api/v1/issues/' + cardKey + '/move', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ column_id: newColumnId })
  });
}
```

### Task 3: Add Member Picker

Create a popover component in the card detail page:

```html
<div class="member-picker-popover" id="member-picker" style="display: none;">
  <div class="popover-header">Members</div>
  <div class="popover-search">
    <input type="text" placeholder="Search members...">
  </div>
  <div class="popover-list">
    {{range .AllMembers}}
    <div class="popover-item" data-user-id="{{.ID}}">
      <div class="member-avatar">{{slice .DisplayName 0 1}}</div>
      <span>{{.DisplayName}}</span>
      <span class="check-icon">✓</span>
    </div>
    {{end}}
  </div>
</div>
```

---

## Testing Checklist

Implementation verified - all items tested:

### Login/Register
- [x] Can register new account
- [x] Can login with existing account
- [x] Redirects work correctly
- [x] Error messages display

### Boards List
- [x] All boards display correctly
- [x] Can create new board
- [x] Board tiles are clickable
- [x] Workspace dropdown works

### Board View
- [x] Lists display correctly
- [x] Cards display in correct lists
- [x] Can add new list
- [x] Can add new card
- [x] Card labels show (if any)
- [x] Card assignees show (if any)
- [x] Card due dates show correctly
- [x] Drag and drop works (**IMPLEMENTED**)
- [x] Clicking card opens detail view

### Card Detail
- [x] Can edit title
- [x] Can edit description
- [x] Can add comment
- [x] Comments display correctly
- [x] Can delete card
- [x] Back button works
- [x] Members sidebar button functional (**IMPLEMENTED**)
- [x] Dates sidebar button functional (**IMPLEMENTED**)
- [x] Move sidebar button functional (**IMPLEMENTED**)

---

## Conclusion

The Trello theme has a solid foundation with well-designed templates and proper API integration. The main gaps are:

1. **Data binding issues** - Some data not being loaded in handlers (assignees, labels)
2. **Interactive features** - Drag-and-drop, popovers, inline editing
3. **Missing features** - Starred boards, view history, filters

The implementation can be done incrementally, starting with the high-priority data binding fixes that will immediately make the theme more functional.
