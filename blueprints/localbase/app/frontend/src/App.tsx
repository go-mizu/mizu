import { Routes, Route, useLocation, Navigate } from 'react-router-dom';
import { AppShell, Box, Burger, Group, LoadingOverlay } from '@mantine/core';
import { useDisclosure, useMediaQuery } from '@mantine/hooks';
import { Suspense, lazy, useEffect } from 'react';
import { Sidebar } from './components/layout/Sidebar';
import { Header } from './components/layout/Header';
import { useAppStore } from './stores/appStore';

// Lazy load all pages for code splitting
const ProjectOverviewPage = lazy(() => import('./pages/project-overview/ProjectOverview').then(m => ({ default: m.ProjectOverviewPage })));
const UsersPage = lazy(() => import('./pages/auth/Users').then(m => ({ default: m.UsersPage })));
const StoragePage = lazy(() => import('./pages/storage/Storage').then(m => ({ default: m.StoragePage })));
const TableEditorPage = lazy(() => import('./pages/database/TableEditor').then(m => ({ default: m.TableEditorPage })));
const SQLEditorPage = lazy(() => import('./pages/database/SQLEditor').then(m => ({ default: m.SQLEditorPage })));
const PoliciesPage = lazy(() => import('./pages/database/Policies').then(m => ({ default: m.PoliciesPage })));
const IndexesPage = lazy(() => import('./pages/database/Indexes').then(m => ({ default: m.IndexesPage })));
const ViewsPage = lazy(() => import('./pages/database/Views').then(m => ({ default: m.ViewsPage })));
const TriggersPage = lazy(() => import('./pages/database/Triggers').then(m => ({ default: m.TriggersPage })));
const RolesPage = lazy(() => import('./pages/database/Roles').then(m => ({ default: m.RolesPage })));
const SchemaVisualizerPage = lazy(() => import('./pages/database/SchemaVisualizer').then(m => ({ default: m.SchemaVisualizerPage })));
const ExtensionsPage = lazy(() => import('./pages/database/Extensions').then(m => ({ default: m.ExtensionsPage })));
const DatabaseFunctionsPage = lazy(() => import('./pages/database/DatabaseFunctions').then(m => ({ default: m.DatabaseFunctionsPage })));
const RealtimePage = lazy(() => import('./pages/realtime/Realtime').then(m => ({ default: m.RealtimePage })));
const FunctionsPage = lazy(() => import('./pages/functions/Functions').then(m => ({ default: m.FunctionsPage })));
const LogsExplorerPage = lazy(() => import('./pages/logs/LogsExplorer').then(m => ({ default: m.LogsExplorerPage })));
const ReportsPage = lazy(() => import('./pages/reports/Reports').then(m => ({ default: m.ReportsPage })));
const ApiDocsPage = lazy(() => import('./pages/ApiDocs').then(m => ({ default: m.ApiDocsPage })));
const ApiPlaygroundPage = lazy(() => import('./pages/ApiPlayground').then(m => ({ default: m.ApiPlaygroundPage })));
const SettingsPage = lazy(() => import('./pages/settings/Settings').then(m => ({ default: m.SettingsPage })));
const AdvisorsPage = lazy(() => import('./pages/advisors/Advisors').then(m => ({ default: m.AdvisorsPage })));
const IntegrationsPage = lazy(() => import('./pages/integrations/Integrations').then(m => ({ default: m.IntegrationsPage })));

// Database Layout
const DatabaseLayout = lazy(() => import('./components/database/DatabaseLayout').then(m => ({ default: m.DatabaseLayout })));

export default function App() {
  const { sidebarCollapsed } = useAppStore();
  const [mobileOpened, { toggle: toggleMobile, close: closeMobile }] = useDisclosure();
  const isMobile = useMediaQuery('(max-width: 768px)');
  const isTablet = useMediaQuery('(max-width: 1024px)');
  const location = useLocation();

  // Close mobile sidebar on route change
  useEffect(() => {
    closeMobile();
  }, [location.pathname, closeMobile]);

  // Calculate navbar width based on screen size
  const getNavbarWidth = () => {
    if (isMobile) return 280;
    if (isTablet) return sidebarCollapsed ? 70 : 200;
    return sidebarCollapsed ? 70 : 250;
  };

  return (
    <AppShell
      header={{ height: 48 }}
      navbar={{
        width: getNavbarWidth(),
        breakpoint: 'sm',
        collapsed: { mobile: !mobileOpened },
      }}
      padding={0}
    >
      <AppShell.Header>
        <Group h="100%" px="md" style={{ width: '100%' }}>
          {/* Mobile hamburger */}
          <Burger
            opened={mobileOpened}
            onClick={toggleMobile}
            hiddenFrom="sm"
            size="sm"
            color="var(--supabase-text)"
          />
          <Box style={{ flex: 1 }}>
            <Header />
          </Box>
        </Group>
      </AppShell.Header>

      <AppShell.Navbar>
        <Sidebar onNavigate={closeMobile} />
      </AppShell.Navbar>

      <AppShell.Main style={{ backgroundColor: 'var(--supabase-bg-surface)' }}>
        <Box style={{ height: 'calc(100vh - 48px)', overflow: 'auto' }}>
          <Suspense fallback={<LoadingOverlay visible loaderProps={{ type: 'dots' }} />}>
            <Routes>
              <Route path="/" element={<ProjectOverviewPage />} />

              {/* Legacy routes - redirect to new database routes */}
              <Route path="/table-editor" element={<Navigate to="/database/tables" replace />} />
              <Route path="/sql-editor" element={<Navigate to="/database/sql" replace />} />

              {/* Database routes with unified layout */}
              <Route path="/database" element={<DatabaseLayout />}>
                <Route index element={<Navigate to="/database/tables" replace />} />
                <Route path="tables" element={<TableEditorPage />} />
                <Route path="sql" element={<SQLEditorPage />} />
                <Route path="schema" element={<SchemaVisualizerPage />} />
                <Route path="schema-visualizer" element={<Navigate to="/database/schema" replace />} />
                <Route path="policies" element={<PoliciesPage />} />
                <Route path="indexes" element={<IndexesPage />} />
                <Route path="views" element={<ViewsPage />} />
                <Route path="triggers" element={<TriggersPage />} />
                <Route path="roles" element={<RolesPage />} />
                <Route path="functions" element={<DatabaseFunctionsPage />} />
                <Route path="extensions" element={<ExtensionsPage />} />
              </Route>

              <Route path="/auth/users" element={<UsersPage />} />
              <Route path="/storage" element={<StoragePage />} />
              <Route path="/realtime" element={<RealtimePage />} />
              <Route path="/functions" element={<FunctionsPage />} />
              <Route path="/reports" element={<ReportsPage />} />
              <Route path="/logs" element={<LogsExplorerPage />} />
              <Route path="/api-docs" element={<ApiDocsPage />} />
              <Route path="/api-playground" element={<ApiPlaygroundPage />} />
              <Route path="/settings" element={<SettingsPage />} />
              <Route path="/advisors" element={<AdvisorsPage />} />
              <Route path="/integrations" element={<IntegrationsPage />} />
            </Routes>
          </Suspense>
        </Box>
      </AppShell.Main>
    </AppShell>
  );
}
