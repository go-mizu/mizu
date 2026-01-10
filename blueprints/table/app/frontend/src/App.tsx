import { useEffect, useState } from 'react';
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

function App() {
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
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
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
        return <div className="p-4 text-gray-500">Timeline view coming soon...</div>;
      case 'form':
        return <div className="p-4 text-gray-500">Form view coming soon...</div>;
      case 'list':
        return <div className="p-4 text-gray-500">List view coming soon...</div>;
      default:
        return <GridView />;
    }
  };

  return (
    <div className="min-h-screen flex bg-gray-50">
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
        {currentTable && currentView && <ViewToolbar />}

        {/* Main view area */}
        <div className="flex-1 overflow-hidden">
          {isLoading ? (
            <div className="flex items-center justify-center h-full">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
          ) : currentTable ? (
            renderView()
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
