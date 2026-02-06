import { useSearchStore } from '../stores/searchStore'
import { PageHeader } from '../components/PageHeader'

const REGIONS = [
  { value: 'us', label: 'United States' },
  { value: 'uk', label: 'United Kingdom' },
  { value: 'ca', label: 'Canada' },
  { value: 'au', label: 'Australia' },
  { value: 'de', label: 'Germany' },
  { value: 'fr', label: 'France' },
  { value: 'jp', label: 'Japan' },
  { value: 'in', label: 'India' },
  { value: 'br', label: 'Brazil' },
  { value: 'mx', label: 'Mexico' },
]

const LANGUAGES = [
  { value: 'en', label: 'English' },
  { value: 'es', label: 'Spanish' },
  { value: 'fr', label: 'French' },
  { value: 'de', label: 'German' },
  { value: 'it', label: 'Italian' },
  { value: 'pt', label: 'Portuguese' },
  { value: 'ja', label: 'Japanese' },
  { value: 'zh', label: 'Chinese' },
  { value: 'ko', label: 'Korean' },
  { value: 'ar', label: 'Arabic' },
]

const RESULTS_PER_PAGE = [
  { value: '10', label: '10 results' },
  { value: '20', label: '20 results' },
  { value: '30', label: '30 results' },
  { value: '50', label: '50 results' },
]

const SAFE_SEARCH = [
  { value: 'off', label: 'Off' },
  { value: 'moderate', label: 'Moderate' },
  { value: 'strict', label: 'Strict' },
]

export default function SettingsPage() {
  const { settings, updateSettings, clearRecentSearches } = useSearchStore()

  return (
    <div className="min-h-screen bg-white">
      <PageHeader title="Settings" />

      {/* Main content */}
      <main>
        <div className="max-w-2xl mx-auto px-4 py-4 space-y-6">
          {/* Search preferences */}
          <div className="bg-white rounded-lg border border-[#dadce0] p-6">
            <h2 className="font-semibold text-[#202124] mb-4">
              Search Preferences
            </h2>

            <div className="space-y-4">
              <div className="settings-group">
                <label className="settings-label">Region</label>
                <p className="text-xs text-[#70757a] mb-2">Show results relevant to your region</p>
                <select
                  value={settings.region}
                  onChange={(e) => updateSettings({ region: e.target.value })}
                  className="settings-select"
                >
                  {REGIONS.map(option => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>

              <div className="settings-group">
                <label className="settings-label">Language</label>
                <p className="text-xs text-[#70757a] mb-2">Preferred language for search results</p>
                <select
                  value={settings.language}
                  onChange={(e) => updateSettings({ language: e.target.value })}
                  className="settings-select"
                >
                  {LANGUAGES.map(option => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>

              <div className="settings-group">
                <label className="settings-label">Results per page</label>
                <p className="text-xs text-[#70757a] mb-2">Number of results to show per page</p>
                <select
                  value={String(settings.results_per_page)}
                  onChange={(e) => updateSettings({ results_per_page: parseInt(e.target.value, 10) })}
                  className="settings-select"
                >
                  {RESULTS_PER_PAGE.map(option => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>

              <div className="settings-group">
                <label className="settings-label">SafeSearch</label>
                <p className="text-xs text-[#70757a] mb-2">Filter explicit content from results</p>
                <select
                  value={settings.safe_search}
                  onChange={(e) => updateSettings({ safe_search: e.target.value })}
                  className="settings-select"
                >
                  {SAFE_SEARCH.map(option => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>
            </div>
          </div>

          {/* Display settings */}
          <div className="bg-white rounded-lg border border-[#dadce0] p-6">
            <h2 className="font-semibold text-[#202124] mb-4">
              Display Settings
            </h2>

            <div className="space-y-4">
              <div className="settings-toggle">
                <input
                  type="checkbox"
                  id="openInNewTab"
                  checked={settings.open_in_new_tab}
                  onChange={(e) => updateSettings({ open_in_new_tab: e.target.checked })}
                  className="settings-checkbox"
                />
                <div>
                  <label htmlFor="openInNewTab" className="settings-label cursor-pointer" style={{ marginBottom: 0 }}>
                    Open links in new tab
                  </label>
                  <p className="text-xs text-[#70757a]">Open search result links in a new browser tab</p>
                </div>
              </div>

              <div className="settings-toggle">
                <input
                  type="checkbox"
                  id="showInstantAnswers"
                  checked={settings.show_instant_answers}
                  onChange={(e) => updateSettings({ show_instant_answers: e.target.checked })}
                  className="settings-checkbox"
                />
                <div>
                  <label htmlFor="showInstantAnswers" className="settings-label cursor-pointer" style={{ marginBottom: 0 }}>
                    Show instant answers
                  </label>
                  <p className="text-xs text-[#70757a]">Display instant answers for calculations, conversions, etc.</p>
                </div>
              </div>

              <div className="settings-toggle">
                <input
                  type="checkbox"
                  id="showKnowledgePanel"
                  checked={settings.show_knowledge_panel}
                  onChange={(e) => updateSettings({ show_knowledge_panel: e.target.checked })}
                  className="settings-checkbox"
                />
                <div>
                  <label htmlFor="showKnowledgePanel" className="settings-label cursor-pointer" style={{ marginBottom: 0 }}>
                    Show knowledge panels
                  </label>
                  <p className="text-xs text-[#70757a]">Display knowledge panels for people, places, and things</p>
                </div>
              </div>
            </div>
          </div>

          {/* Privacy */}
          <div className="bg-white rounded-lg border border-[#dadce0] p-6">
            <h2 className="font-semibold text-[#202124] mb-4">
              Privacy
            </h2>

            <div className="space-y-4">
              <div className="settings-toggle">
                <input
                  type="checkbox"
                  id="saveHistory"
                  checked={settings.save_history}
                  onChange={(e) => updateSettings({ save_history: e.target.checked })}
                  className="settings-checkbox"
                />
                <div>
                  <label htmlFor="saveHistory" className="settings-label cursor-pointer" style={{ marginBottom: 0 }}>
                    Save search history
                  </label>
                  <p className="text-xs text-[#70757a]">Keep a record of your searches for quick access</p>
                </div>
              </div>

              <div className="settings-toggle">
                <input
                  type="checkbox"
                  id="autocomplete"
                  checked={settings.autocomplete_enabled}
                  onChange={(e) => updateSettings({ autocomplete_enabled: e.target.checked })}
                  className="settings-checkbox"
                />
                <div>
                  <label htmlFor="autocomplete" className="settings-label cursor-pointer" style={{ marginBottom: 0 }}>
                    Enable autocomplete
                  </label>
                  <p className="text-xs text-[#70757a]">Show search suggestions as you type</p>
                </div>
              </div>

              <div>
                <button
                  type="button"
                  onClick={clearRecentSearches}
                  className="px-4 py-2 text-sm font-medium text-[#d93025] border border-[#d93025]/30 rounded-lg hover:bg-[#d93025]/5 transition-colors"
                >
                  Clear search history
                </button>
                <p className="text-xs text-[#70757a] mt-2">
                  This will remove all your recent searches
                </p>
              </div>
            </div>
          </div>

          {/* Domain preferences info */}
          <div className="bg-white rounded-lg border border-[#dadce0] p-6">
            <h2 className="font-semibold text-[#202124] mb-4">
              Domain Preferences
            </h2>

            <p className="text-sm text-[#70757a]">
              You can upvote, downvote, or block specific domains directly from search results.
              Upvoted domains will appear higher in results, downvoted domains will appear lower,
              and blocked domains will be hidden entirely.
            </p>

            <p className="text-sm text-[#70757a] mt-3">
              Look for the domain preference icons next to each search result.
            </p>
          </div>

          {/* Lenses info */}
          <div className="bg-white rounded-lg border border-[#dadce0] p-6">
            <h2 className="font-semibold text-[#202124] mb-4">
              Search Lenses
            </h2>

            <p className="text-sm text-[#70757a]">
              Search lenses allow you to filter results to specific types of content or domains.
              Use them to focus your search on programming resources, academic papers, news, or other categories.
            </p>

            <p className="text-sm text-[#70757a] mt-3">
              Add a lens filter to your search using the filter dropdown on the results page.
            </p>
          </div>
        </div>
      </main>
    </div>
  )
}
