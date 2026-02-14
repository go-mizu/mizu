import { Link } from "react-router-dom"
import { Database, ChevronRight } from "lucide-react"
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { PageHeader } from "@/components/layout/header"
import { useCrawls } from "@/hooks/use-api"
import { crawlToDate } from "@/lib/format"

export default function CrawlsPage() {
  const { data: crawls, isLoading, error } = useCrawls()

  return (
    <div>
      <PageHeader
        title="Crawls"
        breadcrumbs={[{ label: "Crawls" }]}
      />

      {error && (
        <div className="text-center py-12 text-muted-foreground">
          Failed to load crawls. Please try again later.
        </div>
      )}

      {isLoading && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 9 }).map((_, i) => (
            <Skeleton key={i} className="h-36 rounded-xl" />
          ))}
        </div>
      )}

      {crawls && (
        <>
          <p className="text-sm text-muted-foreground mb-6">
            {crawls.length} crawls available
          </p>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {crawls.map((crawl) => (
              <Link key={crawl.id} to={`/crawl/${encodeURIComponent(crawl.id)}`}>
                <Card className="hover:border-primary/40 transition-colors cursor-pointer h-full group">
                  <CardHeader className="pb-2">
                    <div className="flex items-center justify-between">
                      <Database className="h-4 w-4 text-primary" />
                      <ChevronRight className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
                    </div>
                    <CardTitle className="text-sm font-medium mt-2">{crawl.id}</CardTitle>
                    <CardDescription className="text-xs">{crawlToDate(crawl.id)}</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <p className="text-xs text-muted-foreground mb-2 line-clamp-1">{crawl.name}</p>
                    <div className="flex gap-2">
                      {crawl.from && (
                        <Badge variant="secondary" className="text-[10px]">
                          {crawl.from} - {crawl.to}
                        </Badge>
                      )}
                    </div>
                  </CardContent>
                </Card>
              </Link>
            ))}
          </div>
        </>
      )}
    </div>
  )
}
