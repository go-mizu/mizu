# Localbase Dashboard Restyle Plan - Supabase 2025/2026 Design Match

## Overview

This document outlines the comprehensive restyling of the Localbase dashboard to achieve 100% visual parity with the Supabase Studio dashboard (2025/2026 design).

## Key Design Analysis from Supabase Screenshot

### 1. Color Palette

#### Dark Sidebar Colors
```css
--sidebar-bg: #1C1C1C;              /* Main sidebar background */
--sidebar-bg-hover: #2A2A2A;        /* Hover state */
--sidebar-bg-active: rgba(62, 207, 142, 0.15); /* Active item bg */
--sidebar-text: #8B8B8B;            /* Secondary text */
--sidebar-text-active: #FFFFFF;     /* Active/primary text */
--sidebar-border: #2E2E2E;          /* Borders in dark context */
```

#### Brand Colors
```css
--brand: #3ECF8E;                   /* Supabase green */
--brand-hover: #2DB77A;             /* Hover state */
--brand-muted: rgba(62, 207, 142, 0.15); /* Muted backgrounds */
```

#### Content Area Colors
```css
--bg-primary: #FFFFFF;              /* Main content background */
--bg-secondary: #F8F9FA;            /* Secondary surfaces */
--bg-tertiary: #F1F3F5;             /* Tertiary/hover states */
--border-default: #E6E8EB;          /* Default borders */
--border-muted: #EAEAEA;            /* Muted borders */
```

#### Text Colors
```css
--text-primary: #1C1C1C;            /* Primary text */
--text-secondary: #666666;          /* Secondary text */
--text-muted: #888888;              /* Muted text */
--text-placeholder: #A1A1AA;        /* Placeholder text */
```

### 2. Typography

- **Font Family**: Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif
- **Monospace**: 'Source Code Pro', Menlo, Monaco, monospace
- **Base Size**: 14px (0.875rem)
- **Small Size**: 12px (0.75rem)
- **Weights**: 400 (regular), 500 (medium), 600 (semibold)

### 3. Spacing & Sizing

- **Border Radius**: 4px (sm), 6px (md), 8px (lg)
- **Sidebar Width**: 250px (expanded), 70px (collapsed)
- **Header Height**: 48px
- **Button Heights**: 28px (xs), 32px (sm), 36px (md)

---

## Components to Restyle

### Phase 1: Core Layout (High Priority)

#### 1.1 Sidebar (`Sidebar.tsx`)
**Current**: Light background (#FAFAFA)
**Target**: Dark background (#1C1C1C)

Changes:
- Dark background with dark borders
- Light text on dark background
- Green accent for active items
- Hover states with subtle lightening
- Logo section with green gradient icon
- "FREE"/"Local" badge styling
- Section dividers
- Collapse toggle styling

#### 1.2 Header (`Header.tsx`)
**Current**: Basic breadcrumb
**Target**: Full breadcrumb with dropdowns, badges, action icons

Changes:
- Organization/Project breadcrumb with dropdown menus
- Status badges (PRODUCTION, FREE, Local)
- Connect button with green accent
- Search shortcut (Cmd+K)
- Feedback, Help, Notifications icons
- Settings gear icon
- User avatar

#### 1.3 App Shell (`App.tsx`)
Changes:
- Ensure proper background colors
- Transition animations for sidebar collapse

---

### Phase 2: Table Editor (High Priority)

#### 2.1 TableEditor Page (`TableEditor.tsx`)

**Left Panel (Table List)**:
- Dark-ish surface background
- "Table Editor" title
- Schema dropdown selector
- "New table" button
- Search tables input
- Table list with icons and row counts
- Table icon grid pattern

**Main Content Area**:
- Tabs bar with table name tab
- Close button on tab
- "+" button for new table tab

**Toolbar**:
- Filter button (outline)
- Sort button (outline)
- Insert button (filled green with checkmark icon)
- Separator
- RLS policies button
- Index Advisor button
- Enable Realtime button with toggle
- Role dropdown (postgres)
- View toggle icons

**Data Grid**:
- Checkbox column (sticky)
- Row number column
- Column headers with:
  - Key icon for primary key
  - Column name (bold)
  - Data type (gray)
  - Sort indicator
- Cell styling:
  - Text truncation with ellipsis
  - NULL values styled italic/gray
  - Hover highlighting
- Pagination footer:
  - Page navigation arrows
  - Page input
  - "of X" indicator
  - Rows per page selector
  - Total records count
  - "Data | Definition" tab toggle

---

### Phase 3: Other Pages (Medium Priority)

#### 3.1 SQL Editor (`SQLEditor.tsx`)
- Monaco editor container styling
- Query tabs
- Run button styling
- Results panel
- Query templates sidebar

#### 3.2 Project Overview (`ProjectOverview.tsx`)
- Stat cards with hover effects
- Service health indicators
- Activity feed styling
- Quick actions cards
- Getting started checklist

#### 3.3 Authentication Users (`Users.tsx`)
- User list table
- User detail panel
- Action buttons

#### 3.4 Storage (`Storage.tsx`)
- File browser styling
- Bucket list
- Upload dropzone
- File/folder icons

#### 3.5 Realtime (`Realtime.tsx`)
- Channel list
- Connection status indicators
- Message inspector

#### 3.6 Functions (`Functions.tsx`)
- Function list
- Deploy button
- Logs panel

#### 3.7 Database Pages
- Policies, Indexes, Views, Triggers, Roles
- Schema Visualizer (graph styling)

#### 3.8 Logs Explorer (`LogsExplorer.tsx`)
- Log table
- Filters
- Time range selector

#### 3.9 Settings (`Settings.tsx`)
- Settings form
- Danger zone styling

---

### Phase 4: Common Components

#### 4.1 DataTable (`DataTable.tsx`)
- Header styling
- Row hover
- Selection checkbox
- Pagination controls
- Empty state

#### 4.2 EmptyState (`EmptyState.tsx`)
- Icon styling
- Text hierarchy
- Action button

#### 4.3 PageContainer (`PageContainer.tsx`)
- Header section
- Description text
- Action buttons area

#### 4.4 SearchInput (`SearchInput.tsx`)
- Input styling
- Icon positioning

---

## CSS Variable Updates (`supabase-theme.css`)

### New Variables to Add

```css
:root {
  /* Dark Sidebar Theme */
  --supabase-sidebar-bg: #1C1C1C;
  --supabase-sidebar-bg-hover: #2A2A2A;
  --supabase-sidebar-bg-active: rgba(62, 207, 142, 0.15);
  --supabase-sidebar-text: #8B8B8B;
  --supabase-sidebar-text-hover: #BBBBBB;
  --supabase-sidebar-text-active: #FFFFFF;
  --supabase-sidebar-border: #2E2E2E;
  --supabase-sidebar-divider: #333333;

  /* Enhanced Badge Colors */
  --supabase-badge-production: #F59E0B;
  --supabase-badge-production-bg: rgba(245, 158, 11, 0.15);
  --supabase-badge-free: #6366F1;
  --supabase-badge-free-bg: rgba(99, 102, 241, 0.15);

  /* Table Editor Specific */
  --supabase-table-header-bg: #F8F9FA;
  --supabase-table-row-hover: #F1F3F5;
  --supabase-table-row-selected: rgba(62, 207, 142, 0.1);
  --supabase-table-border: #E6E8EB;

  /* Toolbar */
  --supabase-toolbar-bg: #FFFFFF;
  --supabase-toolbar-border: #E6E8EB;
}
```

### Updated Component Styles

```css
/* Dark Sidebar NavLink Override */
.mantine-AppShell-navbar {
  background-color: var(--supabase-sidebar-bg) !important;
  border-right-color: var(--supabase-sidebar-border) !important;
}

.mantine-AppShell-navbar .mantine-NavLink-root {
  color: var(--supabase-sidebar-text);
}

.mantine-AppShell-navbar .mantine-NavLink-root:hover {
  background-color: var(--supabase-sidebar-bg-hover);
  color: var(--supabase-sidebar-text-hover);
}

.mantine-AppShell-navbar .mantine-NavLink-root[data-active] {
  background-color: var(--supabase-sidebar-bg-active);
  color: var(--supabase-sidebar-text-active);
  border-left: 2px solid var(--supabase-brand);
}
```

---

## Implementation Order

1. **supabase-theme.css** - Add new CSS variables and dark sidebar styles
2. **Sidebar.tsx** - Implement dark theme with new styling
3. **Header.tsx** - Update breadcrumb and badges
4. **TableEditor.tsx** - Comprehensive toolbar and data grid restyle
5. **All other pages** - Apply consistent styling
6. **Common components** - Update shared components

---

## Testing Checklist

- [ ] Sidebar appears dark with proper contrast
- [ ] Active nav items have green accent
- [ ] Header breadcrumbs work correctly
- [ ] Table Editor matches screenshot layout
- [ ] Data grid has proper column types display
- [ ] Pagination works and looks correct
- [ ] All buttons have correct variant styling
- [ ] Hover states work throughout
- [ ] Modal styling is consistent
- [ ] Form inputs have proper focus states
- [ ] Responsive behavior maintained

---

## Files to Modify

### Styles
- `src/styles/supabase-theme.css` - Core theme variables and overrides

### Layout
- `src/components/layout/Sidebar.tsx` - Dark sidebar implementation
- `src/components/layout/Header.tsx` - Header breadcrumbs and actions
- `src/components/layout/PageContainer.tsx` - Page wrapper updates
- `src/App.tsx` - Shell configuration

### Pages
- `src/pages/database/TableEditor.tsx` - Complete restyle
- `src/pages/database/SQLEditor.tsx` - Editor styling
- `src/pages/project-overview/ProjectOverview.tsx` - Dashboard cards
- `src/pages/auth/Users.tsx` - User management
- `src/pages/storage/Storage.tsx` - File browser
- `src/pages/realtime/Realtime.tsx` - Realtime panel
- `src/pages/functions/Functions.tsx` - Functions list
- `src/pages/database/Policies.tsx` - Policies table
- `src/pages/database/Indexes.tsx` - Indexes table
- `src/pages/database/Views.tsx` - Views table
- `src/pages/database/Triggers.tsx` - Triggers table
- `src/pages/database/Roles.tsx` - Roles table
- `src/pages/database/SchemaVisualizer/` - Graph styling
- `src/pages/logs/LogsExplorer.tsx` - Logs table
- `src/pages/settings/Settings.tsx` - Settings form
- `src/pages/advisors/Advisors.tsx` - Advisors panel
- `src/pages/integrations/Integrations.tsx` - Integrations list
- `src/pages/ApiDocs.tsx` - API docs viewer

### Components
- `src/components/common/DataTable.tsx` - Table component
- `src/components/common/EmptyState.tsx` - Empty state
- `src/components/common/ConfirmModal.tsx` - Modal styling
- `src/components/common/StatusBadge.tsx` - Badge styling
- `src/components/forms/SearchInput.tsx` - Search input
