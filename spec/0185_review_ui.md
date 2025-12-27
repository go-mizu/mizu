# Trello UI Review and Enhancement Plan

## Overview

This document provides a comprehensive review of the current Trello-style UI implementation and outlines the enhancements needed to achieve pixel-perfect Trello-like appearance and behavior.

---

## 1. Layout: default.html

### Current State
- Header with transparent glass effect
- Logo, Workspaces dropdown, Recent button, Create button
- Search bar with expand animation
- Notifications button and user avatar dropdown

### Issues Found
1. **Header Height**: Current 44px, Trello uses 48px
2. **Logo**: Missing the characteristic Trello gradient hover effect
3. **Workspaces Button**: Missing the down chevron icon animation on hover
4. **Starred Boards**: Missing "Starred" dropdown in header-left
5. **Templates**: Missing "Templates" link in header
6. **Create Button**: Should have `+` icon, not just text
7. **Search Bar**: Placeholder should be "Search" with magnifying glass
8. **Notification Bell**: Missing badge count indicator
9. **Help Button**: Missing `?` help button before user avatar

### Enhancements Needed
- [ ] Adjust header height to 48px
- [ ] Add "Starred" dropdown menu
- [ ] Add "Templates" link
- [ ] Add `+` icon to Create button
- [ ] Add `?` help button
- [ ] Add notification badge support

---

## 2. Board Page: board.html

### Current State
- Board header with title, star button, members
- List containers with cards
- Drag and drop for cards
- Add card/list forms

### Issues Found

#### A. Board Header
1. **Board Title**: Should be editable inline (currently just a button)
2. **Star Button**: Missing filled/unfilled toggle states
3. **Visibility Button**: Missing "Workspace visible" indicator with lock icon
4. **Board Views**: Missing "Board", "Table", "Calendar" view tabs
5. **Power-Ups**: Missing Power-Ups button
6. **Automation**: Missing Automation button
7. **Filter**: Filter icon is edit pencil, should be funnel icon
8. **Share Button**: Missing prominent share button
9. **Menu Button**: Should say "..." and show side panel

#### B. Lists
1. **List Header**: Missing drag handle appearance on hover
2. **List Title**: Should be editable inline on click
3. **List Menu**: Missing dropdown with options (Copy, Move, Archive, etc.)
4. **Watch List**: Missing watch icon option
5. **Drag Lists**: Lists themselves are not draggable (only cards)

#### C. Cards
1. **Card Cover**: Missing color cover/image cover at top of card
2. **Card Edit Pencil**: Missing pencil icon on hover
3. **Labels on Hover**: Labels should expand to show text on hover
4. **Checkbox Badge**: Missing checkbox progress badge
5. **Attachment Badge**: Missing attachment count badge
6. **Card Due Date**: Should show actual date, not just "Due"
7. **Card Quick Edit**: Missing quick edit popup on pencil click

#### D. Drag & Drop (CRITICAL BUG)
1. **Flickering Issue**: Placeholder flickers rapidly when dragging over list
   - Cause: `dragover` event fires continuously, each creating/removing placeholder
   - The `dragleave` fires incorrectly when moving over child elements
2. **Smooth Animation**: Missing smooth card movement animation
3. **Drag Preview**: Card drag preview should show card snapshot
4. **Auto-scroll**: Missing auto-scroll when dragging near edges

#### E. Add Card Form
1. **Button Styling**: "Add a card" should have `+` icon properly aligned
2. **Textarea**: Should auto-resize
3. **Template Button**: Missing template/copy from template option

#### F. Add List Form
1. **Position**: Should appear as last list, not separate element
2. **Focus Management**: Should close when clicking outside

---

## 3. Card Detail Page: card.html

### Current State
- Card title (editable)
- Labels section
- Members section
- Description with edit mode
- Activity/comments section
- Sidebar with action buttons

### Issues Found

#### A. Header
1. **Cover Image**: Missing cover image/color at top
2. **Title**: Missing card icon before title
3. **In List Link**: "in list" text should be clickable to move card
4. **Watch Button**: Missing watch/subscribe button

#### B. Main Content
1. **Notifications**: Missing "Notifications" toggle section
2. **Due Date Display**: Should show as chip with checkbox in main area
3. **Checklist**: Missing checklist section entirely
4. **Attachments**: Missing attachments section
5. **Custom Fields**: Missing custom fields section
6. **Activity Toggle**: Missing "Show Details" toggle for activity

#### C. Description
1. **Markdown Support**: Should support markdown preview
2. **Emoji Picker**: Missing emoji button
3. **@ Mentions**: Missing @mention autocomplete
4. **Link Detection**: Should auto-detect and linkify URLs

#### D. Comments
1. **Edit/Delete**: Missing edit and delete buttons on own comments
2. **Reactions**: Missing emoji reactions
3. **Attachment in Comment**: Missing attach file button in comment

#### E. Sidebar
1. **Join Button**: Missing "Join" button at top of sidebar
2. **Checklist Button**: Missing "Checklist" button
3. **Attachment Button**: Missing "Attachment" button
4. **Cover Button**: Missing "Cover" button
5. **Custom Fields**: Missing "Custom Fields" button
6. **Copy Button**: Missing "Copy" action
7. **Make Template**: Missing "Make template" action
8. **Archive Button**: Missing "Archive" action
9. **Share Button**: Missing "Share" action

#### F. Popovers
1. **Labels Popover**: Missing complete labels management
2. **Members Search**: Filter should work case-insensitive
3. **Due Date**: Missing time picker, reminder options

---

## 4. Boards List Page: boards.html

### Current State
- Workspace header with avatar
- Recently viewed boards section
- Your boards grid
- Create board modal

### Issues Found
1. **Left Sidebar**: Missing persistent left sidebar with workspace navigation
2. **Starred Section**: Missing "Starred Boards" section
3. **Templates Section**: Missing "Templates" section
4. **Closed Boards**: Missing link to view closed boards
5. **Board Tile Star**: Missing star icon on board tiles
6. **Board Tile Menu**: Missing `...` menu on board tile hover
7. **Board Backgrounds**: Should support gradient and image backgrounds
8. **Create Board Preview**: Modal should show live preview of board with selected color/image

---

## 5. Auth Pages: login.html, register.html

### Current State
- Clean auth card with gradient background
- Form with email/password fields
- Social login placeholder

### Issues Found
1. **Social Logins**: Missing Google, Microsoft, Apple, Slack login buttons
2. **Password Visibility**: Missing eye icon to show/hide password
3. **Remember Me**: Missing "Keep me logged in" checkbox
4. **Forgot Password**: Missing "Can't log in?" link
5. **Terms**: Missing terms of service links

---

## 6. Critical Bug Fix: Drag & Drop Flickering

### Root Cause Analysis
The flickering occurs because:
1. Each `dragover` event removes and recreates the placeholder
2. `dragleave` fires when entering child elements
3. No debouncing/throttling on placeholder updates

### Solution
```javascript
// Use a throttle mechanism
let lastDropTarget = null;
let throttleTimeout = null;

container.addEventListener('dragover', (e) => {
  e.preventDefault();

  // Throttle updates to prevent flickering
  if (throttleTimeout) return;
  throttleTimeout = setTimeout(() => {
    throttleTimeout = null;
  }, 50);

  // Only update if drop position changed
  const afterElement = getDragAfterElement(container, e.clientY);
  if (afterElement !== lastDropTarget) {
    lastDropTarget = afterElement;
    updatePlaceholder(container, afterElement);
  }
});

// Check if relatedTarget is truly outside container
container.addEventListener('dragleave', (e) => {
  const rect = container.getBoundingClientRect();
  const isOutside = e.clientX < rect.left || e.clientX > rect.right ||
                    e.clientY < rect.top || e.clientY > rect.bottom;
  if (isOutside) {
    removePlaceholder();
  }
});
```

---

## Enhancement Priority Order

### Phase 1: Critical Bug Fixes
1. Fix drag & drop flickering (HIGH PRIORITY)
2. Fix list dragleave detection

### Phase 2: Board Page Polish
1. Add card cover colors
2. Add edit pencil on card hover
3. Expand labels on hover
4. Add due date actual display
5. Add attachment/checklist badges
6. Make list title editable
7. Add list drag & drop

### Phase 3: Card Detail Enhancement
1. Add cover image section
2. Add checklist functionality
3. Add attachments section
4. Add activity details toggle
5. Improve sidebar with all actions

### Phase 4: Navigation & Layout
1. Add left sidebar to boards page
2. Add starred boards section
3. Improve header with all Trello elements
4. Add help button and info button

### Phase 5: Auth & Polish
1. Add social login buttons (UI only)
2. Add password visibility toggle
3. Add remember me checkbox
4. Add terms links

---

## Implementation Checklist

- [x] Fix drag & drop flickering
- [x] Add card edit pencil on hover
- [x] Add card cover colors
- [x] Make labels expandable on hover
- [x] Show actual due dates on cards
- [x] Add checklist/attachment badges
- [x] Make list titles editable inline
- [x] Add list menu dropdown with actions
- [x] Add card detail cover section
- [x] Polish header elements (Starred, Templates, Help buttons)
- [ ] Add list drag & drop (future enhancement)
- [ ] Add left sidebar navigation (future enhancement)
- [ ] Add activity details toggle (future enhancement)

---

## Completed Enhancements Summary

### 1. Drag & Drop Flickering Fix
- Added position tracking with `lastTargetContainer` and `lastAfterElement`
- Only update placeholder when target position actually changes
- Use bounding rect check for `dragleave` to detect true container exit
- Smooth height transition on placeholder

### 2. Card UI Enhancements
- Edit pencil button appears on card hover
- Card cover colors support (green, yellow, orange, red, purple, blue, sky, lime, pink, black)
- Labels expand on hover to show text with smooth animation
- Due dates show actual formatted date (Jan 2 format)
- Checklist progress badge support
- Attachment count badge support

### 3. List Enhancements
- List titles are now editable inline (textarea)
- Auto-resize on input
- Save on blur, Enter key submits, Escape cancels
- List menu dropdown with actions:
  - Add card
  - Copy list
  - Move list
  - Sort by date
  - Archive all cards
  - Archive list (with confirmation)

### 4. Card Detail Page
- Cover section with color support
- Join button for quick membership
- Checklist button
- Attachment button
- Cover button
- Copy, Archive, Share action buttons
- Delete button styled in danger color

### 5. Header Improvements
- Header height updated to 48px (Trello standard)
- Added "Starred" dropdown
- Added "Templates" dropdown
- Added + icon to Create button
- Added Information button
- Added Help button
- Enhanced user dropdown with Settings and Theme options

---

## Template Error Fixes

Fixed template errors where fields were referenced that don't exist in the Go structs:

### Removed from board.html:
- `.CoverColor` - Issue struct doesn't have this field
- `.HasChecklist` - TrelloCard doesn't have this field
- `.ChecklistProgress` - TrelloCard doesn't have this field
- `.AttachmentCount` - TrelloCard doesn't have this field
- `.IsComplete` - TrelloCard doesn't have this field (use IsOverdue/IsDueSoon instead)

### Removed from card.html:
- `.Card.CoverColor` - Issue struct doesn't have this field

### Notes:
- To add cover colors, checklist, and attachment features, the Issue model and TrelloCard wrapper need to be extended
- The cover CSS styles are preserved for future use when the backend supports it
