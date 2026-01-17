import { Routes, Route } from 'react-router-dom';
import { AppShell } from '@mantine/core';
import { Sidebar } from './components/layout/Sidebar';
import { Dashboard } from './pages/Dashboard';
import { UsersPage } from './pages/auth/Users';
import { StoragePage } from './pages/storage/Storage';
import { TableEditorPage } from './pages/database/TableEditor';
import { SQLEditorPage } from './pages/database/SQLEditor';
import { PoliciesPage } from './pages/database/Policies';
import { IndexesPage } from './pages/database/Indexes';
import { ViewsPage } from './pages/database/Views';
import { TriggersPage } from './pages/database/Triggers';
import { RolesPage } from './pages/database/Roles';
import { RealtimePage } from './pages/realtime/Realtime';
import { FunctionsPage } from './pages/functions/Functions';
import { LogsExplorerPage } from './pages/logs/LogsExplorer';
import { ApiDocsPage } from './pages/ApiDocs';
import { SettingsPage } from './pages/settings/Settings';
import { useAppStore } from './stores/appStore';

export default function App() {
  const { sidebarCollapsed } = useAppStore();

  return (
    <AppShell
      navbar={{
        width: sidebarCollapsed ? 70 : 250,
        breakpoint: 'sm',
      }}
      padding={0}
    >
      <AppShell.Navbar>
        <Sidebar />
      </AppShell.Navbar>

      <AppShell.Main style={{ backgroundColor: 'var(--supabase-bg-surface)' }}>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/table-editor" element={<TableEditorPage />} />
          <Route path="/sql-editor" element={<SQLEditorPage />} />
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
        </Routes>
      </AppShell.Main>
    </AppShell>
  );
}
