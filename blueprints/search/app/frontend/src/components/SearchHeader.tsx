import { Link, useNavigate } from 'react-router-dom'
import { Settings, Sparkles, Image, Video, Newspaper, Search as SearchIcon } from 'lucide-react'
import { SearchBox } from './SearchBox'
import { useAIStore } from '../stores/aiStore'

export type SearchTab = 'all' | 'ai' | 'images' | 'videos' | 'news'

interface SearchHeaderProps {
  query: string
  activeTab: SearchTab
  onSearch?: (query: string) => void
  /** Extra content rendered after the tabs row (e.g., filters, view toggle) */
  tabsRight?: React.ReactNode
  /** Extra row below tabs (e.g., filter chips) */
  belowTabs?: React.ReactNode
}

export function SearchHeader({
  query,
  activeTab,
  onSearch,
  tabsRight,
  belowTabs,
}: SearchHeaderProps) {
  const navigate = useNavigate()
  const { aiAvailable } = useAIStore()

  const handleSearch = (newQuery: string) => {
    if (onSearch) {
      onSearch(newQuery)
    } else {
      navigate(`/search?q=${encodeURIComponent(newQuery)}`)
    }
  }

  const handleTabClick = (tab: SearchTab) => {
    const encoded = encodeURIComponent(query)
    switch (tab) {
      case 'all':
        navigate(`/search?q=${encoded}`)
        break
      case 'ai':
        navigate(`/ai?q=${encoded}`)
        break
      case 'images':
        navigate(`/images?q=${encoded}`)
        break
      case 'videos':
        navigate(`/videos?q=${encoded}`)
        break
      case 'news':
        navigate(`/news?q=${encoded}`)
        break
    }
  }

  return (
    <header className="sticky top-0 bg-white z-50 border-b border-[#e8eaed]">
      <div className="max-w-7xl mx-auto px-4 py-3">
        {/* Top row: Logo + Search + Actions */}
        <div className="flex items-center gap-6">
          <Link to="/" className="flex-shrink-0">
            <span className="text-2xl font-semibold text-[#202124] tracking-tight">
              Mizu
            </span>
          </Link>

          <div className="flex-1 max-w-xl">
            <SearchBox
              initialValue={query}
              size="sm"
              onSearch={handleSearch}
            />
          </div>

          <div className="flex items-center gap-1">
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
              to="/settings"
              className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
              title="Settings"
            >
              <Settings size={20} />
            </Link>
          </div>
        </div>

        {/* Tabs row */}
        <div className="flex items-center justify-between mt-2">
          <div className="search-tabs" style={{ paddingLeft: 0 }}>
            <button
              type="button"
              className={`search-tab ${activeTab === 'all' ? 'active' : ''}`}
              onClick={() => handleTabClick('all')}
            >
              <SearchIcon size={16} />
              All
            </button>
            {aiAvailable && (
              <button
                type="button"
                className={`search-tab ${activeTab === 'ai' ? 'active' : ''}`}
                onClick={() => handleTabClick('ai')}
              >
                <Sparkles size={16} />
                AI
              </button>
            )}
            <button
              type="button"
              className={`search-tab ${activeTab === 'images' ? 'active' : ''}`}
              onClick={() => handleTabClick('images')}
            >
              <Image size={16} />
              Images
            </button>
            <button
              type="button"
              className={`search-tab ${activeTab === 'videos' ? 'active' : ''}`}
              onClick={() => handleTabClick('videos')}
            >
              <Video size={16} />
              Videos
            </button>
            <button
              type="button"
              className={`search-tab ${activeTab === 'news' ? 'active' : ''}`}
              onClick={() => handleTabClick('news')}
            >
              <Newspaper size={16} />
              News
            </button>
          </div>

          {tabsRight && <div className="flex items-center gap-2">{tabsRight}</div>}
        </div>
      </div>

      {belowTabs}
    </header>
  )
}
