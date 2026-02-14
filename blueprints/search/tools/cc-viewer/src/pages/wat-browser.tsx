import { useState, useMemo } from "react"
import { useSearchParams } from "react-router-dom"
import { FileJson, Search } from "lucide-react"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Badge } from "@/components/ui/badge"
import { PageHeader } from "@/components/layout/header"
import { useWAT } from "@/hooks/use-api"
import { truncURL } from "@/lib/format"

export default function WATBrowserPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const file = searchParams.get("file") || ""
  const offset = searchParams.get("offset") || ""
  const length = searchParams.get("length") || ""

  const [fileInput, setFileInput] = useState(file)
  const [offsetInput, setOffsetInput] = useState(offset)
  const [lengthInput, setLengthInput] = useState(length)

  const hasParams = file && offset && length
  const { data, isLoading, error } = useWAT(
    hasParams ? file : "",
    hasParams ? parseInt(offset) : 0,
    hasParams ? parseInt(length) : 0
  )

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSearchParams({
      file: fileInput.trim(),
      offset: offsetInput.trim(),
      length: lengthInput.trim(),
    })
  }

  // WAT body is JSON metadata; parse it
  const metadata = useMemo(() => {
    if (!data?.body) return null
    try { return JSON.parse(data.body) } catch { return null }
  }, [data?.body])

  const warcHeaders = metadata?.["WARC-Header-Metadata"] || metadata?.warcHeaders || {}
  const httpHeaders = metadata?.["HTTP-Response-Metadata"]?.["Headers"] || metadata?.httpHeaders || {}
  const htmlMeta = metadata?.["HTML-Metadata"] || metadata?.htmlMetadata || {}
  const headMetas = htmlMeta?.Head?.Metas || htmlMeta?.metas || []
  const links = htmlMeta?.Head?.Link || htmlMeta?.links || htmlMeta?.Links || []
  const title = htmlMeta?.Head?.Title || htmlMeta?.title || ""

  return (
    <div className="mx-auto max-w-5xl px-6 py-8">
      <PageHeader
        title="WAT Browser"
        breadcrumbs={[
          { label: "WAT Browser" },
        ]}
      />

      {/* Input form */}
      <Card className="mb-6">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm">Locate WAT Record</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="grid grid-cols-1 sm:grid-cols-4 gap-3">
            <div className="sm:col-span-2">
              <label className="text-xs text-muted-foreground mb-1 block">WARC File</label>
              <Input
                value={fileInput}
                onChange={(e) => setFileInput(e.target.value)}
                placeholder="crawl-data/CC-MAIN-.../warc/..."
                className="font-mono text-xs"
              />
            </div>
            <div>
              <label className="text-xs text-muted-foreground mb-1 block">Offset</label>
              <Input
                value={offsetInput}
                onChange={(e) => setOffsetInput(e.target.value)}
                placeholder="0"
                type="number"
              />
            </div>
            <div className="flex items-end gap-2">
              <div className="flex-1">
                <label className="text-xs text-muted-foreground mb-1 block">Length</label>
                <Input
                  value={lengthInput}
                  onChange={(e) => setLengthInput(e.target.value)}
                  placeholder="0"
                  type="number"
                />
              </div>
              <Button type="submit" size="default" className="shrink-0">
                <Search className="h-4 w-4" />
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {!hasParams && !isLoading && (
        <div className="text-center py-16 text-muted-foreground">
          <FileJson className="h-12 w-12 mx-auto mb-4 opacity-30" />
          <p>Enter a WARC file path, offset, and length to view WAT metadata.</p>
        </div>
      )}

      {error && (
        <div className="text-center py-8 text-muted-foreground">
          Failed to load WAT metadata.
        </div>
      )}

      {isLoading && (
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-40 rounded-xl" />
          ))}
        </div>
      )}

      {metadata && (
        <div className="space-y-6">
          {/* WARC Headers */}
          {Object.keys(warcHeaders).length > 0 && (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm">WARC Headers</CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-48">Field</TableHead>
                      <TableHead>Value</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {Object.entries(warcHeaders).map(([key, value]) => (
                      <TableRow key={key}>
                        <TableCell className="font-mono text-xs font-medium">{key}</TableCell>
                        <TableCell className="font-mono text-xs break-all">{String(value)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}

          {/* HTTP Headers */}
          {Object.keys(httpHeaders).length > 0 && (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm">HTTP Headers</CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-48">Header</TableHead>
                      <TableHead>Value</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {Object.entries(httpHeaders).map(([key, value]) => (
                      <TableRow key={key}>
                        <TableCell className="font-mono text-xs font-medium">{key}</TableCell>
                        <TableCell className="font-mono text-xs break-all">{String(value)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}

          {/* HTML Metadata */}
          {(title || headMetas.length > 0) && (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm">HTML Metadata</CardTitle>
              </CardHeader>
              <CardContent>
                {title && (
                  <div className="mb-4">
                    <p className="text-xs text-muted-foreground mb-1">Title</p>
                    <p className="text-sm font-medium">{title}</p>
                  </div>
                )}
                {headMetas.length > 0 && (
                  <div>
                    <p className="text-xs text-muted-foreground mb-2">Meta Tags</p>
                    <div className="space-y-1.5">
                      {headMetas.map((meta: any, i: number) => (
                        <div key={i} className="flex items-start gap-2 text-xs">
                          <Badge variant="secondary" className="text-[10px] shrink-0">
                            {meta.name || meta.property || meta.httpEquiv || "meta"}
                          </Badge>
                          <span className="text-muted-foreground break-all">{meta.content || JSON.stringify(meta)}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Links */}
          {links.length > 0 && (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm">Links ({links.length})</CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <ScrollArea className="max-h-96">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>URL</TableHead>
                        <TableHead className="w-20">Rel</TableHead>
                        <TableHead className="w-20">Type</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {links.map((link: any, i: number) => (
                        <TableRow key={i}>
                          <TableCell className="text-xs font-mono truncate max-w-md">
                            {truncURL(link.url || link.href || "", 80)}
                          </TableCell>
                          <TableCell className="text-xs text-muted-foreground">{link.rel || "--"}</TableCell>
                          <TableCell className="text-xs text-muted-foreground">{link.type || "--"}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </ScrollArea>
              </CardContent>
            </Card>
          )}

          {/* Raw JSON fallback */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">Raw Metadata</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <ScrollArea className="h-80">
                <pre className="p-4 text-xs font-mono whitespace-pre-wrap break-all">
                  {JSON.stringify(metadata, null, 2)}
                </pre>
              </ScrollArea>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  )
}
