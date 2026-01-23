import { lazy, Suspense } from 'react'
import { Routes, Route } from 'react-router-dom'
import { Box, LoadingOverlay } from '@mantine/core'
import { useHotkeys } from '@mantine/hooks'
import Sidebar from './components/layout/Sidebar'
import CommandPalette from './components/core/CommandPalette'
import { useUIStore } from './stores/uiStore'

// Lazy load pages for better performance
const Home = lazy(() => import('./pages/Home'))
const Browse = lazy(() => import('./pages/Browse'))
const Collection = lazy(() => import('./pages/Collection'))
const DatabaseBrowser = lazy(() => import('./pages/Browse/DatabaseBrowser'))
const Question = lazy(() => import('./pages/Question'))
const Dashboard = lazy(() => import('./pages/Dashboard'))
const DataModel = lazy(() => import('./pages/admin/DataModel'))
const Settings = lazy(() => import('./pages/admin/Settings'))
const People = lazy(() => import('./pages/admin/People'))

function PageLoader() {
  return <LoadingOverlay visible={true} />
}

function App() {
  const { openCommandPalette, toggleCommandPalette } = useUIStore()

  // Global keyboard shortcuts
  useHotkeys([
    ['mod+k', () => toggleCommandPalette()],
    ['/', () => openCommandPalette()],
  ])

  return (
    <Box style={{ display: 'flex', minHeight: '100vh' }}>
      <Sidebar />
      <Box
        component="main"
        style={{
          flex: 1,
          backgroundColor: '#f9fbfc',
          minHeight: '100vh',
          overflow: 'auto',
        }}
      >
        <Suspense fallback={<PageLoader />}>
          <Routes>
            <Route path="/" element={<Home />} />
            <Route path="/browse" element={<Browse />} />
            <Route path="/browse/databases" element={<Browse view="databases" />} />
            <Route path="/browse/database/:datasourceId" element={<DatabaseBrowser />} />
            <Route path="/browse/database/:datasourceId/table/:tableId" element={<DatabaseBrowser />} />
            <Route path="/browse/models" element={<Browse view="models" />} />
            <Route path="/browse/metrics" element={<Browse view="metrics" />} />
            <Route path="/browse/:id" element={<Browse />} />
            {/* Collection routes */}
            <Route path="/collection/root" element={<Collection type="root" />} />
            <Route path="/collection/personal" element={<Collection type="personal" />} />
            <Route path="/collection/trash" element={<Collection type="trash" />} />
            <Route path="/collection/:id" element={<Collection />} />
            <Route path="/question/new" element={<Question />} />
            <Route path="/question/:id" element={<Question />} />
            <Route path="/question/:id/edit" element={<Question mode="edit" />} />
            <Route path="/dashboard/new" element={<Dashboard />} />
            <Route path="/dashboard/:id" element={<Dashboard />} />
            <Route path="/dashboard/:id/edit" element={<Dashboard mode="edit" />} />
            <Route path="/admin/datamodel" element={<DataModel />} />
            <Route path="/admin/datamodel/:tableId" element={<DataModel />} />
            <Route path="/admin/people" element={<People />} />
            <Route path="/admin/settings" element={<Settings />} />
          </Routes>
        </Suspense>
      </Box>
      <CommandPalette />
    </Box>
  )
}

export default App
