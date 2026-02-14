import { useState, useCallback } from "react"
import { useQuery } from "@tanstack/react-query"
import { ArrowRight, Copy, Check, ExternalLink, AlertCircle, Globe } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { LanguageSelect } from "@/components/language-select"
import { fetchLanguages } from "@/api/client"

const LS_KEY = "page-translator-lang"

function getStoredLang(): string {
  try {
    return localStorage.getItem(LS_KEY) || "vi"
  } catch {
    return "vi"
  }
}

function storeLang(code: string) {
  try {
    localStorage.setItem(LS_KEY, code)
  } catch {
    // ignore
  }
}

export function PageTranslator() {
  const [url, setUrl] = useState("")
  const [targetLang, setTargetLang] = useState(getStoredLang)
  const [translatedUrl, setTranslatedUrl] = useState<string | null>(null)
  const [iframeLoading, setIframeLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const { data: languages = [] } = useQuery({
    queryKey: ["languages"],
    queryFn: fetchLanguages,
  })

  // Filter out auto-detect for page translation
  const filteredLanguages = languages.filter((l) => l.code !== "auto")

  const handleLangChange = useCallback((code: string) => {
    setTargetLang(code)
    storeLang(code)
  }, [])

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault()
      setError(null)

      const trimmed = url.trim()
      if (!trimmed) {
        setError("Please enter a URL")
        return
      }
      if (!trimmed.startsWith("http://") && !trimmed.startsWith("https://")) {
        setError("URL must start with http:// or https://")
        return
      }

      try {
        new URL(trimmed)
      } catch {
        setError("Please enter a valid URL")
        return
      }

      const path = `/page/${targetLang}?url=${encodeURIComponent(trimmed)}`
      setTranslatedUrl(path)
      setIframeLoading(true)
    },
    [url, targetLang]
  )

  const displayUrl = translatedUrl

  const handleCopy = useCallback(async () => {
    if (!translatedUrl) return
    const fullUrl = `${window.location.origin}${translatedUrl}`
    await navigator.clipboard.writeText(fullUrl)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }, [translatedUrl])

  const handleOpenNewTab = useCallback(() => {
    if (!translatedUrl) return
    window.open(translatedUrl, "_blank")
  }, [translatedUrl])

  return (
    <div className="space-y-4">
      {/* Input form */}
      <form onSubmit={handleSubmit} className="space-y-3">
        <label className="text-sm font-medium text-muted-foreground">
          Enter a URL to translate
        </label>
        <div className="flex flex-col sm:flex-row gap-2">
          <div className="flex-1">
            <input
              type="text"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://example.com"
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            />
          </div>
          <div className="w-full sm:w-[180px]">
            <LanguageSelect
              languages={filteredLanguages}
              value={targetLang}
              onChange={handleLangChange}
            />
          </div>
          <Button type="submit" className="shrink-0">
            <span>Translate</span>
            <ArrowRight className="h-4 w-4" />
          </Button>
        </div>
      </form>

      {/* Error */}
      {error && (
        <div className="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          <AlertCircle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      {/* Translated URL bar */}
      {translatedUrl && (
        <div className="flex flex-col sm:flex-row items-start sm:items-center gap-2 rounded-lg border border-border bg-secondary/30 px-4 py-3">
          <span className="text-sm text-muted-foreground shrink-0">Translated URL:</span>
          <code className="flex-1 text-sm break-all">{displayUrl}</code>
          <div className="flex gap-1">
            <Button variant="outline" size="sm" onClick={handleOpenNewTab}>
              <ExternalLink className="h-3.5 w-3.5 mr-1" />
              Open
            </Button>
            <Button variant="outline" size="sm" onClick={handleCopy}>
              {copied ? (
                <>
                  <Check className="h-3.5 w-3.5 mr-1" />
                  Copied
                </>
              ) : (
                <>
                  <Copy className="h-3.5 w-3.5 mr-1" />
                  Copy
                </>
              )}
            </Button>
          </div>
        </div>
      )}

      {/* Iframe */}
      {translatedUrl && (
        <div className="relative rounded-lg border border-border overflow-hidden">
          {iframeLoading && (
            <div className="absolute inset-0 z-10 bg-background/80 p-6 space-y-4">
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Globe className="h-4 w-4 animate-spin" />
                Translating page...
              </div>
              <Skeleton className="h-6 w-2/3" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-5/6" />
              <Skeleton className="h-4 w-4/6" />
              <Skeleton className="h-32 w-full" />
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-full" />
            </div>
          )}
          <iframe
            src={translatedUrl}
            title="Translated page"
            className="w-full border-0"
            style={{ minHeight: "500px", height: "70vh" }}
            onLoad={() => setIframeLoading(false)}
          />
        </div>
      )}

      {/* Empty state */}
      {!translatedUrl && (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-16 text-muted-foreground">
          <Globe className="h-10 w-10 mb-3 opacity-40" />
          <p className="text-sm">Enter a URL above to see the translated page here</p>
        </div>
      )}
    </div>
  )
}
