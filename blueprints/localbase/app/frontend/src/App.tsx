import { Routes, Route, useLocation } from 'react-router-dom';
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
const RealtimePage = lazy(() => import('./pages/realtime/Realtime').then(m => ({ default: m.RealtimePage })));
const FunctionsPage = lazy(() => import('./pages/functions/Functions').then(m => ({ default: m.FunctionsPage })));
const LogsExplorerPage = lazy(() => import('./pages/logs/LogsExplorer').then(m => ({ default: m.LogsExplorerPage })));
const ReportsPage = lazy(() => import('./pages/reports/Reports').then(m => ({ default: m.ReportsPage })));
const ApiDocsPage = lazy(() => import('./pages/ApiDocs').then(m => ({ default: m.ApiDocsPage })));
const SettingsPage = lazy(() => import('./pages/settings/Settings').then(m => ({ default: m.SettingsPage })));
const AdvisorsPage = lazy(() => import('./pages/advisors/Advisors').then(m => ({ default: m.AdvisorsPage })));
const IntegrationsPage = lazy(() => import('./pages/integrations/Integrations').then(m => ({ default: m.IntegrationsPage })));

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
              <Route path="/table-editor" element={<TableEditorPage />} />
              <Route path="/sql-editor" element={<SQLEditorPage />} />
              <Route path="/database/schema-visualizer" element={<SchemaVisualizerPage />} />
              <Route path="/database/policies" element={<PoliciesPage />} />
              <Route path="/database/indexes" element={<IndexesPage />} />
              <Route path="/database/views" element={<ViewsPage />} />
              <Route path="/database/triggers" element={<TriggersPage />} />
              <Route path="/database/roles" element={<RolesPage />} />
              <Route path="/auth/users" element={<UsersPage />} />
              <Route path="/storage" element={<StoragePage />} />
              <Route path="/realtime" element={<RealtimePage />} />
              <Route path="/functions" element={<FunctionsPage />} />
              <Route path="/reports" element={<ReportsPage />} />
              <Route path="/logs" element={<LogsExplorerPage />} />
              <Route path="/api-docs" element={<ApiDocsPage />} />
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
