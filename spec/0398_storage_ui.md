# Storage UI Enhancement Spec

**Spec ID:** 0398
**Date:** 2026-01-17
**Status:** Implementation

## Overview

This spec documents the design research and implementation plan for enhancing the Localbase Storage UI to match Supabase Studio 2025/2026 design patterns, including light theme sidebar, modern icons, and a comprehensive file preview panel with metadata display.

## Design Research: Supabase Studio 2025/2026

### Reference Analysis

Based on analysis of Supabase Studio screenshots (January 2026):

#### 1. Sidebar Design - Light Theme

Supabase has transitioned from a dark sidebar to a **light theme sidebar**:

```css
/* Old (Dark) - DEPRECATED */
--supabase-sidebar-bg: #1C1C1C;

/* New (Light) - 2025/2026 */
--supabase-sidebar-bg: #FFFFFF;
--supabase-sidebar-bg-hover: #F5F5F5;
--supabase-sidebar-bg-active: rgba(62, 207, 142, 0.1);
--supabase-sidebar-text: #6B6B6B;
--supabase-sidebar-text-hover: #3F3F46;
--supabase-sidebar-text-active: #1C1C1C;
--supabase-sidebar-border: #E6E8EB;
--supabase-sidebar-divider: #E6E8EB;
--supabase-sidebar-icon: #9CA3AF;
--supabase-sidebar-icon-active: #3ECF8E;
```

Key visual changes:
- White/light gray background
- Muted gray text with dark text on hover/active
- Green accent for active items (left border + icon color)
- Subtle dividers in light gray

#### 2. Storage UI Layout (Miller Column)

Supabase Storage uses a 3-column layout:

```
+------------------+------------------------+------------------+
| Sidebar (MANAGE) | File Browser (Miller)  | Preview Panel    |
| - Files          | - Breadcrumb nav       | - File preview   |
| - Analytics      | - Folder tree view     | - Metadata       |
| - Vectors        | - File list            | - Actions        |
| CONFIGURATION    |                        |                  |
| - S3             |                        |                  |
+------------------+------------------------+------------------+
```

#### 3. File Preview Panel

When a file is selected, a right-side panel displays:

- **Preview Area**: Visual preview based on file type
  - Images: Rendered image
  - Video: Video player
  - Audio: Audio player
  - Text/Code: Syntax-highlighted content
  - Markdown: Rendered markdown
  - Documents: PDF viewer or icon

- **Metadata Section**:
  - File name
  - MIME type + file size (e.g., "image/jpeg - 129.19 KB")
  - Added on: datetime
  - Last modified: datetime

- **Actions**:
  - Download button
  - Get URL dropdown (public URL, signed URL)
  - Delete file button

#### 4. Icon System

Supabase uses consistent, modern icons:
- Tabler Icons with stroke width 1.5
- Size: 18px for navigation, 16px for actions
- Muted gray color (#9CA3AF) for inactive
- Brand green (#3ECF8E) for active/selected

## Implementation Plan

### Phase 1: Backend Enhancements

#### 1.1 Fix ListObjects Response

Current response is missing critical fields. Update to include:

```go
// Storage handler - ListObjects response
map[string]any{
    "id":           obj.ID,
    "name":         name,
    "bucket_id":    obj.BucketID,
    "owner":        obj.Owner,
    "content_type": obj.ContentType,  // ADD
    "size":         obj.Size,          // ADD
    "created_at":   obj.CreatedAt,
    "updated_at":   obj.UpdatedAt,
    "metadata":     obj.Metadata,
}
```

#### 1.2 Enhance GetObjectInfo Response

Return comprehensive metadata:

```go
map[string]any{
    "id":               obj.ID,
    "name":             filepath.Base(obj.Name),
    "bucket_id":        obj.BucketID,
    "owner":            obj.Owner,
    "content_type":     obj.ContentType,
    "size":             obj.Size,
    "version":          obj.Version,
    "created_at":       obj.CreatedAt,
    "updated_at":       obj.UpdatedAt,
    "last_accessed_at": obj.LastAccessedAt,
    "metadata":         obj.Metadata,
}
```

### Phase 2: Theme Updates

#### 2.1 Light Sidebar Theme

Update CSS variables in `supabase-theme.css`:

```css
/* Light Sidebar Theme (2025/2026) */
--supabase-sidebar-bg: #FFFFFF;
--supabase-sidebar-bg-hover: #F5F5F5;
--supabase-sidebar-bg-active: rgba(62, 207, 142, 0.1);
--supabase-sidebar-text: #6B6B6B;
--supabase-sidebar-text-hover: #3F3F46;
--supabase-sidebar-text-active: #1C1C1C;
--supabase-sidebar-border: #E6E8EB;
--supabase-sidebar-divider: #E6E8EB;
--supabase-sidebar-icon: #9CA3AF;
--supabase-sidebar-icon-active: #3ECF8E;
```

#### 2.2 Update AppShell CSS

```css
.mantine-AppShell-navbar {
  background-color: var(--supabase-sidebar-bg) !important;
  border-right: 1px solid var(--supabase-sidebar-border) !important;
}
```

### Phase 3: Storage UI Enhancement

#### 3.1 Storage Page Layout

Update Storage.tsx with 3-panel layout:

```tsx
<Box style={{ display: 'flex', height: '100%' }}>
  {/* Left: Storage Navigation */}
  <StorageSidebar />

  {/* Center: File Browser */}
  <FileBrowser />

  {/* Right: Preview Panel (conditional) */}
  {selectedFile && <FilePreviewPanel file={selectedFile} />}
</Box>
```

#### 3.2 File Preview Component

Create `FilePreviewPanel.tsx`:

```tsx
interface FilePreviewPanelProps {
  file: StorageObject;
  bucket: Bucket;
  onClose: () => void;
  onDownload: () => void;
  onDelete: () => void;
  onCopyUrl: () => void;
}

function FilePreviewPanel({ file, bucket, onClose, ... }) {
  return (
    <Box style={{ width: 320, borderLeft: '1px solid ...' }}>
      {/* Close button */}
      <Header>
        <CloseButton onClick={onClose} />
      </Header>

      {/* Preview Area */}
      <FilePreview file={file} bucket={bucket} />

      {/* Metadata */}
      <MetadataSection file={file} />

      {/* Actions */}
      <ActionButtons ... />
    </Box>
  );
}
```

#### 3.3 File Preview by Type

```tsx
function FilePreview({ file, bucket }: { file: StorageObject; bucket: Bucket }) {
  const contentType = file.content_type || '';

  // Image preview
  if (contentType.startsWith('image/')) {
    return <ImagePreview file={file} bucket={bucket} />;
  }

  // Video preview
  if (contentType.startsWith('video/')) {
    return <VideoPreview file={file} bucket={bucket} />;
  }

  // Audio preview
  if (contentType.startsWith('audio/')) {
    return <AudioPreview file={file} bucket={bucket} />;
  }

  // Text/Code preview
  if (isTextFile(contentType, file.name)) {
    return <CodePreview file={file} bucket={bucket} />;
  }

  // Markdown preview
  if (file.name.endsWith('.md')) {
    return <MarkdownPreview file={file} bucket={bucket} />;
  }

  // PDF preview
  if (contentType === 'application/pdf') {
    return <PDFPreview file={file} bucket={bucket} />;
  }

  // Default: File icon
  return <FileIconPreview file={file} />;
}
```

#### 3.4 Supported Preview Types

| File Type | Extension | Preview Method |
|-----------|-----------|----------------|
| Images | .jpg, .png, .gif, .webp, .svg | `<img>` element |
| Video | .mp4, .webm, .mov | `<video>` element |
| Audio | .mp3, .wav, .ogg | `<audio>` element |
| Text | .txt, .log | Plain text display |
| Source Code | .js, .ts, .py, .go, .rs, etc. | Syntax highlighted |
| Markdown | .md | Rendered markdown |
| JSON | .json | Formatted JSON |
| PDF | .pdf | PDF.js viewer |
| Archives | .zip, .tar, .gz | File list (future) |
| Other | * | Generic file icon |

### Phase 4: Icon System

#### 4.1 File Type Icons

```tsx
function getFileIcon(file: StorageObject) {
  const contentType = file.content_type || '';
  const ext = file.name.split('.').pop()?.toLowerCase();

  // By content type
  if (contentType.startsWith('image/')) return IconPhoto;
  if (contentType.startsWith('video/')) return IconVideo;
  if (contentType.startsWith('audio/')) return IconMusic;
  if (contentType === 'application/pdf') return IconFileTypePdf;

  // By extension
  switch (ext) {
    case 'js':
    case 'ts':
    case 'jsx':
    case 'tsx':
      return IconBrandJavascript;
    case 'py':
      return IconBrandPython;
    case 'go':
      return IconBrandGolang;
    case 'rs':
      return IconBrandRust;
    case 'md':
      return IconMarkdown;
    case 'json':
      return IconBraces;
    case 'html':
      return IconBrandHtml5;
    case 'css':
      return IconBrandCss3;
    case 'zip':
    case 'tar':
    case 'gz':
      return IconFileZip;
    default:
      return IconFile;
  }
}
```

## API Compatibility

### Supabase Storage API Endpoints

| Endpoint | Method | Status |
|----------|--------|--------|
| `/storage/v1/bucket` | GET | Implemented |
| `/storage/v1/bucket` | POST | Implemented |
| `/storage/v1/bucket/:id` | GET | Implemented |
| `/storage/v1/bucket/:id` | PUT | Implemented |
| `/storage/v1/bucket/:id` | DELETE | Implemented |
| `/storage/v1/object/:bucket/:path` | POST | Implemented |
| `/storage/v1/object/:bucket/:path` | GET | Implemented |
| `/storage/v1/object/:bucket/:path` | PUT | Implemented |
| `/storage/v1/object/:bucket/:path` | DELETE | Implemented |
| `/storage/v1/object/list/:bucket` | POST | Enhanced |
| `/storage/v1/object/move` | POST | Implemented |
| `/storage/v1/object/copy` | POST | Implemented |
| `/storage/v1/object/sign/:bucket/:path` | POST | Implemented |
| `/storage/v1/object/info/:bucket/:path` | GET | Enhanced |
| `/storage/v1/object/public/:bucket/:path` | GET | Implemented |

## File Structure

```
app/frontend/src/
├── pages/storage/
│   ├── Storage.tsx              # Main storage page
│   ├── components/
│   │   ├── StorageSidebar.tsx   # MANAGE/CONFIGURATION sidebar
│   │   ├── FileBrowser.tsx      # File listing with Miller columns
│   │   ├── FilePreviewPanel.tsx # Right-side preview panel
│   │   ├── FilePreview.tsx      # Preview renderer by type
│   │   ├── ImagePreview.tsx     # Image preview component
│   │   ├── VideoPreview.tsx     # Video preview component
│   │   ├── AudioPreview.tsx     # Audio preview component
│   │   ├── CodePreview.tsx      # Syntax-highlighted code
│   │   ├── MarkdownPreview.tsx  # Markdown renderer
│   │   └── FileMetadata.tsx     # Metadata display
│   └── utils/
│       ├── fileTypes.ts         # File type detection
│       └── formatters.ts        # Size/date formatters
```

## Phase 5: Fix UUID/Data Type Display in Table Editor

### Problem

UUIDs are displayed as byte arrays `[115,108,75,80,217,27,74,46,...]` instead of formatted UUID strings like `6f6c4b50-d91b-2a2e-a26c-c2bf5521774f`.

### Root Cause

The pgx driver's `rows.Values()` returns raw PostgreSQL types. UUIDs come through as `[16]byte` arrays, which JSON serializes to number arrays.

### Solution: Backend Type Conversion

Add type conversion in `store/postgres/database.go`:

```go
import (
    "github.com/jackc/pgx/v5/pgtype"
)

// convertValue converts PostgreSQL-specific types to JSON-serializable types
func convertValue(v interface{}) interface{} {
    if v == nil {
        return nil
    }

    switch val := v.(type) {
    case [16]byte:
        // UUID - convert to string format
        return formatUUID(val)
    case []byte:
        // Check if it's a UUID (16 bytes)
        if len(val) == 16 {
            var arr [16]byte
            copy(arr[:], val)
            return formatUUID(arr)
        }
        // Otherwise return as string
        return string(val)
    case pgtype.UUID:
        if val.Valid {
            return formatUUID(val.Bytes)
        }
        return nil
    case time.Time:
        return val.Format(time.RFC3339Nano)
    case pgtype.Timestamp:
        if val.Valid {
            return val.Time.Format(time.RFC3339Nano)
        }
        return nil
    case pgtype.Timestamptz:
        if val.Valid {
            return val.Time.Format(time.RFC3339Nano)
        }
        return nil
    case pgtype.Numeric:
        if val.Valid {
            f, _ := val.Float64Value()
            return f.Float64
        }
        return nil
    default:
        return v
    }
}

// formatUUID formats a 16-byte UUID as a string
func formatUUID(b [16]byte) string {
    return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
        binary.BigEndian.Uint32(b[0:4]),
        binary.BigEndian.Uint16(b[4:6]),
        binary.BigEndian.Uint16(b[6:8]),
        binary.BigEndian.Uint16(b[8:10]),
        b[10:16])
}
```

Apply in Query function:

```go
for rows.Next() {
    values, err := rows.Values()
    if err != nil {
        return nil, err
    }

    row := make(map[string]interface{})
    for i, col := range columns {
        row[col] = convertValue(values[i])  // Convert types
    }
    results = append(results, row)
}
```

### Frontend Fallback

For cases where backend conversion isn't applied, add frontend fallback:

```typescript
// Format UUID from byte array
const formatUUID = (bytes: number[]): string => {
  const hex = bytes.map(b => b.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0,8)}-${hex.slice(8,12)}-${hex.slice(12,16)}-${hex.slice(16,20)}-${hex.slice(20)}`;
};

// Enhanced formatCellValue
const formatCellValue = (value: any, columnType?: string): string => {
  if (value === null) return 'NULL';
  if (typeof value === 'boolean') return value ? 'true' : 'false';

  // Handle UUID byte arrays
  if (Array.isArray(value) && value.length === 16 && value.every(v => typeof v === 'number' && v >= 0 && v <= 255)) {
    return formatUUID(value);
  }

  // Format timestamps nicely
  if (columnType?.includes('timestamp') && typeof value === 'string') {
    try {
      return new Date(value).toLocaleString();
    } catch {
      return value;
    }
  }

  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
};
```

---

## Phase 6: Enhanced Seeding with Realistic Data

### 6.1 Database Tables to Create

```sql
-- Comments table with nested structure
CREATE TABLE IF NOT EXISTS public.comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID REFERENCES public.posts(id) ON DELETE CASCADE,
    author_id UUID REFERENCES auth.users(id),
    parent_id UUID REFERENCES public.comments(id),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Products table with various data types
CREATE TABLE IF NOT EXISTS public.products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    stock INTEGER DEFAULT 0,
    category VARCHAR(100),
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Orders table
CREATE TABLE IF NOT EXISTS public.orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id),
    status VARCHAR(50) DEFAULT 'pending',
    total DECIMAL(10,2),
    items JSONB NOT NULL,
    shipping_address JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Tags table
CREATE TABLE IF NOT EXISTS public.tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    color VARCHAR(7),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Post-tags junction table
CREATE TABLE IF NOT EXISTS public.post_tags (
    post_id UUID REFERENCES public.posts(id) ON DELETE CASCADE,
    tag_id UUID REFERENCES public.tags(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);

-- Order items junction table
CREATE TABLE IF NOT EXISTS public.order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID REFERENCES public.orders(id) ON DELETE CASCADE,
    product_id UUID REFERENCES public.products(id),
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Test users table (public, separate from auth.users)
CREATE TABLE IF NOT EXISTS public.test_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE,
    display_name VARCHAR(255),
    avatar_url TEXT,
    bio TEXT,
    website VARCHAR(255),
    social_links JSONB DEFAULT '{}',
    preferences JSONB DEFAULT '{}',
    is_verified BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 6.2 Realistic Sample Data

```sql
-- Insert tags
INSERT INTO public.tags (name, slug, color) VALUES
('Technology', 'technology', '#3B82F6'),
('Design', 'design', '#EC4899'),
('Business', 'business', '#10B981'),
('Tutorial', 'tutorial', '#F59E0B'),
('News', 'news', '#6366F1'),
('Open Source', 'open-source', '#8B5CF6'),
('Database', 'database', '#EF4444'),
('Frontend', 'frontend', '#06B6D4'),
('Backend', 'backend', '#84CC16'),
('DevOps', 'devops', '#F97316');

-- Insert products
INSERT INTO public.products (name, description, price, stock, category, tags, metadata) VALUES
('Wireless Headphones Pro', 'Premium noise-canceling wireless headphones with 40hr battery', 299.99, 50, 'Electronics',
 ARRAY['audio', 'wireless', 'premium'], '{"brand": "AudioMax", "warranty": "2 years", "weight": "250g"}'),
('Ergonomic Keyboard', 'Split mechanical keyboard with Cherry MX switches', 179.99, 30, 'Electronics',
 ARRAY['keyboard', 'ergonomic', 'mechanical'], '{"switches": "Cherry MX Brown", "layout": "ANSI"}'),
('Standing Desk Pro', 'Electric height-adjustable desk with memory presets', 599.99, 15, 'Furniture',
 ARRAY['desk', 'standing', 'electric'], '{"max_height": "48 inches", "weight_capacity": "300 lbs"}'),
('4K Monitor 27"', 'Professional 4K IPS monitor with USB-C', 449.99, 25, 'Electronics',
 ARRAY['monitor', '4k', 'usb-c'], '{"resolution": "3840x2160", "refresh_rate": "60Hz"}'),
('Webcam HD', '1080p webcam with auto-focus and noise reduction', 79.99, 100, 'Electronics',
 ARRAY['webcam', 'streaming', 'video'], '{"resolution": "1080p", "fps": 30}');

-- Insert test users
INSERT INTO public.test_users (email, username, display_name, bio, website, is_verified) VALUES
('alice@example.com', 'alice', 'Alice Johnson', 'Full-stack developer passionate about open source', 'https://alice.dev', true),
('bob@example.com', 'bob', 'Bob Smith', 'UX designer and product enthusiast', 'https://bobsmith.design', true),
('charlie@example.com', 'charlie', 'Charlie Brown', 'DevOps engineer | K8s | Terraform', NULL, false),
('diana@example.com', 'diana', 'Diana Prince', 'Tech lead at StartupCo', 'https://diana.io', true),
('eve@example.com', 'eve', 'Eve Wilson', 'Backend developer | Go | Rust', NULL, false);

-- Insert posts
INSERT INTO public.posts (author_id, title, content, published)
SELECT u.id, 'Getting Started with Supabase',
'Supabase is an open source Firebase alternative. This guide will help you get started with authentication, database, and storage.

## Key Features

1. **Authentication** - Built-in auth with social providers
2. **Database** - PostgreSQL with real-time subscriptions
3. **Storage** - S3-compatible object storage
4. **Edge Functions** - Serverless functions at the edge

Let''s dive in!', true
FROM auth.users u WHERE u.email = 'admin@localbase.dev';

INSERT INTO public.posts (author_id, title, content, published)
SELECT u.id, 'Building Real-time Applications',
'Learn how to build real-time collaborative features using Supabase Realtime and PostgreSQL LISTEN/NOTIFY.

## Prerequisites

- Node.js 18+
- Supabase account
- Basic PostgreSQL knowledge

## Getting Started

First, enable realtime on your table...', true
FROM auth.users u WHERE u.email = 'admin@localbase.dev';

INSERT INTO public.posts (author_id, title, content, published)
SELECT u.id, 'Draft: Advanced RLS Patterns',
'This post covers advanced row-level security patterns including multi-tenancy, hierarchical permissions, and time-based access control.

TODO: Add code examples', false
FROM auth.users u WHERE u.email = 'user@localbase.dev';

-- Insert comments
INSERT INTO public.comments (post_id, author_id, content)
SELECT p.id, u.id, 'Great introduction! This helped me get started quickly.'
FROM public.posts p, auth.users u
WHERE p.title LIKE '%Getting Started%' AND u.email = 'user@localbase.dev';

INSERT INTO public.comments (post_id, author_id, content)
SELECT p.id, u.id, 'Can you add more examples about storage policies?'
FROM public.posts p, auth.users u
WHERE p.title LIKE '%Getting Started%' AND u.email = 'admin@localbase.dev';

-- Insert orders
INSERT INTO public.orders (user_id, status, total, items, shipping_address)
SELECT u.id, 'completed', 479.98,
'[{"product": "Wireless Headphones Pro", "quantity": 1, "price": 299.99}, {"product": "Ergonomic Keyboard", "quantity": 1, "price": 179.99}]'::jsonb,
'{"street": "123 Main St", "city": "San Francisco", "state": "CA", "zip": "94102", "country": "USA"}'::jsonb
FROM auth.users u WHERE u.email = 'admin@localbase.dev';

INSERT INTO public.orders (user_id, status, total, items, shipping_address)
SELECT u.id, 'pending', 599.99,
'[{"product": "Standing Desk Pro", "quantity": 1, "price": 599.99}]'::jsonb,
'{"street": "456 Oak Ave", "city": "New York", "state": "NY", "zip": "10001", "country": "USA"}'::jsonb
FROM auth.users u WHERE u.email = 'user@localbase.dev';

-- Link posts to tags
INSERT INTO public.post_tags (post_id, tag_id)
SELECT p.id, t.id
FROM public.posts p, public.tags t
WHERE p.title LIKE '%Supabase%' AND t.slug IN ('technology', 'tutorial', 'database');

INSERT INTO public.post_tags (post_id, tag_id)
SELECT p.id, t.id
FROM public.posts p, public.tags t
WHERE p.title LIKE '%Real-time%' AND t.slug IN ('technology', 'backend', 'database');
```

### 6.3 Storage Files Structure

Create sample files in buckets:

```
avatars/
├── default.svg (SVG placeholder avatar)
├── user-1.jpg
├── user-2.png
└── team/
    ├── alice.jpg
    └── bob.png

documents/
├── reports/
│   ├── 2024/
│   │   └── annual-report.pdf
│   └── 2025/
│       └── q1-summary.pdf
├── contracts/
│   └── nda-template.docx
└── README.md

public/
├── assets/
│   ├── logo.svg
│   └── favicon.ico
├── downloads/
│   └── user-guide.pdf
└── examples/
    ├── sample.json
    ├── config.yaml
    └── script.py

media/
├── images/
│   ├── hero.jpg
│   └── gallery/
│       ├── photo-001.jpg
│       └── photo-002.png
├── videos/
│   └── intro.mp4
└── audio/
    └── notification.mp3
```

---

## Testing

### E2E Tests to Add

1. File selection shows preview panel
2. Image files display correctly
3. Text files show syntax highlighting
4. Metadata displays accurate information
5. Download button works
6. Get URL dropdown functions
7. Delete confirmation works
8. Light sidebar theme applied
9. **UUID columns display as formatted strings**
10. **All data types render correctly in Table Editor**
11. **Seeded data is visible and correct**

## Sources

- [Supabase Storage](https://supabase.com/storage)
- [Supabase Storage Docs](https://supabase.com/docs/guides/storage)
- [Supabase Storage Schema](https://supabase.com/docs/guides/storage/schema/design)
- [Supabase GitHub](https://github.com/supabase/storage)
