# Header and Navigation Enhancement Spec

This document details the comparison between GitHome and GitHub's UI (using golang/go as reference) and the enhancements needed to achieve 100% visual parity.

## 1. Global Header Bar

### Current GitHome
- Dark background (#24292f)
- Logo (octicon) on left
- Search input (when authenticated)
- Links: "Pull requests", "Issues", "Explore"
- Actions: Notifications, Create new (+), User avatar
- Sign in/Sign up buttons when not authenticated

### GitHub (golang/go)
- Same dark background
- Logo with dropdown menu for navigation
- **Search command palette** with keyboard shortcut indicator (Type / to search)
- **Product navigation**: Product, Solutions, Resources, Open Source, Enterprise, Pricing
- **Different authenticated nav**: Pull requests, Issues, Codespaces, Marketplace, Explore
- **Better notification icon** with indicator dot
- **Create new dropdown** with more options
- **User avatar dropdown** with full menu

### Required Enhancements
1. Add keyboard shortcut hint to search ("Type / to search" or "Search or jump to...")
2. Add proper search focus expansion behavior
3. Keep nav links simple for our clone (Pull requests, Issues, Explore is good)
4. Ensure notification bell has proper hover state
5. Improve spacing and alignment to match exactly

---

## 2. Repository Header (Title + Actions)

### Current GitHome
```html
<div class="repo-header-title">
    [lock/book icon] owner / repo-name [Public badge]
</div>
<div class="repo-header-actions">
    [Watch btn] [count] | [Star btn] [count] | [Fork btn] [count]
</div>
```

### GitHub (golang/go)
- **Visibility icon**: Uses specific repo icon (not lock for public)
- **Owner link**: Lighter color, not bold
- **Repo name**: Bold, accent color
- **Visibility label**: Smaller, border-style label
- **Watch/Star/Fork buttons**:
  - Each has dropdown caret for options
  - Counts are in separate connected button segment
  - Has hover states with background change
  - Watch has "Unwatch" state with filled eye icon
  - Star has filled star when starred (yellow)
  - Fork shows "Fork" count separately

### Required Enhancements
1. Add dropdown carets to Watch/Star/Fork buttons
2. Improve button group styling with proper borders
3. Add proper hover states matching GitHub
4. Star button: filled yellow star when starred
5. Watch button: filled eye icon when watching
6. Add proper "Pin" option icon (optional)
7. Button counters should be clickable links to respective pages

---

## 3. UnderlineNav (Repository Tabs)

### Current GitHome
```html
<nav class="UnderlineNav">
    <a href="..." class="UnderlineNav-item selected">
        <span>Code</span>
        <span class="Counter">5</span>
    </a>
</nav>
```
- No icons on tabs
- Orange underline on selected (#fd8c73)
- Basic text only

### GitHub (golang/go)
- **Each tab has an icon** before the text:
  - Code: `<>` code icon
  - Issues: circle-dot issue icon
  - Pull requests: git-pull-request icon
  - Discussions: comment-discussion icon
  - Actions: play icon
  - Projects: table/kanban icon
  - Wiki: book icon
  - Security: shield icon
  - Insights: graph icon
  - Settings: gear icon (only for maintainers)
- **Counter pills** styled consistently
- **Hover effect**: Gray underline appears on hover
- **Selected state**: Font weight 600, orange underline

### Required Enhancements
1. **Add icons to each tab** (Code, Issues, Pull requests, etc.)
2. Ensure proper icon SVGs are used (Octicons)
3. Add hover underline effect (gray, not orange)
4. Improve selected state with proper font weight
5. Counter styling should match GitHub exactly

---

## 4. File Navigation Bar

### Current GitHome
```html
<div class="file-navigation">
    [Branch selector btn] [X branches text]
    [Go to file btn] [Code btn (primary)]
</div>
```

### GitHub (golang/go)
- **Branch/tag selector**:
  - Has branch icon
  - Shows "master" or branch name
  - Dropdown caret
  - Opens modal with tabs (Branches/Tags)
  - Has search within dropdown
- **Branches link**: "X Branches" as clickable link
- **Tags link**: "X Tags" as clickable link (separate from branches)
- **Commit count link**: Shows "X Commits" with history icon
- **Right side**:
  - "Go to file" button (with `t` keyboard shortcut)
  - "Add file" dropdown (with + icon)
  - "Code" button (green, primary) with dropdown for clone options

### Required Enhancements
1. Add branch icon to branch selector
2. Make branches/tags/commits links properly styled and clickable
3. Add tags count link
4. Add proper branch selector dropdown with search
5. "Add file" dropdown button
6. "Code" button should be green and have clone dropdown
7. Add keyboard shortcut hints where applicable

---

## 5. File List Box (Tree View)

### Current GitHome
- Latest commit bar with avatar, author, message, SHA, time
- File rows with folder/file icons, name, commit message, date

### GitHub (golang/go)
- **Latest commit section**:
  - Has avatar (linked to user)
  - Author name (linked, bold)
  - Commit message (linked, truncated)
  - "Verified" badge if GPG signed
  - Short SHA (monospace, linked)
  - Relative time
  - History button/link
- **File list**:
  - Folder icon (blue) for directories
  - File icon (gray) for files
  - Name as link
  - Commit message as link (truncated)
  - Relative date (right-aligned)
  - Subtle hover effect

### Current Status: Good, minor improvements needed
1. Add "History" link/button in commit bar
2. Ensure proper link styling
3. Add verified badge support

---

## 6. File View Header (Blob Page)

### Current GitHome
```html
<div class="Box-header">
    [file icon] filename.go | X lines (X KB)
    [Preview/Code toggle] [Raw] [Blame] [Copy]
</div>
```

### GitHub (golang/go)
- **Left side**:
  - File icon (document icon)
  - File name (bold)
  - Line count text
  - File size in parentheses
- **Right side** (button group):
  - "Preview" / "Code" toggle (for markdown)
  - "Raw" button (opens raw file)
  - "Copy raw file" button (icon only)
  - "Download raw file" button (icon only)
  - "Edit" button (pencil icon, if authorized)
  - More options (...) dropdown
  - "Blame" as separate action
  - Line wrapping toggle

### Required Enhancements
1. Add "Download" button
2. Add "Edit" button (even if just icon, can be disabled)
3. Add more options dropdown (...)
4. Group buttons properly
5. Add copy/download as icon-only buttons
6. Improve button styling to match GitHub exactly

---

## 7. Blame View Header

### Current GitHome
- Similar to blob view but with "Normal" button to go back

### GitHub (golang/go)
- Same structure as blob view
- "Normal view" button to exit blame
- Blame info columns properly styled
- Each blame section has distinct visual grouping

### Current Status: Acceptable, needs button styling alignment

---

## 8. Sidebar (Repository Root - About Section)

### Current GitHome
```html
<div class="Layout-sidebar">
    <h2>About</h2>
    <p>Description</p>
    <a>Homepage link</a>
    <div>Topic tags</div>
    <div>Stars | Watching | Forks links</div>
    <h2>Languages</h2>
    <span>Go (with color dot)</span>
</div>
```

### GitHub (golang/go)
- **About section**:
  - "About" heading with gear icon (for editing, if authorized)
  - Description text (larger font)
  - Website link with external link icon
  - Topics as pill tags (clickable, blue bg on hover)
  - **Resource links** (each with icon):
    - Readme (book icon)
    - License (law/scale icon) with license name
    - Code of conduct (if exists)
    - Security policy (if exists)
    - Stars count with star icon
    - Watching count with eye icon
    - Forks count with fork icon
- **Releases section** (separate):
  - "Releases" heading
  - Latest release with tag name
  - "X Releases" link
- **Packages section** (if applicable)
- **Contributors section**:
  - Avatar stack of top contributors
  - "+ X contributors" link
- **Languages section**:
  - Color bar showing language percentages
  - Language breakdown with colored dots and percentages
- **Suggested workflows** (optional)
- **Used by** section (if popular)

### Required Enhancements
1. Add section icons (gear for About header if editable)
2. **Add Readme link** with book icon
3. **Add License link** with scale icon (show license type like "MIT License")
4. **Add Activity/Stats row**: Stars, Watching, Forks with icons (vertical list style)
5. **Add Releases section** if releases exist
6. **Add Contributors section** with avatar stack
7. **Improve Languages section**:
   - Add horizontal color bar at top showing language percentages
   - Show percentage next to each language
   - Make it collapsible if many languages
8. Better spacing between sections

---

## 9. Link Styling Consistency

### GitHub Patterns
- **Primary links** (accent color): User names, repo names, commit links
- **Secondary links** (muted): Dates, meta info
- **Hover behavior**: Underline on hover for most links
- **No underline by default** for navigation

### Required Enhancements
1. Ensure all owner/repo links use accent color
2. Commit SHAs should be monospace and linked
3. Author names should be bold and linked
4. Dates should be muted color
5. Consistent hover states across all links

---

## 10. Button Styling

### GitHub Patterns
- **Default button**: Light gray background, dark border, dark text
- **Primary button**: Green background (#238636), white text
- **Danger button**: Red text, red bg on hover
- **Button groups**: Connected with shared borders, first/last radius
- **Small buttons**: Reduced padding, smaller font
- **Icon-only buttons**: Square-ish, icon centered
- **Dropdown buttons**: Has caret indicator

### Required Enhancements
1. Button group borders should collapse properly
2. Hover states need subtle background change
3. Focus states need proper outline
4. Icon buttons should have proper sizing
5. Dropdown carets should be properly styled

---

## Summary of Priority Changes

### High Priority
1. Add icons to UnderlineNav tabs
2. Enhance sidebar with Readme/License/Releases/Contributors sections
3. Fix button group styling for Watch/Star/Fork
4. Add proper branch selector dropdown

### Medium Priority
1. File view header button improvements
2. Language bar visualization in sidebar
3. Search input styling improvements
4. Link hover state consistency

### Low Priority
1. Keyboard shortcut hints
2. Additional dropdown menus
3. Edit/settings buttons (placeholder)

---

## Implementation Notes

### Icon Sources (Octicons)
Use GitHub's Octicons library. Key icons needed:
- `code` - Code tab
- `issue-opened` - Issues tab
- `git-pull-request` - Pull requests tab
- `comment-discussion` - Discussions
- `play` - Actions
- `table` - Projects
- `book` - Wiki/Readme
- `shield` - Security
- `graph` - Insights
- `gear` - Settings
- `law` - License
- `eye` - Watch/Watching
- `star` - Star
- `repo-forked` - Fork
- `tag` - Tags/Releases
- `history` - Commits/History
- `people` - Contributors

### CSS Class Naming
Follow GitHub's Primer CSS naming conventions:
- `UnderlineNav-octicon` for tab icons
- `Link--primary`, `Link--secondary`, `Link--muted`
- `Counter`, `Counter--primary`
- `Label`, `Label--secondary`
- `BtnGroup`, `BtnGroup-item`
