import { Routes, Route, useLocation } from 'react-router-dom';
import { AppShell, Box, Burger, Group } from '@mantine/core';
import { useDisclosure, useMediaQuery } from '@mantine/hooks';
import { useEffect } from 'react';
import { Sidebar } from './components/layout/Sidebar';
import { Header } from './components/layout/Header';
import { ProjectOverviewPage } from './pages/project-overview/ProjectOverview';
import { UsersPage } from './pages/auth/Users';
import { StoragePage } from './pages/storage/Storage';
import { TableEditorPage } from './pages/database/TableEditor';
import { SQLEditorPage } from './pages/database/SQLEditor';
import { PoliciesPage } from './pages/database/Policies';
import { IndexesPage } from './pages/database/Indexes';
import { ViewsPage } from './pages/database/Views';
import { TriggersPage } from './pages/database/Triggers';
import { RolesPage } from './pages/database/Roles';
import { SchemaVisualizerPage } from './pages/database/SchemaVisualizer';
import { RealtimePage } from './pages/realtime/Realtime';
import { FunctionsPage } from './pages/functions/Functions';
import { LogsExplorerPage } from './pages/logs/LogsExplorer';
import { ApiDocsPage } from './pages/ApiDocs';
import { SettingsPage } from './pages/settings/Settings';
import { AdvisorsPage } from './pages/advisors/Advisors';
import { IntegrationsPage } from './pages/integrations/Integrations';
import { useAppStore } from './stores/appStore';

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
            <Route path="/logs" element={<LogsExplorerPage />} />
            <Route path="/api-docs" element={<ApiDocsPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/advisors" element={<AdvisorsPage />} />
            <Route path="/integrations" element={<IntegrationsPage />} />
          </Routes>
        </Box>
      </AppShell.Main>
    </AppShell>
  );
}
