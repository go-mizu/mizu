import { useState, useRef } from 'react'
import { Upload, Download, FileText, Check, BookOpen, Loader2 } from 'lucide-react'
import Header from '../components/Header'
import { booksApi } from '../api/books'

export default function ImportExportPage() {
  const [importing, setImporting] = useState(false)
  const [importResult, setImportResult] = useState<{ imported: number } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [dragActive, setDragActive] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)
  const [grUrl, setGrUrl] = useState('')
  const [grImporting, setGrImporting] = useState(false)
  const [grResult, setGrResult] = useState<{ title: string } | null>(null)
  const [grError, setGrError] = useState<string | null>(null)

  const handleFile = async (file: File) => {
    if (!file.name.endsWith('.csv')) {
      setError('Please upload a CSV file')
      return
    }
    setImporting(true)
    setError(null)
    setImportResult(null)
    try {
      const result = await booksApi.importCSV(file)
      setImportResult(result)
    } catch {
      setError('Failed to import CSV file')
    } finally {
      setImporting(false)
    }
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setDragActive(false)
    const file = e.dataTransfer.files[0]
    if (file) handleFile(file)
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) handleFile(file)
  }

  const handleCatalogImport = async () => {
    if (!grUrl.trim()) return
    setGrImporting(true)
    setGrError(null)
    setGrResult(null)
    try {
      const book = await booksApi.importSourceBook(grUrl.trim())
      setGrResult({ title: book.title })
      setGrUrl('')
    } catch {
      setGrError('Failed to import from the catalog source. Check the URL and try again.')
    } finally {
      setGrImporting(false)
    }
  }

  return (
    <>
      <Header />
      <div className="page-container page-narrow-wide">
        <h1 className="page-title page-title-lg">Import & Export</h1>

        <section className="import-section">
          <h2 className="section-title section-title-with-icon">
            <Upload size={20} />
            Import Library
          </h2>
          <p className="page-subtitle">
            Upload a CSV file containing your book library. The CSV should include columns like
            Title, Author, ISBN, Rating, and Shelf.
          </p>

          <div
            className={`dropzone${dragActive ? ' active' : ''}`}
            onDragOver={(e) => { e.preventDefault(); setDragActive(true) }}
            onDragLeave={() => setDragActive(false)}
            onDrop={handleDrop}
            onClick={() => fileRef.current?.click()}
          >
            <FileText size={36} className="dropzone-icon" />
            <p className="dropzone-title">
              Drop your CSV file here, or <span className="dropzone-link">click to browse</span>
            </p>
            <p className="dropzone-subtitle">Supports standard book library CSV format</p>
            <input
              ref={fileRef}
              type="file"
              accept=".csv"
              className="hidden"
              onChange={handleChange}
            />
          </div>

          {importing && (
            <div className="import-status-row">
              <div className="spinner spinner-sm" /> Importing...
            </div>
          )}

          {importResult && (
            <div className="import-status-row success">
              <Check size={16} /> Successfully imported {importResult.imported} books
            </div>
          )}

          {error && <div className="import-status-row error">{error}</div>}
        </section>

        <section className="import-section">
          <h2 className="section-title section-title-with-icon">
            <BookOpen size={20} />
            Import from Catalog URL
          </h2>
          <p className="page-subtitle">
            Paste a supported book URL to import all book details including ratings,
            description, genres, and reviews.
          </p>
          <div className="catalog-import-row">
            <input
              type="text"
              value={grUrl}
              onChange={(e) => setGrUrl(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleCatalogImport()}
              placeholder="https://example.com/book/..."
              className="form-input"
              disabled={grImporting}
            />
            <button
              className="btn btn-primary"
              onClick={handleCatalogImport}
              disabled={grImporting || !grUrl.trim()}
            >
              {grImporting ? <Loader2 size={16} className="spin" /> : <Upload size={16} />}
              {grImporting ? 'Importing...' : 'Import'}
            </button>
          </div>

          {grResult && (
            <div className="import-status-row success">
              <Check size={16} /> Imported "{grResult.title}" successfully
            </div>
          )}

          {grError && <div className="import-status-row error">{grError}</div>}
        </section>

        <section className="import-section">
          <h2 className="section-title section-title-with-icon">
            <Download size={20} />
            Export Library
          </h2>
          <p className="page-subtitle">
            Download your library as a CSV file. Includes all books,
            ratings, reviews, shelves, and reading dates.
          </p>
          <button className="btn btn-primary" onClick={() => booksApi.exportCSV()}>
            <Download size={16} /> Export as CSV
          </button>
        </section>
      </div>
    </>
  )
}
