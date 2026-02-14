import { useState, useMemo } from "react"
import { useParams, useSearchParams, Link } from "react-router-dom"
import { Globe, FileText, Eye, ChevronRight } from "lucide-react"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Pagination } from "@/components/ui/pagination"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table"
import { PageHeader } from "@/components/layout/header"
import { KPICard } from "@/components/kpi-card"
import { StatusBadge } from "@/components/status-badge"
import { useCrawls, useDomain } from "@/hooks/use-api"
import { fmtNum, fmtBytes, fmtTimestamp, truncURL } from "@/lib/format"
import { cn } from "@/lib/utils"

const STATUS_COLORS: Record<string, string> = {
  "2": "bg-emerald-500",
  "3": "bg-amber-500",
  "4": "bg-red-400",
  "5": "bg-red-600",
}

function statusCategory(code: string): string {
  return code.charAt(0)
}

export default function DomainDetailPage() {
  const { domain } = useParams<{ domain: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const page = parseInt(searchParams.get("page") || "0")
  const crawlParam = searchParams.get("crawl") || ""
  const [expandedURL, setExpandedURL] = useState<string | null>(null)

  const { data: crawls } = useCrawls()
  const [selectedCrawl, setSelectedCrawl] = useState(crawlParam)
  const activeCrawl = selectedCrawl || crawls?.[0]?.id || ""

  const { data, isLoading, error } = useDomain(domain || "", activeCrawl, page)

  const stats = data?.stats

  // Group entries by URL
  const groups = useMemo(() => {
    if (!data?.entries) return []
    const map = new Map<string, { url: string; path: string; entries: typeof data.entries; latestStatus: string; latestMime: string }>()
    for (const entry of data.entries) {
      let path = entry.url
      try { path = new URL(entry.url).pathname } catch {}
      const existing = map.get(entry.url)
      if (existing) {
        existing.entries.push(entry)
      } else {
        map.set(entry.url, {
          url: entry.url,
          path,
          entries: [entry],
          latestStatus: entry.status,
          latestMime: entry.mime,
        })
      }
    }
    return Array.from(map.values())
  }, [data?.entries])

  function setPage(p: number) {
    setSearchParams((prev) => {
      prev.set("page", String(p))
      return prev
    })
  }

  // Compute status distribution percentages
  const statusEntries = stats?.statusCounts ? Object.entries(stats.statusCounts) : []
  const totalStatusPages = statusEntries.reduce((sum, [, count]) => sum + count, 0)

  // Compute MIME distribution percentages
  const mimeEntries = stats?.mimeCounts ? Object.entries(stats.mimeCounts).sort((a, b) => b[1] - a[1]).slice(0, 8) : []
  const totalMimePages = mimeEntries.reduce((sum, [, count]) => sum + count, 0)

  // Success rate
  const successCount = statusEntries
    .filter(([code]) => code.startsWith("2"))
    .reduce((sum, [, count]) => sum + count, 0)
  const successRate = totalStatusPages > 0 ? Math.round((successCount / totalStatusPages) * 100) : 0

  return (
    <div className="mx-auto max-w-5xl px-6 py-8">
      <PageHeader
        title={domain || ""}
        breadcrumbs={[
          { label: "Domains", href: "/domains" },
          { label: domain || "" },
        ]}
      />

      {/* Crawl selector */}
      <div className="mb-6">
        <label className="text-xs text-muted-foreground mb-1 block">Crawl</label>
        <select
          value={activeCrawl}
          onChange={(e) => {
            setSelectedCrawl(e.target.value)
            setSearchParams((prev) => {
              prev.set("crawl", e.target.value)
              prev.set("page", "0")
              return prev
            })
          }}
          className="w-full sm:w-72 h-9 px-3 rounded-md border border-input bg-background text-sm"
        >
          {crawls?.map((c) => (
            <option key={c.id} value={c.id}>{c.id}</option>
          ))}
        </select>
      </div>

      {error && (
        <div className="text-center py-12 text-muted-foreground">
          Failed to load domain data.
        </div>
      )}

      {isLoading && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-xl" />
          ))}
        </div>
      )}

      {stats && (
        <>
          {/* KPIs */}
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
            <KPICard title="Total Pages" value={fmtNum(stats.totalPages)} icon={<FileText className="h-4 w-4" />} />
            <KPICard title="Unique Paths" value={fmtNum(stats.uniquePaths)} icon={<Globe className="h-4 w-4" />} />
            <KPICard title="Total Size" value={fmtBytes(stats.totalSize)} icon={<Eye className="h-4 w-4" />} />
            <KPICard title="Success Rate" value={`${successRate}%`} icon={<Eye className="h-4 w-4" />} />
          </div>

          {/* Status distribution */}
          <Card className="mb-6">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">HTTP Status Distribution</CardTitle>
            </CardHeader>
            <CardContent>
              {totalStatusPages > 0 && (
                <>
                  <div className="flex h-3 rounded-full overflow-hidden mb-3">
                    {statusEntries.map(([code, count]) => {
                      const pct = (count / totalStatusPages) * 100
                      const cat = statusCategory(code)
                      return (
                        <div
                          key={code}
                          className={cn("transition-all", STATUS_COLORS[cat] || "bg-gray-400")}
                          style={{ width: `${pct}%` }}
                          data-tooltip={`${code}: ${fmtNum(count)} (${pct.toFixed(1)}%)`}
                        />
                      )
                    })}
                  </div>
                  <div className="flex flex-wrap gap-3 text-xs">
                    {statusEntries.map(([code, count]) => (
                      <div key={code} className="flex items-center gap-1.5">
                        <div className={cn("w-2.5 h-2.5 rounded-sm", STATUS_COLORS[statusCategory(code)] || "bg-gray-400")} />
                        <span className="text-muted-foreground">{code}:</span>
                        <span className="font-medium">{fmtNum(count)}</span>
                      </div>
                    ))}
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {/* MIME distribution */}
          {mimeEntries.length > 0 && (
            <Card className="mb-8">
              <CardHeader className="pb-3">
                <CardTitle className="text-sm">Content Types</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  {mimeEntries.map(([mime, count]) => {
                    const pct = totalMimePages > 0 ? (count / totalMimePages) * 100 : 0
                    return (
                      <div key={mime} className="flex items-center gap-3 text-xs">
                        <span className="w-32 truncate text-muted-foreground">{mime}</span>
                        <div className="flex-1 h-2 bg-muted rounded-full overflow-hidden">
                          <div className="h-full bg-primary rounded-full" style={{ width: `${pct}%` }} />
                        </div>
                        <span className="w-16 text-right font-medium">{fmtNum(count)}</span>
                      </div>
                    )
                  })}
                </div>
              </CardContent>
            </Card>
          )}
        </>
      )}

      {/* URL groups table */}
      {groups.length > 0 && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm">URLs</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <div className="border-t">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>URL</TableHead>
                    <TableHead className="w-16">Status</TableHead>
                    <TableHead className="w-20">MIME</TableHead>
                    <TableHead className="w-16 text-right">Count</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {groups.map((group) => (
                    <>
                      <TableRow
                        key={group.url}
                        className="cursor-pointer hover:bg-muted/50"
                        onClick={() => setExpandedURL(expandedURL === group.url ? null : group.url)}
                      >
                        <TableCell className="text-xs">
                          <div className="flex items-center gap-1.5">
                            <ChevronRight className={cn(
                              "h-3 w-3 text-muted-foreground transition-transform shrink-0",
                              expandedURL === group.url && "rotate-90"
                            )} />
                            <span className="truncate max-w-md">{truncURL(group.path || group.url, 70)}</span>
                          </div>
                        </TableCell>
                        <TableCell><StatusBadge status={group.latestStatus} /></TableCell>
                        <TableCell className="text-xs text-muted-foreground">{group.latestMime}</TableCell>
                        <TableCell className="text-right text-xs font-medium">{group.entries.length}</TableCell>
                      </TableRow>
                      {expandedURL === group.url && group.entries.map((entry, i) => (
                        <TableRow key={`${group.url}-${i}`} className="bg-muted/30">
                          <TableCell className="text-xs pl-10">
                            <Link
                              to={`/view?file=${encodeURIComponent(entry.filename)}&offset=${entry.offset}&length=${entry.length}`}
                              className="text-primary hover:underline"
                            >
                              {fmtTimestamp(entry.timestamp)}
                            </Link>
                          </TableCell>
                          <TableCell><StatusBadge status={entry.status} /></TableCell>
                          <TableCell className="text-xs text-muted-foreground">{entry.mime}</TableCell>
                          <TableCell className="text-right text-xs text-muted-foreground">
                            {fmtBytes(parseInt(entry.length) || 0)}
                          </TableCell>
                        </TableRow>
                      ))}
                    </>
                  ))}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      )}

      {data && data.totalPages > 1 && (
        <div className="mt-6 flex justify-center">
          <Pagination
            currentPage={page}
            totalPages={data.totalPages}
            onPageChange={setPage}
          />
        </div>
      )}
    </div>
  )
}
