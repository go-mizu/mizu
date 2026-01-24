import { Settings, History } from 'lucide-react'
import { Link } from 'react-router-dom'
import { SearchBox } from '../components/SearchBox'

export default function HomePage() {
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
        <div className="w-full max-w-xl px-4">
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

      {/* Footer */}
      <footer className="footer">
        <div className="max-w-6xl mx-auto flex justify-between items-center">
          <span className="text-xs text-[#70757a]">Built with Go + React</span>
          <div className="flex gap-6">
            <a href="#" className="text-xs text-[#70757a] hover:underline">
              Privacy
            </a>
            <a href="#" className="text-xs text-[#70757a] hover:underline">
              Terms
            </a>
          </div>
        </div>
      </footer>
    </div>
  )
}
