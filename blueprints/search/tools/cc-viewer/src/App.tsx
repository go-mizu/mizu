import { lazy, Suspense } from "react"
import { Routes, Route } from "react-router-dom"
import { AppLayout } from "@/components/layout/app-layout"
import { Skeleton } from "@/components/ui/skeleton"

const Dashboard = lazy(() => import("@/pages/dashboard"))
const Crawls = lazy(() => import("@/pages/crawls"))
const CrawlDetail = lazy(() => import("@/pages/crawl-detail"))
const Domains = lazy(() => import("@/pages/domains"))
const DomainDetail = lazy(() => import("@/pages/domain-detail"))
const URLLookup = lazy(() => import("@/pages/url-lookup"))
const Search = lazy(() => import("@/pages/search"))
const WARCViewer = lazy(() => import("@/pages/warc-viewer"))
const WATBrowser = lazy(() => import("@/pages/wat-browser"))
const WETBrowser = lazy(() => import("@/pages/wet-browser"))
const Statistics = lazy(() => import("@/pages/statistics"))
const WebGraph = lazy(() => import("@/pages/web-graph"))
const News = lazy(() => import("@/pages/news"))

function PageSkeleton() {
  return (
    <div className="space-y-4 p-2">
      <Skeleton className="h-8 w-48" />
      <Skeleton className="h-4 w-96" />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mt-6">
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
      </div>
      <Skeleton className="h-64 mt-6" />
    </div>
  )
}

export default function App() {
  return (
    <AppLayout>
      <Suspense fallback={<PageSkeleton />}>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/crawls" element={<Crawls />} />
          <Route path="/crawl/:id" element={<CrawlDetail />} />
          <Route path="/crawl/:id/files" element={<CrawlDetail />} />
          <Route path="/crawl/:id/cdx" element={<CrawlDetail />} />
          <Route path="/domains" element={<Domains />} />
          <Route path="/domain/:domain" element={<DomainDetail />} />
          <Route path="/url/*" element={<URLLookup />} />
          <Route path="/search" element={<Search />} />
          <Route path="/view" element={<WARCViewer />} />
          <Route path="/wat" element={<WATBrowser />} />
          <Route path="/wet" element={<WETBrowser />} />
          <Route path="/stats" element={<Statistics />} />
          <Route path="/statistics" element={<Statistics />} />
          <Route path="/graph" element={<WebGraph />} />
          <Route path="/news" element={<News />} />
          <Route
            path="*"
            element={
              <div className="flex flex-col items-center justify-center py-24 text-center">
                <h1 className="text-4xl font-semibold mb-2">404</h1>
                <p className="text-muted-foreground">Page not found.</p>
              </div>
            }
          />
        </Routes>
      </Suspense>
    </AppLayout>
  )
}
