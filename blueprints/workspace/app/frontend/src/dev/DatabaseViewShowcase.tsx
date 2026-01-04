/**
 * Database View Showcase
 * Comprehensive preview component for all database view types
 * Use this in development to test and preview all views
 */
import { useState, useMemo, useEffect } from 'react'
import { TableView } from '../database/views/TableView'
import { BoardView } from '../database/views/BoardView'
import { TimelineView } from '../database/views/TimelineView'
import { CalendarView } from '../database/views/CalendarView'
import { GalleryView } from '../database/views/GalleryView'
import { ChartView } from '../database/views/ChartView'
import { ListView } from '../database/views/ListView'
import { Database, DatabaseRow, Property } from '../api/client'
import {
  Table2,
  LayoutGrid,
  Clock,
  Calendar,
  Image,
  BarChart3,
  List,
} from 'lucide-react'

type ViewType = 'table' | 'board' | 'timeline' | 'calendar' | 'gallery' | 'chart' | 'list'

// Sample database schema
const sampleProperties: Property[] = [
  {
    id: 'title',
    name: 'Task Name',
    type: 'text',
  },
  {
    id: 'status',
    name: 'Status',
    type: 'select',
    options: [
      { id: 'todo', name: 'To Do', color: 'gray' },
      { id: 'progress', name: 'In Progress', color: 'blue' },
      { id: 'review', name: 'In Review', color: 'orange' },
      { id: 'done', name: 'Done', color: 'green' },
    ],
  },
  {
    id: 'priority',
    name: 'Priority',
    type: 'select',
    options: [
      { id: 'high', name: 'High', color: 'red' },
      { id: 'medium', name: 'Medium', color: 'yellow' },
      { id: 'low', name: 'Low', color: 'gray' },
    ],
  },
  {
    id: 'assignee',
    name: 'Assignee',
    type: 'person',
  },
  {
    id: 'due_date',
    name: 'Due Date',
    type: 'date',
  },
  {
    id: 'estimate',
    name: 'Estimate (hrs)',
    type: 'number',
  },
  {
    id: 'tags',
    name: 'Tags',
    type: 'multi_select',
    options: [
      { id: 'frontend', name: 'Frontend', color: 'purple' },
      { id: 'backend', name: 'Backend', color: 'orange' },
      { id: 'design', name: 'Design', color: 'pink' },
      { id: 'bug', name: 'Bug', color: 'red' },
      { id: 'feature', name: 'Feature', color: 'green' },
    ],
  },
  {
    id: 'cover',
    name: 'Cover Image',
    type: 'files',
  },
  {
    id: 'progress_pct',
    name: 'Progress %',
    type: 'number',
  },
  {
    id: 'completed',
    name: 'Completed',
    type: 'checkbox',
  },
]

// Helper to generate dates relative to today
const daysFromNow = (days: number): string => {
  const date = new Date()
  date.setDate(date.getDate() + days)
  return date.toISOString()
}

// Sample rows
const generateSampleRows = (): DatabaseRow[] => [
  {
    id: 'row-1',
    database_id: 'demo-db',
    properties: {
      title: 'Implement user authentication',
      status: 'done',
      priority: 'high',
      assignee: 'Alice',
      due_date: daysFromNow(-5),
      estimate: 16,
      tags: ['backend', 'feature'],
      cover: 'https://images.unsplash.com/photo-1555066931-4365d14bab8c?w=400',
      progress_pct: 100,
      completed: true,
    },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'row-2',
    database_id: 'demo-db',
    properties: {
      title: 'Design dashboard mockups',
      status: 'done',
      priority: 'high',
      assignee: 'Bob',
      due_date: daysFromNow(-3),
      estimate: 8,
      tags: ['design', 'feature'],
      cover: 'https://images.unsplash.com/photo-1561070791-2526d30994b5?w=400',
      progress_pct: 100,
      completed: true,
    },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'row-3',
    database_id: 'demo-db',
    properties: {
      title: 'Build API endpoints',
      status: 'progress',
      priority: 'high',
      assignee: 'Charlie',
      due_date: daysFromNow(2),
      estimate: 24,
      tags: ['backend', 'feature'],
      cover: 'https://images.unsplash.com/photo-1516116216624-53e697fedbea?w=400',
      progress_pct: 65,
      completed: false,
    },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'row-4',
    database_id: 'demo-db',
    properties: {
      title: 'Create React components',
      status: 'progress',
      priority: 'medium',
      assignee: 'Alice',
      due_date: daysFromNow(5),
      estimate: 20,
      tags: ['frontend', 'feature'],
      cover: 'https://images.unsplash.com/photo-1633356122544-f134324a6cee?w=400',
      progress_pct: 40,
      completed: false,
    },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'row-5',
    database_id: 'demo-db',
    properties: {
      title: 'Fix navigation bug',
      status: 'review',
      priority: 'high',
      assignee: 'Diana',
      due_date: daysFromNow(0),
      estimate: 4,
      tags: ['frontend', 'bug'],
      progress_pct: 90,
      completed: false,
    },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'row-6',
    database_id: 'demo-db',
    properties: {
      title: 'Write unit tests',
      status: 'todo',
      priority: 'medium',
      assignee: 'Eve',
      due_date: daysFromNow(7),
      estimate: 12,
      tags: ['backend', 'frontend'],
      cover: 'https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=400',
      progress_pct: 0,
      completed: false,
    },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'row-7',
    database_id: 'demo-db',
    properties: {
      title: 'Set up CI/CD pipeline',
      status: 'todo',
      priority: 'low',
      assignee: 'Frank',
      due_date: daysFromNow(14),
      estimate: 8,
      tags: ['backend'],
      cover: 'https://images.unsplash.com/photo-1618401471353-b98afee0b2eb?w=400',
      progress_pct: 0,
      completed: false,
    },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'row-8',
    database_id: 'demo-db',
    properties: {
      title: 'Database optimization',
      status: 'todo',
      priority: 'medium',
      assignee: 'Charlie',
      due_date: daysFromNow(10),
      estimate: 6,
      tags: ['backend', 'bug'],
      progress_pct: 0,
      completed: false,
    },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'row-9',
    database_id: 'demo-db',
    properties: {
      title: 'Mobile responsive design',
      status: 'progress',
      priority: 'medium',
      assignee: 'Bob',
      due_date: daysFromNow(3),
      estimate: 10,
      tags: ['frontend', 'design'],
      cover: 'https://images.unsplash.com/photo-1512941937669-90a1b58e7e9c?w=400',
      progress_pct: 25,
      completed: false,
    },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'row-10',
    database_id: 'demo-db',
    properties: {
      title: 'Documentation update',
      status: 'review',
      priority: 'low',
      assignee: 'Grace',
      due_date: daysFromNow(1),
      estimate: 3,
      tags: ['feature'],
      progress_pct: 85,
      completed: false,
    },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'row-11',
    database_id: 'demo-db',
    properties: {
      title: 'Performance profiling',
      status: 'todo',
      priority: 'high',
      assignee: 'Henry',
      due_date: daysFromNow(8),
      estimate: 5,
      tags: ['backend', 'bug'],
      cover: 'https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=400',
      progress_pct: 0,
      completed: false,
    },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'row-12',
    database_id: 'demo-db',
    properties: {
      title: 'User onboarding flow',
      status: 'progress',
      priority: 'high',
      assignee: 'Ivy',
      due_date: daysFromNow(4),
      estimate: 14,
      tags: ['frontend', 'design', 'feature'],
      cover: 'https://images.unsplash.com/photo-1559136555-9303baea8ebd?w=400',
      progress_pct: 55,
      completed: false,
    },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
]

const VIEW_TYPES: { type: ViewType; label: string; icon: React.ReactNode; description: string }[] = [
  { type: 'table', label: 'Table', icon: <Table2 size={16} />, description: 'Spreadsheet-style view with rows and columns' },
  { type: 'board', label: 'Board', icon: <LayoutGrid size={16} />, description: 'Kanban-style board grouped by status' },
  { type: 'timeline', label: 'Timeline', icon: <Clock size={16} />, description: 'Gantt chart for project planning' },
  { type: 'calendar', label: 'Calendar', icon: <Calendar size={16} />, description: 'Calendar view for date-based items' },
  { type: 'gallery', label: 'Gallery', icon: <Image size={16} />, description: 'Card-based gallery with images' },
  { type: 'chart', label: 'Chart', icon: <BarChart3 size={16} />, description: 'Visualize data with charts' },
  { type: 'list', label: 'List', icon: <List size={16} />, description: 'Simple list view' },
]

export function DatabaseViewShowcase() {
  const [activeView, setActiveView] = useState<ViewType>('table')
  const [rows, setRows] = useState<DatabaseRow[]>(generateSampleRows)
  const [properties, setProperties] = useState<Property[]>(sampleProperties)
  const [hiddenProperties, setHiddenProperties] = useState<string[]>([])

  // Listen for view type switch events from sidebar
  useEffect(() => {
    const handleSwitchView = (e: CustomEvent<{ viewType: string }>) => {
      const viewType = e.detail.viewType as ViewType
      if (VIEW_TYPES.some(v => v.type === viewType)) {
        setActiveView(viewType)
      }
    }
    window.addEventListener('switch-database-view', handleSwitchView as EventListener)
    return () => {
      window.removeEventListener('switch-database-view', handleSwitchView as EventListener)
    }
  }, [])

  // Sample database object
  const database: Database = useMemo(() => ({
    id: 'demo-db',
    workspace_id: 'demo-workspace',
    name: 'Project Tasks',
    properties: properties,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  }), [properties])

  // View handlers
  const handleAddRow = async (initialProperties?: Record<string, unknown>) => {
    const newRow: DatabaseRow = {
      id: `row-${Date.now()}`,
      database_id: 'demo-db',
      properties: {
        title: 'New Task',
        status: 'todo',
        ...initialProperties,
      },
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    setRows((prev) => [...prev, newRow])
    return newRow
  }

  const handleUpdateRow = (rowId: string, updates: Record<string, unknown>) => {
    setRows((prev) =>
      prev.map((row) =>
        row.id === rowId
          ? { ...row, properties: { ...row.properties, ...updates }, updated_at: new Date().toISOString() }
          : row
      )
    )
  }

  const handleDeleteRow = (rowId: string) => {
    setRows((prev) => prev.filter((row) => row.id !== rowId))
  }

  const handleAddProperty = (property: Omit<Property, 'id'>) => {
    const newProperty: Property = {
      ...property,
      id: `prop-${Date.now()}`,
    }
    setProperties((prev) => [...prev, newProperty])
  }

  const handleUpdateProperty = (propertyId: string, updates: Partial<Property>) => {
    setProperties((prev) =>
      prev.map((prop) =>
        prop.id === propertyId ? { ...prop, ...updates } : prop
      )
    )
  }

  const handleDeleteProperty = (propertyId: string) => {
    setProperties((prev) => prev.filter((prop) => prop.id !== propertyId))
  }

  // Common props for all views
  const viewProps = {
    rows,
    properties,
    database,
    hiddenProperties,
    onAddRow: handleAddRow,
    onUpdateRow: handleUpdateRow,
    onDeleteRow: handleDeleteRow,
    onAddProperty: handleAddProperty,
    onUpdateProperty: handleUpdateProperty,
    onDeleteProperty: handleDeleteProperty,
    onHiddenPropertiesChange: setHiddenProperties,
  }

  const renderView = () => {
    switch (activeView) {
      case 'table':
        return <TableView {...viewProps} groupBy={null} />
      case 'board':
        return <BoardView {...viewProps} groupBy="status" />
      case 'timeline':
        return <TimelineView {...viewProps} groupBy={null} />
      case 'calendar':
        return <CalendarView {...viewProps} groupBy={null} />
      case 'gallery':
        return <GalleryView {...viewProps} groupBy={null} />
      case 'chart':
        return <ChartView {...viewProps} groupBy={null} />
      case 'list':
        return <ListView {...viewProps} groupBy="status" />
      default:
        return null
    }
  }

  return (
    <div className="view-showcase">
      <div className="showcase-header">
        <h1>Database Views Showcase</h1>
        <p>Preview and test all database view types</p>
      </div>

      <div className="view-tabs">
        {VIEW_TYPES.map(({ type, label, icon, description }) => (
          <button
            key={type}
            className={`view-tab ${activeView === type ? 'active' : ''}`}
            onClick={() => setActiveView(type)}
            title={description}
          >
            {icon}
            <span>{label}</span>
          </button>
        ))}
      </div>

      <div className="view-info">
        <span className="info-label">
          {VIEW_TYPES.find((v) => v.type === activeView)?.description}
        </span>
        <span className="info-count">{rows.length} items</span>
      </div>

      <div className="showcase-view-panel" style={{ position: 'relative', height: 600 }}>
        {renderView()}
      </div>

      {/* Debug info */}
      <div className="debug-info" style={{
        marginTop: 16,
        padding: 12,
        background: 'rgba(55, 53, 47, 0.04)',
        borderRadius: 6,
        fontSize: 12,
        color: '#787774',
      }}>
        <strong>Debug:</strong> {rows.length} rows, {properties.length} properties
      </div>

      <style>{`
        .view-showcase {
          padding: 24px;
          max-width: 1400px;
          margin: 0 auto;
          font-family: ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
        }

        .showcase-header {
          margin-bottom: 24px;
        }

        .showcase-header h1 {
          font-size: 24px;
          font-weight: 600;
          color: #37352f;
          margin: 0 0 4px;
        }

        .showcase-header p {
          font-size: 14px;
          color: #787774;
          margin: 0;
        }

        .view-tabs {
          display: flex;
          gap: 4px;
          padding: 4px;
          background: rgba(55, 53, 47, 0.04);
          border-radius: 8px;
          margin-bottom: 16px;
          overflow-x: auto;
        }

        .view-tab {
          display: flex;
          align-items: center;
          gap: 6px;
          padding: 8px 16px;
          border: none;
          background: none;
          border-radius: 6px;
          cursor: pointer;
          font-size: 13px;
          font-weight: 500;
          color: #787774;
          white-space: nowrap;
          transition: all 0.15s;
        }

        .view-tab:hover {
          background: rgba(55, 53, 47, 0.08);
          color: #37352f;
        }

        .view-tab.active {
          background: #fff;
          color: #37352f;
          box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
        }

        .view-info {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 8px 0;
          margin-bottom: 16px;
          border-bottom: 1px solid rgba(55, 53, 47, 0.09);
        }

        .info-label {
          font-size: 13px;
          color: #787774;
        }

        .info-count {
          font-size: 12px;
          color: #9a9a97;
          background: rgba(55, 53, 47, 0.06);
          padding: 2px 8px;
          border-radius: 4px;
        }

        .showcase-view-panel {
          background: #fff;
          border: 1px solid rgba(55, 53, 47, 0.09);
          border-radius: 8px;
          min-height: 500px;
          overflow: visible;
        }

        .showcase-view-panel .table-view {
          height: 100%;
        }

        /* CSS Variables for views */
        :root {
          --bg-primary: #ffffff;
          --bg-secondary: #fbfbfa;
          --text-primary: #37352f;
          --text-secondary: #787774;
          --text-tertiary: #9a9a97;
          --border-color: rgba(55, 53, 47, 0.09);
          --border-color-light: rgba(55, 53, 47, 0.06);
          --accent-color: #2383e2;
          --error-color: #eb5757;
          --shadow-lg: rgba(15, 15, 15, 0.05) 0px 0px 0px 1px, rgba(15, 15, 15, 0.1) 0px 3px 6px, rgba(15, 15, 15, 0.2) 0px 9px 24px;
          --radius-sm: 4px;
          --radius-md: 6px;
          --tag-gray: #e9e9e7;
          --tag-brown: #eee0da;
          --tag-orange: #fadec9;
          --tag-yellow: #fdecc8;
          --tag-green: #dbeddb;
          --tag-blue: #d3e5ef;
          --tag-purple: #e8deee;
          --tag-pink: #f5e0e9;
          --tag-red: #ffd5d2;
        }
      `}</style>
    </div>
  )
}

export default DatabaseViewShowcase
