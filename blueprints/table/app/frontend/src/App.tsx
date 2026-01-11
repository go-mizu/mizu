import { useEffect, useState, useMemo } from 'react';
import { useAuthStore } from './stores/authStore';
import { useBaseStore } from './stores/baseStore';
import { LoginForm } from './components/auth/LoginForm';
import { Sidebar } from './components/base/Sidebar';
import { BaseHeader } from './components/base/BaseHeader';
import { TableTabs } from './components/base/TableTabs';
import { ViewToolbar } from './components/views/ViewToolbar';
import { GridView } from './components/views/grid/GridView';
import { KanbanView } from './components/views/kanban/KanbanView';
import { CalendarView } from './components/views/calendar/CalendarView';
import { GalleryView } from './components/views/gallery/GalleryView';
import { TimelineView } from './components/views/timeline/TimelineView';
import { FormView } from './components/views/form/FormView';
import { ListView } from './components/views/list/ListView';
import { DashboardView } from './components/views/dashboard/DashboardView';
import { PublicFormPage } from './pages/PublicFormPage';
import { RecordPage } from './pages/RecordPage';

function App() {
  // Check for public form route
  const publicFormViewId = useMemo(() => {
    const path = window.location.pathname;
    const match = path.match(/^\/form\/([^/]+)$/);
    return match ? match[1] : null;
  }, []);

  // Check for record page route
  const recordPageParams = useMemo(() => {
    const path = window.location.pathname;
    const search = window.location.search;
    const match = path.match(/^\/record\/([^/]+)$/);
    if (match) {
      const viewId = new URLSearchParams(search).get('view') || undefined;
      return { recordId: match[1], viewId };
    }
    return null;
  }, []);

  // If this is a public form route, render the public form page
  if (publicFormViewId) {
    return <PublicFormPage viewId={publicFormViewId} />;
  }
  const { isAuthenticated, isLoading: authLoading, checkAuth } = useAuthStore();
  const {
    workspaces,
    currentWorkspace,
    currentBase,
    currentTable,
    currentView,
    isLoading,
    error,
    loadWorkspaces,
    selectWorkspace,
  } = useBaseStore();

  const [sidebarOpen, setSidebarOpen] = useState(true);

  // Check auth on mount
  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  // Load workspaces when authenticated
  useEffect(() => {
    if (isAuthenticated) {
      loadWorkspaces();
    }
  }, [isAuthenticated, loadWorkspaces]);

  // Auto-select first workspace
  useEffect(() => {
    if (workspaces.length > 0 && !currentWorkspace) {
      selectWorkspace(workspaces[0].id);
    }
  }, [workspaces, currentWorkspace, selectWorkspace]);

  // Loading state
  if (authLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[#f6f7fb]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading...</p>
        </div>
      </div>
    );
  }

  // Not authenticated - show login
  if (!isAuthenticated) {
    return <LoginForm />;
  }

  // If this is a record page route, render the record page
  if (recordPageParams && currentTable) {
    return (
      <RecordPage
        recordId={recordPageParams.recordId}
        tableId={currentTable.id}
        viewId={recordPageParams.viewId}
        onBack={() => {
          window.history.pushState({}, '', '/');
          window.location.reload();
        }}
      />
    );
  }

  // Render current view
  const renderView = () => {
    if (!currentView) return null;

    switch (currentView.type) {
      case 'grid':
        return <GridView />;
      case 'kanban':
        return <KanbanView />;
      case 'calendar':
        return <CalendarView />;
      case 'gallery':
        return <GalleryView />;
      case 'timeline':
        return <TimelineView />;
      case 'form':
        return <FormView />;
      case 'list':
        return <ListView />;
      case 'dashboard':
        return <DashboardView />;
      default:
        return <GridView />;
    }
  };

  return (
    <div className="min-h-screen flex bg-[#f6f7fb]">
      {/* Sidebar */}
      <Sidebar
        isOpen={sidebarOpen}
        onToggle={() => setSidebarOpen(!sidebarOpen)}
      />

      {/* Main content */}
      <div className={`flex-1 flex flex-col min-w-0 ${sidebarOpen ? 'ml-64' : 'ml-0'}`}>
        {/* Error banner */}
        {error && (
          <div className="bg-danger text-white px-4 py-2 flex items-center justify-between">
            <span>{error}</span>
            <button
              onClick={() => useBaseStore.getState().clearError()}
              className="text-white hover:text-gray-200"
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Base header */}
        {currentBase && <BaseHeader />}

        {/* Table tabs */}
        {currentBase && <TableTabs />}

        {/* View toolbar */}
        {currentTable && <ViewToolbar />}

        {/* Main view area */}
        <div className="flex-1 overflow-hidden">
          {isLoading ? (
            <div className="flex items-center justify-center h-full">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
          ) : currentTable && currentView ? (
            renderView()
          ) : currentTable ? (
            <div className="flex items-center justify-center h-full text-gray-500">
              <div className="text-center">
                <p className="text-lg">No views yet</p>
                <p className="text-sm mt-1">Use "Create view" to add your first view</p>
              </div>
            </div>
          ) : currentBase ? (
            <div className="flex items-center justify-center h-full text-gray-500">
              <div className="text-center">
                <p className="text-lg">No tables yet</p>
                <p className="text-sm mt-1">Click "+ Add table" to create your first table</p>
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center h-full text-gray-500">
              <div className="text-center">
                <p className="text-lg">Select a base to get started</p>
                <p className="text-sm mt-1">or create a new one from the sidebar</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default App;
