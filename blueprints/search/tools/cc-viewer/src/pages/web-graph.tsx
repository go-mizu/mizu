import { useState } from "react"
import { useSearchParams } from "react-router-dom"
import { Network, Search, Download, ExternalLink } from "lucide-react"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { PageHeader } from "@/components/layout/header"
import { KPICard } from "@/components/kpi-card"
import { useCrawls, useGraph, useGraphRank } from "@/hooks/use-api"
import { fmtNum } from "@/lib/format"

export default function WebGraphPage() {
  const [searchParams] = useSearchParams()
  const crawlParam = searchParams.get("crawl") || ""
  const { data: crawls } = useCrawls()
  const [selectedCrawl, setSelectedCrawl] = useState(crawlParam)
  const activeCrawl = selectedCrawl || crawls?.[0]?.id || ""

  const [domainSearch, setDomainSearch] = useState("")
  const [searchedDomain, setSearchedDomain] = useState("")

  const graphType = (searchParams.get("type") || "host") as "host" | "domain"
  const { data: graphData, isLoading, error } = useGraph(activeCrawl, graphType)
  const { data: rankData, isLoading: rankLoading } = useGraphRank(searchedDomain)

  function handleDomainSearch(e: React.FormEvent) {
    e.preventDefault()
    setSearchedDomain(domainSearch.trim())
  }

  const topRanked = graphData?.topRanked?.slice(0, 50) || []

  return (
    <div className="mx-auto max-w-5xl px-6 py-8">
      <PageHeader
        title="Web Graph"
        breadcrumbs={[{ label: "Web Graph" }]}
      />

      {/* Crawl selector */}
      <div className="mb-6">
        <label className="text-xs text-muted-foreground mb-1 block">Crawl</label>
        <select
          value={activeCrawl}
          onChange={(e) => setSelectedCrawl(e.target.value)}
          className="w-full sm:w-72 h-9 px-3 rounded-md border border-input bg-background text-sm"
        >
          {crawls?.map((c) => (
            <option key={c.id} value={c.id}>{c.id}</option>
          ))}
        </select>
      </div>

      {error && (
        <div className="text-center py-12 text-muted-foreground">
          Failed to load web graph data.
        </div>
      )}

      {isLoading && (
        <div className="space-y-4">
          <div className="grid grid-cols-3 gap-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-24 rounded-xl" />
            ))}
          </div>
          <Skeleton className="h-96 rounded-xl" />
        </div>
      )}

      {graphData && (
        <>
          {/* KPIs */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
            <KPICard
              title="Total Nodes"
              value={fmtNum(graphData.totalNodes)}
              icon={<Network className="h-4 w-4" />}
            />
            <KPICard
              title="Total Edges"
              value={fmtNum(graphData.totalEdges)}
              icon={<Network className="h-4 w-4" />}
            />
            <KPICard
              title="Avg Degree"
              value={graphData.totalNodes > 0
                ? (graphData.totalEdges / graphData.totalNodes).toFixed(1)
                : "--"
              }
              icon={<Network className="h-4 w-4" />}
            />
          </div>

          {/* Domain lookup */}
          <Card className="mb-8">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">Domain Lookup</CardTitle>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleDomainSearch} className="flex gap-2 mb-4">
                <Input
                  value={domainSearch}
                  onChange={(e) => setDomainSearch(e.target.value)}
                  placeholder="Search for a domain's ranking..."
                  className="max-w-md"
                />
                <Button type="submit" size="default">
                  <Search className="h-4 w-4" />
                </Button>
              </form>

              {rankLoading && <Skeleton className="h-16 rounded-lg" />}

              {rankData && searchedDomain && (
                <div className="flex items-center gap-6 py-2">
                  <div>
                    <p className="text-xs text-muted-foreground">Domain</p>
                    <p className="text-sm font-medium">{searchedDomain}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Harmonic Centrality</p>
                    <p className="text-sm font-medium">{rankData.harmonicCentrality?.toFixed(6) ?? "N/A"}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">PageRank</p>
                    <p className="text-sm font-medium">{rankData.pageRank?.toFixed(6) ?? "N/A"}</p>
                  </div>
                </div>
              )}

              {!rankLoading && searchedDomain && !rankData && (
                <p className="text-sm text-muted-foreground">Domain not found in the graph.</p>
              )}
            </CardContent>
          </Card>

          {/* Top 50 table */}
          <Card>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm">Top 50 Domains by Harmonic Centrality</CardTitle>
                <a
                  href="https://data.commoncrawl.org/projects/hyperlinkgraph/"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Button variant="outline" size="sm" className="text-xs">
                    <Download className="h-3 w-3 mr-1.5" />
                    Full Graph Data
                    <ExternalLink className="h-3 w-3 ml-1.5" />
                  </Button>
                </a>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-12">#</TableHead>
                    <TableHead>Domain</TableHead>
                    <TableHead className="w-40 text-right">Harmonic Centrality</TableHead>
                    <TableHead className="w-32 text-right">PageRank</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {topRanked.map((item, i) => (
                    <TableRow key={item.domain}>
                      <TableCell className="text-xs text-muted-foreground font-medium">{i + 1}</TableCell>
                      <TableCell>
                        <a
                          href={`/domain/${item.domain}`}
                          className="text-sm text-primary hover:underline"
                        >
                          {item.domain}
                        </a>
                      </TableCell>
                      <TableCell className="text-right font-mono text-xs">
                        {item.harmonicCentrality.toFixed(6)}
                      </TableCell>
                      <TableCell className="text-right font-mono text-xs">
                        {item.pageRank.toFixed(6)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}
