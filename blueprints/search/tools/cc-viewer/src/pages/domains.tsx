import { useState } from "react"
import { Link, useSearchParams } from "react-router-dom"
import { Globe, ChevronRight } from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Pagination } from "@/components/ui/pagination"
import { PageHeader } from "@/components/layout/header"
import { useCrawls, useDomains } from "@/hooks/use-api"
import { fmtNum, fmtBytes } from "@/lib/format"

export default function DomainsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const page = parseInt(searchParams.get("page") || "0")
  const crawlParam = searchParams.get("crawl") || ""

  const { data: crawls } = useCrawls()
  const [selectedCrawl, setSelectedCrawl] = useState(crawlParam)
  const activeCrawl = selectedCrawl || crawls?.[0]?.id || ""

  const { data, isLoading, error } = useDomains(activeCrawl, page)

  function handleCrawlChange(e: React.ChangeEvent<HTMLSelectElement>) {
    const crawl = e.target.value
    setSelectedCrawl(crawl)
    setSearchParams((prev) => {
      prev.set("crawl", crawl)
      prev.set("page", "0")
      return prev
    })
  }

  function setPage(p: number) {
    setSearchParams((prev) => {
      prev.set("page", String(p))
      return prev
    })
  }

  return (
    <div className="mx-auto max-w-4xl px-6 py-8">
      <PageHeader
        title="Browse Domains"
        breadcrumbs={[{ label: "Domains" }]}
      />

      {/* Crawl selector */}
      <div className="mb-6">
        <label className="text-xs text-muted-foreground mb-1 block">Crawl</label>
        <select
          value={activeCrawl}
          onChange={handleCrawlChange}
          className="w-full sm:w-72 h-9 px-3 rounded-md border border-input bg-background text-sm"
        >
          {crawls?.map((c) => (
            <option key={c.id} value={c.id}>{c.id}</option>
          ))}
        </select>
      </div>

      {error && (
        <div className="text-center py-12 text-muted-foreground">
          Failed to load domains.
        </div>
      )}

      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 10 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-lg" />
          ))}
        </div>
      )}

      {data && (
        <>
          <p className="text-sm text-muted-foreground mb-4">
            {data.domains.length > 0 ? `${fmtNum(data.domains.length)} domains shown` : "No domains found"}
          </p>

          <div className="space-y-2">
            {data.domains.map((domain) => (
              <Link key={domain.domain} to={`/domain/${domain.domain}`}>
                <Card className="hover:border-primary/30 transition-colors cursor-pointer group">
                  <CardContent className="py-3 flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <Globe className="h-4 w-4 text-muted-foreground shrink-0" />
                      <div>
                        <p className="text-sm font-medium">{domain.domain}</p>
                        <p className="text-xs text-muted-foreground">
                          {fmtNum(domain.pages)} pages
                          {domain.size > 0 && <> &middot; {fmtBytes(domain.size)}</>}
                        </p>
                      </div>
                    </div>
                    <ChevronRight className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
                  </CardContent>
                </Card>
              </Link>
            ))}
          </div>

          {data.totalPages > 1 && (
            <div className="mt-6 flex justify-center">
              <Pagination
                currentPage={page}
                totalPages={data.totalPages}
                onPageChange={setPage}
              />
            </div>
          )}
        </>
      )}
    </div>
  )
}
