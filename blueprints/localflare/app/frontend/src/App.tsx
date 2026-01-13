import { Routes, Route } from 'react-router-dom'
import { AppShell } from '@mantine/core'
import { Sidebar } from './components/layout/Sidebar'
import { Dashboard } from './pages/Dashboard'
import { ZonesList } from './pages/ZonesList'
import { ZoneDetail } from './pages/ZoneDetail'
import { DNSRecords } from './pages/DNSRecords'
import { SSLSettings } from './pages/SSLSettings'
import { FirewallRules } from './pages/FirewallRules'
import { CacheSettings } from './pages/CacheSettings'
import { WorkersList } from './pages/WorkersList'
import { WorkerDetail } from './pages/WorkerDetail'
import { KVNamespaces } from './pages/KVNamespaces'
import { KVDetail } from './pages/KVDetail'
import { R2Buckets } from './pages/R2Buckets'
import { R2Detail } from './pages/R2Detail'
import { D1Databases } from './pages/D1Databases'
import { D1Detail } from './pages/D1Detail'
import { Analytics } from './pages/Analytics'
import { Login } from './pages/Login'

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
          <Route path="/" element={<Dashboard />} />
          <Route path="/login" element={<Login />} />
          <Route path="/zones" element={<ZonesList />} />
          <Route path="/zones/:id" element={<ZoneDetail />} />
          <Route path="/zones/:id/dns" element={<DNSRecords />} />
          <Route path="/zones/:id/ssl" element={<SSLSettings />} />
          <Route path="/zones/:id/firewall" element={<FirewallRules />} />
          <Route path="/zones/:id/caching" element={<CacheSettings />} />
          <Route path="/zones/:id/analytics" element={<Analytics />} />
          <Route path="/workers" element={<WorkersList />} />
          <Route path="/workers/:id" element={<WorkerDetail />} />
          <Route path="/kv" element={<KVNamespaces />} />
          <Route path="/kv/:id" element={<KVDetail />} />
          <Route path="/r2" element={<R2Buckets />} />
          <Route path="/r2/:id" element={<R2Detail />} />
          <Route path="/d1" element={<D1Databases />} />
          <Route path="/d1/:id" element={<D1Detail />} />
          <Route path="/analytics" element={<Analytics />} />
        </Routes>
      </AppShell.Main>
    </AppShell>
  )
}

export default App
