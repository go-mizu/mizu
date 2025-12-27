# UI Review and Refactoring Spec

## Overview

This document details UI issues found in the Kanban application and the required fixes for consistency and improved UX.

---

## Issue 1: Remove Topbar and Relocate User Menu

### Current State
- Topbar exists at `default.html:136-202` with:
  - Sidebar toggle button
  - Breadcrumb navigation
  - Search trigger (Cmd+K)
  - Create button
  - User avatar dropdown

### Problems
- Topbar looks out of place in a modern Linear-style interface
- Takes up vertical space unnecessarily
- User menu is disconnected from sidebar

### Fix
1. Remove the entire topbar element
2. Move user info to bottom of sidebar (before Projects section)
3. Move search trigger to sidebar header
4. Move Create button to sidebar or keep floating button
5. Move breadcrumbs inline with page content

### Files to Modify
- `assets/views/default/layouts/default.html`
- `assets/static/css/default.css` (remove topbar styles: lines 1314-1430)

---

## Issue 2: Rework Issues List View

### Current State
- `issues.html:110-116` has a "New Issue" button in header
- Chevron on workspace name dropdown looks off
- View switcher is on the left, controls are scattered

### Problems
- "New Issue" button redundant (already in topbar/sidebar)
- Chevron direction/styling inconsistent
- Layout not clean

### Fix
1. Remove "New Issue" button from issues page header
2. Move "New Issue" to sidebar as icon-only button with tooltip
3. Fix dropdown chevron styling - use consistent `dropdown-chevron` class
4. Move view switcher to the right side of header

### Files to Modify
- `assets/views/default/pages/issues.html`
- `assets/views/default/layouts/default.html`
- `assets/static/css/default.css`

---

## Issue 3: Issues Page - View Controls and Gantt Chart

### Current State
- `issues.html:8-43` - view switcher on left
- Table shows issue count: `<p class="text-muted"><span id="visible-count">{{.TotalCount}}</span> issues</p>`
- Gantt chart week labels broken after w3
- Scale/Group dropdowns have no effect

### Problems
- View switcher should be on the right for consistency
- Issue count in table view inconsistent with other pages
- Gantt timeline calculation is broken for weeks beyond 3
- `gantt.html:82` uses `--timeline-days` CSS variable but week offset calculation is wrong

### Fix
1. Move view switcher to right side of header
2. Remove issue count from table view header (or add to all views)
3. Fix Gantt chart week rendering - check `HeaderDates` calculation in handler
4. Ensure Scale/Group query params actually filter data

### Files to Modify
- `assets/views/default/pages/issues.html`
- `assets/views/default/pages/gantt.html`
- `app/web/handler/page.go` (backend Gantt data)

---

## Issue 4: Calendar View - No Interaction on Date Click

### Current State
- `calendar.html:82-105` - calendar days rendered but no click handler
- Only issue links are clickable

### Problems
- Clicking a date does nothing
- Should allow creating an issue on that date or showing a popover

### Fix
1. Add click handler to `.calendar-day` elements
2. On click, either:
   - Open create issue modal with date pre-filled, OR
   - Show popover with issues for that date and "Add issue" button
3. Add `cursor: pointer` to calendar days

### Files to Modify
- `assets/views/default/pages/calendar.html` (add JS)
- `assets/static/css/default.css` (add cursor style)

---

## Issue 5: Kanban Drag & Drop - Improve Animation

### Current State
- `app.js:418-561` - basic HTML5 drag/drop
- `default.css:1597-1617` - minimal card styling
- Card gets `opacity: 0.5` and `rotate(2deg)` when dragging

### Problems
- Drag feels "flickery" - no smooth transition
- Drop indicator is basic (just a colored line)
- No ghost/preview of card while dragging
- Priority styling inconsistent with other views

### Fix - Learn from Linear.app
1. Add smooth CSS transitions for drag states:
   ```css
   .issue-card {
     transition: transform 200ms cubic-bezier(0.2, 0, 0, 1),
                 box-shadow 200ms ease,
                 opacity 200ms ease;
   }
   ```
2. Create proper drag ghost with card preview
3. Animate cards moving up/down when hovering
4. Use spring-like easing for drop animation
5. Standardize priority display: icon + title + color (same as everywhere else)

### Priority Styling Standardization
Current priority in cards (`board.html:87-92`) uses simple SVG
Should use same badge style as dropdown options

### Files to Modify
- `assets/static/js/app.js` (kanban section)
- `assets/static/css/default.css`
- `assets/views/default/pages/board.html`

---

## Issue 6: Status Badge Consistency

### Current State
Status badges appear in multiple places with varying styles:
- `issues.html:168-183` - table cells
- `inbox.html:229` - issue list
- `issue.html:85-104` - property chips
- `board.html:193-209` - modal dropdown
- Form dropdowns

### Problems
- Some show icon + text, some just text
- Colors are consistent but presentation varies
- In form values vs dropdowns, styling differs

### Fix
Create a single reusable status badge component pattern:
```html
<span class="status-badge {{lower .Name}}">
  <!-- Status Icon SVG -->
  {{.Name}}
</span>
```

Apply this EVERYWHERE:
- Table cells
- Inbox list items
- Property chips in issue detail
- Dropdown options
- Form selected values

### Files to Modify
All template files using status badges

---

## Issue 7: Inbox Create Issue Form - Border on Focus

### Current State
- `inbox.html:36-48` - title and description inputs
- Uses `create-box-title` and `create-box-description` classes
- `default.css` line ~500 - `.form-control-borderless:focus` adds border-bottom

### Problems
- Focus state shows border which looks jarring
- Should be seamless/borderless for modern look

### Fix
1. Remove border-bottom on focus for these specific inputs
2. Use subtle background color change instead:
   ```css
   .create-box-title:focus,
   .create-box-description:focus {
     box-shadow: none;
     border: none;
     background: hsl(var(--muted) / 0.3);
   }
   ```

### Files to Modify
- `assets/static/css/default.css`

---

## Issue 8: Issue View Page Sidebar Redesign

### Current State
- `issue.html:81-279` - right sidebar with property cards
- Status: shows all options as clickable chips
- Priority: shows all options as clickable chips
- Sprint: dropdown but broken
- Assignees: modal popup to add
- Project: static text link
- Created: raw timestamp
- Actions: "More Actions" dropdown

### Problems
- Status/Priority show ALL options - too much visual noise
- Should show single selected value, dropdown to change
- Sprint dropdown doesn't work (handler issue)
- Assignees uses modal instead of inline dropdown
- "Created" shows ugly timestamp, no calendar picker
- "Copy Link" buried in dropdown, should be prominent
- White card backgrounds for each field looks dated

### Fix
1. **Status**: Show only selected value with icon, click to dropdown
2. **Priority**: Same - single value display, dropdown to change
3. **Sprint**: Fix JS handler, show as dropdown
4. **Assignees**: Convert to multi-select dropdown (not modal)
5. **Project**: Dropdown to change project
6. **Created**: Format as "Dec 27, 2024" with optional calendar
7. **Copy Link**: Add prominent button next to issue key
8. **Remove white boxes**: Use borderless property rows

### New Layout
```
[Issue Key] [Copy Link Button]

Status: [Backlog v]
Priority: [High v]
Sprint: [Sprint 1 v]
Assignees: [User1, User2 +]
Project: [Project Name v]
Created: Dec 27, 2024

[Delete Issue - destructive]
```

### Files to Modify
- `assets/views/default/pages/issue.html`
- `assets/static/css/default.css`
- `assets/static/js/app.js`

---

## Issue 9: Comment Box and Activity List Redesign

### Current State
- `issue.html:44-76` - comment form and list
- Form: textarea + submit button below
- List: avatar + content layout

### Problems
- Comment form looks basic/ugly
- Activity list should be ABOVE comment box (like GitHub)
- Inconsistent with other forms in the app

### Fix
1. Move activity/comment list ABOVE the comment form
2. Redesign comment form to match other inputs:
   - Borderless textarea
   - Inline action buttons
   - Consistent with inbox create form style
3. Add avatar next to comment input
4. Format timestamps nicely ("2 hours ago")

### New Layout
```
### Activity

[Avatar] User Name - 2 hours ago
Comment text here...

[Avatar] User Name - 1 day ago
Comment text here...

---

[Avatar] [      Write a comment...      ] [Send]
```

### Files to Modify
- `assets/views/default/pages/issue.html`
- `assets/static/css/default.css`

---

## Issue 10: Inbox Page Tab Styling

### Current State
- `inbox.html:202-214` - tab navigation
- CSS likely gives active tab background color change

### Problems
- Background color change on active tab looks ugly
- Should use subtle indicator (bottom border or text weight)

### Fix
Use Linear-style tabs:
```css
.inbox-tab {
  border-bottom: 2px solid transparent;
  background: transparent;
}
.inbox-tab.is-active {
  border-bottom-color: hsl(var(--foreground));
  font-weight: 500;
}
```

### Files to Modify
- `assets/static/css/default.css`

---

## Issue 11: Color Scheme - Use Consistent Tailwind Gray

### Current State
- CSS uses custom gray scale (lines 20-29):
  ```css
  --gray-50: 210 20% 98%;
  --gray-100: 220 14% 96%;
  /* ... etc */
  ```

### Problems
- Mix of custom grays throughout
- Main content might need lighter background
- Sidebar could be slightly grayer
- Inconsistent gray usage

### Fix
Use Tailwind's **Zinc** color palette consistently:
```css
/* Zinc from Tailwind */
--zinc-50: 240 5% 96%;
--zinc-100: 240 5% 90%;
--zinc-200: 240 6% 84%;
--zinc-300: 240 5% 65%;
--zinc-400: 240 4% 46%;
--zinc-500: 240 4% 35%;
--zinc-600: 240 5% 26%;
--zinc-700: 240 5% 20%;
--zinc-800: 240 4% 16%;
--zinc-900: 240 6% 10%;
--zinc-950: 240 10% 4%;
```

Apply:
- Main content background: `zinc-50`
- Sidebar background: `zinc-100`
- Cards: `white`
- Muted text: `zinc-500`
- Borders: `zinc-200`

### Files to Modify
- `assets/static/css/default.css` (CSS variables section)

---

## Issue 12: Reduce Custom CSS, Use Tailwind Utilities

### Current State
- `default.css` is 2000+ lines
- Many utility classes defined that duplicate Tailwind
- Custom margin/padding/flex classes

### Problems
- Bloated CSS
- Hard to maintain
- Inconsistent with Tailwind conventions

### Fix
1. Remove duplicate utility classes (`.flex`, `.gap-2`, `.items-center`, etc.)
2. These are already available if using Tailwind CDN
3. Keep only custom component styles
4. Consider adding Tailwind CDN to layout

### Files to Modify
- `assets/static/css/default.css`
- `assets/views/default/layouts/default.html` (add Tailwind CDN)

---

## Issue 13: Remove All Rounded Corners

### Current State
- `default.css` defines border-radius:
  ```css
  --radius-sm: 0.375rem;
  --radius: 0.5rem;
  --radius-md: 0.5rem;
  --radius-lg: 0.75rem;
  ```
- Used throughout: cards, buttons, inputs, badges, modals

### Problems
- Rounded corners are overused, look dated
- Inconsistent radius values

### Fix
Set all radius to 0 or minimal values:
```css
--radius-sm: 0;
--radius: 0;
--radius-md: 0;
--radius-lg: 0;
--radius-xl: 0;
--radius-full: 0; /* Keep this for avatars only */
```

**Exception**: Keep rounded for:
- Avatars (circle)
- Status/priority badges (pill shape for legibility)

### Files to Modify
- `assets/static/css/default.css`

---

## Implementation Order

1. **Phase 1 - Foundation**
   - Update color scheme to Zinc
   - Remove rounded corners
   - Reduce custom CSS

2. **Phase 2 - Layout**
   - Remove topbar
   - Relocate user menu to sidebar
   - Fix sidebar styling

3. **Phase 3 - Components**
   - Standardize status badges
   - Standardize priority badges
   - Fix form input focus states

4. **Phase 4 - Pages**
   - Fix issues list view layout
   - Fix Gantt chart rendering
   - Add calendar click handlers
   - Improve Kanban drag/drop

5. **Phase 5 - Issue Detail**
   - Redesign sidebar
   - Redesign comment section
   - Fix dropdown behaviors

6. **Phase 6 - Polish**
   - Fix Inbox tabs
   - Add animations
   - Final consistency pass

---

## CSS Variable Updates Required

```css
:root {
  /* Colors - Switch to Zinc */
  --background: 240 5% 96%;  /* zinc-50 */
  --foreground: 240 6% 10%;  /* zinc-900 */

  --card: 0 0% 100%;  /* white */
  --card-foreground: 240 6% 10%;

  --muted: 240 5% 90%;  /* zinc-100 */
  --muted-foreground: 240 4% 46%;  /* zinc-400 */

  --border: 240 6% 84%;  /* zinc-200 */

  /* Remove rounded corners */
  --radius-sm: 0;
  --radius: 0;
  --radius-md: 0;
  --radius-lg: 2px;  /* subtle for cards */
  --radius-xl: 0;
}
```

---

## Implementation Summary

The following changes have been implemented:

### Completed Changes

#### 1. Color Scheme - Zinc Palette
- Updated CSS variables to use Tailwind Zinc colors
- Background: `zinc-50`, Sidebar: `zinc-100`, Cards: white
- All grays now consistent throughout the application

#### 2. Removed Rounded Corners
- Set `--radius-sm`, `--radius`, `--radius-md`, `--radius-xl` to 0
- Kept `--radius-lg: 2px` for subtle card borders
- Preserved `--radius-full` for avatars only

#### 3. Topbar Removed, User Menu in Sidebar
- Removed entire topbar from `default.html`
- Added sidebar footer with:
  - Search trigger (Cmd+K)
  - New Issue button
  - User avatar and dropdown menu (opens upward)
- Main content now uses full height

#### 4. Issues Page Layout
- Removed "New Issue" button from header (now in sidebar)
- Removed issue count from header
- Moved view switcher to right side of header
- Cleaner, more focused layout

#### 5. Inbox Tabs Styling
- Changed from background color change to bottom border indicator
- Active tab: `border-bottom-color: foreground`, `font-weight: 500`
- Transparent backgrounds for all tabs

#### 6. Inbox Create Form Focus
- Removed border effects on focus for title and description
- Seamless, borderless input experience

#### 7. Kanban Drag & Drop Animations
- Added smooth CSS transitions with cubic-bezier easing
- `cursor: grab/grabbing` states
- Improved drag opacity and scale effects
- Pulsing drop indicator animation
- Cards animate smoothly during reordering

#### 8. Issue View Sidebar Redesign
- Removed white card backgrounds for property rows
- Flat, borderless property list with subtle dividers
- Sidebar now has a card background with border
- Copy link button moved next to issue key
- Comment list moved above comment form
- New inline comment form with avatar

#### 9. Calendar Day Click Interactions
- Added `data-date` attribute to calendar days
- Click handler opens create issue modal with date pre-filled
- Added hover effect and cursor pointer
- Does not interfere with issue link clicks

### Files Modified

- `assets/static/css/default.css` - Color scheme, rounded corners, all styling
- `assets/views/default/layouts/default.html` - Removed topbar, added sidebar footer
- `assets/views/default/pages/issues.html` - Removed New Issue button, moved view switcher
- `assets/views/default/pages/issue.html` - Redesigned sidebar and comments
- `assets/views/default/pages/calendar.html` - Added date click interactions

### Remaining Work (Future)

1. **Gantt Chart Fix** - Week labels broken after w3, Scale/Group dropdowns need backend work
2. **Status Badge Standardization** - Full consistency across all templates
3. **Priority Badge Standardization** - Same icon+text+color pattern everywhere
4. **Sprint Dropdown** - Backend handler needs fixing
5. **Assignees Dropdown** - Convert from modal to inline multi-select
