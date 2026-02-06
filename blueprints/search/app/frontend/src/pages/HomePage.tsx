import { Settings, History, Zap, Sparkles, Image, Newspaper } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { SearchBox } from '../components/SearchBox'
import { useAIStore } from '../stores/aiStore'

// Showcase queries that demonstrate different features
const SHOWCASE_QUERIES = [
  { query: '2+2', label: 'Calculator', icon: '🔢' },
  { query: '10 km to miles', label: 'Conversion', icon: '📏' },
  { query: '100 usd to eur', label: 'Currency', icon: '💱' },
  { query: 'weather tokyo', label: 'Weather', icon: '🌤️' },
  { query: 'time in london', label: 'Time', icon: '🕐' },
  { query: 'define programming', label: 'Define', icon: '📖' },
]

// Bang shortcuts for quick access
const QUICK_BANGS = [
  { bang: '!g', name: 'Google', color: '#4285F4' },
  { bang: '!yt', name: 'YouTube', color: '#FF0000' },
  { bang: '!gh', name: 'GitHub', color: '#333' },
  { bang: '!w', name: 'Wikipedia', color: '#000' },
  { bang: '!r', name: 'Reddit', color: '#FF5700' },
  { bang: '!ai', name: 'AI Mode', color: '#9333EA' },
]

export default function HomePage() {
  const navigate = useNavigate()
  const { aiAvailable } = useAIStore()

  const handleSuggestionClick = (query: string) => {
    navigate(`/search?q=${encodeURIComponent(query)}`)
  }

  const handleBangClick = (bang: string) => {
    // Focus the search box with the bang pre-filled
    navigate(`/search?q=${encodeURIComponent(bang + ' ')}`)
  }

  return (
    <div className="min-h-screen flex flex-col bg-gradient-to-b from-white to-[#fafafa]">
      {/* Header */}
      <header className="flex justify-end p-4 gap-2">
        {aiAvailable && (
          <Link
            to="/ai/sessions"
            className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
            title="AI Sessions"
          >
            <Sparkles size={20} />
          </Link>
        )}
        <Link
          to="/history"
          className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
          title="History"
        >
          <History size={20} />
        </Link>
        <Link
          to="/settings"
          className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
          title="Settings"
        >
          <Settings size={20} />
        </Link>
      </header>

      {/* Main content */}
      <main className="flex-1 flex items-center justify-center -mt-16">
        <div className="w-full max-w-2xl px-4">
          <div className="flex flex-col items-center gap-6">
            {/* Logo */}
            <div className="text-center mb-2">
              <h1 className="text-6xl font-semibold text-[#202124] tracking-tight">
                Mizu
              </h1>
              <p className="text-sm text-[#70757a] mt-2">
                Private search, powered by you
              </p>
            </div>

            {/* Search box */}
            <SearchBox size="lg" autoFocus />

            {/* Bang shortcuts */}
            <div className="flex flex-wrap justify-center gap-2 mt-2">
              {QUICK_BANGS.map((item) => (
                <button
                  key={item.bang}
                  onClick={() => handleBangClick(item.bang)}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-white hover:bg-[#f8f9fa] border border-[#e8eaed] hover:border-[#dadce0] rounded-full transition-colors text-[#5f6368]"
                  title={`Search with ${item.name}`}
                >
                  <Zap size={12} style={{ color: item.color }} />
                  <span>{item.bang}</span>
                </button>
              ))}
            </div>

            {/* Instant answers showcase */}
            <div className="mt-6 w-full max-w-xl">
              <p className="text-xs text-[#9aa0a6] text-center mb-3 uppercase tracking-wide">
                Instant Answers
              </p>
              <div className="flex flex-wrap justify-center gap-2">
                {SHOWCASE_QUERIES.map((item) => (
                  <button
                    key={item.query}
                    onClick={() => handleSuggestionClick(item.query)}
                    className="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-[#f8f9fa] hover:bg-[#e8f0fe] border border-[#e8eaed] hover:border-[#c2dbff] rounded-lg transition-all text-[#202124]"
                  >
                    <span className="text-base">{item.icon}</span>
                    <span>{item.label}</span>
                  </button>
                ))}
              </div>
            </div>

            {/* Quick category links */}
            <div className="flex items-center gap-6 mt-6">
              <Link
                to="/images"
                className="flex items-center gap-2 text-sm text-[#5f6368] hover:text-[#1a73e8] transition-colors"
              >
                <Image size={16} />
                Images
              </Link>
              <Link
                to="/news"
                className="flex items-center gap-2 text-sm text-[#5f6368] hover:text-[#1a73e8] transition-colors"
              >
                <Newspaper size={16} />
                News
              </Link>
              {aiAvailable && (
                <Link
                  to="/ai"
                  className="flex items-center gap-2 text-sm text-[#5f6368] hover:text-[#9333ea] transition-colors"
                >
                  <Sparkles size={16} />
                  AI Mode
                </Link>
              )}
            </div>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="py-4 text-center text-xs text-[#9aa0a6]">
        <p>Type <code className="px-1 py-0.5 bg-[#f1f3f4] rounded text-[#5f6368]">!</code> for bangs • <code className="px-1 py-0.5 bg-[#f1f3f4] rounded text-[#5f6368]">!ai</code> for AI mode</p>
      </footer>
    </div>
  )
}
