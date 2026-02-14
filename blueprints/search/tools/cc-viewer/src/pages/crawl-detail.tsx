import { useState } from "react"
import { useParams, useSearchParams, Link } from "react-router-dom"
import { Database, FileText, Search, Server, HardDrive, Layers, Hash } from "lucide-react"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table"
import { Pagination } from "@/components/ui/pagination"
import { PageHeader } from "@/components/layout/header"
import { KPICard } from "@/components/kpi-card"
import { StatusBadge } from "@/components/status-badge"
import { useCrawl, useCrawlFiles, useCrawlCDX } from "@/hooks/use-api"
import { fmtNum, fmtBytes, fmtTimestamp, truncURL, crawlToDate } from "@/lib/format"

export default function CrawlDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = searchParams.get("tab") || "overview"
  const filesPage = parseInt(searchParams.get("fp") || "0")
  const cdxPage = parseInt(searchParams.get("cp") || "0")
  const [cdxPrefix, setCdxPrefix] = useState(searchParams.get("prefix") || "")
  const [prefixInput, setPrefixInput] = useState(cdxPrefix)

  const crawlId = id ? decodeURIComponent(id) : ""
  const { data: crawlData, isLoading, error } = useCrawl(crawlId)
  const { data: filesData, isLoading: filesLoading } = useCrawlFiles(crawlId, filesPage)
  const { data: cdxData, isLoading: cdxLoading } = useCrawlCDX(crawlId, cdxPrefix, cdxPage)

  function setTab(t: string) {
    setSearchParams((prev) => {
      prev.set("tab", t)
      return prev
    })
  }

  function setFilesPage(p: number) {
    setSearchParams((prev) => {
      prev.set("fp", String(p))
      return prev
    })
  }

  function setCdxPage(p: number) {
    setSearchParams((prev) => {
      prev.set("cp", String(p))
      return prev
    })
  }

  function handlePrefixSearch(e: React.FormEvent) {
    e.preventDefault()
    setCdxPrefix(prefixInput.trim())
    setSearchParams((prev) => {
      prev.set("prefix", prefixInput.trim())
      prev.set("cp", "0")
      return prev
    })
  }

  const stats = crawlData?.stats

  return (
    <div className="mx-auto max-w-5xl px-6 py-8">
      <PageHeader
        title={crawlId}
        breadcrumbs={[
          { label: "Crawls", href: "/crawls" },
          { label: crawlId },
        ]}
      />

      {error && (
        <div className="text-center py-12 text-muted-foreground">
          Failed to load crawl details.
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
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
          <KPICard title="WARC Files" value={fmtNum(stats.warcFiles)} icon={<FileText className="h-4 w-4" />} />
          <KPICard title="Segments" value={fmtNum(stats.segments)} icon={<Layers className="h-4 w-4" />} />
          <KPICard title="Index Files" value={fmtNum(stats.indexFiles)} icon={<Hash className="h-4 w-4" />} />
          <KPICard title="Estimated Size" value={fmtBytes(stats.estimatedSizeBytes)} icon={<HardDrive className="h-4 w-4" />} />
        </div>
      )}

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList className="mb-6">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="files">Files</TabsTrigger>
          <TabsTrigger value="cdx">CDX Index</TabsTrigger>
        </TabsList>

        <TabsContent value="overview">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Link to={`/crawl/${id}?tab=files`} onClick={() => setTab("files")}>
              <Card className="hover:border-primary/40 transition-colors cursor-pointer h-full">
                <CardContent className="pt-5">
                  <Server className="h-5 w-5 text-primary mb-3" />
                  <p className="font-medium text-sm mb-1">Browse WARC Files</p>
                  <p className="text-xs text-muted-foreground">
                    View the raw WARC files for this crawl, organized by segment.
                  </p>
                </CardContent>
              </Card>
            </Link>
            <Card className="hover:border-primary/40 transition-colors cursor-pointer" onClick={() => setTab("cdx")}>
              <CardContent className="pt-5">
                <Database className="h-5 w-5 text-primary mb-3" />
                <p className="font-medium text-sm mb-1">Browse CDX Index</p>
                <p className="text-xs text-muted-foreground">
                  Search the CDX index by URL prefix for this crawl.
                </p>
              </CardContent>
            </Card>
          </div>

          {crawlData?.crawl && (
            <Card className="mt-6">
              <CardHeader>
                <CardTitle className="text-sm">Crawl Information</CardTitle>
              </CardHeader>
              <CardContent>
                <dl className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
                  <div>
                    <dt className="text-muted-foreground text-xs mb-1">Crawl ID</dt>
                    <dd className="font-medium">{crawlData.crawl.id}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground text-xs mb-1">Date</dt>
                    <dd className="font-medium">{crawlToDate(crawlData.crawl.id)}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground text-xs mb-1">Name</dt>
                    <dd className="font-medium">{crawlData.crawl.name}</dd>
                  </div>
                  {crawlData.crawl.from && (
                    <div>
                      <dt className="text-muted-foreground text-xs mb-1">Date Range</dt>
                      <dd className="font-medium">{crawlData.crawl.from} &ndash; {crawlData.crawl.to}</dd>
                    </div>
                  )}
                </dl>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="files">
          {filesLoading && (
            <div className="space-y-2">
              {Array.from({ length: 10 }).map((_, i) => (
                <Skeleton key={i} className="h-10 rounded" />
              ))}
            </div>
          )}

          {filesData && (
            <>
              <p className="text-sm text-muted-foreground mb-4">
                {fmtNum(filesData.totalFiles)} files across {stats?.segments ?? "?"} segments
              </p>
              <div className="border rounded-lg overflow-hidden">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>File Path</TableHead>
                      <TableHead className="w-28 text-right">Segment</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filesData.files.map((file: string, i: number) => {
                      const parts = file.split("/")
                      const segment = parts.find((p: string) => /^\d{13}\.\d+$/.test(p)) || ""
                      return (
                        <TableRow key={i}>
                          <TableCell className="font-mono text-xs truncate max-w-lg">
                            {truncURL(file, 100)}
                          </TableCell>
                          <TableCell className="text-right text-xs text-muted-foreground">
                            {segment ? segment.substring(0, 10) + "..." : "--"}
                          </TableCell>
                        </TableRow>
                      )
                    })}
                  </TableBody>
                </Table>
              </div>
              {filesData.totalPages > 1 && (
                <div className="mt-4 flex justify-center">
                  <Pagination
                    currentPage={filesData.page}
                    totalPages={filesData.totalPages}
                    onPageChange={setFilesPage}
                  />
                </div>
              )}
            </>
          )}
        </TabsContent>

        <TabsContent value="cdx">
          <form onSubmit={handlePrefixSearch} className="flex gap-2 mb-6">
            <Input
              value={prefixInput}
              onChange={(e) => setPrefixInput(e.target.value)}
              placeholder="URL prefix (e.g. com,example)/"
              className="font-mono text-sm"
            />
            <Button type="submit" size="default">
              <Search className="h-4 w-4" />
            </Button>
          </form>

          {cdxLoading && (
            <div className="space-y-2">
              {Array.from({ length: 10 }).map((_, i) => (
                <Skeleton key={i} className="h-10 rounded" />
              ))}
            </div>
          )}

          {cdxData && cdxData.entries.length > 0 && (
            <>
              <div className="border rounded-lg overflow-hidden">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>URL</TableHead>
                      <TableHead className="w-20">Status</TableHead>
                      <TableHead className="w-28">MIME</TableHead>
                      <TableHead className="w-36">Timestamp</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {cdxData.entries.map((entry: any, i: number) => (
                      <TableRow key={i}>
                        <TableCell className="text-xs truncate max-w-sm">
                          <Link
                            to={`/view?file=${encodeURIComponent(entry.filename)}&offset=${entry.offset}&length=${entry.length}`}
                            className="text-primary hover:underline"
                          >
                            {truncURL(entry.url, 60)}
                          </Link>
                        </TableCell>
                        <TableCell>
                          <StatusBadge status={entry.status} />
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">{entry.mime}</TableCell>
                        <TableCell className="text-xs text-muted-foreground">{fmtTimestamp(entry.timestamp)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
              {cdxData.totalPages > 1 && (
                <div className="mt-4 flex justify-center">
                  <Pagination
                    currentPage={cdxPage}
                    totalPages={cdxData.totalPages}
                    onPageChange={setCdxPage}
                  />
                </div>
              )}
            </>
          )}

          {cdxData && cdxData.entries.length === 0 && cdxPrefix && (
            <div className="text-center py-8 text-muted-foreground text-sm">
              No CDX entries found for prefix "{cdxPrefix}".
            </div>
          )}

          {!cdxPrefix && !cdxLoading && (
            <div className="text-center py-8 text-muted-foreground text-sm">
              Enter a URL prefix to browse the CDX index.
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}
