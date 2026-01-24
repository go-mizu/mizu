import { useEffect } from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import HomePage from './pages/HomePage'
import SearchPage from './pages/SearchPage'
import ImagesPage from './pages/ImagesPage'
import VideosPage from './pages/VideosPage'
import NewsPage from './pages/NewsPage'
import SettingsPage from './pages/SettingsPage'
import HistoryPage from './pages/HistoryPage'
import AISessionsPage from './pages/AISessionsPage'
import AISessionPage from './pages/AISessionPage'
import { useAIStore } from './stores/aiStore'
import { aiApi } from './api/ai'

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
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/search" element={<SearchPage />} />
        <Route path="/images" element={<ImagesPage />} />
        <Route path="/videos" element={<VideosPage />} />
        <Route path="/news" element={<NewsPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/history" element={<HistoryPage />} />
        <Route path="/ai/sessions" element={<AISessionsPage />} />
        <Route path="/ai/session/:id" element={<AISessionPage />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
