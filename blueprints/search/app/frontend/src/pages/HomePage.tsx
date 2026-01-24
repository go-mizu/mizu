import { Settings, History } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { SearchBox } from '../components/SearchBox'

// Showcase queries that demonstrate different features
const SHOWCASE_QUERIES = [
  { query: '2+2', label: 'Calculator', icon: '🔢' },
  { query: '10 km to miles', label: 'Unit Conversion', icon: '📏' },
  { query: '100 usd to eur', label: 'Currency', icon: '💱' },
  { query: 'weather tokyo', label: 'Weather', icon: '🌤️' },
  { query: 'time in london', label: 'World Time', icon: '🕐' },
  { query: 'define programming', label: 'Dictionary', icon: '📖' },
  { query: 'Go', label: 'Knowledge Panel', icon: '📚' },
  { query: 'Python', label: 'Search Results', icon: '🔍' },
]

export default function HomePage() {
  const navigate = useNavigate()

  const handleSuggestionClick = (query: string) => {
    navigate(`/search?q=${encodeURIComponent(query)}`)
  }

  return (
    <div className="min-h-screen flex flex-col">
      {/* Header */}
      <header className="flex justify-end p-4 gap-4">
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
      <main className="flex-1 flex items-center justify-center -mt-20">
        <div className="w-full max-w-2xl px-4">
          <div className="flex flex-col items-center gap-8">
            {/* Logo */}
            <div className="text-center">
              <h1
                className="text-7xl font-bold tracking-tight"
                style={{
                  background: 'linear-gradient(90deg, #4285F4, #EA4335, #FBBC05, #34A853)',
                  WebkitBackgroundClip: 'text',
                  WebkitTextFillColor: 'transparent',
                }}
              >
                Search
              </h1>
            </div>

            {/* Search box */}
            <SearchBox size="lg" autoFocus />

            {/* Try these searches */}
            <div className="mt-6 w-full max-w-xl">
              <p className="text-sm text-[#70757a] text-center mb-4">Try these searches:</p>
              <div className="flex flex-wrap justify-center gap-2">
                {SHOWCASE_QUERIES.map((item) => (
                  <button
                    key={item.query}
                    onClick={() => handleSuggestionClick(item.query)}
                    className="flex items-center gap-2 px-3 py-2 text-sm bg-[#f8f9fa] hover:bg-[#e8f0fe] border border-[#dadce0] hover:border-[#1a73e8] rounded-full transition-colors text-[#202124]"
                  >
                    <span>{item.icon}</span>
                    <span>{item.query}</span>
                  </button>
                ))}
              </div>
            </div>

            {/* Quick links */}
            <div className="flex gap-8 mt-4">
              <Link
                to="/images"
                className="text-sm text-[#70757a] hover:underline"
              >
                Images
              </Link>
              <Link
                to="/search?time=day"
                className="text-sm text-[#70757a] hover:underline"
              >
                News
              </Link>
              <Link
                to="/settings"
                className="text-sm text-[#70757a] hover:underline"
              >
                Settings
              </Link>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}
