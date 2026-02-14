import { Settings } from 'lucide-react'
import Header from '../components/Header'
import { useUIStore } from '../stores/uiStore'

export default function SettingsPage() {
  const { shelfView, setShelfView, sortBy, setSortBy } = useUIStore()

  return (
    <>
      <Header />
      <div className="page-container page-narrow settings-page">
        <h1 className="page-title page-title-lg section-title-with-icon">
          <Settings size={24} />
          Settings
        </h1>

        <div className="settings-stack">
          <section>
            <h2 className="settings-heading">Display</h2>

            <div className="form-group">
              <label className="form-label">Default Book View</label>
              <select
                className="form-input"
                value={shelfView}
                onChange={(e) => setShelfView(e.target.value as 'grid' | 'list' | 'table')}
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
                onChange={(e) => setSortBy(e.target.value)}
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
          </section>

          <section>
            <h2 className="settings-heading">Data</h2>
            <p className="page-subtitle">
              Your library data is stored locally in a SQLite database. Use the Import/Export
              page to back up or transfer your data.
            </p>
            <a href="/import-export" className="meta-link">
              Go to Import & Export
            </a>
          </section>
        </div>
      </div>
    </>
  )
}
