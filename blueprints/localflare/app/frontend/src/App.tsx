import { Routes, Route } from 'react-router-dom'
import { AppShell } from '@mantine/core'
import { Sidebar } from './components/layout/Sidebar'
import { Dashboard } from './pages/Dashboard'
import { DurableObjects } from './pages/DurableObjects'
import { DurableObjectDetail } from './pages/DurableObjectDetail'
import { Queues } from './pages/Queues'
import { QueueDetail } from './pages/QueueDetail'
import { Vectorize } from './pages/Vectorize'
import { VectorizeDetail } from './pages/VectorizeDetail'
import { AnalyticsEngine } from './pages/AnalyticsEngine'
import { AnalyticsEngineDetail } from './pages/AnalyticsEngineDetail'
import { WorkersAI } from './pages/WorkersAI'
import { AIGatewayPage } from './pages/AIGateway'
import { AIGatewayDetail } from './pages/AIGatewayDetail'
import { AIGatewayLogs } from './pages/AIGatewayLogs'
import { Hyperdrive } from './pages/Hyperdrive'
import { HyperdriveDetail } from './pages/HyperdriveDetail'
import { Cron } from './pages/Cron'
import { CronDetail } from './pages/CronDetail'

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
          {/* Dashboard */}
          <Route path="/" element={<Dashboard />} />

          {/* Durable Objects */}
          <Route path="/durable-objects" element={<DurableObjects />} />
          <Route path="/durable-objects/:id" element={<DurableObjectDetail />} />

          {/* Queues */}
          <Route path="/queues" element={<Queues />} />
          <Route path="/queues/:id" element={<QueueDetail />} />

          {/* Vectorize */}
          <Route path="/vectorize" element={<Vectorize />} />
          <Route path="/vectorize/:name" element={<VectorizeDetail />} />

          {/* Analytics Engine */}
          <Route path="/analytics-engine" element={<AnalyticsEngine />} />
          <Route path="/analytics-engine/:name" element={<AnalyticsEngineDetail />} />

          {/* Workers AI */}
          <Route path="/ai" element={<WorkersAI />} />

          {/* AI Gateway */}
          <Route path="/ai-gateway" element={<AIGatewayPage />} />
          <Route path="/ai-gateway/:id" element={<AIGatewayDetail />} />
          <Route path="/ai-gateway/:id/logs" element={<AIGatewayLogs />} />

          {/* Hyperdrive */}
          <Route path="/hyperdrive" element={<Hyperdrive />} />
          <Route path="/hyperdrive/:id" element={<HyperdriveDetail />} />

          {/* Cron Triggers */}
          <Route path="/cron" element={<Cron />} />
          <Route path="/cron/:id" element={<CronDetail />} />
        </Routes>
      </AppShell.Main>
    </AppShell>
  )
}

export default App
