import { useState } from 'react'
import { ArrowLeft, Search, Trash2, MoreVertical, Clock } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { useSearchStore } from '../stores/searchStore'

export default function HistoryPage() {
  const navigate = useNavigate()
  const { recentSearches, removeRecentSearch, clearRecentSearches } = useSearchStore()
  const [filter, setFilter] = useState('')
  const [openMenuIndex, setOpenMenuIndex] = useState<number | null>(null)

  const filteredSearches = recentSearches.filter((search) =>
    search.toLowerCase().includes(filter.toLowerCase())
  )

  const handleSearchClick = (query: string) => {
    navigate(`/search?q=${encodeURIComponent(query)}`)
  }

  return (
    <div className="min-h-screen bg-[#f8f9fa]">
      {/* Header */}
      <header className="bg-white border-b border-[#dadce0]">
        <div className="max-w-2xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <Link
                to="/"
                className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
              >
                <ArrowLeft size={20} />
              </Link>
              <h1 className="text-xl font-semibold text-[#202124]">
                Search History
              </h1>
            </div>

            {recentSearches.length > 0 && (
              <button
                type="button"
                onClick={clearRecentSearches}
                className="flex items-center gap-2 px-3 py-2 text-sm text-[#d93025] hover:bg-[#d93025]/5 rounded transition-colors"
              >
                <Trash2 size={16} />
                Clear all
              </button>
            )}
          </div>
        </div>
      </header>

      {/* Main content */}
      <main>
        <div className="max-w-2xl mx-auto px-4 py-6">
          {recentSearches.length === 0 ? (
            <div className="bg-white rounded-lg border border-[#dadce0] p-12 text-center">
              <Clock size={48} className="mx-auto mb-4 text-[#9aa0a6]" />
              <p className="text-lg font-medium text-[#202124] mb-2">
                No search history
              </p>
              <p className="text-[#70757a]">
                Your recent searches will appear here
              </p>
              <Link
                to="/"
                className="inline-block mt-6 px-4 py-2 text-sm font-medium text-[#1a73e8] bg-[#e8f0fe] hover:bg-[#d2e3fc] rounded transition-colors"
              >
                Start searching
              </Link>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Search filter */}
              <div className="relative">
                <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-[#9aa0a6]" />
                <input
                  type="text"
                  placeholder="Filter history..."
                  value={filter}
                  onChange={(e) => setFilter(e.target.value)}
                  className="w-full pl-10 pr-4 py-2 border border-[#dadce0] rounded-lg text-sm focus:outline-none focus:border-[#1a73e8]"
                />
              </div>

              {/* History list */}
              <div className="bg-white rounded-lg border border-[#dadce0] overflow-hidden">
                {filteredSearches.length === 0 ? (
                  <div className="p-8 text-center">
                    <p className="text-[#70757a]">No matches found</p>
                  </div>
                ) : (
                  <div>
                    {filteredSearches.map((search, index) => (
                      <div
                        key={`${search}-${index}`}
                        className={`flex items-center justify-between px-4 py-3 hover:bg-[#f1f3f4] cursor-pointer ${
                          index > 0 ? 'border-t border-[#dadce0]' : ''
                        }`}
                        onClick={() => handleSearchClick(search)}
                      >
                        <div className="flex items-center gap-3">
                          <Clock size={18} className="text-[#9aa0a6]" />
                          <span className="text-[#202124]">{search}</span>
                        </div>

                        <div className="relative">
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation()
                              setOpenMenuIndex(openMenuIndex === index ? null : index)
                            }}
                            className="p-1 text-[#5f6368] hover:bg-[#e8eaed] rounded-full transition-colors"
                          >
                            <MoreVertical size={16} />
                          </button>

                          {openMenuIndex === index && (
                            <div className="dropdown-menu" style={{ right: 0 }}>
                              <button
                                type="button"
                                className="dropdown-item"
                                onClick={(e) => {
                                  e.stopPropagation()
                                  setOpenMenuIndex(null)
                                  handleSearchClick(search)
                                }}
                              >
                                <Search size={14} />
                                Search again
                              </button>
                              <button
                                type="button"
                                className="dropdown-item danger"
                                onClick={(e) => {
                                  e.stopPropagation()
                                  setOpenMenuIndex(null)
                                  removeRecentSearch(search)
                                }}
                              >
                                <Trash2 size={14} />
                                Remove
                              </button>
                            </div>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Stats */}
              <p className="text-sm text-[#70757a] text-center">
                {filteredSearches.length} of {recentSearches.length} searches shown
              </p>
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
