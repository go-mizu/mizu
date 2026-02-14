import { useParams, useSearchParams, Link } from "react-router-dom"
import { ExternalLink, Clock, Eye } from "lucide-react"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { PageHeader } from "@/components/layout/header"
import { StatusBadge } from "@/components/status-badge"
import { useURLLookup, useCrawls } from "@/hooks/use-api"
import { fmtDate, fmtTimestamp, fmtBytes, truncURL } from "@/lib/format"

export default function URLLookupPage() {
  const params = useParams()
  const [searchParams] = useSearchParams()

  // The URL is everything after /url/ in the path
  const rawURL = params["*"] || ""
  const url = decodeURIComponent(rawURL)
  const crawlParam = searchParams.get("crawl") || ""

  const { data: crawls } = useCrawls()
  const { data, isLoading, error } = useURLLookup(url, crawlParam || undefined)

  const waybackURL = url ? `https://web.archive.org/web/*/${url}` : ""

  return (
    <div className="mx-auto max-w-4xl px-6 py-8">
      <PageHeader
        title="URL Lookup"
        breadcrumbs={[
          { label: "URL Lookup" },
        ]}
      />

      {url && (
        <Card className="mb-6">
          <CardContent className="py-4">
            <p className="text-xs text-muted-foreground mb-1">Looking up</p>
            <p className="text-sm font-mono break-all">{url}</p>
            <div className="flex gap-2 mt-3">
              <a href={url} target="_blank" rel="noopener noreferrer">
                <Button variant="outline" size="sm" className="text-xs">
                  <ExternalLink className="h-3 w-3 mr-1.5" />
                  Visit URL
                </Button>
              </a>
              <a href={waybackURL} target="_blank" rel="noopener noreferrer">
                <Button variant="outline" size="sm" className="text-xs">
                  <Clock className="h-3 w-3 mr-1.5" />
                  Wayback Machine
                </Button>
              </a>
            </div>
          </CardContent>
        </Card>
      )}

      {!url && (
        <div className="text-center py-16 text-muted-foreground">
          <p>No URL specified. Navigate here from a search or domain page.</p>
        </div>
      )}

      {error && (
        <div className="text-center py-8 text-muted-foreground">
          Failed to look up URL captures.
        </div>
      )}

      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-20 rounded-lg" />
          ))}
        </div>
      )}

      {data && (
        <>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-medium">
              {data.entries?.length || 0} capture{(data.entries?.length || 0) !== 1 ? "s" : ""} found
            </h2>
            {data.crawl && (
              <Badge variant="secondary" className="text-xs">{data.crawl}</Badge>
            )}
          </div>

          {data.entries && data.entries.length > 0 ? (
            <div className="space-y-3">
              {data.entries.map((entry, i) => (
                <Link
                  key={`${entry.timestamp}-${i}`}
                  to={`/view?file=${encodeURIComponent(entry.filename)}&offset=${entry.offset}&length=${entry.length}`}
                >
                  <Card className="hover:border-primary/30 transition-colors cursor-pointer">
                    <CardContent className="py-3">
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center gap-2">
                          <StatusBadge status={entry.status} />
                          <Badge variant="secondary" className="text-xs">{entry.mime}</Badge>
                        </div>
                        <Eye className="h-3.5 w-3.5 text-muted-foreground" />
                      </div>
                      <div className="flex items-center justify-between text-xs text-muted-foreground">
                        <span>{fmtTimestamp(entry.timestamp)}</span>
                        <span>{fmtBytes(parseInt(entry.length) || 0)}</span>
                      </div>
                    </CardContent>
                  </Card>
                </Link>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground text-sm">
              No captures found for this URL.
            </div>
          )}
        </>
      )}
    </div>
  )
}
