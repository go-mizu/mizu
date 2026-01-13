import { Routes, Route } from 'react-router-dom'
import { AppShell } from '@mantine/core'
import { Sidebar } from './components/layout/Sidebar'
import { Dashboard } from './pages/Dashboard'
import { Workers } from './pages/Workers'
import { WorkerDetail } from './pages/WorkerDetail'
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
import { KV } from './pages/KV'
import { KVDetail } from './pages/KVDetail'
import { R2 } from './pages/R2'
import { R2Detail } from './pages/R2Detail'
import { D1 } from './pages/D1'
import { D1Detail } from './pages/D1Detail'
import { Pages } from './pages/Pages'
import { Images } from './pages/Images'
import { Stream } from './pages/Stream'
import { Observability } from './pages/Observability'
import { Settings } from './pages/Settings'

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

          {/* Workers */}
          <Route path="/workers" element={<Workers />} />
          <Route path="/workers/:id" element={<WorkerDetail />} />

          {/* Durable Objects */}
          <Route path="/durable-objects" element={<DurableObjects />} />
          <Route path="/durable-objects/:id" element={<DurableObjectDetail />} />

          {/* Cron Triggers */}
          <Route path="/cron" element={<Cron />} />
          <Route path="/cron/:id" element={<CronDetail />} />

          {/* KV */}
          <Route path="/kv" element={<KV />} />
          <Route path="/kv/:id" element={<KVDetail />} />

          {/* R2 */}
          <Route path="/r2" element={<R2 />} />
          <Route path="/r2/:name" element={<R2Detail />} />
          <Route path="/r2/:name/*" element={<R2Detail />} />

          {/* D1 */}
          <Route path="/d1" element={<D1 />} />
          <Route path="/d1/:id" element={<D1Detail />} />

          {/* Queues */}
          <Route path="/queues" element={<Queues />} />
          <Route path="/queues/:id" element={<QueueDetail />} />

          {/* Vectorize */}
          <Route path="/vectorize" element={<Vectorize />} />
          <Route path="/vectorize/:name" element={<VectorizeDetail />} />

          {/* Pages */}
          <Route path="/pages" element={<Pages />} />

          {/* Images */}
          <Route path="/images" element={<Images />} />

          {/* Stream */}
          <Route path="/stream" element={<Stream />} />

          {/* Workers AI */}
          <Route path="/ai" element={<WorkersAI />} />

          {/* AI Gateway */}
          <Route path="/ai-gateway" element={<AIGatewayPage />} />
          <Route path="/ai-gateway/:id" element={<AIGatewayDetail />} />
          <Route path="/ai-gateway/:id/logs" element={<AIGatewayLogs />} />

          {/* Hyperdrive */}
          <Route path="/hyperdrive" element={<Hyperdrive />} />
          <Route path="/hyperdrive/:id" element={<HyperdriveDetail />} />

          {/* Analytics Engine */}
          <Route path="/analytics-engine" element={<AnalyticsEngine />} />
          <Route path="/analytics-engine/:name" element={<AnalyticsEngineDetail />} />

          {/* Observability */}
          <Route path="/observability" element={<Observability />} />

          {/* Settings */}
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </AppShell.Main>
    </AppShell>
  )
}

export default App
