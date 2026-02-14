import { useEffect, useState } from "react"
import { useSearchParams, useNavigate } from "react-router-dom"
import { Search, Globe, FileText, ArrowRight } from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { PageHeader } from "@/components/layout/header"
import { useSearch } from "@/hooks/use-api"

export default function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()
  const queryParam = searchParams.get("q") || ""
  const [input, setInput] = useState(queryParam)

  const isURL = /^https?:\/\//i.test(queryParam)
  const isDomain = !isURL && /^[a-z0-9]([a-z0-9-]*\.)+[a-z]{2,}$/i.test(queryParam)

  // Auto-redirect for URL and domain queries
  useEffect(() => {
    if (!queryParam) return
    if (isURL) {
      navigate(`/url/${encodeURIComponent(queryParam)}`, { replace: true })
    } else if (isDomain) {
      navigate(`/domain/${queryParam}`, { replace: true })
    }
  }, [queryParam, isURL, isDomain, navigate])

  const shouldSearch = queryParam && !isURL && !isDomain
  const type = searchParams.get("type") || undefined
  const { data, isLoading, error } = useSearch(
    shouldSearch ? queryParam : "",
    type
  )

  function handleSearch(e: React.FormEvent) {
    e.preventDefault()
    const q = input.trim()
    if (!q) return
    setSearchParams({ q })
  }

  return (
    <div className="mx-auto max-w-4xl px-6 py-8">
      <PageHeader title="Search" breadcrumbs={[{ label: "Search" }]} />

      <form onSubmit={handleSearch} className="flex gap-2 mb-8">
        <Input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Enter a URL, domain, or search term..."
          className="h-10"
        />
        <Button type="submit" className="h-10 px-5">
          <Search className="h-4 w-4" />
        </Button>
      </form>

      {!queryParam && (
        <div className="text-center py-16 text-muted-foreground">
          <Search className="h-12 w-12 mx-auto mb-4 opacity-30" />
          <p className="text-lg mb-2">Search Common Crawl</p>
          <div className="max-w-md mx-auto text-sm space-y-2">
            <p>Enter a full URL to find all captures of that page.</p>
            <p>Enter a domain name (e.g. <code className="bg-muted px-1.5 py-0.5 rounded text-xs">example.com</code>) to browse all pages.</p>
            <p>Or enter any keyword to search.</p>
          </div>
        </div>
      )}

      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-lg" />
          ))}
        </div>
      )}

      {error && (
        <div className="text-center py-8 text-muted-foreground">
          Failed to load search results.
        </div>
      )}

      {data && !isLoading && (
        <div className="space-y-3">
          {data.entries.length === 0 && (
            <div className="text-center py-12 text-muted-foreground">
              No results found for "{queryParam}".
            </div>
          )}

          {data.entries.length > 0 && (
            <>
              <div className="flex items-center gap-2 mb-2">
                <Badge variant="secondary" className="text-xs">{data.type}</Badge>
                <span className="text-sm text-muted-foreground">
                  {data.entries.length} result{data.entries.length !== 1 ? "s" : ""}
                  {data.crawl && <> from {data.crawl}</>}
                </span>
              </div>
              {data.entries.map((entry, i: number) => {
                const viewLink = `/view?file=${encodeURIComponent(entry.filename)}&offset=${entry.offset}&length=${entry.length}`
                let domain = ""
                try { domain = new URL(entry.url).hostname } catch {}
                return (
                  <a key={`${entry.timestamp}-${i}`} href={viewLink}>
                    <Card className="hover:border-primary/30 transition-colors cursor-pointer">
                      <CardContent className="py-3 flex items-center justify-between">
                        <div className="flex items-center gap-3 min-w-0">
                          {data.type === "domain" ? (
                            <Globe className="h-4 w-4 text-muted-foreground shrink-0" />
                          ) : (
                            <FileText className="h-4 w-4 text-muted-foreground shrink-0" />
                          )}
                          <div className="min-w-0">
                            <p className="text-sm font-medium truncate">{entry.url}</p>
                            <p className="text-xs text-muted-foreground">
                              {domain && <>{domain} &middot; </>}{entry.status} &middot; {entry.mime}
                            </p>
                          </div>
                        </div>
                        <ArrowRight className="h-3.5 w-3.5 text-muted-foreground shrink-0 ml-4" />
                      </CardContent>
                    </Card>
                  </a>
                )
              })}
            </>
          )}
        </div>
      )}
    </div>
  )
}
