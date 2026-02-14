import { useState } from "react"
import { useSearchParams } from "react-router-dom"
import { ScrollText, Search, Hash, FileText } from "lucide-react"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Badge } from "@/components/ui/badge"
import { PageHeader } from "@/components/layout/header"
import { KPICard } from "@/components/kpi-card"
import { useWET } from "@/hooks/use-api"
import { fmtNum } from "@/lib/format"

export default function WETBrowserPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const file = searchParams.get("file") || ""
  const offset = searchParams.get("offset") || ""
  const length = searchParams.get("length") || ""

  const [fileInput, setFileInput] = useState(file)
  const [offsetInput, setOffsetInput] = useState(offset)
  const [lengthInput, setLengthInput] = useState(length)

  const hasParams = file && offset && length
  const { data, isLoading, error } = useWET(
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

  // WET body is the extracted plaintext
  const text = data?.body || ""
  const wordCount = text ? text.trim().split(/\s+/).filter(Boolean).length : 0
  const charCount = text ? text.length : 0
  const detectedLang = data?.httpHeaders?.["Content-Language"] || ""

  return (
    <div className="mx-auto max-w-5xl px-6 py-8">
      <PageHeader
        title="WET Browser"
        breadcrumbs={[
          { label: "WET Browser" },
        ]}
      />

      {/* Input form */}
      <Card className="mb-6">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm">Locate WET Record</CardTitle>
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
          <ScrollText className="h-12 w-12 mx-auto mb-4 opacity-30" />
          <p>Enter a WARC file path, offset, and length to view extracted plaintext.</p>
        </div>
      )}

      {error && (
        <div className="text-center py-8 text-muted-foreground">
          Failed to load WET text.
        </div>
      )}

      {isLoading && (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <Skeleton className="h-24 rounded-xl" />
            <Skeleton className="h-24 rounded-xl" />
          </div>
          <Skeleton className="h-96 rounded-xl" />
        </div>
      )}

      {data && (
        <>
          {/* Stats */}
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-4 mb-6">
            <KPICard
              title="Words"
              value={fmtNum(wordCount)}
              icon={<FileText className="h-4 w-4" />}
            />
            <KPICard
              title="Characters"
              value={fmtNum(charCount)}
              icon={<Hash className="h-4 w-4" />}
            />
            {detectedLang && (
              <KPICard
                title="Language"
                value={detectedLang}
                icon={<ScrollText className="h-4 w-4" />}
              />
            )}
          </div>

          {/* Text content */}
          <Card>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm">Extracted Text</CardTitle>
                <Badge variant="secondary" className="text-xs">
                  {fmtNum(wordCount)} words
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              <ScrollArea className="h-[500px]">
                <div className="p-5">
                  <p className="text-sm leading-relaxed whitespace-pre-wrap">{text}</p>
                </div>
              </ScrollArea>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}
