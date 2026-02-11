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

  const handleGoodreadsImport = async () => {
    if (!grUrl.trim()) return
    setGrImporting(true)
    setGrError(null)
    setGrResult(null)
    try {
      const book = await booksApi.importGoodreads(grUrl.trim())
      setGrResult({ title: book.title })
      setGrUrl('')
    } catch {
      setGrError('Failed to import from Goodreads. Check the URL and try again.')
    } finally {
      setGrImporting(false)
    }
  }

  return (
    <>
      <Header />
      <div className="page-container" style={{ maxWidth: 700, margin: '0 auto' }}>
        <h1 className="font-serif text-2xl font-bold text-gr-brown mb-8">Import & Export</h1>

        {/* Import Section */}
        <div className="mb-10">
          <h2 className="section-title mb-4">
            <Upload size={20} className="inline mr-2" />
            Import Library
          </h2>
          <p className="text-sm text-gr-light mb-4">
            Upload a CSV file containing your book library. The CSV should include columns like
            Title, Author, ISBN, Rating, and Shelf.
          </p>

          <div
            className={`border-2 border-dashed rounded-lg p-10 text-center transition-colors cursor-pointer ${
              dragActive ? 'border-gr-teal bg-gr-cream' : 'border-gr-border hover:border-gr-teal'
            }`}
            onDragOver={e => { e.preventDefault(); setDragActive(true) }}
            onDragLeave={() => setDragActive(false)}
            onDrop={handleDrop}
            onClick={() => fileRef.current?.click()}
          >
            <FileText size={36} className="mx-auto text-gr-light mb-3" />
            <p className="text-sm text-gr-text mb-1">
              Drop your CSV file here, or <span className="text-gr-teal font-bold">click to browse</span>
            </p>
            <p className="text-xs text-gr-light">Supports standard book library CSV format</p>
            <input
              ref={fileRef}
              type="file"
              accept=".csv"
              className="hidden"
              onChange={handleChange}
            />
          </div>

          {importing && (
            <div className="mt-4 flex items-center gap-2 text-sm text-gr-light">
              <div className="spinner" style={{ width: 16, height: 16 }} /> Importing...
            </div>
          )}

          {importResult && (
            <div className="mt-4 flex items-center gap-2 text-sm text-gr-green">
              <Check size={16} /> Successfully imported {importResult.imported} books
            </div>
          )}

          {error && (
            <div className="mt-4 text-sm text-red-600">{error}</div>
          )}
        </div>

        {/* Goodreads Import */}
        <div className="mb-10">
          <h2 className="section-title mb-4">
            <BookOpen size={20} className="inline mr-2" />
            Import from Goodreads
          </h2>
          <p className="text-sm text-gr-light mb-4">
            Paste a Goodreads book URL to import all book details including ratings,
            description, genres, and reviews.
          </p>
          <div className="flex gap-2">
            <input
              type="text"
              value={grUrl}
              onChange={e => setGrUrl(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && handleGoodreadsImport()}
              placeholder="https://www.goodreads.com/book/show/..."
              className="flex-1 px-3 py-2 border border-gr-border rounded-md text-sm focus:outline-none focus:border-gr-teal"
              disabled={grImporting}
            />
            <button
              className="btn btn-primary"
              onClick={handleGoodreadsImport}
              disabled={grImporting || !grUrl.trim()}
            >
              {grImporting ? <Loader2 size={16} className="animate-spin" /> : <Upload size={16} />}
              {grImporting ? 'Importing...' : 'Import'}
            </button>
          </div>

          {grResult && (
            <div className="mt-4 flex items-center gap-2 text-sm text-gr-green">
              <Check size={16} /> Imported "{grResult.title}" successfully
            </div>
          )}

          {grError && (
            <div className="mt-4 text-sm text-red-600">{grError}</div>
          )}
        </div>

        {/* Export Section */}
        <div>
          <h2 className="section-title mb-4">
            <Download size={20} className="inline mr-2" />
            Export Library
          </h2>
          <p className="text-sm text-gr-light mb-4">
            Download your library as a CSV file. Includes all books,
            ratings, reviews, shelves, and reading dates.
          </p>
          <button className="btn btn-primary" onClick={() => booksApi.exportCSV()}>
            <Download size={16} /> Export as CSV
          </button>
        </div>
      </div>
    </>
  )
}
