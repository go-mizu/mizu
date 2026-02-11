import { Settings } from 'lucide-react'
import Header from '../components/Header'
import { useUIStore } from '../stores/uiStore'

export default function SettingsPage() {
  const { shelfView, setShelfView, sortBy, setSortBy } = useUIStore()

  return (
    <>
      <Header />
      <div className="page-container" style={{ maxWidth: 600, margin: '0 auto' }}>
        <h1 className="font-serif text-2xl font-bold text-gr-brown mb-8">
          <Settings size={24} className="inline mr-2" />
          Settings
        </h1>

        <div className="space-y-8">
          {/* Display Settings */}
          <div>
            <h2 className="text-sm font-bold text-gr-brown uppercase tracking-wider mb-4">
              Display
            </h2>

            <div className="form-group">
              <label className="form-label">Default Book View</label>
              <select
                className="form-input"
                value={shelfView}
                onChange={e => setShelfView(e.target.value as 'grid' | 'list' | 'table')}
              >
                <option value="grid">Grid (Covers)</option>
                <option value="list">List (Cards)</option>
                <option value="table">Table</option>
              </select>
            </div>

            <div className="form-group">
              <label className="form-label">Default Sort</label>
              <select
                className="form-input"
                value={sortBy}
                onChange={e => setSortBy(e.target.value)}
              >
                <option value="date_added">Date Added</option>
                <option value="title">Title</option>
                <option value="author">Author</option>
                <option value="rating">Rating</option>
                <option value="date_read">Date Read</option>
                <option value="pages">Pages</option>
                <option value="year">Publication Year</option>
              </select>
            </div>
          </div>

          {/* Data */}
          <div>
            <h2 className="text-sm font-bold text-gr-brown uppercase tracking-wider mb-4">
              Data
            </h2>
            <p className="text-sm text-gr-light mb-3">
              Your library data is stored locally in a SQLite database. Use the Import/Export
              page to back up or transfer your data.
            </p>
            <a href="/import-export" className="text-sm text-gr-teal font-bold hover:underline">
              Go to Import & Export →
            </a>
          </div>
        </div>
      </div>
    </>
  )
}
