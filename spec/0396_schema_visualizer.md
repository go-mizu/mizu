# 0396 Schema Visualizer

## Overview

Schema Visualizer is an interactive Entity Relationship Diagram (ERD) tool that provides a visual representation of database schema structure. It displays tables as interactive cards, shows column details with constraint indicators, and draws relationship lines between tables based on foreign key relationships.

This feature matches Supabase's Schema Visualizer design and functionality.

## Features

### Core Features

1. **Interactive Canvas**
   - Pan and zoom capabilities using mouse/trackpad
   - Drag-and-drop table positioning
   - Smooth animations for transitions

2. **Table Nodes**
   - Displays table name with external link icon
   - Lists all columns with:
     - Column name
     - Data type (uuid, text, int4, timestamptz, etc.)
     - Constraint indicators (Primary Key, Identity, Unique, Nullable, Non-Nullable)
   - Rounded card design with shadow
   - Hover effects for interactivity

3. **Relationship Edges**
   - Connects foreign key columns between tables
   - Dashed lines for visual clarity
   - Smoothstep edge type for readable connections
   - Animated edges on hover

4. **Auto Layout**
   - Dagre-based automatic layout algorithm
   - Positions tables to minimize edge crossings
   - Respects table size and relationships
   - One-click auto layout button

5. **Schema Selector**
   - Dropdown to select database schema (public, auth, storage, etc.)
   - Filters displayed tables by selected schema

6. **Copy as SQL**
   - Generates CREATE TABLE statements for all visible tables
   - Includes column definitions, constraints, and foreign keys
   - Copies to clipboard

7. **Export/Download**
   - Export diagram as PNG image
   - High-resolution export support

8. **Minimap**
   - Bottom-right corner navigation minimap
   - Shows overview of entire diagram
   - Click to navigate to specific areas

9. **Legend**
   - Bottom status bar showing constraint icons:
     - Primary key (key icon)
     - Identity (hash icon)
     - Unique (fingerprint icon)
     - Nullable (empty diamond)
     - Non-Nullable (filled diamond)

### Visual Design

Following Supabase's design language:

- **Table Card Colors:**
  - Background: `#1c1c1c` (dark mode) / `#ffffff` (light mode)
  - Border: `#3c3c3c` (dark mode) / `#e5e5e5` (light mode)
  - Header background: slightly lighter than body

- **Column Row Design:**
  - Constraint icon on left
  - Column name (bold if primary key)
  - Data type in muted color on right
  - Subtle hover effect

- **Relationship Lines:**
  - Dashed stroke
  - Color: `#4a4a4a`
  - Connects to specific column positions

## Technical Implementation

### Frontend Components

```
app/frontend/src/pages/database/
└── SchemaVisualizer/
    ├── index.tsx              # Main page component
    ├── SchemaVisualizer.tsx   # Canvas and controls
    ├── TableNode.tsx          # Custom ReactFlow node
    ├── ColumnRow.tsx          # Column display component
    ├── RelationshipEdge.tsx   # Custom edge component
    ├── Legend.tsx             # Bottom legend bar
    ├── useAutoLayout.ts       # Dagre layout hook
    └── utils.ts               # Helper functions
```

### Dependencies

```json
{
  "@xyflow/react": "^12.0.0",
  "dagre": "^0.8.5",
  "html-to-image": "^1.11.11"
}
```

### API Endpoint

**GET /api/pg/schema-visualization**

Query parameters:
- `schema` (string): Schema name to visualize (default: "public")

Response:
```json
{
  "tables": [
    {
      "id": 12345,
      "schema": "public",
      "name": "users",
      "columns": [
        {
          "name": "id",
          "type": "uuid",
          "is_nullable": false,
          "is_primary_key": true,
          "is_unique": true,
          "is_identity": false,
          "default_value": "gen_random_uuid()"
        }
      ]
    }
  ],
  "relationships": [
    {
      "id": 1,
      "source_schema": "public",
      "source_table": "posts",
      "source_columns": ["user_id"],
      "target_schema": "public",
      "target_table": "users",
      "target_columns": ["id"],
      "constraint_name": "posts_user_id_fkey"
    }
  ]
}
```

### Backend Handler (Go)

Location: `app/web/handler/api/pgmeta.go`

```go
// GetSchemaVisualization returns data for the schema visualizer
func (h *PGMetaHandler) GetSchemaVisualization(c *mizu.Ctx) error {
    schema := c.QueryParam("schema", "public")

    tables, err := h.store.ListTablesWithColumns(ctx, schema)
    if err != nil {
        return err
    }

    relationships, err := h.store.ListRelationships(ctx, []string{schema})
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, map[string]any{
        "tables": tables,
        "relationships": relationships,
    })
}
```

### ReactFlow Node Data Structure

```typescript
interface TableNodeData {
  id: string;
  schema: string;
  name: string;
  columns: Array<{
    name: string;
    type: string;
    isPrimaryKey: boolean;
    isIdentity: boolean;
    isUnique: boolean;
    isNullable: boolean;
  }>;
}

interface RelationshipEdgeData {
  sourceColumn: string;
  targetColumn: string;
  constraintName: string;
}
```

### Auto-Layout Algorithm

Using Dagre for directed graph layout:

```typescript
const layoutNodes = (nodes: Node[], edges: Edge[]) => {
  const g = new dagre.graphlib.Graph();
  g.setGraph({ rankdir: 'LR', nodesep: 100, ranksep: 200 });
  g.setDefaultEdgeLabel(() => ({}));

  nodes.forEach(node => {
    g.setNode(node.id, {
      width: NODE_WIDTH,
      height: calculateNodeHeight(node.data.columns.length)
    });
  });

  edges.forEach(edge => {
    g.setEdge(edge.source, edge.target);
  });

  dagre.layout(g);

  return nodes.map(node => {
    const nodeWithPosition = g.node(node.id);
    return {
      ...node,
      position: {
        x: nodeWithPosition.x - NODE_WIDTH / 2,
        y: nodeWithPosition.y - nodeWithPosition.height / 2,
      },
    };
  });
};
```

## User Flow

1. User navigates to Database > Schema Visualizer
2. Page loads with schema dropdown defaulting to "public"
3. Tables are fetched and displayed with auto-layout
4. User can:
   - Drag tables to reposition
   - Zoom in/out with scroll
   - Pan by dragging canvas
   - Click "Auto layout" to reset positions
   - Click "Copy as SQL" to get CREATE statements
   - Click download icon to export as PNG
   - Select different schema from dropdown

## Route Configuration

**Route:** `/database/schema-visualizer`

**App.tsx:**
```tsx
<Route path="/database/schema-visualizer" element={<SchemaVisualizerPage />} />
```

**Sidebar.tsx:**
Add under Database children:
```tsx
{ icon: IconSchema, label: 'Schema Visualizer', path: '/database/schema-visualizer' }
```

## Testing

### E2E Tests (Playwright)

```typescript
test.describe('Schema Visualizer', () => {
  test('should display tables with columns', async ({ page }) => {
    await page.goto('/database/schema-visualizer');
    await expect(page.locator('.react-flow__node')).toHaveCount.greaterThan(0);
    await page.screenshot({ path: 'screenshots/schema-visualizer.png' });
  });

  test('should show relationships between tables', async ({ page }) => {
    await page.goto('/database/schema-visualizer');
    await expect(page.locator('.react-flow__edge')).toHaveCount.greaterThan(0);
  });

  test('should auto-layout on button click', async ({ page }) => {
    await page.goto('/database/schema-visualizer');
    await page.click('button:has-text("Auto layout")');
    // Tables should be repositioned
  });

  test('should copy SQL to clipboard', async ({ page }) => {
    await page.goto('/database/schema-visualizer');
    await page.click('button:has-text("Copy as SQL")');
    // Verify clipboard content or notification
  });

  test('should change schema selection', async ({ page }) => {
    await page.goto('/database/schema-visualizer');
    await page.click('[data-testid="schema-select"]');
    await page.click('text=auth');
    // Tables should update
  });
});
```

## Migration Notes

No database migrations required. This feature uses existing pgmeta queries.

## Security Considerations

- Uses existing API key authentication
- Read-only operation (no data modification)
- Schema filtering respects database permissions

## Performance Considerations

- Lazy loading for large schemas (>50 tables)
- Debounced position updates for drag operations
- Efficient edge rendering with ReactFlow optimizations
- Canvas virtualization for smooth performance

## Future Enhancements

1. Table filtering/search
2. Relationship type indicators (one-to-one, one-to-many)
3. Save/load diagram positions
4. Multiple schema overlay view
5. Export as SVG/PDF
6. Column details tooltip
7. Edit schema from visualizer
