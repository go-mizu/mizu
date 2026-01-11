import { useEffect, useState, useCallback, lazy, Suspense } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { DashboardWidget, WidgetData, DashboardConfig } from '../../../types';
import { NumberWidget } from './widgets/NumberWidget';
import { ListWidget } from './widgets/ListWidget';
import { AddWidgetModal } from './AddWidgetModal';

// Lazy load ChartWidget to split recharts (~400KB) into separate chunk
const ChartWidget = lazy(() => import('./widgets/ChartWidget').then(m => ({ default: m.ChartWidget })));

interface DashboardDataResponse {
  widgets: WidgetData[];
  updated_at: string;
}

// Loading placeholder while chart chunk loads
function ChartLoadingPlaceholder({ title }: { title: string }) {
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4 h-full flex flex-col">
      <h3 className="text-sm font-medium text-gray-700 mb-3">{title}</h3>
      <div className="flex-1 flex items-center justify-center">
        <div className="animate-pulse flex flex-col items-center">
          <div className="w-32 h-32 bg-gray-200 rounded-full"></div>
          <div className="mt-4 h-4 bg-gray-200 rounded w-24"></div>
        </div>
      </div>
    </div>
  );
}

export function DashboardView() {
  const { currentView, currentTable, tables, fields, selectTable } = useBaseStore();
  const [widgetData, setWidgetData] = useState<Map<string, WidgetData>>(new Map());
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showAddWidget, setShowAddWidget] = useState(false);
  const [editingWidget, setEditingWidget] = useState<DashboardWidget | null>(null);

  // Parse dashboard config from view
  const dashboardConfig: DashboardConfig = (currentView?.config as unknown as DashboardConfig) || {
    widgets: [],
    grid_cols: 12
  };

  // Fetch dashboard data
  const fetchDashboardData = useCallback(async () => {
    if (!currentView?.id) return;

    try {
      setIsLoading(true);
      const response = await fetch(`/api/v1/views/${currentView.id}/dashboard/data`, {
        credentials: 'include',
      });

      if (!response.ok) {
        throw new Error('Failed to fetch dashboard data');
      }

      const data: DashboardDataResponse = await response.json();

      // Convert to map for easy lookup
      const dataMap = new Map<string, WidgetData>();
      data.widgets.forEach(w => {
        dataMap.set(w.widget_id, w);
      });
      setWidgetData(dataMap);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load dashboard');
    } finally {
      setIsLoading(false);
    }
  }, [currentView?.id]);

  // Refresh view data after widget changes
  const refreshViewData = async () => {
    if (currentTable?.id) {
      await selectTable(currentTable.id);
      await fetchDashboardData();
    }
  };

  // Add widget via API
  const handleAddWidget = async (widget: DashboardWidget) => {
    if (!currentView?.id) return;

    try {
      const response = await fetch(`/api/v1/views/${currentView.id}/dashboard/widgets`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ widget }),
      });

      if (!response.ok) {
        throw new Error('Failed to add widget');
      }

      // Refresh views to get updated config
      await refreshViewData();
      setShowAddWidget(false);
    } catch (err) {
      console.error('Failed to add widget:', err);
      alert('Failed to add widget');
    }
  };

  // Update widget via API
  const handleUpdateWidget = async (widget: DashboardWidget) => {
    if (!currentView?.id) return;

    try {
      const response = await fetch(`/api/v1/views/${currentView.id}/dashboard/widgets/${widget.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ widget }),
      });

      if (!response.ok) {
        throw new Error('Failed to update widget');
      }

      await refreshViewData();
      setEditingWidget(null);
    } catch (err) {
      console.error('Failed to update widget:', err);
      alert('Failed to update widget');
    }
  };

  // Delete widget via API
  const handleDeleteWidget = async (widgetId: string) => {
    if (!currentView?.id) return;
    if (!confirm('Are you sure you want to delete this widget?')) return;

    try {
      const response = await fetch(`/api/v1/views/${currentView.id}/dashboard/widgets/${widgetId}`, {
        method: 'DELETE',
        credentials: 'include',
      });

      if (!response.ok) {
        throw new Error('Failed to delete widget');
      }

      await refreshViewData();
    } catch (err) {
      console.error('Failed to delete widget:', err);
      alert('Failed to delete widget');
    }
  };

  useEffect(() => {
    fetchDashboardData();
  }, [fetchDashboardData]);

  // Render individual widget with wrapper
  const renderWidget = (widget: DashboardWidget) => {
    const data = widgetData.get(widget.id);

    const WidgetComponent = () => {
      switch (widget.type) {
        case 'chart':
          return (
            <Suspense fallback={<ChartLoadingPlaceholder title={widget.title} />}>
              <ChartWidget
                widget={widget}
                data={data}
                isLoading={isLoading}
              />
            </Suspense>
          );
        case 'number':
          return (
            <NumberWidget
              widget={widget}
              data={data}
              isLoading={isLoading}
            />
          );
        case 'list':
          return (
            <ListWidget
              widget={widget}
              data={data}
              isLoading={isLoading}
            />
          );
        default:
          return (
            <div className="bg-white rounded-lg border border-gray-200 p-4 h-full">
              <p className="text-gray-500">Unknown widget type: {widget.type}</p>
            </div>
          );
      }
    };

    return (
      <div key={widget.id} className="relative group h-full">
        <WidgetComponent />
        {/* Widget actions overlay */}
        <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity flex gap-1">
          <button
            onClick={() => setEditingWidget(widget)}
            className="p-1.5 bg-white rounded shadow-sm border border-gray-200 text-gray-500 hover:text-gray-700 hover:bg-gray-50"
            title="Edit widget"
          >
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
          </button>
          <button
            onClick={() => handleDeleteWidget(widget.id)}
            className="p-1.5 bg-white rounded shadow-sm border border-gray-200 text-gray-500 hover:text-red-600 hover:bg-red-50"
            title="Delete widget"
          >
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>
    );
  };

  // Calculate grid position styles
  const getWidgetStyle = (widget: DashboardWidget): React.CSSProperties => {
    return {
      gridColumn: `span ${widget.size.width}`,
      gridRow: `span ${widget.size.height}`,
    };
  };

  if (error) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <p className="text-danger mb-2">{error}</p>
          <button
            onClick={fetchDashboardData}
            className="px-4 py-2 bg-primary text-white rounded-md hover:bg-primary-dark"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-auto bg-[var(--at-bg)] p-6">
      {/* Dashboard Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-semibold text-[var(--at-text)]">
            {currentView?.name || 'Dashboard'}
          </h1>
          <p className="text-sm text-[var(--at-text-secondary)] mt-1">
            {currentTable?.name} overview
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={fetchDashboardData}
            className="toolbar-btn"
            disabled={isLoading}
          >
            <svg className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            Refresh
          </button>
          <button
            onClick={() => setShowAddWidget(true)}
            className="btn btn-primary"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Add Widget
          </button>
        </div>
      </div>

      {/* Dashboard Grid */}
      {dashboardConfig.widgets.length === 0 ? (
        <div className="flex items-center justify-center h-64 bg-white rounded-xl border-2 border-dashed border-[var(--at-border)]">
          <div className="empty-state animate-fade-in">
            <div className="empty-state-icon-wrapper">
              <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 5a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM14 5a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1V5zM4 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1v-4zM14 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
              </svg>
            </div>
            <h3 className="empty-state-title">No widgets yet</h3>
            <p className="empty-state-description">Add charts, numbers, and lists to visualize your data</p>
            <button onClick={() => setShowAddWidget(true)} className="btn btn-primary mt-2">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              Add Your First Widget
            </button>
          </div>
        </div>
      ) : (
        <div
          className="grid gap-4"
          style={{
            gridTemplateColumns: `repeat(${dashboardConfig.grid_cols || 12}, minmax(0, 1fr))`,
            gridAutoRows: 'minmax(80px, auto)',
          }}
        >
          {dashboardConfig.widgets.map(widget => (
            <div key={widget.id} style={getWidgetStyle(widget)}>
              {renderWidget(widget)}
            </div>
          ))}
        </div>
      )}

      {/* Add Widget Modal */}
      {showAddWidget && (
        <AddWidgetModal
          tables={tables}
          fields={fields}
          currentTableId={currentTable?.id || ''}
          onClose={() => setShowAddWidget(false)}
          onAdd={handleAddWidget}
        />
      )}

      {/* Edit Widget Modal */}
      {editingWidget && (
        <AddWidgetModal
          tables={tables}
          fields={fields}
          currentTableId={currentTable?.id || ''}
          editingWidget={editingWidget}
          onClose={() => setEditingWidget(null)}
          onAdd={handleUpdateWidget}
        />
      )}
    </div>
  );
}
