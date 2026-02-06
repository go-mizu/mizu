import { useEffect, lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import HomePage from './pages/HomePage'
import { useAIStore } from './stores/aiStore'
import { aiApi } from './api/ai'

// Lazy load pages for code splitting
const SearchPage = lazy(() => import('./pages/SearchPage'))
const AIPage = lazy(() => import('./pages/AIPage'))
const ImagesPage = lazy(() => import('./pages/ImagesPage'))
const VideosPage = lazy(() => import('./pages/VideosPage'))
const NewsPage = lazy(() => import('./pages/NewsPage'))
const CodePage = lazy(() => import('./pages/CodePage'))
const SciencePage = lazy(() => import('./pages/SciencePage'))
const SocialPage = lazy(() => import('./pages/SocialPage'))
const MusicPage = lazy(() => import('./pages/MusicPage'))
const MapsPage = lazy(() => import('./pages/MapsPage'))
const SettingsPage = lazy(() => import('./pages/SettingsPage'))
const HistoryPage = lazy(() => import('./pages/HistoryPage'))
const AISessionsPage = lazy(() => import('./pages/AISessionsPage'))
const AISessionPage = lazy(() => import('./pages/AISessionPage'))

// Minimal loading fallback
function PageLoader() {
  return (
    <div className="flex items-center justify-center min-h-screen">
      <div className="animate-pulse text-gray-500">Loading...</div>
    </div>
  )
}

function App() {
  const { setAIAvailable, setAvailableModes } = useAIStore()

  // Check AI availability on mount
  useEffect(() => {
    const checkAI = async () => {
      try {
        const { modes } = await aiApi.getModes()
        const available = modes.filter(m => m.available).map(m => m.id)
        setAvailableModes(available)
        setAIAvailable(available.length > 0)
      } catch {
        setAIAvailable(false)
        setAvailableModes([])
      }
    }
    checkAI()
  }, [])

  return (
    <BrowserRouter>
      <Suspense fallback={<PageLoader />}>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/search" element={<SearchPage />} />
          <Route path="/ai" element={<AIPage />} />
          <Route path="/images" element={<ImagesPage />} />
          <Route path="/videos" element={<VideosPage />} />
          <Route path="/news" element={<NewsPage />} />
          <Route path="/code" element={<CodePage />} />
          <Route path="/science" element={<SciencePage />} />
          <Route path="/social" element={<SocialPage />} />
          <Route path="/music" element={<MusicPage />} />
          <Route path="/maps" element={<MapsPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="/history" element={<HistoryPage />} />
          <Route path="/ai/sessions" element={<AISessionsPage />} />
          <Route path="/ai/session/:id" element={<AISessionPage />} />
        </Routes>
      </Suspense>
    </BrowserRouter>
  )
}

export default App
