import { Routes, Route } from 'react-router-dom'
import { AppShell } from '@mantine/core'
import Sidebar from './components/layout/Sidebar'
import Home from './pages/Home'
import Browse from './pages/Browse'
import Question from './pages/Question'
import Dashboard from './pages/Dashboard'
import DataModel from './pages/admin/DataModel'
import Settings from './pages/admin/Settings'

function App() {
  return (
    <AppShell
      navbar={{ width: 260, breakpoint: 'sm' }}
      padding="md"
    >
      <AppShell.Navbar>
        <Sidebar />
      </AppShell.Navbar>

      <AppShell.Main>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/browse" element={<Browse />} />
          <Route path="/browse/:id" element={<Browse />} />
          <Route path="/question/new" element={<Question />} />
          <Route path="/question/:id" element={<Question />} />
          <Route path="/dashboard/new" element={<Dashboard />} />
          <Route path="/dashboard/:id" element={<Dashboard />} />
          <Route path="/admin/datamodel" element={<DataModel />} />
          <Route path="/admin/settings" element={<Settings />} />
        </Routes>
      </AppShell.Main>
    </AppShell>
  )
}

export default App
