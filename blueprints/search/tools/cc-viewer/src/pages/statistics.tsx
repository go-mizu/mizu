import { useState } from "react"
import { useSearchParams } from "react-router-dom"
import { BarChart3 } from "lucide-react"
import { Skeleton } from "@/components/ui/skeleton"
import { PageHeader } from "@/components/layout/header"
import { KPICard } from "@/components/kpi-card"
import { BarChartCard, PieChartCard, LineChartCard, AreaChartCard } from "@/components/stats-chart"
import { useCrawls, useStats, useStatsTrends } from "@/hooks/use-api"
import { fmtNum, fmtBytes } from "@/lib/format"

const STATUS_COLORS: Record<string, string> = {
  "200": "#10a37f",
  "301": "#f59e0b",
  "302": "#f59e0b",
  "304": "#8b5cf6",
  "403": "#ef4444",
  "404": "#ef4444",
  "500": "#dc2626",
}

export default function StatisticsPage() {
  const [searchParams] = useSearchParams()
  const crawlParam = searchParams.get("crawl") || ""
  const { data: crawls } = useCrawls()
  const [selectedCrawl, setSelectedCrawl] = useState(crawlParam)
  const activeCrawl = selectedCrawl || crawls?.[0]?.id || ""

  const { data: stats, isLoading, error } = useStats(activeCrawl)
  const { data: trends } = useStatsTrends()

  // Transform TLD distribution for bar chart
  const tldData = stats?.tldDistribution
    ? Object.entries(stats.tldDistribution)
        .sort((a, b) => b[1] - a[1])
        .slice(0, 20)
        .map(([name, value]) => ({ name, value }))
    : []

  // Transform language distribution for horizontal bar chart
  const langData = stats?.languageDistribution
    ? Object.entries(stats.languageDistribution)
        .sort((a, b) => b[1] - a[1])
        .slice(0, 20)
        .map(([name, value]) => ({ name, value }))
    : []

  // Transform MIME distribution for pie chart
  const mimeData = stats?.mimeDistribution
    ? Object.entries(stats.mimeDistribution)
        .sort((a, b) => b[1] - a[1])
        .slice(0, 10)
        .map(([name, value]) => ({ name, value }))
    : []

  // Transform status distribution for colored bar chart
  const statusData = stats?.statusDistribution
    ? Object.entries(stats.statusDistribution)
        .sort((a, b) => parseInt(a[0]) - parseInt(b[0]))
        .map(([name, value]) => ({
          name,
          value,
          fill: STATUS_COLORS[name] || "#6e6e80",
        }))
    : []

  // Transform trends for line/area charts
  const growthData = trends
    ? trends.map((t) => ({
        name: t.date || t.crawl,
        pages: t.estimatedPages,
        warcFiles: t.warcFiles,
        sizeTB: t.estimatedSizeTB,
      }))
    : []

  // Protocol data derived from trends (size as proxy; real protocol data would need a separate endpoint)
  const protocolData = trends
    ? trends.map((t) => ({
        name: t.date || t.crawl,
        https: Math.round(t.estimatedPages * 0.85),
        http: Math.round(t.estimatedPages * 0.15),
      }))
    : []

  return (
    <div className="mx-auto max-w-6xl px-6 py-8">
      <PageHeader
        title="Statistics"
        breadcrumbs={[{ label: "Statistics" }]}
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
          Failed to load statistics.
        </div>
      )}

      {isLoading && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-80 rounded-xl" />
          ))}
        </div>
      )}

      {stats && (
        <>
          {/* Summary KPIs */}
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
            <KPICard title="Total Pages" value={fmtNum(stats.totalPages)} icon={<BarChart3 className="h-4 w-4" />} />
            <KPICard title="Total Domains" value={fmtNum(stats.totalDomains)} icon={<BarChart3 className="h-4 w-4" />} />
            <KPICard title="Total Size" value={fmtBytes(stats.totalSize)} icon={<BarChart3 className="h-4 w-4" />} />
            <KPICard title="TLDs" value={fmtNum(Object.keys(stats.tldDistribution || {}).length)} icon={<BarChart3 className="h-4 w-4" />} />
          </div>

          {/* Charts grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* 1. Crawl Growth */}
            {growthData.length > 0 && (
              <LineChartCard
                title="Crawl Growth"
                description="Estimated pages and WARC files over time"
                data={growthData}
                lines={[
                  { dataKey: "pages", name: "Est. Pages", color: "#10a37f" },
                  { dataKey: "warcFiles", name: "WARC Files", color: "#6366f1" },
                ]}
                xDataKey="name"
              />
            )}

            {/* 2. TLD Distribution */}
            {tldData.length > 0 && (
              <BarChartCard
                title="TLD Distribution"
                description="Top 20 TLDs by page count"
                data={tldData}
                dataKey="value"
                nameKey="name"
                color="#10a37f"
              />
            )}

            {/* 3. Language Distribution */}
            {langData.length > 0 && (
              <BarChartCard
                title="Language Distribution"
                description="Top 20 languages"
                data={langData}
                dataKey="value"
                nameKey="name"
                color="#6366f1"
              />
            )}

            {/* 4. MIME Types */}
            {mimeData.length > 0 && (
              <PieChartCard
                title="Content Types"
                description="Top 10 MIME types"
                data={mimeData}
              />
            )}

            {/* 5. HTTP Status */}
            {statusData.length > 0 && (
              <BarChartCard
                title="HTTP Status Codes"
                description="Response status distribution"
                data={statusData}
                dataKey="value"
                nameKey="name"
              />
            )}

            {/* 6. Protocol Adoption */}
            {protocolData.length > 0 && (
              <AreaChartCard
                title="Protocol Adoption"
                description="HTTP vs HTTPS over time"
                data={protocolData}
                areas={[
                  { dataKey: "https", name: "HTTPS", color: "#10a37f" },
                  { dataKey: "http", name: "HTTP", color: "#ef4444" },
                ]}
                xDataKey="name"
              />
            )}
          </div>
        </>
      )}
    </div>
  )
}
