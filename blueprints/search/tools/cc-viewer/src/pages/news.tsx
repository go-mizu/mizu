import { useState } from "react"
import { useSearchParams, Link } from "react-router-dom"
import { Newspaper, Clock, Eye } from "lucide-react"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Pagination } from "@/components/ui/pagination"
import { PageHeader } from "@/components/layout/header"
import { StatusBadge } from "@/components/status-badge"
import { useNews, useNewsDates } from "@/hooks/use-api"
import { fmtTimestamp, truncURL } from "@/lib/format"

export default function NewsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const dateParam = searchParams.get("date") || ""
  const page = parseInt(searchParams.get("page") || "0")

  const { data: datesResult, isLoading: datesLoading } = useNewsDates()
  const datesList = datesResult?.dates || []
  const [selectedDate, setSelectedDate] = useState(dateParam)
  const activeDate = selectedDate || datesList[0] || ""

  const { data, isLoading, error } = useNews(activeDate, page)

  function handleDateChange(e: React.ChangeEvent<HTMLSelectElement>) {
    const date = e.target.value
    setSelectedDate(date)
    setSearchParams((prev) => {
      prev.set("date", date)
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
        title="CC-NEWS Archive"
        breadcrumbs={[{ label: "News" }]}
      />

      {/* Date selector */}
      <div className="mb-6">
        <label className="text-xs text-muted-foreground mb-1 block">Date</label>
        {datesLoading ? (
          <Skeleton className="h-9 w-72 rounded-md" />
        ) : (
          <select
            value={activeDate}
            onChange={handleDateChange}
            className="w-full sm:w-72 h-9 px-3 rounded-md border border-input bg-background text-sm"
          >
            {datesList.map((d) => (
              <option key={d} value={d}>{d}</option>
            ))}
          </select>
        )}
      </div>

      {!activeDate && !datesLoading && (
        <div className="text-center py-16 text-muted-foreground">
          <Newspaper className="h-12 w-12 mx-auto mb-4 opacity-30" />
          <p>No news dates available.</p>
        </div>
      )}

      {error && (
        <div className="text-center py-8 text-muted-foreground">
          Failed to load news entries.
        </div>
      )}

      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-20 rounded-lg" />
          ))}
        </div>
      )}

      {data && data.entries && (
        <>
          <p className="text-sm text-muted-foreground mb-4">
            {data.entries.length} articles for {activeDate}
          </p>

          <div className="space-y-2">
            {data.entries.map((entry: any, i: number) => {
              const viewLink = `/view?file=${encodeURIComponent(entry.filename)}&offset=${entry.offset}&length=${entry.length}`

              // Extract domain from URL for display
              let domain = ""
              try {
                domain = new URL(entry.url).hostname
              } catch {
                domain = ""
              }

              return (
                <Link key={`${entry.timestamp}-${i}`} to={viewLink}>
                  <Card className="hover:border-primary/30 transition-colors cursor-pointer mb-2">
                    <CardContent className="py-3">
                      <div className="flex items-start justify-between gap-4">
                        <div className="min-w-0 flex-1">
                          <p className="text-sm font-medium truncate mb-1">
                            {truncURL(entry.url, 80)}
                          </p>
                          <div className="flex items-center gap-2 text-xs text-muted-foreground">
                            {domain && <span>{domain}</span>}
                            <span>&middot;</span>
                            <span>{fmtTimestamp(entry.timestamp)}</span>
                          </div>
                        </div>
                        <div className="flex items-center gap-2 shrink-0">
                          <StatusBadge status={entry.status} />
                          <Badge variant="secondary" className="text-[10px]">{entry.mime}</Badge>
                          <Eye className="h-3.5 w-3.5 text-muted-foreground" />
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </Link>
              )
            })}
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
