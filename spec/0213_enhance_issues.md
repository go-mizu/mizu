# Spec 0213: Enhance Issues List Page to Match GitHub

## Overview

Enhance the GitHome Issues List page to match GitHub's exact design and functionality. This includes visual styling, filter dropdowns, sorting, and functional links/buttons.

## Reference

![GitHub Issues Page](reference: golang/go issues page screenshot)

## Current State

The existing implementation includes:
- Basic issues list with open/closed toggle
- Search bar with basic query support
- Labels and Milestones filter buttons (non-functional)
- Sort dropdown with options
- Issue items showing title, labels, metadata, assignees, comments

## Enhancements Required

### 1. Pinned/Announcement Issue Banner

GitHub shows a highlighted announcement banner at the top of the issues page for pinned issues.

**Implementation:**
- Add `PinnedIssues` field to `RepoIssuesData` struct
- Create service method to get pinned issues for repo
- Add styled banner component in template with:
  - Green open icon
  - Issue title as link
  - Issue number and metadata
  - Comment count

### 2. Enhanced Search Bar with Filter Syntax Display

**Current:** Simple search input with placeholder
**Target:** Show filter syntax "is:issue state:open" as colored badges

**Implementation:**
- Update search input to display current filter state as visual badges
- Add a separate search button icon on the right side
- Style the badges to match GitHub's blue text styling

### 3. Labels & Milestones Filter Buttons with Icons

**Implementation:**
- Add tag icon to Labels button
- Add milestone icon to Milestones button
- Make buttons functional with dropdown menus showing available labels/milestones

### 4. Issues Table Header with Filter Dropdowns

**GitHub shows these in the header row:**
- Open count (left, with icon)
- Closed count (with icon)
- Author dropdown
- Labels dropdown
- Projects dropdown (placeholder)
- Milestones dropdown
- Assignees dropdown
- Types dropdown (placeholder)
- Sort dropdown (shows "Newest" label)

**Implementation:**
- Reorganize header layout to match GitHub
- Add functional dropdown menus for:
  - Author filter
  - Labels filter (multi-select)
  - Milestones filter
  - Assignees filter
- Add Sort dropdown that shows current sort option (e.g., "Newest")

### 5. Issue List Item Enhancements

Each issue row should show:
- State icon (green for open, purple for closed)
- Title as link
- Labels as colored pills
- Metadata: `#number · username opened X hours ago`
- Milestone with icon (if set)
- Comment count on the right (if > 0)

**Implementation:**
- Update issue row layout to match GitHub exactly
- Ensure milestone appears inline after metadata
- Style comment count consistently

### 6. Pagination Enhancements

**Implementation:**
- Add "Previous" and "Next" buttons styled like GitHub
- Show page numbers when multiple pages exist

## Files to Modify

### Templates
- `blueprints/githome/assets/views/default/pages/repo_issues.html` - Main issues list template
- `blueprints/githome/assets/views/default/pages/issue_view.html` - Single issue view (nil checks)

### Styles
- `blueprints/githome/assets/static/css/main.css` - Additional CSS for GitHub-like styling

### Go Code
- `blueprints/githome/app/web/handler/page.go` - Add filter support, sorting, assignees data
- `blueprints/githome/feature/issues/service.go` - Add populateUser to Get method

### Seed Data
- `blueprints/githome/pkg/seed/github/client.go` - Ensure proper data import
- `blueprints/githome/pkg/seed/github/seeder.go` - Add more test data variety

## Implementation Steps

1. **Fix nil pointer issues** (DONE)
   - Add nil checks for User in templates
   - Add populateUser call in Get method

2. **Update repo_issues.html template**
   - Add pinned issue banner section
   - Enhance search bar with filter badges
   - Add icons to filter buttons
   - Reorganize header with filter dropdowns
   - Update issue list item structure
   - Add functional dropdown menus with JavaScript

3. **Update CSS**
   - Add styles for filter badges
   - Add styles for dropdown menus
   - Polish issue list item appearance
   - Add pinned banner styles

4. **Update page handler**
   - Add support for additional filter parameters
   - Pass assignees list to template
   - Support multiple label/assignee filtering

5. **Test thoroughly**
   - Verify all links work
   - Verify filter dropdowns function
   - Verify sorting works
   - Verify pagination works

## Test Data Requirements

To properly test the enhanced issues page:
- Multiple issues with various states (open/closed)
- Issues with labels (variety of colors)
- Issues with milestones
- Issues with assignees
- Issues with comments
- At least one pinned/announcement issue

## Acceptance Criteria

- [ ] Issues page visually matches GitHub reference screenshot
- [ ] All filter dropdowns are functional (Author, Labels, Milestones, Assignees)
- [ ] Sort dropdown shows current sort option and works
- [ ] Issue rows show all metadata correctly
- [ ] Labels display with correct colors and contrast
- [ ] Milestones show with icon
- [ ] Comment counts link to issue comments section
- [ ] Pagination works correctly
- [ ] No JavaScript console errors
- [ ] No Go template errors
- [ ] Mobile responsive design maintained
