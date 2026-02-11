import { lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import HomePage from './pages/HomePage'

const SearchPage = lazy(() => import('./pages/SearchPage'))
const BookDetailPage = lazy(() => import('./pages/BookDetailPage'))
const AuthorPage = lazy(() => import('./pages/AuthorPage'))
const MyBooksPage = lazy(() => import('./pages/MyBooksPage'))
const BrowsePage = lazy(() => import('./pages/BrowsePage'))
const GenrePage = lazy(() => import('./pages/GenrePage'))
const ChallengePage = lazy(() => import('./pages/ChallengePage'))
const ListsPage = lazy(() => import('./pages/ListsPage'))
const ListDetailPage = lazy(() => import('./pages/ListDetailPage'))
const StatsPage = lazy(() => import('./pages/StatsPage'))
const ImportExportPage = lazy(() => import('./pages/ImportExportPage'))
const SettingsPage = lazy(() => import('./pages/SettingsPage'))

function PageLoader() {
  return (
    <div className="loading-spinner">
      <div className="spinner" />
    </div>
  )
}

function App() {
  return (
    <BrowserRouter>
      <Suspense fallback={<PageLoader />}>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/search" element={<SearchPage />} />
          <Route path="/book/:id" element={<BookDetailPage />} />
          <Route path="/author/:id" element={<AuthorPage />} />
          <Route path="/my-books" element={<MyBooksPage />} />
          <Route path="/browse" element={<BrowsePage />} />
          <Route path="/genre/:genre" element={<GenrePage />} />
          <Route path="/challenge" element={<ChallengePage />} />
          <Route path="/lists" element={<ListsPage />} />
          <Route path="/list/:id" element={<ListDetailPage />} />
          <Route path="/stats" element={<StatsPage />} />
          <Route path="/import-export" element={<ImportExportPage />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Routes>
      </Suspense>
    </BrowserRouter>
  )
}

export default App
