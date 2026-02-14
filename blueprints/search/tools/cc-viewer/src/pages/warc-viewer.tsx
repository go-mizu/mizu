import { useSearchParams, Link } from "react-router-dom"
import { Eye, Code, FileText, ExternalLink, Clock, FileJson, ScrollText } from "lucide-react"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table"
import { ScrollArea } from "@/components/ui/scroll-area"
import { PageHeader } from "@/components/layout/header"
import { KPICard } from "@/components/kpi-card"
import { StatusBadge } from "@/components/status-badge"
import { useView } from "@/hooks/use-api"
import { fmtBytes, fmtTimestamp } from "@/lib/format"

const MAX_PREVIEW_BYTES = 1_000_000

export default function WARCViewerPage() {
  const [searchParams] = useSearchParams()
  const file = searchParams.get("file") || ""
  const offset = searchParams.get("offset") || ""
  const length = searchParams.get("length") || ""

  const hasParams = file && offset && length
  const { data: record, isLoading, error } = useView(
    hasParams ? file : "",
    hasParams ? parseInt(offset) : 0,
    hasParams ? parseInt(length) : 0
  )

  const waybackURL = record?.targetURI
    ? `https://web.archive.org/web/*/${record.targetURI}`
    : ""

  const watLink = `/wat?file=${encodeURIComponent(file)}&offset=${offset}&length=${length}`
  const wetLink = `/wet?file=${encodeURIComponent(file)}&offset=${offset}&length=${length}`

  const isHTML = record?.contentType?.includes("text/html") || record?.body?.trimStart().startsWith("<")
  const isTruncated = record ? record.body.length >= MAX_PREVIEW_BYTES : false

  const httpHeaders = record?.httpHeaders ? Object.entries(record.httpHeaders) : []

  return (
    <div className="mx-auto max-w-5xl px-6 py-8">
      <PageHeader
        title="WARC Viewer"
        breadcrumbs={[
          { label: "WARC Viewer" },
        ]}
      />

      {!hasParams && (
        <div className="text-center py-16 text-muted-foreground">
          <Eye className="h-12 w-12 mx-auto mb-4 opacity-30" />
          <p>No WARC record specified. Navigate here from a URL lookup or domain page.</p>
        </div>
      )}

      {error && (
        <div className="text-center py-8 text-muted-foreground">
          Failed to load WARC record.
        </div>
      )}

      {isLoading && (
        <div className="space-y-4">
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-24 rounded-xl" />
            ))}
          </div>
          <Skeleton className="h-96 rounded-xl" />
        </div>
      )}

      {record && (
        <>
          {/* Meta KPIs */}
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
            <KPICard
              title="HTTP Status"
              value={String(record.httpStatus)}
              icon={<Eye className="h-4 w-4" />}
            />
            <KPICard
              title="Content Type"
              value={record.contentType?.split(";")[0] || "unknown"}
              icon={<FileText className="h-4 w-4" />}
            />
            <KPICard
              title="Size"
              value={fmtBytes(record.contentLength)}
              icon={<Code className="h-4 w-4" />}
            />
            <KPICard
              title="Date"
              value={record.date ? fmtTimestamp(record.date.replace(/[-T:Z]/g, "").substring(0, 14)) : "--"}
              icon={<Clock className="h-4 w-4" />}
            />
          </div>

          {/* URL and actions */}
          <Card className="mb-6">
            <CardContent className="py-4">
              <p className="text-xs text-muted-foreground mb-1">Target URI</p>
              <p className="text-sm font-mono break-all mb-1">{record.targetURI}</p>
              <div className="flex items-center gap-2 mt-3 text-xs">
                <Badge variant="secondary">{record.warcType}</Badge>
                <Badge variant="secondary">{record.recordID}</Badge>
              </div>
              <div className="flex flex-wrap gap-2 mt-3">
                {record.targetURI && (
                  <a href={record.targetURI} target="_blank" rel="noopener noreferrer">
                    <Button variant="outline" size="sm" className="text-xs">
                      <ExternalLink className="h-3 w-3 mr-1.5" />
                      Visit URL
                    </Button>
                  </a>
                )}
                {waybackURL && (
                  <a href={waybackURL} target="_blank" rel="noopener noreferrer">
                    <Button variant="outline" size="sm" className="text-xs">
                      <Clock className="h-3 w-3 mr-1.5" />
                      Wayback Machine
                    </Button>
                  </a>
                )}
                <Link to={watLink}>
                  <Button variant="outline" size="sm" className="text-xs">
                    <FileJson className="h-3 w-3 mr-1.5" />
                    View WAT
                  </Button>
                </Link>
                <Link to={wetLink}>
                  <Button variant="outline" size="sm" className="text-xs">
                    <ScrollText className="h-3 w-3 mr-1.5" />
                    View WET
                  </Button>
                </Link>
              </div>
            </CardContent>
          </Card>

          {/* Truncation warning */}
          {isTruncated && (
            <div className="mb-4 px-4 py-2 bg-warning/10 border border-warning/30 rounded-lg text-xs text-warning">
              Body was truncated at {fmtBytes(MAX_PREVIEW_BYTES)}. The full record may be larger.
            </div>
          )}

          {/* Content tabs */}
          <Tabs defaultValue={isHTML ? "preview" : "source"}>
            <TabsList className="mb-4">
              {isHTML && <TabsTrigger value="preview">Preview</TabsTrigger>}
              <TabsTrigger value="source">Source</TabsTrigger>
              <TabsTrigger value="headers">Headers</TabsTrigger>
            </TabsList>

            {isHTML && (
              <TabsContent value="preview">
                <Card>
                  <CardContent className="p-0">
                    <iframe
                      srcDoc={record.body}
                      sandbox="allow-same-origin"
                      className="w-full h-[600px] border-0 rounded-lg"
                      title="WARC Preview"
                    />
                  </CardContent>
                </Card>
              </TabsContent>
            )}

            <TabsContent value="source">
              <Card>
                <CardContent className="p-0">
                  <ScrollArea className="h-[600px]">
                    <pre className="p-4 text-xs font-mono whitespace-pre-wrap break-all leading-relaxed">
                      {record.body}
                    </pre>
                  </ScrollArea>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="headers">
              <Card>
                <CardContent className="p-0">
                  {httpHeaders.length > 0 ? (
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead className="w-48">Header</TableHead>
                          <TableHead>Value</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {httpHeaders.map(([key, value]) => (
                          <TableRow key={key}>
                            <TableCell className="font-mono text-xs font-medium">{key}</TableCell>
                            <TableCell className="font-mono text-xs break-all">{value}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  ) : (
                    <div className="text-center py-8 text-muted-foreground text-sm">
                      No HTTP headers available.
                    </div>
                  )}
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </>
      )}
    </div>
  )
}
