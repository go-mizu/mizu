# UI Enhancement Spec: Match GitHub 100%

## Overview

This document outlines the UI issues identified in GitHome compared to GitHub's actual styling (using golang/go repository as reference) and the fixes required to achieve 100% parity.

## Reference Sources

- GitHub Repository: https://github.com/golang/go
- GitHub Code View: https://github.com/golang/go/blob/master/README.md
- GitHub Commits: https://github.com/golang/go/commits/master/
- GitHub Commit Detail: https://github.com/golang/go/commit/f4cec7917cc53c8c7ef2ea456b4bf0474c41189a

---

## Issues Identified

### 1. Avatar Styling

**Current Issues:**
- Avatar sizes inconsistent
- Border styling not matching GitHub
- AvatarStack overlap not correct (-10px is too much)

**GitHub Reference:**
- Commits list avatars: 20px with 2px white border
- Commit detail author avatar: 20px inline with text
- AvatarStack overlap: -8px (not -10px)
- Border: 2px solid white (--color-canvas-default)

**Fix:**
```css
.AvatarStack .Avatar + .Avatar {
  margin-left: -8px; /* Was -10px */
}
```

---

### 2. Commits List Page

**Current Issues:**
- Commit entry padding inconsistent
- Avatar alignment with content
- Meta text spacing
- SHA pill styling

**GitHub Reference:**
- Commit entry padding: 16px
- Avatar stack aligned to top of content
- Commit message: font-weight 600, color: fg-default
- Meta text: font-size 12px, color: fg-muted, margin-top 4px
- SHA: monospace font, 12px, padding 5px 10px, bg subtle, border-radius 6px

**Fix:**
```css
.commit-entry {
  padding: 16px;
  align-items: flex-start;
}

.commit-sha {
  padding: 5px 10px;
  border-radius: 6px;
  font-size: 12px;
}
```

---

### 3. Commit Detail Page

**Current Issues:**
- Commit title size too large (using 2xl instead of proper sizing)
- Author info layout needs improvement
- SHA display inline styling issues
- Browse files button position

**GitHub Reference:**
- Commit title: 24px (not 32px), font-weight 600
- Author/committer avatars: 20px, inline with text
- Author info format: "author **authored** and **committer** committed X ago"
- Full SHA: monospace, 12px, with copy button
- Parent SHA: link with subtle background, border-radius 6px

**Fix:**
```css
.commit-title {
  font-size: 24px; /* Was 32px */
}

.commit-author-info {
  gap: 6px; /* Was 8px */
}
```

---

### 4. Code View Page

**Current Issues:**
- Line number padding incorrect
- Code content overflow handling
- Long lines causing horizontal scroll on entire table
- Line number column width

**GitHub Reference:**
- Line numbers: width 50px, padding-right 16px, padding-left 16px
- Line numbers: right-aligned, color fg-muted (#656d76)
- Code content: padding-left 16px
- Table: `table-layout: auto` for line numbers, content expands
- Long lines: horizontal scroll only on code content, not line numbers

**Fix:**
```css
.code-line-number {
  width: 50px;
  min-width: 50px;
  padding: 0 16px;
  position: sticky;
  left: 0;
  background-color: var(--color-canvas-subtle);
}

.code-line-content {
  overflow-x: auto;
}
```

---

### 5. File Diff Table

**Current Issues:**
- Line number columns too narrow
- Diff marker (+/-) alignment
- Hunk header styling

**GitHub Reference:**
- Line number width: 50px each column
- Diff marker: 1ch width, same line as content
- Hunk header: light blue background, padding 4px 10px

**Fix:**
```css
.diff-line-num {
  min-width: 50px;
  width: 50px;
}

.diff-hunk .diff-line-code {
  padding: 4px 10px;
}
```

---

### 6. Icon Alignment

**Current Issues:**
- Some octicons not vertically aligned correctly
- Icon spacing inconsistent

**GitHub Reference:**
- Octicons: vertical-align: text-bottom
- Icon + text spacing: 6px (--space-1 is 4px, should use 6px for icons)

**Fix:**
```css
.octicon {
  vertical-align: text-bottom;
}

.mr-1 { margin-right: 6px; } /* For icons only */
```

---

### 7. Breadcrumb Navigation

**Current Issues:**
- Separator spacing
- Font size

**GitHub Reference:**
- Separator: " / " with spaces
- Font size: 14px
- Current file: font-weight 600

**Fix:** Already correct in template, verify CSS.

---

### 8. Box Header Padding

**Current Issues:**
- File header (blob view) padding inconsistent

**GitHub Reference:**
- Box header padding: 8px 16px
- Border-radius on header: 6px top corners

**Fix:**
```css
.Box-header {
  padding: 8px 16px;
}
```

---

## Implementation Plan

### Phase 1: CSS Fixes (main.css)

1. Update Avatar styles:
   - Fix AvatarStack overlap
   - Ensure consistent sizing

2. Update Commits List styles:
   - Commit entry padding
   - SHA pill styling
   - Meta text spacing

3. Update Commit Detail styles:
   - Title font size
   - Author info layout
   - Parent SHA styling

4. Update Code View styles:
   - Line number sticky positioning
   - Code content overflow
   - Table layout

5. Update Diff Table styles:
   - Line number width
   - Hunk header padding

### Phase 2: Template Fixes

1. repo_commits.html:
   - Verify avatar markup
   - Check icon placement

2. commit_detail.html:
   - Author/committer display format
   - SHA copy button placement

3. repo_blob.html:
   - Breadcrumb spacing
   - Header button alignment

---

## CSS Changes Summary

```css
/* Avatar Stack */
.AvatarStack .Avatar + .Avatar {
  margin-left: -8px;
}

/* Commit Entry */
.commit-entry {
  padding: 16px;
}

.commit-sha {
  padding: 5px 10px;
  border-radius: 6px;
}

/* Commit Title */
.commit-title {
  font-size: 24px;
}

/* Code View */
.code-line-number {
  width: 50px;
  min-width: 50px;
  position: sticky;
  left: 0;
  z-index: 1;
}

.code-view {
  overflow-x: auto;
}

.code-table {
  min-width: 100%;
}

/* Diff Table */
.diff-line-num {
  width: 50px;
  min-width: 50px;
}
```

---

## Implementation Status

**Completed: 2025-12-29**

All CSS and template fixes have been implemented:
- Avatar styling fixed (proper sizes, border, overlap)
- Commits list styling fixed (padding, font sizes, colors)
- Commit detail styling fixed (title size, author info layout)
- Code view styling fixed (line numbers, sticky position, overflow)
- Diff table styling fixed (line numbers, code content)

## Testing Checklist

- [x] Commits list page matches GitHub
- [x] Commit detail page matches GitHub
- [x] Code view page matches GitHub
- [x] Avatar stack displays correctly
- [x] Long lines scroll properly in code view
- [x] Diff display matches GitHub styling
- [ ] Dark mode works correctly
- [ ] Responsive layout works

---

## Files to Modify

1. `/assets/static/css/main.css` - Core styling fixes
2. `/assets/views/default/pages/repo_commits.html` - Template adjustments
3. `/assets/views/default/pages/commit_detail.html` - Template adjustments
4. `/assets/views/default/pages/repo_blob.html` - Template adjustments
