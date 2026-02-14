import { useState } from "react"
import { useNavigate, Link } from "react-router-dom"
import { Globe, Search, Database, BarChart3, GitGraph, Newspaper, ArrowRight } from "lucide-react"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { KPICard } from "@/components/kpi-card"
import { useCrawls } from "@/hooks/use-api"
import { crawlToDate } from "@/lib/format"

const QUICK_LINKS = [
  { label: "Browse Domains", href: "/domains", icon: Globe, desc: "Explore the most-crawled domains" },
  { label: "Statistics", href: "/stats", icon: BarChart3, desc: "Charts and analytics across crawls" },
  { label: "Web Graph", href: "/graph", icon: GitGraph, desc: "Link structure and domain rankings" },
  { label: "News Archive", href: "/news", icon: Newspaper, desc: "Browse CC-NEWS captures" },
]

export default function DashboardPage() {
  const [query, setQuery] = useState("")
  const navigate = useNavigate()
  const { data: crawls, isLoading, error } = useCrawls()

  function handleSearch(e: React.FormEvent) {
    e.preventDefault()
    const q = query.trim()
    if (!q) return
    navigate(`/search?q=${encodeURIComponent(q)}`)
  }

  const latestCrawl = crawls?.[0]
  const recentCrawls = crawls?.slice(0, 4)

  return (
    <div>
      {/* Hero search */}
      <div className="flex flex-col items-center gap-6 py-16">
        <div className="flex items-center gap-3">
          <Database className="h-8 w-8 text-primary" />
          <h1 className="text-3xl font-semibold tracking-tight">Common Crawl Viewer</h1>
        </div>
        <p className="text-muted-foreground text-center max-w-lg">
          Search and explore billions of web pages from the Common Crawl open dataset.
        </p>
        <form onSubmit={handleSearch} className="flex w-full max-w-xl gap-2">
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Enter a URL, domain, or search term..."
            className="h-11 text-base"
          />
          <Button type="submit" className="h-11 px-6">
            <Search className="h-4 w-4 mr-2" />
            Search
          </Button>
        </form>
      </div>

      {/* KPIs */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-12">
        {isLoading ? (
          <>
            <Skeleton className="h-24 rounded-xl" />
            <Skeleton className="h-24 rounded-xl" />
            <Skeleton className="h-24 rounded-xl" />
          </>
        ) : error ? (
          <div className="col-span-3 text-center text-muted-foreground py-8">
            Failed to load crawl data.
          </div>
        ) : (
          <>
            <KPICard
              title="Latest Crawl"
              value={latestCrawl ? crawlToDate(latestCrawl.id) : "--"}
              icon={<Database className="h-4 w-4" />}
            />
            <KPICard
              title="Total Crawls"
              value={crawls ? String(crawls.length) : "--"}
              icon={<Globe className="h-4 w-4" />}
            />
            <KPICard
              title="Total Pages"
              value="300B+"
              icon={<BarChart3 className="h-4 w-4" />}
            />
          </>
        )}
      </div>

      {/* Recent crawls */}
      <div className="mb-12">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-medium">Recent Crawls</h2>
          <Link to="/crawls" className="text-sm text-primary hover:underline flex items-center gap-1">
            View all <ArrowRight className="h-3 w-3" />
          </Link>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {isLoading ? (
            Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-28 rounded-xl" />
            ))
          ) : recentCrawls?.map((crawl) => (
            <Link key={crawl.id} to={`/crawl/${encodeURIComponent(crawl.id)}`}>
              <Card className="hover:border-primary/40 transition-colors cursor-pointer h-full">
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium">{crawl.id}</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-xs text-muted-foreground">{crawlToDate(crawl.id)}</p>
                  <p className="text-xs text-muted-foreground mt-1">{crawl.name}</p>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      </div>

      {/* Quick links */}
      <div>
        <h2 className="text-lg font-medium mb-4">Explore</h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {QUICK_LINKS.map((link) => (
            <Link key={link.href} to={link.href}>
              <Card className="hover:border-primary/40 transition-colors cursor-pointer h-full">
                <CardContent className="pt-5">
                  <link.icon className="h-5 w-5 text-primary mb-3" />
                  <p className="font-medium text-sm mb-1">{link.label}</p>
                  <p className="text-xs text-muted-foreground">{link.desc}</p>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      </div>
    </div>
  )
}
